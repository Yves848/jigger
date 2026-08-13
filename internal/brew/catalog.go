// Package brew récupère (et met en cache) les noms connus de Homebrew : toutes les
// formulae, tous les casks, et les paquets installés. Ces listes alimentent la
// complétion contextuelle.
package brew

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Catalog contient les ensembles de noms nécessaires à la complétion.
type Catalog struct {
	Formulae  []string
	Casks     []string
	Installed map[string]bool   // nom -> installé
	Versions  map[string]string // nom -> version installée (vide si inconnue)
	formulaeM map[string]bool
	casksM    map[string]bool
}

// Version renvoie la version installée d'un paquet (vide si non installé/inconnue).
func (c *Catalog) Version(name string) string { return c.Versions[name] }

// IsCask indique si un nom est un cask connu.
func (c *Catalog) IsCask(name string) bool { return c.casksM[name] }

// IsFormula indique si un nom est une formula connue.
func (c *Catalog) IsFormula(name string) bool { return c.formulaeM[name] }

// brewPath renvoie le chemin du binaire brew (Apple Silicon puis Intel).
func brewPath() string {
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
	return filepath.Dir(filepath.Dir(brewPath())) // …/bin/brew -> …
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

// cacheDir renvoie ~/.cache/jigger (créé si besoin).
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "jigger")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// cachedLines renvoie les lignes de `brew <args>`, mises en cache <ttl>. Le fichier
// de cache accélère massivement les invocations répétées (chaque appel du widget).
func cachedLines(cacheName string, ttl time.Duration, args ...string) []string {
	path := filepath.Join(cacheDir(), cacheName)
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < ttl {
		if data, err := os.ReadFile(path); err == nil {
			return splitLines(data)
		}
	}

	out, err := exec.Command(brewPath(), args...).Output()
	if err != nil {
		// En cas d'échec, on se rabat sur un cache périmé s'il existe.
		if data, rerr := os.ReadFile(path); rerr == nil {
			return splitLines(data)
		}
		return nil
	}
	_ = os.WriteFile(path, out, 0o644)
	return splitLines(out)
}

func splitLines(data []byte) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// Load construit le catalogue. Formulae/casks sont mis en cache 24 h (ils changent
// rarement) ; les installés sont lus sur le disque à chaque fois — toujours frais, et
// pour un coût négligeable (cf. installedFromDirs).
func Load() *Catalog {
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

	return NewCatalogVersions(formulae, casks, installed)
}

// NewCatalog construit un catalogue à partir de listes de noms (sans versions).
// Conservé pour les tests qui n'ont pas besoin des versions.
func NewCatalog(formulae, casks, installed []string) *Catalog {
	return NewCatalogVersions(formulae, casks, installed)
}

// NewCatalogVersions construit un catalogue. `installed` accepte soit des noms simples
// (« git »), soit des lignes « nom version » (sortie de `brew list --versions`) : le
// premier champ est le nom, le second (s'il existe) la version.
func NewCatalogVersions(formulae, casks, installed []string) *Catalog {
	c := &Catalog{
		Formulae:  formulae,
		Casks:     casks,
		Installed: make(map[string]bool, len(installed)),
		Versions:  make(map[string]string, len(installed)),
		formulaeM: make(map[string]bool, len(formulae)),
		casksM:    make(map[string]bool, len(casks)),
	}
	for _, n := range formulae {
		c.formulaeM[n] = true
	}
	for _, n := range casks {
		c.casksM[n] = true
	}
	for _, line := range installed {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		c.Installed[name] = true
		if len(fields) > 1 {
			c.Versions[name] = fields[1]
		}
	}
	return c
}
