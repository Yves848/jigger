// Package pm définit le contrat que jigger attend d'un gestionnaire de paquets —
// Homebrew sur macOS, winget et scoop sous Windows — et fournit ce que leurs
// implémentations partagent : le dossier de cache, des listes de noms mises en cache,
// et le réchauffement de ces listes en tâche de fond.
//
// La règle qui gouverne tout le paquet : **rien de lent dans le chemin d'un rendu**.
// `jigger render` tourne à chaque frappe ; il ne lit donc que des fichiers de cache ou
// des répertoires, jamais un gestionnaire de paquets. Quand un cache est périmé, on
// s'en contente pour cette frappe-ci et on lance `jigger warm` détaché, qui le refera
// pour la suivante (cf. TriggerWarm).
package pm

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

// Item est un candidat de complétion tel que l'affiche le popup.
type Item struct {
	Name      string
	Badge     string // classe du paquet ; "" pour une sous-commande ou une option
	Installed bool
	Version   string // version installée (vide si non installé/inconnue)
	// PM nomme le gestionnaire d'où vient le candidat, et n'est rempli que par la
	// façade : sur `brew install ⇥`, il n'y a rien à désambiguïser. Le badge ne
	// suffirait pas — BadgeOther est partagé par winget et scoop.
	PM string
}

// PlusieursPM dit si au moins deux gestionnaires distincts figurent dans une liste de
// codes PM (chaînes vides ignorées : elles ne comptent pour aucun gestionnaire — c'est le
// cas du chemin natif, qui ne remplit jamais PM). La colonne PM des tableaux de sortie
// (facade.Formater, §4) et celle du popup (ui.Frame, §5) obéissent à la même règle
// visuelle — « n'apparaît que si plus d'un gestionnaire a contribué » — et l'appliquaient
// jusqu'ici chacune à sa façon, l'une correcte, l'autre non (elle s'affichait dès qu'UN
// item portait un PM, même un seul gestionnaire disponible). Elle vit ici, une fois, pour
// ne pas diverger une troisième fois.
func PlusieursPM(pms []string) bool {
	vu := ""
	for _, p := range pms {
		if p == "" {
			continue
		}
		if vu == "" {
			vu = p
		} else if vu != p {
			return true
		}
	}
	return false
}

// Badges. Chaque gestionnaire range ses paquets en deux classes, que le popup distingue
// par un glyphe et une couleur (cf. internal/ui). Ce qui les sépare varie — nature du
// paquet, provenance — mais la première est toujours le cas ordinaire.
const (
	BadgeFormula = "F" // brew : formula
	BadgeCask    = "C" // brew : cask
	BadgeWinget  = "W" // winget : paquet du catalogue
	BadgeOther   = "X" // winget : installé hors catalogue (ARP/MSIX) ; scoop : autre bucket
	BadgeScoop   = "S" // scoop : bucket main
	BadgeRepo    = "R" // pacman : paquet d'un dépôt binaire (core, extra, multilib…)
	BadgeAUR     = "A" // pacman/yay : paquet de l'AUR, ou installé hors dépôts
)

// Catalog rassemble tout ce que la complétion doit savoir des paquets d'un
// gestionnaire : les noms connus, leur classe, et ceux qui sont installés.
type Catalog struct {
	Names     []string          // noms connus, triés
	Badges    map[string]string // nom -> badge
	Installed map[string]bool   // nom -> installé
	Versions  map[string]string // nom -> version installée
	// Qualified donne, pour les rares noms qui en ont besoin, le texte qui lève une
	// ambiguïté à l'insertion : « extras/flux » quand `flux` existe dans deux buckets.
	Qualified map[string]string
	// Note remplace « aucun candidat » quand le catalogue est vide pour une raison
	// qu'on sait passagère — typiquement : il est en cours de constitution.
	Note string
}

// NewCatalog rend un catalogue vide, prêt à être rempli.
func NewCatalog() *Catalog { return NewCatalogDe(0) }

// NewCatalogDe est NewCatalog dimensionné d'avance pour `n` paquets.
//
// Ce n'est pas une micro-optimisation : une table de hachage Go qui reçoit cent mille
// entrées sans indication de taille les réalloue une vingtaine de fois en chemin, en
// recopiant tout à chaque fois. Sur le catalogue de l'AUR — 118 000 noms, lus à CHAQUE
// frappe (cf. la règle d'ouverture de ce paquet) — la différence se voit à l'œil nu.
// Names est dimensionné pour la même raison.
//
// n est une estimation : trop bas, on retombe simplement sur le comportement d'avant.
func NewCatalogDe(n int) *Catalog {
	return &Catalog{
		Names:     make([]string, 0, n),
		Badges:    make(map[string]string, n),
		Installed: map[string]bool{},
		Versions:  map[string]string{},
		Qualified: map[string]string{},
	}
}

