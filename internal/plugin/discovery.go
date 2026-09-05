// Package plugin découvre et charge des gestionnaires de paquets tiers définis par un
// descripteur JSON (config.json) sur le disque.
//
// jigger parcourt les répertoires suivants à la recherche de descripteurs valides :
//
//	~/.config/jigger/plugins/<nom>/config.json   (utilisateur, non versionné)
//	/usr/local/lib/jigger-plugins/<nom>/         (système, lecture seule)
//	<cache jigger>/plugins/<nom>/                (installés par un tiers)
//
// Chaque sous-dossier portant un `config.json` valide est un plugin candidat. Si le
// binaire désigné par `cmd` est introuvable, le plugin est ignoré — Available() rend
// false, et il sera repris à la prochaine invocation s'il apparaît plus tard.
//
// Un plugin n'est jamais lancé dans le chemin du rendu : Load() ne lit que des fichiers
// de cache, exactement comme un gestionnaire natif. Le sous-processus n'existe qu'au
// réchauffement (Warm) et à l'exécution d'un verbe (cf. facade.ExecuterAvec).
package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// ttlCatalogue est la durée au-delà de laquelle un cache de plugin est réputé périmé. Un
// cache périmé reste servi — mieux vaut compléter sur le catalogue d'hier que sur rien —
// mais il déclenche un réchauffement détaché (même règle que brew et winget).
const ttlCatalogue = 24 * time.Hour

// Config est la structure lue depuis un descripteur de plugin.
type Config struct {
	Name     string               `json:"name"`
	Version  string               `json:"version"`
	Cmd      string               `json:"cmd"`
	Platform []string             `json:"platforms"`
	Verbs    map[string]Verb      `json:"verbs"`
	Pools    map[string]Vivier    `json:"pools"`
	Warmup   map[string]WarmupCmd `json:"warmup"`
	Parse    Parse                `json:"parse"`
}

// WarmupCmd décrit une commande de réchauffement. `Cmd` désigne le binaire à lancer — il
// vaut celui du plugin quand il est omis — et `Args` porte l'argv complet.
type WarmupCmd struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

// Verb décrit un verbe supporté par le plugin. `Native` est l'argv **complet** passé au
// binaire du plugin, marqueurs pm.MarqueurUn / pm.MarqueurTous compris : jigger n'y
// ajoute rien.
type Verb struct {
	Native []string `json:"native"`

	// Pool nomme le vivier où puiser les candidats de ce verbe : une des trois valeurs
	// historiques — "catalogue", "installees", "aucun" — ou le nom d'une entrée de
	// Config.Pools (ADR-0009).
	Pool string `json:"pool"`

	// Options sont les drapeaux proposés derrière `-` pour ce verbe. Sans elles, un
	// plugin ne pouvait rien proposer là où un natif déclare ses options par
	// sous-commande — c'était la moitié de ce qui manquait à un helper de commande.
	Options []string `json:"options"`
}

// Vivier est une source de candidats nommée, déclarée par le descripteur (ADR-0009).
//
// Deux régimes, et le choix n'est pas de commodité :
//
//   - "cache" — le vivier est réchauffé par `jigger warm` et lu depuis le disque. C'est
//     ce qu'il faut d'un catalogue : gros, lent à produire, stable d'une heure à l'autre.
//   - "direct" — le vivier est demandé AU MOMENT DE LA FRAPPE, dans le répertoire
//     courant. C'est ce qu'il faut de candidats petits et contextuels — les branches d'un
//     dépôt, les fichiers modifiés — qu'un cache rendrait faux plutôt que rapides.
//
// Le binaire interrogé est TOUJOURS celui du plugin : le descripteur ne nomme que les
// arguments. Un descripteur ne doit pas pouvoir faire lancer n'importe quel programme à
// chaque frappe — c'est la contrainte 4 du plan d'injection, et le régime direct la rend
// d'autant plus sensible qu'il s'exécute sans que l'utilisateur ait rien lancé.
type Vivier struct {
	Regime string   `json:"regime"` // "cache" ou "direct"
	Args   []string `json:"args"`
}

// Parse dit comment interpréter la sortie JSON du plugin.
type Parse struct {
	Fields       []string `json:"package_fields"`
	Encoding     string   `json:"encoding"`      // toujours utf-8, présent pour validation
	CatalogField string   `json:"catalog_field"` // "names" si omis
	BadgeField   string   `json:"badge_field"`   // "badges" si omis
}

