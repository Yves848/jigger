// Package scoop branche jigger sur scoop.
//
// scoop est écrit en PowerShell : le moindre `scoop list` coûte près d'une seconde,
// démarrage de l'interpréteur compris. Mais tout ce dont jigger a besoin est **déjà sur
// le disque**, dans une arborescence limpide :
//
//	<racine>/buckets/<bucket>/bucket/<nom>.json   le catalogue
//	<racine>/apps/<nom>/<version>                 les paquets installés
//
// C'est exactement la structure que Homebrew donne à Cellar : un readdir suffit, et
// jigger n'a jamais à lancer scoop — ni pour compléter, ni pour compter les mises à
// jour en attente. Rien à mettre en cache, donc rien qui puisse mentir.
package scoop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Manager implémente pm.Manager pour scoop.
type Manager struct{}

// New rend le gestionnaire scoop.
func New() Manager { return Manager{} }

func (Manager) Cmd() string { return "scoop" }

// Available dit si scoop est installé.
func (Manager) Available() bool { return Present() }

// Root rend la racine de l'installation utilisateur : $SCOOP, sinon ~/scoop.
func Root() string {
	if r := os.Getenv("SCOOP"); r != "" {
		return r
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "scoop")
}

// GlobalRoot rend la racine des paquets installés pour toute la machine (`scoop install
// -g`) : $SCOOP_GLOBAL, sinon %ProgramData%\scoop. Vide si elle n'existe pas.
func GlobalRoot() string {
	r := os.Getenv("SCOOP_GLOBAL")
	if r == "" {
		if pd := os.Getenv("ProgramData"); pd != "" {
			r = filepath.Join(pd, "scoop")
		}
	}
	if r == "" {
		return ""
	}
	if _, err := os.Stat(r); err != nil {
		return ""
	}
	return r
}

// Present dit si scoop est installé. Le prompt s'en sert pour ne pas afficher un
// compteur qui n'aurait aucun sens sur une machine sans scoop.
func Present() bool {
	root := Root()
	if root == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, "apps"))
	return err == nil
}

var subcommands = []string{
	"install", "uninstall", "update", "list", "search", "info", "status", "cleanup",
	"hold", "unhold", "reset", "bucket", "cache", "cat", "checkup", "config", "create",
	"depends", "download", "export", "import", "home", "prefix", "shim", "virustotal",
	"which", "alias", "help",
}

var optionsCommunes = []string{"--help"}

var optionsParSub = map[string][]string{
	"install":    {"--global", "--independent", "--no-cache", "--no-update-scoop", "--skip-hash-check", "--arch"},
	"uninstall":  {"--global", "--purge"},
	"update":     {"--all", "--force", "--global", "--independent", "--no-cache", "--skip-hash-check", "--quiet"},
	"cleanup":    {"--all", "--global", "--cache"},
	"list":       {"--global"},
	"status":     {"--local"},
	"download":   {"--force", "--no-hash-check", "--no-update-scoop", "--arch"},
	"info":       {"--verbose"},
	"hold":       {"--global"},
	"unhold":     {"--global"},
	"reset":      {"--all"},
	"export":     {"--config"},
	"import":     {"--global"},
	"depends":    {"--arch"},
	"shim":       {"--global"},
	"virustotal": {"--all", "--scan", "--no-depends", "--no-update-scoop", "--passthru"},
}

// Sous-commandes dont l'argument est une application déjà installée.
var installedOnly = map[string]bool{
	"uninstall": true, "update": true, "hold": true, "unhold": true, "reset": true,
	"cleanup": true, "prefix": true, "list": true,
}

func (Manager) Subcommands() []string { return subcommands }

func (Manager) Options(sub string) []string {
	if spec, ok := optionsParSub[strings.ToLower(sub)]; ok {
		return append(append([]string{}, spec...), optionsCommunes...)
	}
	return optionsCommunes
}

func (Manager) InstalledOnly(sub string) bool { return installedOnly[strings.ToLower(sub)] }

// Load lit le catalogue et les installés sur le disque. Aucun cache : les répertoires
// concernés tiennent en quelques milliers d'entrées, et un readdir est mille fois moins
// cher qu'un démarrage de PowerShell.
func (Manager) Load() *pm.Catalog {
	return LoadFrom(Root(), GlobalRoot())
}

