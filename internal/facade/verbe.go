// Package facade est le moteur de la syntaxe unique de jigger : `jg install fd` plutôt
// que `brew install fd` ou `scoop install fd`.
//
// Il ne connaît aucun gestionnaire en particulier. Tout ce qu'il sait, il le lit dans les
// tables que ceux-ci déclarent (cf. pm.Bindings) : quels verbes existent, comment ils se
// traduisent, où chercher leurs candidats.
package facade

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// ResoudreVerbe reconnaît le verbe en tête de ligne et rend les gestionnaires installés
// qui savent le rendre.
func ResoudreVerbe(ligne []string) (pm.Verb, []string, []pm.Manager, error) {
	return resoudreVerbe(ligne, managers.Available(), managers.All())
}

// resoudreVerbe est ResoudreVerbe sur des listes données — c'est la forme testable, qui
// permet de simuler une machine où tel gestionnaire manque.
func resoudreVerbe(ligne []string, dispo, tous []pm.Manager) (pm.Verb, []string, []pm.Manager, error) {
	if len(ligne) == 0 {
		return "", nil, nil, fmt.Errorf("jigger : aucun verbe. Essaie « jg install <paquet> » ou « jg outdated »")
	}

	tablesDispo := managers.Tables(dispo)

	// Le verbe composé d'abord : « source add extras » est « source add », pas « source »
	// avec un argument.
	if len(ligne) >= 2 {
		compose := pm.Verb(ligne[0] + " " + ligne[1])
		if capables, ok := tablesDispo[compose]; ok {
			return compose, ligne[2:], trier(capables), nil
		}
	}

	simple := pm.Verb(ligne[0])
	if capables, ok := tablesDispo[simple]; ok {
		return simple, ligne[1:], trier(capables), nil
	}

	return "", nil, nil, verbeIndisponible(ligne, dispo, tous)
}

// verbeIndisponible construit le message qui distingue « personne ne sait faire ça » de
// « quelqu'un saurait, mais il n'est pas installé ». C'est le modèle de capacités qui
// parle : sans cette distinction, l'utilisateur ne sait pas s'il s'est trompé de mot ou
// s'il lui manque un outil.
func verbeIndisponible(ligne []string, dispo, tous []pm.Manager) error {
	mot := ligne[0]
	if len(ligne) >= 2 {
		if _, ok := managers.Tables(tous)[pm.Verb(mot+" "+ligne[1])]; ok {
			mot = mot + " " + ligne[1]
		}
	}

	var ailleurs []string
	for _, m := range tous {
		if estDispo(m, dispo) {
			continue
		}
		b, ok := m.(pm.Bindings)
		if !ok {
			continue
		}
		liaison, ok := b.Verbs()[pm.Verb(mot)]
		if !ok {
			continue
		}
		if natif := liaison.NomNatif(); natif != "" && natif != mot {
			ailleurs = append(ailleurs, fmt.Sprintf("%s le sait (%s)", m.Cmd(), natif))
		} else {
			ailleurs = append(ailleurs, fmt.Sprintf("%s le sait", m.Cmd()))
		}
	}

	if len(ailleurs) == 0 {
		return fmt.Errorf("jigger : « %s » — verbe inconnu. « jg ⇥ » liste ce que jigger sait faire", mot)
	}
	sort.Strings(ailleurs)
	return fmt.Errorf("jigger : « %s » — aucun gestionnaire disponible ne sait faire ça.\n        %s, mais n'est pas installé",
		mot, strings.Join(ailleurs, " ; "))
}

func estDispo(m pm.Manager, dispo []pm.Manager) bool {
	for _, d := range dispo {
		if d.Cmd() == m.Cmd() {
			return true
		}
	}
	return false
}

// trier range les gestionnaires dans l'ordre de managers.All(), pour que l'exécution
// séquentielle soit reproductible d'un appel à l'autre.
func trier(mgrs []pm.Manager) []pm.Manager {
	rang := map[string]int{}
	for i, m := range managers.All() {
		rang[m.Cmd()] = i
	}
	out := append([]pm.Manager(nil), mgrs...)
	sort.Slice(out, func(i, j int) bool { return rang[out[i].Cmd()] < rang[out[j].Cmd()] })
	return out
}
