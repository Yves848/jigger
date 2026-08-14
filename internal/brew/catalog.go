// Package brew récupère (et met en cache) les noms connus de Homebrew : toutes les
// formulae, tous les casks, et les paquets installés. Ces listes alimentent la
// complétion contextuelle.
package brew

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Path renvoie le chemin du binaire brew (Apple Silicon puis Intel).
func Path() string {
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("brew"); err == nil {
		return p
	}
	return "brew"
}

// prefix renvoie le préfixe Homebrew (le parent de bin/brew), d'où pendent Cellar
// et Caskroom.
func prefix() string {
	if p := os.Getenv("HOMEBREW_PREFIX"); p != "" {
		return p
	}
	return filepath.Dir(filepath.Dir(Path())) // …/bin/brew -> …
}

// installedFromDirs lit les paquets installés directement sur le disque, sous la forme
// « nom version » (le format de `brew list --versions`). Homebrew range chaque paquet
// dans <Cellar|Caskroom>/<nom>/<version> : un readdir suffit, là où `brew list` coûte
// ~300 ms de démarrage Ruby — inacceptable pour un rendu à chaque frappe.
//
// Un répertoire absent ou illisible n'est pas une erreur (pas de cask installé, préfixe
// inattendu) : il ne contribue simplement rien.
func installedFromDirs(dirs ...string) []string {
	var lines []string
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if v := latestVersionDir(filepath.Join(dir, e.Name())); v != "" {
				lines = append(lines, e.Name()+" "+v)
			}
		}
	}
	sort.Strings(lines)
	return lines
}

// latestVersionDir renvoie le nom du dernier sous-répertoire par ordre alphabétique —
// la version la plus récente dans l'immense majorité des cas, et de toute façon celle
// que `brew list --versions` citerait en dernier. Vide si le paquet n'a aucun keg
// (répertoire résiduel après une désinstallation).
func latestVersionDir(pkgDir string) string {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return ""
	}
	latest := ""
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.Name() > latest {
			latest = e.Name()
		}
	}
	return latest
}

// cachedLines renvoie les lignes de `brew <args>`, mises en cache <ttl>. Le fichier
// de cache accélère massivement les invocations répétées (chaque appel du widget).
func cachedLines(cacheName string, ttl time.Duration, args ...string) []string {
	if lines, frais := pm.Cached(cacheName, ttl); frais {
		return lines
	}

	out, err := pm.Run(Path(), args...)
	if err != nil {
		// En cas d'échec, on se rabat sur un cache périmé s'il existe.
		lines, _ := pm.Cached(cacheName, ttl)
		return lines
	}
	lines := pm.SplitLines(out)
	_ = pm.Store(cacheName, lines)
	return lines
}

// Load construit le catalogue. Formulae/casks sont mis en cache 24 h (ils changent
// rarement) ; les installés sont lus sur le disque à chaque fois — toujours frais, et
// pour un coût négligeable (cf. installedFromDirs).
func Load() *pm.Catalog {
	const day = 24 * time.Hour

	formulae := cachedLines("formulae", day, "formulae")
	casks := cachedLines("casks", day, "casks")

	p := prefix()
	installed := installedFromDirs(filepath.Join(p, "Cellar"), filepath.Join(p, "Caskroom"))
	if len(installed) == 0 {
		// Préfixe inattendu (ou vraiment rien d'installé) : on repasse par brew, lent
		// mais sûr. « --versions » donne « nom version » par installé.
		installed = cachedLines("installed", 0, "list", "--versions")
	}

	return NewCatalog(formulae, casks, installed)
}

// NewCatalog construit un catalogue à partir des trois listes de Homebrew. `installed`
// accepte soit des noms simples (« git »), soit des lignes « nom version » (sortie de
// `brew list --versions`) : le premier champ est le nom, le second (s'il existe) la
// version.
func NewCatalog(formulae, casks, installed []string) *pm.Catalog {
	cat := pm.NewCatalog()
	for _, n := range formulae {
		cat.Add(n, pm.BadgeFormula)
	}
	// Un nom déjà pris par une formula garde son badge : seul un cask *pur* est un cask
	// aux yeux de la complétion — c'est lui, et lui seul, qui réclame `--cask`.
	for _, n := range casks {
		cat.Add(n, pm.BadgeCask)
	}
	for _, line := range installed {
		nom, version, _ := strings.Cut(line, " ")
		cat.MarkInstalled(nom, strings.TrimSpace(version), pm.BadgeFormula)
	}
	cat.Sort()
	return cat
}
