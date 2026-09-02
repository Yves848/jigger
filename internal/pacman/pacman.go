// Package pacman branche jigger sur pacman et sur yay — le premier gestionnaire Linux de
// jigger (conception du 2 septembre 2026).
//
// Il se distingue des trois autres par un point de grammaire : **les opérations de pacman
// sont des drapeaux**. Là où brew écrit `brew install`, pacman écrit `pacman -S`. Le moteur
// de complétion, lui, route sur `strings.HasPrefix(word, "-")` : tout ce qui commence par
// un tiret part chercher des options, le reste des sous-commandes. Une opération de pacman
// tombe donc du mauvais côté de ce test.
//
// La résolution ne demande rien au moteur : le fournisseur déclare **la même liste des deux
// côtés**. Subcommands() rend les opérations (`pacman ⇥`), et Options("") les rend aussi
// (`pacman -⇥`). Options d'une opération donnée, en revanche, rend ses drapeaux
// secondaires. Les tables sont indexées en minuscules parce que complete minuscule la
// sous-commande avant de la passer.
package pacman

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/config"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Manager implémente pm.Manager pour pacman et pour yay.
//
// Deux fournisseurs plutôt qu'un à deux noms, sur le patron de ssh/scp/sftp : Manager.Cmd()
// ne rend qu'un mot. Ils partagent l'implémentation, le cache des dépôts et la lecture de
// la base locale ; ce qui les sépare tient à trois choses — le catalogue AUR, quelques
// opérations en plus, et la table des verbes (cf. ADR-0007).
type Manager struct{ cmd string }

// New rend le fournisseur pour l'un des deux mots de commande.
func New(cmd string) Manager { return Manager{cmd: cmd} }

func (m Manager) Cmd() string { return m.cmd }

// Opérations de pacman, dans l'ordre où le popup les montre : la famille -S d'abord (celle
// qu'on tape le plus), puis -R, puis -Q, puis les marginales.
//
// La liste est curée, pas exhaustive : `-Sy` y figure parce qu'on le tape, `-Syyuu` non.
// Les combinaisons usuelles sont proposées telles quelles (`-Syu`, `-Rns`, `-Qdt`) plutôt
// que laissées à composer — c'est précisément ce qu'un débutant sous Arch cherche.
var operations = []string{
	"-S", "-Syu", "-Sy", "-Ss", "-Si", "-Sw", "-Sc",
	"-R", "-Rns", "-Rs", "-Rdd",
	"-Q", "-Qu", "-Qi", "-Ql", "-Qo", "-Qs", "-Qe", "-Qdt", "-Qm",
	"-U", "-F", "-Fy",
	"--help", "--version",
}

// Opérations propres à yay, ajoutées après celles de pacman : yay accepte tout pacman, et
// y ajoute les siennes.
var operationsYay = []string{"-Y", "-Yc", "-P", "-Ps", "-G"}

// Drapeaux acceptés partout. `--noconfirm` y est plutôt que dans chaque opération mutante :
// pacman le comprend sur toutes.
var optionsCommunes = []string{"--help", "--noconfirm", "--color", "--verbose", "--quiet"}

// Drapeaux secondaires par opération. Clés en minuscules : complete minuscule la
// sous-commande avant de la passer (`-Rns` arrive en « -rns »).
var optionsParOp = map[string][]string{
	"-s":   {"--needed", "--asdeps", "--asexplicit", "--nodeps", "--overwrite", "--downloadonly", "--ignore"},
	"-syu": {"--needed", "--ignore", "--overwrite", "--downloadonly"},
	"-sy":  {"--needed", "--ignore"},
	"-sw":  {"--needed"},
	"-r":   {"--nosave", "--recursive", "--unneeded", "--cascade", "--nodeps"},
	"-rns": {"--recursive", "--unneeded", "--cascade"},
	"-rs":  {"--nosave", "--unneeded", "--cascade"},
	"-rdd": {"--nosave", "--cascade"},
	"-q":   {"--explicit", "--deps", "--unrequired", "--upgrades", "--foreign", "--native", "--info", "--list", "--owns"},
	"-u":   {"--asdeps", "--overwrite", "--nodeps"},
	"-f":   {"--refresh", "--list", "--owns"},
	"-fy":  {"--refresh"},
}

// Drapeaux que yay ajoute aux opérations de la famille -S : ce sont eux qui décident si
// l'on cherche dans les dépôts, dans l'AUR, ou dans les deux.
var optionsYayParOp = map[string][]string{
	"-s":   {"--aur", "--repo", "--rebuild", "--devel", "--nodiffmenu", "--noeditmenu", "--answerclean", "--answerdiff"},
	"-syu": {"--aur", "--repo", "--devel", "--nodiffmenu", "--noeditmenu"},
	"-ss":  {"--aur", "--repo", "--sortby", "--topdown"},
	"-si":  {"--aur", "--repo"},
	"-y":   {"--gendb"},
	"-p":   {"--stats", "--news", "--numberupgrades"},
}

// Opérations dont les arguments sont des paquets déjà installés : toute la famille -R, et
// toute la famille -Q — ce sont exactement celles qui interrogent la base locale.
//
// `-Qo` en est absent volontairement : son argument est un **chemin de fichier**, pas un
// nom de paquet. Le catalogue est un mauvais vivier pour lui, la liste des installés en
// serait un pire.
var installedOnly = map[string]bool{
	"-r": true, "-rns": true, "-rs": true, "-rdd": true,
	"-q": true, "-qu": true, "-qi": true, "-ql": true, "-qs": true,
	"-qe": true, "-qdt": true, "-qm": true,
}