// discoverPaths rend les répertoires à parcourir lors de la découverte. L'ordre importe :
// le premier descripteur trouvé pour un nom donné gagne, et c'est l'utilisateur qui doit
// pouvoir supplanter un plugin système.
func discoverPaths() []string {
	var paths []string

	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		if home, _ := os.UserHomeDir(); home != "" {
			xdg = filepath.Join(home, ".config")
		}
	}
	if xdg != "" {
		paths = append(paths, filepath.Join(xdg, "jigger", "plugins"))
	}

	paths = append(paths, "/usr/local/lib/jigger-plugins")

	if cache := pm.CacheDir(); cache != "" {
		paths = append(paths, filepath.Join(cache, "plugins"))
	}

	return paths
}

// Discover renvoie les plugins trouvés, valides et disponibles. Un descripteur illisible
// ou mal formé est ignoré en silence : un plugin cassé ne doit pas empêcher jigger de
// compléter une ligne (ADR-0006, même esprit).
func Discover() []*PluginManager {
	var out []*PluginManager
	vus := map[string]bool{}

	for _, dir := range discoverPaths() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // répertoire absent ou non lisible : on passe
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			racine := filepath.Join(dir, e.Name())
			cfg, err := loadConfig(filepath.Join(racine, "config.json"))
			if err != nil {
				continue // descripteur invalide : skip
			}
			if vus[cfg.Name] {
				continue // déjà servi par un répertoire plus prioritaire
			}
			m := NewPluginManager(cfg, racine)
			if !m.Available() {
				continue
			}
			vus[cfg.Name] = true
			out = append(out, m)
		}
	}

	return out
}

