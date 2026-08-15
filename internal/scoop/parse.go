// Ce fichier écrit les parsers de scoop pour les trois verbes normalisés qu'il expose en
// Native — list, search, source (cf. verbs.go ; outdated, lui, est Direct et n'a pas
// besoin d'un parser).
//
// NON VÉRIFIÉ contre un vrai scoop, comme le reste de ce paquet (cf. tâche 1b) : ce Mac
// n'a ni scoop ni PowerShell à interroger. Les gabarits ci-dessous sont écrits contre la
// documentation et le code source publics de scoop — Format-Table de PowerShell pour
// `list`/`bucket list`, la sortie texte de `search` — et les jeux d'essai de testdata/
// sont composés à la main dans le même esprit, avec le même avertissement. À vérifier et
// corriger dès qu'une vraie installation de scoop est accessible.
package scoop

import (
	"regexp"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// colonnes découpe une ligne de tableau scoop (Format-Table de PowerShell) sur au moins
// deux espaces. Contrairement à winget (cf. internal/winget/table.go), scoop n'aligne pas
// ses colonnes à une position fixe d'une commande à l'autre — seule la largeur du champ le
// plus long de chaque colonne compte —, mais l'espacement entre deux colonnes est toujours
// d'au moins deux espaces, jamais un identifiant ou une version ne s'écrit avec deux
// espaces internes.
var colonnes = regexp.MustCompile(`\s{2,}`)

func decouper(ligne string) []string {
	ligne = strings.TrimRight(ligne, " \t")
	if ligne == "" {
		return nil
	}
	return colonnes.Split(ligne, -1)
}

// estSeparateur reconnaît la ligne de tirets que Format-Table pose entre l'en-tête et les
// données (« ----  -------  ------ »).
func estSeparateur(ligne string) bool {
	t := strings.TrimSpace(ligne)
	return t != "" && strings.Trim(t, "- ") == ""
}

// commentaire reconnaît les lignes « # … » que portent les jeux d'essai de testdata/ pour
// se dire non vérifiés : scoop n'émet jamais de ligne commençant par #, ce préfixe est
// donc sans risque de collision avec une vraie sortie.
func commentaire(ligne string) bool { return strings.HasPrefix(ligne, "#") }

// parseList lit « scoop list » : un en-tête (Name, Version, Source, Updated, Info), une
// ligne de tirets, puis une ligne par application installée. Tout ce qui précède la ligne
// de tirets (le titre « Installed apps: », l'en-tête lui-même) est ignoré : lire les noms
// de colonnes ne sert à rien tant que scoop les traduit pas (contrairement à winget), donc
// seule la position compte.
func parseList(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	dansTable := false
	for _, ligne := range pm.SplitLines(out) {
		if commentaire(ligne) {
			continue
		}
		if estSeparateur(ligne) {
			dansTable = true
			continue
		}
		if !dansTable {
			continue
		}
		champs := decouper(ligne)
		if len(champs) < 2 {
			continue
		}
		source := ""
		if len(champs) >= 3 {
			source = champs[2]
		}
		badge := pm.BadgeScoop
		if source != "" && source != "main" {
			badge = pm.BadgeOther
		}
		rows = append(rows, pm.Package{
			Name:    champs[0],
			Version: champs[1],
			Kind:    badge,
			Source:  source,
		})
	}
	return rows, nil
}

// ligneBucket reconnaît l'en-tête de section que `scoop search` imprime avant les
// résultats d'un bucket : « 'main' bucket: ».
var ligneBucket = regexp.MustCompile(`^'([^']+)' bucket:$`)

// ligneResultat reconnaît une ligne de résultat : « nom (version) », éventuellement suivie
// d'un commentaire que scoop ajoute pour un nom trouvé par binaire plutôt que par manifeste
// (« --> includes … ») — capturé mais ignoré, ce n'est pas une donnée normalisée.
var ligneResultat = regexp.MustCompile(`^(\S+)\s+\(([^)]+)\)`)

// parseSearch lit « scoop search <requête> » : des sections par bucket, chacune suivie
// d'une ligne par correspondance.
func parseSearch(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	bucket := ""
	for _, ligne := range pm.SplitLines(out) {
		if commentaire(ligne) {
			continue
		}
		if m := ligneBucket.FindStringSubmatch(ligne); m != nil {
			bucket = m[1]
			continue
		}
		m := ligneResultat.FindStringSubmatch(ligne)
		if m == nil {
			continue // « Results from local buckets… » et autres phrases d'habillage
		}
		badge := pm.BadgeScoop
		if bucket != "" && bucket != "main" {
			badge = pm.BadgeOther
		}
		rows = append(rows, pm.Package{Name: m[1], Version: m[2], Kind: badge, Source: bucket})
	}
	return rows, nil
}

// parseSource lit « scoop bucket list » : même gabarit de tableau que list, mais Name est
// le nom du bucket et la deuxième colonne (Source) est l'URL du dépôt — il n'y a pas de
// version à porter.
func parseSource(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	dansTable := false
	for _, ligne := range pm.SplitLines(out) {
		if commentaire(ligne) {
			continue
		}
		if estSeparateur(ligne) {
			dansTable = true
			continue
		}
		if !dansTable {
			continue
		}
		champs := decouper(ligne)
		if len(champs) < 1 || champs[0] == "" {
			continue
		}
		source := ""
		if len(champs) >= 2 {
			source = champs[1]
		}
		rows = append(rows, pm.Package{Name: champs[0], Source: source})
	}
	return rows, nil
}