func (m Manager) Subcommands() []string {
	if m.cmd == "yay" {
		return append(append([]string{}, operations...), operationsYay...)
	}
	return operations
}

// Options rend les drapeaux secondaires d'une opération — ou, quand aucune opération n'est
// encore posée, les opérations elles-mêmes. C'est ce second cas qui fait marcher
// « pacman -⇥ » : le mot commence par un tiret, donc complete vient ici, alors que ce que
// l'utilisateur tape est bien une opération.
func (m Manager) Options(sub string) []string {
	if sub == "" {
		return m.Subcommands()
	}
	op := strings.ToLower(sub)
	out := append([]string{}, optionsParOp[op]...)
	if m.cmd == "yay" {
		out = append(out, optionsYayParOp[op]...)
	}
	return append(out, optionsCommunes...)
}

func (Manager) InstalledOnly(sub string) bool { return installedOnly[strings.ToLower(sub)] }

// Available dit si le binaire est sur la machine.
func (m Manager) Available() bool { return Present(m.cmd) }

// Present dit si un binaire est installé. Il est appelé dans le chemin d'un rendu (la
// façade demande managers.Available() à chaque frappe), d'où le stat direct avant le
// parcours du PATH : sur Arch, les deux binaires sont dans /usr/bin.
func Present(cmd string) bool { return filepath.IsAbs(Path(cmd)) }

// Path rend le chemin d'un binaire, ou le mot lui-même en dernier recours — auquel cas
// Present rend faux, comme le fait brew.Path.
func Path(cmd string) string {
	p := filepath.Join("/usr/bin", cmd)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if p, err := exec.LookPath(cmd); err == nil {
		return p
	}
	return cmd
}

func (m Manager) Load() *pm.Catalog { return Charger(m.cmd) }

// Insert qualifie par son dépôt le nom que les dépôts et l'AUR portent tous les deux.
//
// Sans cela, `yay -S <nom>` ouvre un menu interactif au milieu de ce que jigger vient
// d'insérer, pour demander laquelle des deux sources on voulait. C'est la correction de
// scoop, au mot près : le catalogue n'a retenu qu'une entrée, celle du dépôt (Fusionner
// range les dépôts en premier), donc c'est le dépôt qu'on nomme.
//
// pacman n'a jamais besoin de rien : quand deux dépôts portent le même nom, il tranche par
// ordre de priorité, sans erreur ni question. Seule la famille -S est corrigée — un `-R`
// porte sur un paquet installé, où il n'y a plus rien à départager.
func (m Manager) Insert(cat *pm.Catalog, sub, _, name string) string {
	if m.cmd != "yay" {
		return name
	}
	op := strings.ToLower(sub)
	if !strings.HasPrefix(op, "-s") && op != "install" {
		return name
	}
	if q := cat.Qualified[name]; q != "" {
		return q
	}
	return name
}

// Warm reconstitue les listes brutes, puis en dépose la **fusion triée** — c'est cette
// seconde étape qui tient le budget de la frappe (cf. l'ouverture de catalog.go).
//
// Les paquets installés ne sont pas réchauffés : ils se lisent dans /var/lib/pacman/local
// en moins d'une milliseconde, donc toujours frais et sans cache à tenir (cf. Charger).
// C'est le cas de scoop, pas celui de brew, qui garde un recours par `brew list`.
func (m Manager) Warm(scope pm.Scope) error {
	if scope == pm.ScopeInstalled {
		return nil // lus sur le disque : rien à refaire
	}
	ttl, ttlAUR := durees()
	if scope == pm.ScopeAll {
		ttl, ttlAUR = 0, 0
	}

	// Les dépôts par pacman même sous yay : les deux fournisseurs partagent la même base
	// alpm, donc la même liste.
	sync := lignesSync(ttl)
	var aur []string
	if m.cmd == "yay" {
		aur = cachedLines(cacheAUR, ttlAUR, "yay", "-Slq", "aur")
	}
	return pm.Store(fichierCatalogue(m.cmd), Fusionner(sync, aur))
}

func durees() (depots, aur time.Duration) {
	return config.Duree("pacman_ttl", 24*time.Hour), config.Duree("aur_ttl", 24*time.Hour)
}

// lignesSync rend la sortie de `pacman -Sl`, une seule fois par processus.
//
// `jigger warm` appelle Warm sur les DEUX fournisseurs, et `warm --all` met le TTL à zéro
// — donc le cache brut n'arrête plus personne, et la commande de 166 ms tournerait deux
// fois. Ce n'est pas théorique : le greffon zsh lance justement `warm --all` après chaque
// `pacman -Sy`. Le TTL retenu est celui du premier appelant ; runWarm les appelle tous
// avec la même portée, donc la même durée.
var (
	syncUneFois sync.Once
	syncLignes  []string
)

func lignesSync(ttl time.Duration) []string {
	syncUneFois.Do(func() { syncLignes = cachedLines(cacheSync, ttl, "pacman", "-Sl") })
	return syncLignes
}

// Les deux réglages du module. Comme chez brew et winget, c'est le gestionnaire qui les
// déclare : l'écran de configuration en dérive sa mise en page sans rien savoir de pacman.
func init() {
	config.Declarer(config.Reglage{
		Cle: "pacman_ttl", CleI18n: "cfg.ttl", Portee: config.Binaire,
		Type: config.TypeDuree, Defaut: "24h", PM: "pacman",
	})
	config.Declarer(config.Reglage{
		Cle: "aur_ttl", CleI18n: "cfg.ttl_aur", Portee: config.Binaire,
		Type: config.TypeDuree, Defaut: "24h", PM: "yay",
	})
}
