package complete

import (
	"regexp"
	"strings"
)

// Filtre décide si un **nom de paquet** correspond au mot en cours de saisie. Deux modes,
// choisis par l'appelant : préfixe — le comportement historique du popup — ou expression
// rationnelle.
//
// Il ne s'applique qu'aux noms de paquets. Les verbes, les sous-commandes et les options
// gardent leur filtre par préfixe : ce sont des vocabulaires de quelques dizaines
// d'entrées, où une expression rationnelle n'apprendrait rien et surprendrait.
type Filtre struct {
	mot    string         // mot saisi, déjà en minuscules
	re     *regexp.Regexp // non nil en mode regex, si le motif compile
	Regex  bool           // mode demandé
	Vide   bool           // le mot est vide : tout correspond
	Fautif bool           // mode regex, et le motif ne compile pas
}

// NouveauFiltre prépare le filtre une fois pour toutes, avant de balayer le catalogue :
// compiler l'expression à chaque nom coûterait le budget de la frappe (16 000 entrées
// côté winget).
func NouveauFiltre(mot string, regex bool) Filtre {
	bas := strings.ToLower(mot)
	f := Filtre{mot: bas, Regex: regex, Vide: bas == ""}
	if regex && !f.Vide {
		// (?i) : la casse est ignorée, comme en mode préfixe. Basculer de mode ne doit
		// pas changer discrètement la sensibilité à la casse.
		re, err := regexp.Compile("(?i)" + mot)
		if err != nil {
			f.Fautif = true
		} else {
			f.re = re
		}
	}
	return f
}

// Correspond dit si le nom est retenu.
//
// Un motif fautif ne retient rien : le cadre affichera « aucun candidat » plutôt que le
// catalogue entier. C'est l'inverse du choix fait dans le sélecteur plein écran, et c'est
// délibéré — ici, le motif est le mot de la ligne de commande, et montrer 16 000 entrées
// parce qu'une parenthèse manque serait une avalanche, pas une aide.
func (f Filtre) Correspond(nom string) bool {
	switch {
	case f.Vide:
		return true
	case f.Fautif:
		return false
	case f.re != nil:
		return f.re.MatchString(nom)
	default:
		return strings.HasPrefix(strings.ToLower(nom), f.mot)
	}
}
