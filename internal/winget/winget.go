// Package winget branche jigger sur le gestionnaire de paquets de Windows.
//
// winget n'expose aucune sortie machine : tout passe par des tableaux de largeur fixe
// aux en-têtes traduits (cf. table.go). Et il est lent — de une à trois secondes par
// appel, démarrage du moteur COM compris. Ces deux contraintes dictent la conception :
// jigger ne l'interroge **jamais** dans le chemin d'un rendu. Deux listes de noms sont
// tenues en cache — le catalogue et les paquets installés — et reconstituées par
// `jigger warm`, lancé détaché quand elles vieillissent.
package winget

import (
	"errors"
	"gitlab.yg-devworks.com/yves/jigger/internal/config"
	"os/exec"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

const (
	// Le catalogue winget bouge de quelques paquets par jour : le relire une fois par
	// jour suffit largement, pour trois secondes d'un processus détaché.
	ttlCatalogueDefaut = 24 * time.Hour
	// Les paquets installés, eux, changent à chaque install/uninstall. Le TTL n'est là
	// que comme filet : le shell demande un réchauffement forcé dès qu'il voit passer
	// une commande winget mutante.
	ttlInstalles = 10 * time.Minute

	ficCatalogue = "winget-catalog"
	ficInstalles = "winget-installed"
)

// Manager implémente pm.Manager pour winget.
type Manager struct{}

// New rend le gestionnaire winget.
func New() Manager { return Manager{} }

func (Manager) Cmd() string { return "winget" }

// Path rend le chemin de winget (« winget » si on ne le trouve pas : l'erreur
// d'exécution parlera d'elle-même).
func Path() string {
	if p, err := exec.LookPath("winget"); err == nil {
		return p
	}
	return "winget"
}

// Available dit si winget est installé.
func (Manager) Available() bool {
	_, err := exec.LookPath("winget")
	return err == nil
}

// Sous-commandes de winget. `dscv3` et `mcp`, réservées à l'outillage, ne sont pas
// proposées.
var subcommands = []string{
	"install", "uninstall", "upgrade", "list", "search", "show", "source", "pin",
	"export", "import", "download", "repair", "hash", "validate", "settings",
	"features", "configure",
}

// Alias reconnus par winget, ramenés à leur sous-commande.
var alias = map[string]string{
	"add": "install", "remove": "uninstall", "rm": "uninstall",
	"update": "upgrade", "ls": "list", "find": "search", "view": "show",
}

// Options communes à toutes les sous-commandes.
var optionsCommunes = []string{
	"--help", "--verbose", "--disable-interactivity", "--nowarn", "--wait", "--logs",
	"--proxy", "--no-proxy",
}

// Options par sous-commande. `--accept-source-agreements` traîne partout où une source
// est interrogée : c'est ce qui débloque un premier appel non interactif.
var optionsParSub = map[string][]string{
	"install": {
		"--id", "--name", "--moniker", "--version", "--exact", "--source", "--scope",
		"--architecture", "--interactive", "--silent", "--location", "--override",
		"--custom", "--force", "--skip-dependencies", "--uninstall-previous",
		"--accept-package-agreements", "--accept-source-agreements",
	},
	"uninstall": {
		"--id", "--name", "--moniker", "--version", "--exact", "--all-versions",
		"--scope", "--interactive", "--silent", "--purge", "--preserve", "--force",
	},
	"upgrade": {
		"--id", "--name", "--exact", "--all", "--include-unknown", "--include-pinned",
		"--version", "--source", "--scope", "--interactive", "--silent", "--purge",
		"--uninstall-previous", "--accept-package-agreements", "--accept-source-agreements",
	},
	"list": {
		"--id", "--name", "--moniker", "--tag", "--command", "--source", "--exact",
		"--count", "--scope", "--upgrade-available", "--include-unknown",
		"--include-pinned", "--details", "--accept-source-agreements",
	},
	"search": {
		"--id", "--name", "--moniker", "--tag", "--command", "--source", "--exact",
		"--count", "--versions", "--accept-source-agreements",
	},
	"show": {
		"--id", "--name", "--moniker", "--version", "--exact", "--source", "--versions",
		"--accept-source-agreements",
	},
	"download": {
		"--id", "--exact", "--version", "--download-directory", "--architecture",
		"--scope", "--installer-type", "--skip-dependencies", "--accept-package-agreements",
		"--accept-source-agreements",
	},
	"export": {"--output", "--source", "--include-versions", "--accept-source-agreements"},
	"import": {"--import-file", "--ignore-unavailable", "--ignore-versions", "--accept-package-agreements"},
	"pin":    {"--id", "--exact", "--version", "--blocking", "--force", "--source"},
	"repair": {"--id", "--exact", "--version", "--interactive", "--silent", "--purge", "--preserve"},
	"source": {"--name", "--type", "--arg", "--trust-level", "--explicit", "--force"},
}

// Sous-commandes dont l'argument est un paquet déjà installé.
var installedOnly = map[string]bool{
	"uninstall": true, "upgrade": true, "list": true, "repair": true,
}

// Les sous-commandes qui changent ce que `winget upgrade` répondra — celles qui font
// rafraîchir caches et prompt sans attendre le TTL — sont tenues côté shell, dans
// shell/jigger.psm1 : c'est lui qui voit passer les commandes.

func normalise(sub string) string {
	sub = strings.ToLower(sub)
	if a, ok := alias[sub]; ok {
		return a
	}
	return sub
}

func (Manager) Subcommands() []string { return subcommands }

func (Manager) Options(sub string) []string {
	if spec, ok := optionsParSub[normalise(sub)]; ok {
		return append(append([]string{}, spec...), optionsCommunes...)
	}
	return optionsCommunes
}

func (Manager) InstalledOnly(sub string) bool { return installedOnly[normalise(sub)] }

// Load construit le catalogue à partir des deux fichiers de cache — et rien d'autre :
// aucun appel à winget, qui coûterait des secondes à chaque frappe. Un cache périmé est
// utilisé tel quel, et déclenche un réchauffement pour la frappe suivante.
func (Manager) Load() *pm.Catalog {
	cat := pm.NewCatalog()

	noms, catalogueFrais := pm.Cached(ficCatalogue, ttlCatalogue())
	for _, n := range noms {
		cat.Add(n, pm.BadgeWinget)
	}

	installes, installesFrais := pm.Cached(ficInstalles, ttlInstalles)
	for _, l := range installes {
		nom, version, _ := strings.Cut(l, "\t")
		// Un paquet installé qui n'est pas au catalogue vient d'ailleurs (installeur
		// classique, appli du Store) : winget sait le désinstaller, pas le mettre à jour.
		badge := pm.BadgeOther
		if _, connu := cat.Badges[nom]; connu {
			badge = pm.BadgeWinget
		}
		cat.MarkInstalled(nom, version, badge)
	}

	cat.Sort()
	if !catalogueFrais || !installesFrais {
		pm.TriggerWarm()
	}
	if len(noms) == 0 {
		cat.Note = i18n.T("popup.catalog_winget")
	}
	return cat
}

// Insert rend le texte à insérer. winget résout un identifiant exact avant toute
// correspondance partielle : le nom seul suffit, et se laisse compléter. Seul un
// identifiant contenant une espace — ce qui arrive aux paquets détectés hors catalogue —
// doit être protégé, sans quoi le shell le couperait en deux arguments.
func (Manager) Insert(_ *pm.Catalog, _, _, name string) string {
	if strings.ContainsAny(name, " \t") {
		return `"` + name + `"`
	}
	return name
}

// Warm reconstitue les deux caches. C'est le chemin lent (quelques secondes), lancé
// détaché par TriggerWarm ou à la main par `jigger warm`.
func (Manager) Warm(scope pm.Scope) error {
	var echecs []error

	if _, frais := pm.Cached(ficCatalogue, ttlCatalogue()); scope == pm.ScopeAll || (scope == pm.ScopeStale && !frais) {
		// winget refuse de tout lister ; on interroge donc le point commun à tous les
		// identifiants de sa source officielle : le point qui sépare l'éditeur du
		// paquet (« Git.Git », « Microsoft.PowerShell »).
		out, err := pm.Run(Path(), "search", "--query", ".", "--source", "winget",
			"--disable-interactivity", "--accept-source-agreements")
		if err != nil {
			echecs = append(echecs, err)
		} else if ids := CatalogIDs(out); len(ids) > 0 {
			if err := pm.Store(ficCatalogue, ids); err != nil {
				echecs = append(echecs, err)
			}
		}
	}

	if _, frais := pm.Cached(ficInstalles, ttlInstalles); scope != pm.ScopeStale || !frais {
		out, err := pm.Run(Path(), "list", "--disable-interactivity", "--accept-source-agreements")
		if err != nil {
			echecs = append(echecs, err)
		} else if lignes := InstalledLines(out); len(lignes) > 0 {
			if err := pm.Store(ficInstalles, lignes); err != nil {
				echecs = append(echecs, err)
			}
		}
	}

	return errors.Join(echecs...)
}

// CatalogIDs extrait les identifiants d'une sortie de `winget search`.
func CatalogIDs(out []byte) []string {
	var ids []string
	vus := map[string]bool{}
	for _, r := range ParseTable(out) {
		id := r.Col(colID)
		if id == "" || tronque(id) || vus[id] {
			continue
		}
		vus[id] = true
		ids = append(ids, id)
	}
	return ids
}

// InstalledLines extrait « identifiant<TAB>version » d'une sortie de `winget list`.
func InstalledLines(out []byte) []string {
	var lignes []string
	vus := map[string]bool{}
	for _, r := range ParseTable(out) {
		id, version := r.Col(colID), r.Col(colVer)
		if id == "" || tronque(id) || vus[id] {
			continue
		}
		vus[id] = true
		// « > 8.12.32.33 » : winget préfixe la version d'un paquet épinglé.
		version = strings.TrimSpace(strings.TrimPrefix(version, ">"))
		if tronque(version) {
			version = ""
		}
		lignes = append(lignes, id+"\t"+version)
	}
	return lignes
}

// CountOutdated compte les paquets d'une sortie de `winget list --upgrade-available`.
func CountOutdated(out []byte) int {
	n := 0
	for _, r := range ParseTable(out) {
		if r.Col(colID) != "" {
			n++
		}
	}
	return n
}

// ParseVersion extrait « 1.29.280 » de la sortie de `winget --version` (« v1.29.280 »,
// parfois suivie d'un suffixe de préversion). Une sortie inattendue donne une chaîne
// vide : le prompt masquera le bloc plutôt que d'afficher n'importe quoi.
func ParseVersion(out string) string {
	ligne, _, _ := strings.Cut(out, "\n")
	v := strings.TrimSpace(ligne)
	v = strings.TrimPrefix(v, "v")
	v, _, _ = strings.Cut(v, "-")
	if v == "" || v[0] < '0' || v[0] > '9' {
		return ""
	}
	return v
}

// tronque repère une valeur coupée par winget faute de largeur de colonne. Un
// identifiant amputé n'est bon à rien : ni à insérer, ni à reconnaître un installé.
func tronque(s string) bool { return strings.HasSuffix(s, "…") }

// Version interroge winget. Rapide (~100 ms) : c'est la seule question qu'on lui pose
// dont la réponse ne coûte pas une seconde.
func Version() (string, error) {
	out, err := pm.Run(Path(), "--version")
	if err != nil {
		return "", err
	}
	return ParseVersion(string(out)), nil
}

// Outdated compte les paquets à mettre à niveau (~1 s : hors du chemin du prompt).
func Outdated() (int, error) {
	out, err := pm.Run(Path(), "list", "--upgrade-available", "--disable-interactivity",
		"--accept-source-agreements")
	if err != nil {
		// « aucun paquet à mettre à niveau » n'est pas toujours un succès pour winget ;
		// une sortie exploitable prime sur le code de retour.
		if len(out) == 0 {
			return 0, err
		}
	}
	return CountOutdated(out), nil
}

// ttlCatalogue rend la durée de validité du catalogue, déclarée donc réglable (A-14).
// Lue au chargement du catalogue, pas à chaque frappe.
func ttlCatalogue() time.Duration { return config.Duree("winget_ttl", ttlCatalogueDefaut) }

// Le gestionnaire déclare ses réglages : l'écran de configuration en dérive sa mise en
// page, sans rien savoir de winget (A-14, esprit de l'ADR-0002).
func init() {
	config.Declarer(config.Reglage{
		Cle: "winget_ttl", CleI18n: "cfg.ttl", Portee: config.Binaire,
		Type: config.TypeDuree, Defaut: "24h", PM: "winget",
	})
}
