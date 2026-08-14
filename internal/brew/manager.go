package brew

import (
	"path/filepath"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Manager implémente pm.Manager pour Homebrew.
type Manager struct{}

// New rend le gestionnaire Homebrew.
func New() Manager { return Manager{} }

func (Manager) Cmd() string { return "brew" }

// Sous-commandes proposées pour le premier mot.
var subcommands = []string{
	"install", "uninstall", "reinstall", "upgrade", "info", "search", "list", "deps",
	"uses", "pin", "unpin", "tap", "untap", "outdated", "cleanup", "autoremove", "doctor",
	"services", "link", "unlink", "home", "desc", "leaves", "missing", "update",
}

// Options communes à toutes les sous-commandes.
var optionsCommunes = []string{"--help", "--verbose", "--debug", "--quiet"}

// Options spécifiques par sous-commande.
var optionsParSub = map[string][]string{
	"install":   {"--cask", "--formula", "--force", "--HEAD", "--build-from-source", "--no-quarantine", "--dry-run"},
	"reinstall": {"--cask", "--formula", "--force", "--no-quarantine"},
	"uninstall": {"--cask", "--formula", "--force", "--zap", "--ignore-dependencies"},
	"upgrade":   {"--cask", "--formula", "--greedy", "--dry-run", "--force"},
	"list":      {"--versions", "--cask", "--formula", "--pinned", "--full-name", "-1"},
	"outdated":  {"--cask", "--formula", "--greedy", "--verbose", "--json"},
	"info":      {"--json", "--cask", "--formula", "--github"},
	"deps":      {"--tree", "--installed", "--cask", "--formula", "--include-build"},
	"uses":      {"--installed", "--recursive", "--cask", "--formula"},
	"search":    {"--cask", "--formula", "--desc"},
	"cleanup":   {"--prune", "--dry-run", "-s"},
	"services":  {"--all"},
}

// Sous-commandes dont les arguments sont des paquets déjà installés.
var installedOnly = map[string]bool{
	"uninstall": true, "remove": true, "rm": true, "reinstall": true, "upgrade": true,
	"pin": true, "unpin": true, "link": true, "unlink": true, "uses": true,
}

// Les sous-commandes qui changent ce que `brew outdated` répondra — celles qui font
// rafraîchir le bloc de prompt sans attendre le TTL — sont tenues côté shell, dans
// shell/jigger.plugin.zsh : c'est lui qui voit passer les commandes.

func (Manager) Subcommands() []string { return subcommands }

func (Manager) Options(sub string) []string {
	if spec, ok := optionsParSub[strings.ToLower(sub)]; ok {
		return append(append([]string{}, spec...), optionsCommunes...)
	}
	return optionsCommunes
}

func (Manager) InstalledOnly(sub string) bool { return installedOnly[strings.ToLower(sub)] }

// Available dit si Homebrew est installé — c'est-à-dire si Path() a trouvé autre chose
// que son dernier recours.
func (Manager) Available() bool { return filepath.IsAbs(Path()) }

func (Manager) Load() *pm.Catalog { return Load() }

// Insert ajoute `--cask` derrière install/reinstall pour un cask pur — sans quoi brew
// refuse la commande (« use --cask ») —, sauf si la ligne tranche déjà.
func (Manager) Insert(cat *pm.Catalog, sub, prefix, name string) string {
	if sub != "install" && sub != "reinstall" {
		return name
	}
	if strings.Contains(prefix, "--cask") || strings.Contains(prefix, "--formula") {
		return name
	}
	if cat.Badge(name) == pm.BadgeCask {
		return "--cask " + name
	}
	return name
}

// Warm reconstitue les listes de noms. Homebrew les sert en une seconde environ ; elles
// sont de toute façon relues à la demande par Load, ce réchauffement n'étant qu'une
// occasion de le faire hors du chemin d'une frappe.
//
// Les paquets installés n'ont pas de cache à refaire : ils sont lus dans Cellar et
// Caskroom, donc toujours frais (cf. installedFromDirs).
func (Manager) Warm(scope pm.Scope) error {
	if scope == pm.ScopeInstalled {
		return nil
	}
	ttl := 24 * time.Hour
	if scope == pm.ScopeAll {
		ttl = 0
	}
	cachedLines("formulae", ttl, "formulae")
	cachedLines("casks", ttl, "casks")
	return nil
}
