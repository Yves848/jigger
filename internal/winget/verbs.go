package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que winget sait faire. Ni cleanup ni doctor n'y figurent : winget n'a
// pas ces concepts, et leur absence est précisément ce que le modèle de capacités lit.
//
// Les valeurs ci-dessous proviennent du cahier des charges et ne sont pas encore
// vérifiées contre une vraie installation de winget (cf. tâche 1b).
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
		"search":    {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue, Parse: parseSearch},
		"info":      {Native: []string{"show", "--id", pm.MarqueurUn}, Pool: pm.PoolCatalogue},

		"source":     {Native: []string{"source", "list"}, Pool: pm.PoolAucun, Parse: parseSource},
		"source add": {Native: []string{"source", "add", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"source rm":  {Native: []string{"source", "remove", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"pin":        {Native: []string{"pin", "add", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
		"unpin":      {Native: []string{"pin", "remove", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
	}
}
