// Package config porte la configuration de jigger : le fichier, sa lecture, et l'arbitrage
// entre l'environnement, le fichier et les défauts.
//
// C'est la SEULE implémentation de cette préséance. Les greffons ne lisent pas le fichier :
// ils demandent au binaire de leur dicter les valeurs (`jigger config --export`), ce qui
// évite trois analyseurs qui divergeraient — voir ADR-0003.
package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Provenance dit d'où vient une valeur. L'écran l'affiche : sans elle, il montrerait une
// valeur choisie pendant que la machine en applique une autre (ADR-0003).
type Provenance int

const (
	// DuDefaut : personne n'a rien dit.
	DuDefaut Provenance = iota
	// DuFichier : le fichier de configuration l'a fixée.
	DuFichier
	// DeLEnvironnement : une variable d'environnement l'emporte sur le fichier.
	DeLEnvironnement
)

func (p Provenance) String() string {
	switch p {
	case DeLEnvironnement:
		return "environnement"
	case DuFichier:
		return "fichier"
	default:
		return "défaut"
	}
}

// Resoudre arbitre entre les trois sources, dans l'ordre environnement > fichier > défaut.
//
// Une variable d'environnement **vide** compte comme absente : c'est la convention de tout
// le reste du projet (la résolution de langue le fait déjà), et c'est ce qui permet à
// `JIGGER_LANG= jigger …` de neutraliser un réglage sans le supprimer.
//
// Une valeur **vide dans le fichier**, en revanche, est un choix délibéré : on l'a écrite.
// Elle l'emporte donc sur le défaut.
func Resoudre(env string, fichier *string, defaut string) (string, Provenance) {
	if env != "" {
		return env, DeLEnvironnement
	}
	if fichier != nil {
		return *fichier, DuFichier
	}
	return defaut, DuDefaut
}

// Fichier est le contenu analysé du fichier de configuration : les clés qui y figurent,
// et rien d'autre. Une clé absente n'est pas une clé vide — d'où la distinction que
// Resoudre exploite.
type Fichier struct {
	valeurs map[string]string
	ordre   []string // ordre d'apparition, pour réécrire sans mélanger
}

// Nouveau rend un fichier vide.
func Nouveau() *Fichier {
	return &Fichier{valeurs: map[string]string{}}
}

// Valeur rend la valeur d'une clé, ou nil si elle n'y figure pas.
func (f *Fichier) Valeur(cle string) *string {
	if f == nil {
		return nil
	}
	if v, ok := f.valeurs[cle]; ok {
		return &v
	}
	return nil
}

// Poser fixe une clé. Une clé nouvelle s'ajoute à la fin ; une clé existante garde sa place.
func (f *Fichier) Poser(cle, valeur string) {
	if _, existe := f.valeurs[cle]; !existe {
		f.ordre = append(f.ordre, cle)
	}
	f.valeurs[cle] = valeur
}

// Retirer supprime une clé, qui reprend alors sa valeur par défaut.
func (f *Fichier) Retirer(cle string) {
	if _, existe := f.valeurs[cle]; !existe {
		return
	}
	delete(f.valeurs, cle)
	for i, c := range f.ordre {
		if c == cle {
			f.ordre = append(f.ordre[:i], f.ordre[i+1:]...)
			break
		}
	}
}

// Cles rend les clés dans leur ordre d'apparition.
func (f *Fichier) Cles() []string {
	if f == nil {
		return nil
	}
	return append([]string(nil), f.ordre...)
}

// Chemin rend l'emplacement du fichier : <config utilisateur>/jigger/config —
// ~/Library/Application Support/jigger/config sur macOS, %APPDATA%\jigger\config sous
// Windows, ~/.config/jigger/config ailleurs. Symétrique du cache (pm.CacheDir).
//
// JIGGER_CONFIG le remplace entièrement, ce qui rend les tests possibles sans toucher au
// répertoire de l'utilisateur.
func Chemin() (string, error) {
	if p := os.Getenv("JIGGER_CONFIG"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "jigger", "config"), nil
}

// Charger lit le fichier. Un fichier absent n'est pas une erreur : c'est le cas ordinaire
// tant que personne n'a rien réglé.
func Charger() (*Fichier, error) {
	chemin, err := Chemin()
	if err != nil {
		return Nouveau(), err
	}
	f, err := os.Open(chemin)
	if err != nil {
		if os.IsNotExist(err) {
			return Nouveau(), nil
		}
		return Nouveau(), err
	}
	defer f.Close()
	return Analyser(f)
}

// Analyser lit un flux au format « clé = valeur ».
//
// Les lignes vides et celles qui commencent par # sont ignorées. Les espaces autour du nom
// et de la valeur sont retirés. Une ligne sans « = » est ignorée plutôt que de faire
// échouer la lecture : un fichier à moitié écrit ne doit pas empêcher un shell de s'ouvrir
// (spec §risques).
func Analyser(r io.Reader) (*Fichier, error) {
	fic := Nouveau()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		ligne := strings.TrimSpace(sc.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		nom, valeur, ok := strings.Cut(ligne, "=")
		if !ok {
			continue
		}
		fic.Poser(strings.TrimSpace(nom), decoter(strings.TrimSpace(valeur)))
	}
	return fic, sc.Err()
}

// Ecrire enregistre le fichier, en créant son répertoire au besoin. L'écriture passe par
// un fichier temporaire renommé : une interruption ne laisse jamais un fichier à moitié
// écrit, que le prochain shell lirait.
func (f *Fichier) Ecrire() error {
	chemin, err := Chemin()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# Configuration de jigger — écrite par « jigger config ».\n")
	b.WriteString("# L'environnement l'emporte sur ce fichier (ADR-0003).\n\n")
	for _, cle := range f.ordre {
		fmt.Fprintf(&b, "%s = %s\n", cle, coter(f.valeurs[cle]))
	}

	tmp := chemin + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, chemin)
}

// coter entoure la valeur de guillemets quand elle contient des espaces significatifs en
// bord de chaîne, ou des guillemets. Sans cela, « key = ^ » — Ctrl-Espace, une valeur
// documentée — se relirait « ^ » sans son espace, et la touche cesserait de fonctionner.
func coter(v string) string {
	if v == strings.TrimSpace(v) && !strings.HasPrefix(v, `"`) {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
}

// decoter est l'inverse : une valeur entre guillemets garde son contenu au caractère près.
func decoter(v string) string {
	if len(v) < 2 || !strings.HasPrefix(v, `"`) || !strings.HasSuffix(v, `"`) {
		return v
	}
	return strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
}

// Triees rend les clés par ordre alphabétique — pour les affichages qui veulent un ordre
// stable plutôt que l'ordre du fichier.
func (f *Fichier) Triees() []string {
	cles := f.Cles()
	sort.Strings(cles)
	return cles
}