// LoadFrom est Load sur des racines données (tests).
func LoadFrom(root, global string) *pm.Catalog {
	cat := pm.NewCatalog()

	// Un même nom peut vivre dans plusieurs buckets : on retient lesquels pour pouvoir
	// qualifier l'insertion (cf. Insert).
	buckets := map[string][]string{}
	for _, dir := range bucketDirs(root) {
		bucket := bucketName(dir)
		for _, nom := range manifests(dir) {
			cat.Add(nom, badge(bucket))
			buckets[nom] = append(buckets[nom], bucket)
		}
	}
	for nom, bs := range buckets {
		if len(bs) > 1 {
			sort.Strings(bs)
			// scoop retient le premier bucket venu et le signale par un avertissement.
			// On écrit le choix dans la ligne : il devient explicite, et modifiable.
			cat.Qualified[nom] = prioritaire(bs) + "/" + nom
		}
	}

	for _, racine := range []string{root, global} {
		for nom, version := range installed(racine) {
			cat.MarkInstalled(nom, version, pm.BadgeScoop)
		}
	}

	cat.Sort()
	return cat
}

// Insert qualifie par son bucket un nom présent dans plusieurs d'entre eux — l'analogue
// du `--cask` de brew : sans lui, la commande passe, mais sur l'application d'un autre
// bucket que celle qu'on croyait choisir (scoop se contente d'un avertissement).
func (Manager) Insert(cat *pm.Catalog, _, prefix, name string) string {
	if q, ok := cat.Qualified[name]; ok && !strings.Contains(prefix, "/") {
		return q
	}
	return name
}

// Warm n'a rien à faire : scoop n'a pas de cache dans jigger, tout est lu sur le disque.
func (Manager) Warm(pm.Scope) error { return nil }

// bucketDirs rend les dossiers de manifestes des buckets installés. Les buckets récents
// rangent leurs manifestes dans un sous-dossier `bucket/` ; les plus anciens les
// déposent à la racine.
func bucketDirs(root string) []string {
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "buckets"))
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, "buckets", e.Name())
		if info, err := os.Stat(filepath.Join(dir, "bucket")); err == nil && info.IsDir() {
			dir = filepath.Join(dir, "bucket")
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

// bucketName rend le nom du bucket d'après le dossier de ses manifestes.
func bucketName(dir string) string {
	if filepath.Base(dir) == "bucket" {
		dir = filepath.Dir(dir)
	}
	return filepath.Base(dir)
}

// manifests rend les noms d'applications d'un dossier de manifestes.
func manifests(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	noms := make([]string, 0, len(entries))
	for _, e := range entries {
		nom, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || strings.HasPrefix(nom, ".") {
			continue
		}
		noms = append(noms, nom)
	}
	return noms
}

// installed lit les applications installées sous <racine>/apps, sous la forme
// nom -> version. scoop range chaque application dans apps/<nom>/<version>, avec un
// lien `current` vers celle en service : c'est lui qu'on suit, plutôt que de deviner
// la plus récente d'un tri.
func installed(root string) map[string]string {
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "apps"))
	if err != nil {
		return nil
	}
	apps := map[string]string{}
	for _, e := range entries {
		if !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		apps[e.Name()] = versionInstallee(filepath.Join(root, "apps", e.Name()))
	}
	return apps
}

// versionInstallee rend la version en service d'une application. Le manifeste copié
// sous `current` fait foi ; à défaut — installation abîmée, lien cassé — on se rabat
// sur le dernier dossier de version par ordre alphabétique.
func versionInstallee(appDir string) string {
	if v := versionManifeste(filepath.Join(appDir, "current", "manifest.json")); v != "" {
		return v
	}
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return ""
	}
	derniere := ""
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "current" || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.Name() > derniere {
			derniere = e.Name()
		}
	}
	return derniere
}

// versionManifeste lit le champ `version` d'un manifeste scoop.
func versionManifeste(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	return doc.Version
}

// prioritaire choisit le bucket à nommer quand plusieurs contiennent la même
// application : `main` d'abord — c'est le bucket officiel, et celui que scoop retient —
// puis l'ordre alphabétique.
func prioritaire(buckets []string) string {
	for _, b := range buckets {
		if b == "main" {
			return b
		}
	}
	return buckets[0]
}

// badge distingue le bucket officiel des autres, comme brew distingue formulae et casks.
func badge(bucket string) string {
	if bucket == "main" {
		return pm.BadgeScoop
	}
	return pm.BadgeOther
}