// loadConfig lit et valide un descripteur JSON.
func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if err := validate(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// validate vérifie qu'un descripteur est bien formé et complète ses valeurs par défaut.
// Elle prend un pointeur : les défauts qu'elle pose doivent parvenir à l'appelant.
func validate(cfg *Config) error {
	if cfg.Name == "" {
		return errors.New("plugin : le champ name est obligatoire")
	}
	if cfg.Cmd == "" {
		return errors.New("plugin : le champ cmd est obligatoire")
	}
	if len(cfg.Verbs) == 0 {
		return errors.New("plugin : au moins un verbe est requis")
	}
	for k, v := range cfg.Verbs {
		if len(v.Native) == 0 {
			return errors.New("plugin : le verbe " + k + " n'a pas de commande native")
		}
		switch v.Pool {
		case "catalogue", "installees", "aucun":
			// Valeurs historiques : contrat public depuis la 0.16.0, elles restent.
		default:
			// Sinon, le pool doit désigner un vivier déclaré. Un nom qui ne correspond à
			// rien est une faute de frappe silencieuse : le verbe ne proposerait jamais
			// rien, et personne ne saurait pourquoi.
			if _, ok := cfg.Pools[v.Pool]; !ok {
				return errors.New("plugin : le verbe " + k + " puise dans un vivier non déclaré : " + v.Pool)
			}
		}
	}
	for nom, vi := range cfg.Pools {
		switch vi.Regime {
		case "cache", "direct":
			// valide
		default:
			return errors.New("plugin : le vivier " + nom + " a un régime inconnu : " + vi.Regime)
		}
		if len(vi.Args) == 0 {
			return errors.New("plugin : le vivier " + nom + " n'a pas d'arguments")
		}
	}
	if len(cfg.Parse.Fields) == 0 {
		cfg.Parse.Fields = []string{"name", "version", "kind", "source"}
	}
	if cfg.Parse.CatalogField == "" {
		cfg.Parse.CatalogField = "names"
	}
	if cfg.Parse.BadgeField == "" {
		cfg.Parse.BadgeField = "badges"
	}
	return nil
}

// poolFromString convertit la chaîne du descripteur en pm.Pool.
func poolFromString(s string) pm.Pool {
	switch s {
	case "catalogue":
		return pm.PoolCatalogue
	case "installees":
		return pm.PoolInstalles
	default:
		return pm.PoolAucun
	}
}

// cleSure rend un nom de fichier de cache sûr à partir d'un nom de plugin. pm.Store écrit
// par os.CreateTemp, qui refuse tout motif contenant un séparateur : un nom de plugin
// exotique ne doit pas rendre le cache inécrivable — en silence, qui plus est.
func cleSure(nom string) string {
	var b strings.Builder
	for _, r := range nom {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// PluginManager implémente pm.Manager et pm.Bindings pour un plugin tiers.
type PluginManager struct {
	cfg     Config
	dir     string // dossier du plugin (celui qui porte config.json)
	cle     string // préfixe des fichiers de cache, sans séparateur
	binaire string // chemin résolu du binaire, vide s'il est introuvable
}

// NewPluginManager crée un gestionnaire de plugin à partir d'un descripteur et du dossier
// dont il vient.
func NewPluginManager(cfg Config, dir string) *PluginManager {
	m := &PluginManager{
		cfg: cfg,
		dir: dir,
		cle: "plugin-" + cleSure(cfg.Name),
	}
	m.binaire = m.resoudre(cfg.Cmd)
	return m
}

// resoudre cherche un binaire de plugin. Le dossier du plugin passe avant le PATH : c'est
// le seul chemin que le descripteur contrôle, et le plan de sécurité veut qu'un plugin
// livré avec son binaire ne dépende pas de ce qui traîne dans le PATH. Le repli sur le
// PATH reste nécessaire — l'installation documentée place le binaire dans ~/.local/bin.
func (m *PluginManager) resoudre(cmd string) string {
	if cmd == "" {
		return ""
	}
	if filepath.IsAbs(cmd) {
		if executable(cmd) {
			return cmd
		}
		return ""
	}
	if m.dir != "" {
		if p := filepath.Join(m.dir, cmd); executable(p) {
			return p
		}
	}
	if p, err := exec.LookPath(cmd); err == nil {
		return p
	}
	return ""
}

// executable dit si un chemin désigne un fichier ordinaire exécutable.
func executable(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return false
	}
	return fi.Mode()&0o111 != 0
}

// Cmd est le mot de commande du plugin — celui que l'utilisateur tape. Ce n'est **pas**
// le binaire : cf. Binaire.
func (m *PluginManager) Cmd() string { return m.cfg.Name }

// Binaire rend le chemin du programme à lancer pour ce gestionnaire, et false si ce n'en
// est pas un ou s'il est introuvable. La façade s'en sert pour lancer le bon exécutable
// sans rien savoir des plugins.
func Binaire(m pm.Manager) (string, bool) {
	p, ok := m.(*PluginManager)
	if !ok || p.binaire == "" {
		return "", false
	}
	return p.binaire, true
}

// Subcommands rend les verbes de premier niveau déclarés par le plugin, triés.
func (m *PluginManager) Subcommands() []string {
	vu := map[string]bool{}
	for v := range m.cfg.Verbs {
		premier, _, _ := strings.Cut(v, " ")
		vu[premier] = true
	}
	out := make([]string, 0, len(vu))
	for k := range vu {
		out = append(out, k)
	}
	trierMots(out)
	return out
}

// VerbesExhaustifs : oui, toujours. Les verbes d'un plugin viennent de son `config.json`
// et de nulle part ailleurs — jigger n'en connaît pas d'autres, et le plugin n'en exécutera
// pas d'autres. La liste rendue par Subcommands est donc un inventaire, pas une sélection,
// ce qui autorise la complétion à se taire sur tout le reste plutôt que d'y proposer des
// paquets. C'est ce qui distingue un plugin d'un natif comme brew (cf. pm.Exhaustif, #141).
func (*PluginManager) VerbesExhaustifs() bool { return true }

// trierMots range des mots par ordre alphabétique. La liste tient en quelques entrées :
// une insertion suffit, et évite d'importer sort pour cela.
func trierMots(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Options rend les options proposées derrière `-`. Les descripteurs ne les déclarent pas
// encore (phase P4 du plan) : un plugin ne propose donc aucune option.
// Options rend les drapeaux déclarés pour ce verbe. Le descripteur en est la seule
// source : jigger n'a aucun moyen de deviner ce qu'une commande tierce accepte.
func (m *PluginManager) Options(sub string) []string {
	return m.cfg.Verbs[strings.ToLower(sub)].Options
}

// Candidats rend le vivier propre à un verbe, et false si ce verbe n'en déclare pas —
// auquel cas la complétion retombe sur le catalogue, comme avant l'ADR-0009.
//
// Le second retour n'est pas décoratif : il distingue « ce verbe n'a pas de vivier » de
// « le vivier a répondu vide ». Le premier fait retomber sur le catalogue, le second doit
// afficher zéro candidat — confondre les deux ferait proposer tout le catalogue derrière
// `git checkout` d'un dépôt sans branche.
func (m *PluginManager) Candidats(sub string) (*pm.Catalog, bool) {
	v, ok := m.cfg.Verbs[strings.ToLower(sub)]
	if !ok {
		return nil, false
	}
	vi, ok := m.cfg.Pools[v.Pool]
	if !ok || vi.Regime != "direct" {
		return nil, false // vivier historique, ou réchauffé : Load() s'en charge
	}
	if m.binaire == "" {
		return nil, false
	}

	// Le répertoire courant est celui du shell : jigger y est lancé à chaque frappe, et
	// le plugin en hérite sans qu'on ait à le lui passer. C'est ce qui rend le vivier
	// contextuel.
	sortie, err := runDelai(m.binaire, vi.Args, delaiVivier())
	if err != nil {
		// Un vivier qui échoue ou qui traîne ne dit rien, et ne dit pas pourquoi : on est
		// dans le chemin du rendu, où une ligne d'erreur par frappe serait pire que le
		// silence (ADR-0006).
		return pm.NewCatalog(), true
	}

	cat := pm.NewCatalog()
	for _, ligne := range strings.Split(string(sortie), "\n") {
		nom, badge, _ := strings.Cut(strings.TrimRight(ligne, "\r"), "\t")
		if nom = strings.TrimSpace(nom); nom == "" {
			continue
		}
		cat.Add(nom, strings.TrimSpace(badge))
	}
	cat.Sort()
	return cat, true
}

// InstalledOnly indique qu'une sous-commande n'accepte que des paquets déjà installés.
func (m *PluginManager) InstalledOnly(sub string) bool {
	v, ok := m.cfg.Verbs[sub]
	return ok && v.Pool == "installees"
}

// Available dit si le binaire du plugin a été trouvé, et si le plugin se déclare sur
// cette plateforme.
func (m *PluginManager) Available() bool {
	if m.binaire == "" {
		return false
	}
	return m.surCettePlateforme()
}

// surCettePlateforme lit le champ `platforms`. Une liste vide vaut « partout ».
func (m *PluginManager) surCettePlateforme() bool {
	if len(m.cfg.Platform) == 0 {
		return true
	}
	for _, p := range m.cfg.Platform {
		if p == runtime.GOOS {
			return true
		}
	}
	return false
}

// Load construit le catalogue à partir des seuls fichiers de cache. Cet appel est dans le
// chemin du rendu : il ne lance rien.
func (m *PluginManager) Load() *pm.Catalog {
	cat := pm.NewCatalog()

	catalogue, catFrais := pm.Cached(m.cle+"-catalog", ttlCatalogue)
	installes, insFrais := pm.Cached(m.cle+"-installed", ttlCatalogue)
	if !catFrais || !insFrais {
		pm.TriggerWarm()
	}

	// Catalogue : « nom<TAB>badge ».
	for _, ligne := range catalogue {
		nom, badge, _ := strings.Cut(ligne, "\t")
		if nom == "" {
			continue
		}
		cat.Add(nom, badge)
	}

	// Installés : « nom<TAB>badge<TAB>version<TAB>source ».
	for _, ligne := range installes {
		champs := strings.Split(ligne, "\t")
		if champs[0] == "" {
			continue
		}
		badge, version := "", ""
		if len(champs) > 1 {
			badge = champs[1]
		}
		if len(champs) > 2 {
			version = champs[2]
		}
		cat.MarkInstalled(champs[0], version, badge)
	}

	cat.Sort()
	return cat
}

// Insert rend le nom tel quel : un plugin tiers n'a pas le pouvoir de réécrire la ligne
// (décision §9 du plan — les corrections d'insertion sont des bugs de façade).
func (m *PluginManager) Insert(_ *pm.Catalog, _, _, name string) string { return name }

// Warm reconstitue les caches du plugin en lançant les commandes `warmup` déclarées.
// C'est, avec l'exécution d'un verbe, le seul endroit où un sous-processus tiers tourne.
func (m *PluginManager) Warm(scope pm.Scope) error {
	if scope == pm.ScopeInstalled {
		return m.warmInstalled()
	}

	libere, ok := pm.Lock(filepath.Join(pm.CacheDir(), m.cle+".lock"))
	if !ok {
		return nil // un autre réchauffement de ce plugin est déjà en cours
	}
	defer libere()

	// Les deux caches sont indépendants : l'échec du catalogue ne doit pas priver la
	// complétion de la liste des installés, qui est la plus utile des deux.
	errCat := m.warmCatalog()
	errIns := m.warmInstalled()
	if errCat != nil {
		return errCat
	}
	return errIns
}

// warmCatalog rejoue la commande `warmup.catalog` et range son JSON en « nom<TAB>badge ».
func (m *PluginManager) warmCatalog() error {
	out, err := m.warmup("catalog")
	if err != nil || out == nil {
		return err
	}

	var catalogue struct {
		Names  []string          `json:"names"`
		Badges map[string]string `json:"badges"`
	}
	if err := json.Unmarshal(out, &catalogue); err != nil {
		return err
	}

	lignes := make([]string, 0, len(catalogue.Names))
	for _, n := range catalogue.Names {
		lignes = append(lignes, n+"\t"+catalogue.Badges[n])
	}
	return pm.Store(m.cle+"-catalog", lignes)
}

// warmInstalled rejoue `warmup.installed` et range son JSON en
// « nom<TAB>badge<TAB>version<TAB>source ».
func (m *PluginManager) warmInstalled() error {
	out, err := m.warmup("installed")
	if err != nil || out == nil {
		return err
	}

	paquets, err := parsePluginOutput(out)
	if err != nil {
		return err
	}

	lignes := make([]string, 0, len(paquets))
	for _, p := range paquets {
		lignes = append(lignes, strings.Join([]string{p.Name, p.Kind, p.Version, p.Source}, "\t"))
	}
	return pm.Store(m.cle+"-installed", lignes)
}

// warmup lance une commande de réchauffement déclarée. Elle rend (nil, nil) quand le
// descripteur n'en déclare pas — un plugin sans catalogue est parfaitement licite.
func (m *PluginManager) warmup(nom string) ([]byte, error) {
	w, ok := m.cfg.Warmup[nom]
	if !ok || len(w.Args) == 0 {
		return nil, nil
	}
	binaire := m.binaire
	if w.Cmd != "" && w.Cmd != m.cfg.Cmd {
		// Un descripteur peut désigner un autre programme pour le réchauffement ; il est
		// résolu par les mêmes règles que le binaire principal.
		if b := m.resoudre(w.Cmd); b != "" {
			binaire = b
		}
	}
	if binaire == "" {
		return nil, errors.New("plugin " + m.cfg.Name + " : binaire introuvable")
	}
	return Run(binaire, w.Args)
}

// IsPlugin dit si un gestionnaire est un plugin.
func IsPlugin(m pm.Manager) bool {
	_, ok := m.(*PluginManager)
	return ok
}

// Verbs implémente pm.Bindings. Chaque verbe du descripteur devient une liaison dont
// l'argv est celui déclaré — jigger ne préfixe rien.
//
// Un verbe **normalisé** (list, outdated, search, source) voit sa sortie capturée et
// analysée comme du JSON ; tout autre verbe est relayé, terminal compris, ce qui laisse
// passer une invite d'authentification ou une barre de progression. C'est pm.Normalise
// qui tranche, et non le pool : `install` puise ses candidats dans le catalogue, mais
// c'est une écriture.
func (m *PluginManager) Verbs() map[pm.Verb]pm.Binding {
	out := make(map[pm.Verb]pm.Binding, len(m.cfg.Verbs))
	for verbe, v := range m.cfg.Verbs {
		b := pm.Binding{
			Native: append([]string(nil), v.Native...),
			Pool:   poolFromString(v.Pool),
		}
		if pm.Normalise(pm.Verb(verbe)) {
			b.Parse = parsePluginOutput
		}
		out[pm.Verb(verbe)] = b
	}
	return out
}

// parsePluginOutput transforme la sortie JSON d'un plugin en []pm.Package. Le format
// attendu est un tableau d'objets ; les champs inconnus sont ignorés, et une entrée sans
// nom est écartée.
func parsePluginOutput(out []byte) ([]pm.Package, error) {
	var items []map[string]string
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	paquets := make([]pm.Package, 0, len(items))
	for _, item := range items {
		p := pm.Package{
			Name:      item["name"],
			Version:   item["version"],
			Available: item["available"],
			Kind:      item["kind"],
			Source:    item["source"],
		}
		if p.Name != "" {
			paquets = append(paquets, p)
		}
	}
	return paquets, nil
}
