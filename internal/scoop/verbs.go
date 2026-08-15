package scoop

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que scoop sait faire.
//
// Les valeurs ci-dessous proviennent du cahier des charges et ne sont pas encore
// vérifiées contre une vraie installation de scoop (cf. tâche 1b).
//
// outdated est le seul verbe en Direct de tout jigger : scoop range ses applications dans
// une arborescence qui ressemble au Cellar de Homebrew, et la comparaison des manifestes
// se fait sur le disque (cf. outdated.go). Passer par un sous-processus pour redemander ce
// que jigger sait déjà — en démarrant PowerShell, qui plus est — serait absurde.
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"install":   {Native: []string{"install", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		// `scoop update` seul met à jour scoop et les buckets : viser « * » pour les
		// applications quand aucun nom n'est donné.
		"upgrade": {
			Build: func(args []string) []string {
				if len(args) == 0 {
					return []string{"update", "*"}
				}
				return append([]string{"update"}, args...)
			},
			Pool: pm.PoolInstalles,
		},
		"list":     {Native: []string{"list"}, Pool: pm.PoolAucun},
		"outdated": {Direct: outdatedDirect, Pool: pm.PoolAucun},
		"search":   {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"info":     {Native: []string{"info", pm.MarqueurTous}, Pool: pm.PoolCatalogue},

		"source":     {Native: []string{"bucket", "list"}, Pool: pm.PoolAucun},
		"source add": {Native: []string{"bucket", "add", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"source rm":  {Native: []string{"bucket", "rm", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"pin":        {Native: []string{"hold", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"unpin":      {Native: []string{"unhold", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"cleanup":    {Native: []string{"cleanup", "*"}, Pool: pm.PoolAucun},
		"doctor":     {Native: []string{"checkup"}, Pool: pm.PoolAucun},
	}
}

// outdatedDirect enveloppe la comparaison de manifestes déjà écrite dans outdated.go.
// Il ne la réécrit pas : il traduit OutdatedApps en pm.Package.
func outdatedDirect([]string) ([]pm.Package, error) {
	apps, err := OutdatedApps()
	if err != nil {
		return nil, err
	}
	out := make([]pm.Package, 0, len(apps))
	for _, a := range apps {
		out = append(out, pm.Package{
			Name:      a.Nom,
			Version:   a.Installee,
			Available: a.Disponible,
			Kind:      pm.BadgeScoop,
			Source:    a.Bucket,
			PM:        "scoop",
		})
	}
	return out, nil
}