// Add déclare un paquet connu. Un nom déjà déclaré garde son premier badge : les
// gestionnaires listent leurs sources par ordre de priorité.
func (c *Catalog) Add(name, badge string) {
	if name == "" {
		return
	}
	if _, déjà := c.Badges[name]; déjà {
		return
	}
	c.Badges[name] = badge
	c.Names = append(c.Names, name)
}

// MarkInstalled note un paquet comme installé (et le déclare s'il était inconnu, avec
// le badge donné : un paquet peut être installé sans figurer au catalogue).
func (c *Catalog) MarkInstalled(name, version, badge string) {
	if name == "" {
		return
	}
	c.Add(name, badge)
	c.Installed[name] = true
	if version != "" {
		c.Versions[name] = version
	}
}

// Sort trie les noms. À appeler une fois le catalogue rempli : la complétion parcourt
// Names dans l'ordre et le popup l'affiche tel quel.
//
// Le tri ignore la casse. Un tri brut mettrait tous les identifiants winget capitalisés
// avant les autres — « Microsoft.PowerShell » loin devant « mailspring » —, alors que la
// complétion, elle, filtre sans regarder la casse : la liste paraîtrait mélangée.
func (c *Catalog) Sort() { sortFold(c.Names) }

func sortFold(noms []string) {
	sort.Slice(noms, func(i, j int) bool { return LessFold(noms[i], noms[j]) })
}

// LessFold compare deux noms en ignorant la casse (ASCII), sans rien allouer : le tri
// d'un catalogue winget en compte quatorze mille, et il est fait à chaque frappe.
func LessFold(a, b string) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		x, y := bas(a[i]), bas(b[i])
		if x != y {
			return x < y
		}
	}
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b // même nom à la casse près : un ordre stable vaut mieux qu'aucun
}

