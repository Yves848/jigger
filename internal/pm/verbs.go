// Ce fichier porte le second contrat de pm : savoir **agir**. Le premier — pm.go — ne
// sait que répondre à des questions, et n'est pas modifié (cf. ADR-0002).
//
// Un gestionnaire déclare une table `verbe → liaison`. Les capacités en découlent sans
// drapeau à tenir : un verbe absent de la table est un verbe que ce gestionnaire ne sait
// pas rendre.
package pm

import (
	"errors"
	"strings"
)

// Verb est un membre de phrase du vocabulaire de jigger : « install », « source add ».
// La clé porte la phrase entière, ce qui garde les gabarits d'argv triviaux — sans quoi
// `brew tap` / `untap`, deux mots sans rapport, exigeraient aussitôt Build.
type Verb string

// Pool dit où chercher les candidats d'un verbe. Il reprend la notion de
// Manager.InstalledOnly en l'élargissant au cas « ce verbe ne prend pas de paquet ».
// Manager.InstalledOnly reste en place pour la complétion native : les deux coexistent.
type Pool int

const (
	PoolAucun     Pool = iota // le verbe ne prend pas de nom de paquet
	PoolCatalogue             // tous les paquets connus
	PoolInstalles             // les seuls paquets installés
)

// Parser refond la sortie d'un gestionnaire en lignes normalisées. Fonction pure : elle
// ne lance rien, ce qui la rend testable sur fichier.
type Parser func(out []byte) ([]Package, error)

// Marqueurs de gabarit. La distinction n'est pas cosmétique : winget n'installe qu'un
// paquet par appel, brew en prend plusieurs.
const (
	MarqueurTous = "{args}" // tous les arguments, une seule invocation
	MarqueurUn   = "{arg}"  // un argument par invocation
)

// Binding lie un verbe de jigger à ce qu'il faut faire chez un gestionnaire. Une liaison
// agit d'**une seule** des trois façons ci-dessous.
type Binding struct {
	Native []string                          // gabarit d'argv — le cas ordinaire
	Build  func(args []string) []string      // argv calculé — le cas rétif
	Direct func(args []string) ([]Package, error) // sans sous-processus du tout

	Pool  Pool   // où chercher les candidats
	Parse Parser // nil → la sortie est relayée telle quelle
}

// Bindings est le contrat d'exécution. Un gestionnaire peut implémenter Manager sans
// l'implémenter : on saura le compléter sans savoir le piloter.
type Bindings interface {
	Verbs() map[Verb]Binding
}

// Argv construit les lignes d'arguments passées au gestionnaire. Elle en rend
// **plusieurs** quand le gabarit porte MarqueurUn : un appel par argument.
func (b Binding) Argv(args []string) [][]string {
	if b.Build != nil {
		return [][]string{b.Build(args)}
	}
	if !contient(b.Native, MarqueurUn) {
		return [][]string{developper(b.Native, args)}
	}
	if len(args) == 0 {
		return [][]string{retirer(b.Native, MarqueurUn)}
	}
	lignes := make([][]string, 0, len(args))
	for _, a := range args {
		lignes = append(lignes, developper(b.Native, []string{a}))
	}
	return lignes
}

// developper remplace les marqueurs par les arguments. MarqueurTous s'étale en autant
// d'éléments qu'il y a d'arguments ; MarqueurUn prend le premier.
func developper(gabarit, args []string) []string {
	out := make([]string, 0, len(gabarit)+len(args))
	for _, mot := range gabarit {
		switch mot {
		case MarqueurTous:
			out = append(out, args...)
		case MarqueurUn:
			if len(args) > 0 {
				out = append(out, args[0])
			}
		default:
			out = append(out, mot)
		}
	}
	return out
}

func retirer(gabarit []string, marqueur string) []string {
	out := make([]string, 0, len(gabarit))
	for _, mot := range gabarit {
		if mot != marqueur {
			out = append(out, mot)
		}
	}
	return out
}

func contient(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Valid vérifie qu'une liaison est bien formée. Elle est appelée par les tests de table
// de chaque gestionnaire : une table mal formée doit se voir à la compilation des tests,
// pas au premier usage.
func (b Binding) Valid() error {
	n := 0
	for _, pose := range []bool{b.Native != nil, b.Build != nil, b.Direct != nil} {
		if pose {
			n++
		}
	}
	switch {
	case n == 0:
		return errors.New("liaison sans Native, Build ni Direct")
	case n > 1:
		return errors.New("liaison avec plusieurs façons d'agir")
	case b.Direct != nil && b.Parse != nil:
		return errors.New("Direct rend déjà des Package : Parse est de trop")
	}
	return nil
}

// NomNatif rend le nom que le gestionnaire donne au verbe — « checkup » là où jigger dit
// « doctor ». Vide pour une liaison calculée, dont le nom dépend des arguments. Sert aux
// messages de capacité : « scoop le sait (checkup), mais n'est pas installé ».
func (b Binding) NomNatif() string {
	if len(b.Native) == 0 {
		return ""
	}
	if strings.HasPrefix(b.Native[0], "{") {
		return ""
	}
	return b.Native[0]
}
