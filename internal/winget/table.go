package winget

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Row est une ligne de tableau winget, découpée en colonnes.
type Row []string

// Col rend la colonne d'indice i (vide si elle n'existe pas).
func (r Row) Col(i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return r[i]
}

// winget n'a pas de sortie machine : `search`, `list` et `upgrade` ne parlent qu'en
// tableaux de largeur fixe, dont les en-têtes sont **traduits** (« Nom », « Version »,
// « Disponible »…). On ne peut donc pas reconnaître une colonne à son titre — mais leur
// *ordre*, lui, est fixe : nom, identifiant, version, puis les colonnes propres à la
// commande. C'est cet ordre que le paquet exploite.
const (
	colName = 0
	colID   = 1
	colVer  = 2
)

// ParseTable lit un tableau winget : la ligne de tirets sépare l'en-tête — qui donne la
// position des colonnes — des données.
//
// Les lignes de résumé que winget ajoute après le tableau (« 48 mises à niveau
// disponibles. ») sont écartées : ce sont des phrases, elles chevauchent donc les
// frontières de colonnes, là où une vraie ligne les respecte toutes (cf. aligne).
func ParseTable(out []byte) []Row {
	lignes := nettoyer(out)

	sep := -1
	for i, l := range lignes {
		if estSeparateur(l) {
			sep = i
			break
		}
	}
	if sep < 1 {
		return nil // pas de tableau : winget a répondu autre chose (erreur, liste vide)
	}

	debuts := colonnes(lignes[sep-1])
	if len(debuts) < 2 {
		return nil
	}

	var rows []Row
	for _, l := range lignes[sep+1:] {
		if r, ok := decouper(l, debuts); ok {
			rows = append(rows, r)
		}
	}
	return rows
}

// nettoyer rend les lignes utiles de la sortie : sans retour chariot, sans séquences
// ANSI, et sans les états successifs d'une animation de progression (winget les sépare
// par des `\r` ; seul le dernier compte).
func nettoyer(out []byte) []string {
	var lignes []string
	for _, l := range strings.Split(string(out), "\n") {
		// Le retour chariot de fin est celui du CRLF de Windows ; ceux qui restent
		// séparent les états successifs d'une animation, et seul le dernier compte.
		l = strings.TrimSuffix(l, "\r")
		if i := strings.LastIndexByte(l, '\r'); i >= 0 {
			l = l[i+1:]
		}
		l = sansANSI(l)
		if strings.TrimSpace(l) != "" {
			lignes = append(lignes, l)
		}
	}
	return lignes
}

// sansANSI retire les séquences d'échappement (couleurs, hyperliens) que winget émet
// quand il se croit devant un terminal. Elles ne comptent pour aucune colonne : les
// laisser décalerait tout le découpage.
func sansANSI(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] != 0x1b {
			b.WriteRune(runes[i])
			continue
		}
		i++
		if i >= len(runes) {
			break
		}
		switch runes[i] {
		case '[': // CSI … <lettre>
			for i++; i < len(runes) && (runes[i] < '@' || runes[i] > '~'); i++ {
			}
		case ']': // OSC … BEL ou ST
			for i++; i < len(runes); i++ {
				if runes[i] == 7 {
					break
				}
				if runes[i] == 0x1b && i+1 < len(runes) && runes[i+1] == '\\' {
					i++
					break
				}
			}
		}
	}
	return b.String()
}

// estSeparateur reconnaît la ligne de tirets sous l'en-tête.
func estSeparateur(l string) bool {
	l = strings.TrimSpace(l)
	if len(l) < 10 {
		return false
	}
	return strings.Trim(l, "-") == ""
}

// colonnes rend la position (en cellules d'affichage) où commence chaque colonne, lue
// sur l'en-tête : une colonne commence après au moins deux espaces — un seul peut
// séparer les mots d'un titre traduit.
func colonnes(entete string) []int {
	var debuts []int
	cellule, espaces := 0, 2 // le début de ligne ouvre la première colonne
	for _, r := range entete {
		if r == ' ' {
			espaces++
		} else {
			if espaces >= 2 {
				debuts = append(debuts, cellule)
			}
			espaces = 0
		}
		cellule += runewidth.RuneWidth(r)
	}
	return debuts
}

// decouper découpe une ligne de données aux frontières données. Le second retour est
// faux si la ligne n'est pas un enregistrement : une phrase qui chevauche une frontière
// (résumé de fin de tableau) n'en est pas un.
func decouper(l string, debuts []int) (Row, bool) {
	runes := []rune(l)
	cellules := make([]int, len(runes)+1) // cellule où commence chaque rune
	c := 0
	for i, r := range runes {
		cellules[i] = c
		c += runewidth.RuneWidth(r)
	}
	cellules[len(runes)] = c
	largeur := c

	// Une ligne de données remplit ses colonnes puis les complète d'espaces : la
	// cellule qui précède chaque frontière est donc vide. Une phrase, non.
	for _, d := range debuts[1:] {
		if d == 0 || d > largeur {
			continue
		}
		if r, ok := runeEnCellule(runes, cellules, d-1); ok && r != ' ' {
			return nil, false
		}
	}

	row := make(Row, len(debuts))
	for i, d := range debuts {
		fin := largeur
		if i+1 < len(debuts) {
			fin = debuts[i+1]
		}
		row[i] = strings.TrimSpace(tranche(runes, cellules, d, fin))
	}
	if row[colName] == "" {
		return nil, false
	}
	return row, true
}

// runeEnCellule rend la rune qui occupe une cellule donnée.
func runeEnCellule(runes []rune, cellules []int, cellule int) (rune, bool) {
	for i, r := range runes {
		if cellules[i] <= cellule && cellule < cellules[i+1] {
			return r, true
		}
	}
	return 0, false
}

// tranche rend le texte compris entre deux cellules.
func tranche(runes []rune, cellules []int, debut, fin int) string {
	var b strings.Builder
	for i, r := range runes {
		if cellules[i] >= debut && cellules[i] < fin {
			b.WriteRune(r)
		}
	}
	return b.String()
}
