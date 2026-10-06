// Les chaînes que jigger affiche à ses utilisateurs, dans la langue de leur système.
//
// Un catalogue unique (catalogue.go) porte toutes les langues côte à côte : la dérive de
// clés — un fichier de traduction qui oublie une entrée — devient impossible par
// construction. Le tableau est de taille fixe : le jour où une troisième langue arrive,
// nbLangues change et le compilateur désigne chaque ligne à compléter.
//
// Deux replis, et aucune page blanche : une clé absente est rendue telle quelle (donc
// visible à l'œil dès le premier essai), une traduction vide se replie sur l'anglais.
package i18n

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Langue est l'index d'une colonne du catalogue.
type Langue int

const (
	EN Langue = iota
	FR
	nbLangues
)

// courante est résolue une fois, au chargement du paquet : un processus qui vit huit
// millisecondes n'a pas besoin d'un changement de langue à chaud.
var courante = resoudre()

// Courante rend la langue effectivement retenue.
func Courante() Langue { return courante }

// Recharger relit l'environnement. Réservé aux tests : rien en production ne change de
// langue en cours d'exécution.
func Recharger() { courante = resoudre() }

// T rend la traduction de la clé dans la langue courante.
func T(cle string) string {
	trad, ok := catalogue[cle]
	if !ok {
		return cle // clé oubliée : visible, plutôt qu'un trou
	}
	if s := trad[courante]; s != "" {
		return s
	}
	return trad[EN]
}

// Tf formate la traduction avec des paramètres.
func Tf(cle string, args ...any) string { return fmt.Sprintf(T(cle), args...) }

// Entier écrit un entier avec le séparateur de milliers de la langue : « 16 449 » en
// français, « 16,449 » en anglais. L'espace française est insécable (U+00A0) : une
// ligne étroite ne coupe pas un nombre en deux. Pas l'espace fine (U+202F), que la
// typographie préfère mais que bien des polices de terminal n'ont pas.
func Entier(n int) string {
	s := strconv.Itoa(n)
	signe := ""
	if n < 0 {
		signe, s = "-", s[1:]
	}
	sep := ","
	if courante == FR {
		sep = " "
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(c)
	}
	return signe + b.String()
}

// culture est la dernière source de resoudre — la langue de l'utilisateur sous Windows —
// et la seule qui ne se lise pas dans l'environnement.
//
// C'est une variable, et non l'appel direct, pour que les tests puissent la poser :
// `t.Setenv` ne vide pas GetUserDefaultLocaleName, si bien que « toutes les sources sont
// muettes » était un état inatteignable sur une machine Windows. Les tests qui l'ignoraient
// n'avaient que deux issues, toutes deux mauvaises : assérer la langue de la machine qui
// les lance (i18n_test.go, qui échouait sur un Windows français), ou se dispenser d'y
// tourner (locale_test.go, qui s'y passait en silence). Ils la posent désormais.
var culture = cultureSysteme

// resoudre applique l'ordre de la spec : JIGGER_LANG, puis les variables POSIX, puis la
// culture du système, puis l'anglais.
func resoudre() Langue {
	if l, ok := depuisCode(os.Getenv("JIGGER_LANG")); ok {
		return l
	}
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := depuisCode(os.Getenv(v)); ok {
			return l
		}
	}
	if l, ok := depuisCode(culture()); ok {
		return l
	}
	return EN
}

// depuisCode lit un code de locale — « fr », « fr_FR.UTF-8 », « fr-FR » — et rend la
// langue correspondante. Le deuxième retour dit si le code était reconnu : un code vide
// ou inconnu laisse la résolution continuer.
func depuisCode(code string) (Langue, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return EN, false
	}
	code = strings.ToLower(code)
	if i := strings.IndexAny(code, "_-."); i > 0 {
		code = code[:i]
	}
	switch code {
	case "en":
		return EN, true
	case "fr":
		return FR, true
	}
	return EN, false
}
