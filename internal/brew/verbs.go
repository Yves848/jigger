package brew

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que brew sait faire. La table est le modèle de capacités : un verbe
// absent est un verbe que brew ne sait pas rendre.
//
// Les parsers (Parse) des quatre verbes normalisés — list, outdated, search, source —
// sont branchés ici (tâche 9). Les autres verbes n'en ont pas besoin : leur sortie est
// relayée telle quelle.
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		// Universels
		"install":   {Native: []string{"install", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"upgrade":   {Native: []string{"upgrade", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"list":      {Native: []string{"list", "--versions"}, Pool: pm.PoolAucun, Parse: parseList},
		"outdated":  {Native: []string{"outdated", "--json=v2"}, Pool: pm.PoolAucun, Parse: parseOutdated},
		// search prend une requête, pas un nom de paquet à résoudre au catalogue : la
		// router en PoolCatalogue ferait chercher la requête elle-même comme si
		// c'était le paquet visé, et refuserait de chercher un mot qui n'est
		// justement pas encore un nom connu (cf. tâche « recherche »).
		"search": {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolAucun, Parse: parseSearch},
		"info":      {Native: []string{"info", pm.MarqueurTous}, Pool: pm.PoolCatalogue},

		// Convergents
		"source":     {Native: []string{"tap"}, Pool: pm.PoolAucun, Parse: parseSource},
		"source add": {Native: []string{"tap", pm.MarqueurTous}, Pool: pm.PoolAucun},
		// untap n'est pas « tap » avec une option : d'où Build plutôt qu'un gabarit.
		"source rm": {
			Build: func(args []string) []string {
				return append([]string{"untap"}, args...)
			},
			Pool: pm.PoolAucun,
		},
		"pin":     {Native: []string{"pin", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"unpin":   {Native: []string{"unpin", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"cleanup": {Native: []string{"cleanup"}, Pool: pm.PoolAucun},
		"doctor":  {Native: []string{"doctor"}, Pool: pm.PoolAucun},
	}
}
