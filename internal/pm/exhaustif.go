package pm

import "strings"

// Exhaustif est le gestionnaire dont Subcommands() fait foi : la liste n'est pas un choix
// de commodité, c'est l'inventaire complet de ce qu'il sait faire. Contrat **optionnel**,
// comme Bindings ou Elevateur : celui qui ne l'implémente pas ne dit rien, et jigger s'en
// tient au comportement permissif.
//
// La distinction n'est pas théorique, c'est celle qui sépare un plugin d'un natif.
//
// Un plugin déclare ses verbes dans son `config.json` : la liste EST complète par
// construction, jigger n'en connaît pas d'autres et le plugin n'en exécutera pas d'autres.
// Homebrew, lui, déclare vingt-cinq sous-commandes choisies à la main alors qu'il en a une
// centaine — `brew fetch` n'y figure pas et reste une commande parfaitement valide, dont
// l'argument est bien une formule.
//
// D'où deux comportements, et non un seul : sur un natif, un verbe non déclaré prend
// quand même des paquets ; sur un exhaustif, il n'en prend pas, parce qu'il n'est pas un
// verbe du tout. Le cas qui a rendu la distinction nécessaire est `git checkout ` avec le
// plugin git : proposer des noms de dépôts en argument de `checkout` est faux, et le
// proposer comme exécutable est dangereux (#141).
type Exhaustif interface {
	VerbesExhaustifs() bool
}

// VerbesExhaustifsDe interroge un gestionnaire s'il sait répondre, et rend false sinon. Un
// seul endroit fait le test de type : les appelants n'ont pas à connaître le contrat.
func VerbesExhaustifsDe(m Manager) bool {
	e, ok := m.(Exhaustif)
	return ok && e.VerbesExhaustifs()
}

// VerbeConnu dit si `sub` figure parmi les sous-commandes de `m`, sans égard à la casse.
// La comparaison ne peut pas être exacte : complete minuscule la sous-commande avant de la
// passer, alors que pacman déclare ses opérations en capitales — « -Rns » arrive en
// « -rns ».
func VerbeConnu(m Manager, sub string) bool {
	for _, s := range m.Subcommands() {
		if strings.EqualFold(s, sub) {
			return true
		}
	}
	return false
}
