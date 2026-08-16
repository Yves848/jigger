package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que winget sait faire. Ni cleanup ni doctor n'y figurent : winget n'a
// pas ces concepts, et leur absence est précisément ce que le modèle de capacités lit.
//
// Vérifiées le 16 août 2026 contre winget v1.29.280 sous Windows 10.0.26200, sur les
// captures de tests/captures/ : `winget pin --help` donne bien `add` et `remove`,
// `winget source --help` donne bien `add`, `list` et `remove`. C'étaient les deux
// liaisons les plus incertaines du cahier des charges ; les autres verbes s'exercent à
// chaque usage du popup et de la façade sur cette machine.
//
// MarqueurUn partout où un identifiant est attendu : winget ne prend qu'un `--id` par
// appel, là où brew et scoop acceptent une liste.
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"install":   {Native: []string{"install", "--id", pm.MarqueurUn, "--exact"}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", "--id", pm.MarqueurUn, "--exact"}, Pool: pm.PoolInstalles},
		"upgrade":   {Native: []string{"upgrade", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
		"list":      {Native: []string{"list"}, Pool: pm.PoolAucun, Parse: parseList},
		"outdated":  {Native: []string{"list", "--upgrade-available"}, Pool: pm.PoolAucun, Parse: parseOutdated},
		// search prend une requête, pas un nom de paquet à résoudre au catalogue (cf.
		// internal/brew/verbs.go pour le détail du raisonnement).
		"search": {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolAucun, Parse: parseSearch},
		"info":   {Native: []string{"show", "--id", pm.MarqueurUn}, Pool: pm.PoolCatalogue},

		"source":     {Native: []string{"source", "list"}, Pool: pm.PoolAucun, Parse: parseSource},
		"source add": {Native: []string{"source", "add", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"source rm":  {Native: []string{"source", "remove", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"pin":        {Native: []string{"pin", "add", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
		"unpin":      {Native: []string{"pin", "remove", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
	}
}