func bas(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// Badge rend la classe d'un paquet.
func (c *Catalog) Badge(name string) string { return c.Badges[name] }

// Version rend la version installée d'un paquet (vide si non installé/inconnue).
func (c *Catalog) Version(name string) string { return c.Versions[name] }

// InstalledNames rend les paquets installés, triés.
func (c *Catalog) InstalledNames() []string {
	noms := make([]string, 0, len(c.Installed))
	for n := range c.Installed {
		noms = append(noms, n)
	}
	sortFold(noms)
	return noms
}

// Manager est un gestionnaire de paquets vu par jigger. Une implémentation ne fait que
// répondre à des questions : c'est complete qui décide quoi demander, et ui comment
// l'afficher.
type Manager interface {
	// Cmd est le mot de commande — « brew », « winget », « scoop ». C'est lui qui
	// déclenche jigger sur une ligne, et qui nomme le gestionnaire dans le popup.
	Cmd() string

	// Subcommands rend les sous-commandes proposées pour le premier mot.
	Subcommands() []string

	// Options rend les options proposées derrière `-` pour une sous-commande.
	Options(sub string) []string

	// InstalledOnly indique qu'une sous-commande n'accepte que des paquets déjà
	// installés (`brew uninstall`, `winget upgrade`…).
	InstalledOnly(sub string) bool

	// Available dit si le gestionnaire est installé sur cette machine. Seuls ceux-là
	// sont réchauffés ; la complétion, elle, répond pour tous — un catalogue vide est
	// une réponse suffisante.
	Available() bool

	// Load construit le catalogue. Cet appel est dans le chemin du rendu : il ne doit
	// lire que des caches et des répertoires.
	Load() *Catalog

	// Insert rend le texte à insérer pour le candidat retenu. C'est là que vivent les
	// petites corrections qui évitent une commande fautive — `--cask` de brew,
	// qualification par le bucket de scoop.
	Insert(cat *Catalog, sub, prefix, name string) string

	// Warm reconstitue les caches lents du gestionnaire. Appelé par `jigger warm`,
	// toujours hors du chemin du rendu.
	Warm(Scope) error
}

// Scope dit à Warm ce qu'il doit refaire. La distinction n'est pas cosmétique : après
// une installation, seule la liste des paquets installés a changé — refaire un
// catalogue de quatorze mille noms, qui bouge de trois entrées par jour, coûterait
// quelques secondes pour rien.
type Scope int

const (
	ScopeStale     Scope = iota // ce qui est périmé, et rien d'autre
	ScopeInstalled              // les paquets installés, périmés ou non
	ScopeAll                    // tout, périmé ou non
)

// CacheDir rend le dossier de cache de jigger (créé si besoin) : $JIGGER_CACHE_DIR s'il
// est défini, sinon <cache utilisateur>/jigger — ~/Library/Caches/jigger sur macOS,
// %LOCALAPPDATA%\jigger sous Windows, ~/.cache/jigger ailleurs.
//
// La surcharge sert aux tests, mais aussi aux hooks de prompt : ceux-ci doivent désigner
// le même fichier sans avoir le droit de lancer un processus pour le demander.
func CacheDir() string {
	dir := os.Getenv("JIGGER_CACHE_DIR")
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "jigger")
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Cached relit une liste de noms mise en cache. `fresh` dit si elle a moins de `ttl` :
// une liste périmée est tout de même rendue — mieux vaut compléter sur un catalogue
// d'hier que sur rien — mais l'appelant est censé demander un réchauffement.
func Cached(name string, ttl time.Duration) (lines []string, fresh bool) {
	path := filepath.Join(CacheDir(), name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return SplitLines(data), time.Since(info.ModTime()) < ttl
}

// Store écrit une liste de noms dans le cache, de façon atomique : un rendu concurrent
// lit soit l'ancienne liste, soit la nouvelle, jamais un fichier à moitié écrit.
func Store(name string, lines []string) error {
	dir := CacheDir()
	tmp, err := os.CreateTemp(dir, "."+name+"-*")
	if err != nil {
		return err
	}
	nom := tmp.Name()
	defer os.Remove(nom) // sans effet après un Rename réussi

	w := bufio.NewWriter(tmp)
	for _, l := range lines {
		w.WriteString(l)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(nom, filepath.Join(dir, name))
}

// SplitLines découpe une sortie en lignes non vides, débarrassées de leurs espaces (et
// des retours chariot que Windows laisse traîner).
func SplitLines(data []byte) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// intervalleWarm espace deux réchauffements détachés. Sans lui, chaque frappe d'une
// rafale en lancerait un : le premier n'a pas fini d'écrire son cache que le suivant le
// trouve encore périmé.
const intervalleWarm = 60 * time.Second

// TriggerWarm lance `jigger warm` détaché, si le dernier date d'assez longtemps. Le
// processus fils n'hérite d'aucun descripteur : la sortie de `jigger render` est
// capturée par le shell, et un tuyau qu'un fils garde ouvert fait attendre la
// substitution de commande — donc le shell — jusqu'à ce que le fils meure.
//
// L'échec est silencieux : au pire, le catalogue reste celui d'hier.
func TriggerWarm() {
	stamp := filepath.Join(CacheDir(), "warm.stamp")
	if info, err := os.Stat(stamp); err == nil && time.Since(info.ModTime()) < intervalleWarm {
		return
	}
	now := time.Now()
	if f, err := os.OpenFile(stamp, os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.Close()
		_ = os.Chtimes(stamp, now, now)
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "warm")
	detach(cmd) // pas de console qui clignote sous Windows, pas de terminal de contrôle ailleurs
	if err := cmd.Start(); err == nil {
		go cmd.Wait() // récolte le fils sans l'attendre : jigger render sort tout de suite
	}
}

// PeremptionVerrou : au-delà, un verrou est réputé abandonné (processus tué avant
// d'avoir pu le retirer) et n'empêche plus de travailler.
const PeremptionVerrou = 5 * time.Minute

// Lock prend un verrou de travail de fond. Il rend la fonction de libération et un
// booléen : faux si un autre processus tient déjà le verrou.
//
// C'est ce qui empêche dix terminaux ouverts — ou une rafale de frappes — de lancer dix
// fois le même travail lent.
func Lock(chemin string) (func(), bool) {
	f, err := os.OpenFile(chemin, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Verrou déjà pris — sauf s'il traîne depuis assez longtemps pour que son
		// propriétaire soit certainement mort.
		info, serr := os.Stat(chemin)
		if serr != nil || time.Since(info.ModTime()) < PeremptionVerrou {
			return nil, false
		}
		if err := os.Remove(chemin); err != nil {
			return nil, false
		}
		if f, err = os.OpenFile(chemin, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err != nil {
			return nil, false
		}
	}
	f.Close()

	return func() { os.Remove(chemin) }, true
}

// WarmLock est le verrou de `jigger warm`.
func WarmLock() string { return filepath.Join(CacheDir(), "warm.lock") }

// Run exécute une commande et rend sa sortie standard. La sortie d'erreur est ignorée :
// winget comme brew bavardent volontiers sans que ce soit une panne.
func Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	hide(cmd) // aucune fenêtre de console ne doit apparaître
	return cmd.Output()
}
