package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// validConfig est un descripteur minimal valide pour les tests.
func validConfig(t *testing.T) Config {
	t.Helper()
	cfg := Config{
		Name:    "test-pm",
		Version: "1.0.0",
		Cmd:     "jigger-test-pm",
		Verbs: map[string]Verb{
			"install": {Native: []string{"install", "{arg}"}, Pool: "catalogue"},
		},
		Warmup: map[string]WarmupCmd{
			"catalog":   {Args: []string{"catalog", "--json"}},
			"installed": {Args: []string{"list", "--installed", "--json"}},
		},
		Parse: Parse{
			Fields:       []string{"name", "version", "kind", "source"},
			CatalogField: "names",
			BadgeField:   "badges",
		},
	}
	if err := validate(&cfg); err != nil {
		t.Fatalf("config invalide pour le test : %v", err)
	}
	return cfg
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool // vrai si la validation passe
	}{
		{
			name: "valide",
			cfg:  validConfig(t),
			want: true,
		},
		{
			name: "nom manquant",
			cfg: Config{
				Cmd:   "jigger-test-pm",
				Verbs: map[string]Verb{"install": {Native: []string{"install"}, Pool: "catalogue"}},
			},
			want: false,
		},
		{
			name: "cmd manquant",
			cfg: Config{
				Name:  "test-pm",
				Verbs: map[string]Verb{"install": {Native: []string{"install"}, Pool: "catalogue"}},
			},
			want: false,
		},
		{
			name: "aucun verbe",
			cfg: Config{
				Name:    "test-pm",
				Version: "1.0.0",
				Cmd:     "jigger-test-pm",
				Verbs:   map[string]Verb{},
			},
			want: false,
		},
		{
			name: "verbe sans native",
			cfg: Config{
				Name:    "test-pm",
				Cmd:     "jigger-test-pm",
				Version: "1.0.0",
				Verbs:   map[string]Verb{"install": {Pool: "catalogue"}},
			},
			want: false,
		},
		{
			name: "pool invalide",
			cfg: Config{
				Name:    "test-pm",
				Cmd:     "jigger-test-pm",
				Version: "1.0.0",
				Verbs:   map[string]Verb{"install": {Native: []string{"install"}, Pool: "bizarre"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			err := validate(&cfg)
			got := err == nil
			if got != tt.want {
				t.Errorf("validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestValidateDefaults tient la règle qui a manqué à la première écriture : validate pose
// des valeurs par défaut, et celles-ci doivent parvenir à l'appelant. Prise par valeur,
// elle les posait sur sa propre copie — le descripteur en sortait inchangé.
func TestValidateDefaults(t *testing.T) {
	cfg := validConfig(t)
	cfg.Parse = Parse{} // tout effacer pour tester les valeurs par défaut
	if err := validate(&cfg); err != nil {
		t.Fatalf("validate a rejeté : %v", err)
	}
	if cfg.Parse.CatalogField != "names" {
		t.Errorf("catalog_field = %q, want %q", cfg.Parse.CatalogField, "names")
	}
	if cfg.Parse.BadgeField != "badges" {
		t.Errorf("badge_field = %q, want %q", cfg.Parse.BadgeField, "badges")
	}
	if len(cfg.Parse.Fields) == 0 {
		t.Error("package_fields devrait être rempli par défaut")
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := validConfig(t)
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if got.Name != cfg.Name {
		t.Errorf("name = %q, want %q", got.Name, cfg.Name)
	}
	if got.Cmd != cfg.Cmd {
		t.Errorf("cmd = %q, want %q", got.Cmd, cfg.Cmd)
	}
	if got.Warmup["catalog"].Args[0] != "catalog" {
		t.Errorf("warmup.catalog.args = %v", got.Warmup["catalog"].Args)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// JSON invalide
	if err := os.WriteFile(cfgPath, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(cfgPath); err == nil {
		t.Error("loadConfig() devrait échouer sur du JSON invalide")
	}

	// Fichier inexistant
	if _, err := loadConfig(filepath.Join(dir, "nonexistent.json")); err == nil {
		t.Error("loadConfig() devrait échouer sur un fichier absent")
	}
}

func TestPoolFromString(t *testing.T) {
	tests := []struct {
		in   string
		want pm.Pool
	}{
		{"catalogue", pm.PoolCatalogue},
		{"installees", pm.PoolInstalles},
		{"aucun", pm.PoolAucun},
		{"inconnu", pm.PoolAucun},
	}
	for _, tt := range tests {
		if got := poolFromString(tt.in); got != tt.want {
			t.Errorf("poolFromString(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// TestCleSure vérifie qu'un nom de plugin ne peut pas produire un nom de fichier de cache
// contenant un séparateur : pm.Store passe par os.CreateTemp, qui refuse ce motif, et le
// cache serait alors inécrivable sans que rien ne le dise.
func TestCleSure(t *testing.T) {
	tests := map[string]string{
		"git":        "git",
		"gît":        "g_t",
		"a/b":        "a_b",
		"mon.plugin": "mon.plugin",
		"mon_pm-2":   "mon_pm-2",
	}
	for in, want := range tests {
		if got := cleSure(in); got != want {
			t.Errorf("cleSure(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPluginManagerCmd(t *testing.T) {
	cfg := validConfig(t)
	m := NewPluginManager(cfg, "/test/dir")
	if m.Cmd() != "test-pm" {
		t.Errorf("Cmd() = %q, want %q", m.Cmd(), "test-pm")
	}
}

// TestBinaire tient la distinction qui manquait au premier jet : le mot de la ligne n'est
// pas forcement le binaire. Un helper les fait coincider — le plugin `git` declare
// `cmd: "git"` — mais un gestionnaire tiers non, et la facade doit lancer Binaire().
func TestBinaire(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "jigger-test-pm")
	if runtime.GOOS == "windows" {
		t.Skip("le bit exécutable n'a pas de sens sous Windows")
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := NewPluginManager(validConfig(t), dir)
	got, ok := Binaire(m)
	if !ok {
		t.Fatal("Binaire() = false, le binaire du dossier du plugin devrait être trouvé")
	}
	if got != bin {
		t.Errorf("Binaire() = %q, want %q", got, bin)
	}
	if got == m.Cmd() {
		t.Error("Binaire() ne doit pas rendre le mot de la ligne")
	}
	if !m.Available() {
		t.Error("Available() = false alors que le binaire existe")
	}
}

// TestBinaireIntrouvable : sans binaire, le plugin n'est pas disponible et la façade doit
// retomber sur son propre chemin plutôt que de lancer n'importe quoi.
func TestBinaireIntrouvable(t *testing.T) {
	cfg := validConfig(t)
	cfg.Cmd = "jigger-ce-binaire-nexiste-pas"
	m := NewPluginManager(cfg, t.TempDir())
	if _, ok := Binaire(m); ok {
		t.Error("Binaire() = true pour un binaire absent")
	}
	if m.Available() {
		t.Error("Available() = true pour un binaire absent")
	}
}

// TestBinaireSurNatif : Binaire ne doit répondre que pour un plugin.
func TestBinaireSurNatif(t *testing.T) {
	if _, ok := Binaire(nil); ok {
		t.Error("Binaire(nil) = true, want false")
	}
}

func TestPlateforme(t *testing.T) {
	cfg := validConfig(t)
	cfg.Platform = []string{"plan9-imaginaire"}
	m := NewPluginManager(cfg, "/test/dir")
	if m.surCettePlateforme() {
		t.Error("surCettePlateforme() = true sur une plateforme non déclarée")
	}

	cfg.Platform = []string{runtime.GOOS}
	m = NewPluginManager(cfg, "/test/dir")
	if !m.surCettePlateforme() {
		t.Error("surCettePlateforme() = false sur la plateforme courante")
	}

	cfg.Platform = nil // liste vide : partout
	m = NewPluginManager(cfg, "/test/dir")
	if !m.surCettePlateforme() {
		t.Error("surCettePlateforme() = false pour une liste vide")
	}
}

func TestPluginManagerSubcommands(t *testing.T) {
	cfg := validConfig(t)
	cfg.Verbs["upgrade"] = Verb{Native: []string{"upgrade", "{args}"}, Pool: "installees"}
	cfg.Verbs["source add"] = Verb{Native: []string{"tap"}, Pool: "aucun"}

	subs := NewPluginManager(cfg, "/test/dir").Subcommands()

	// Doit contenir install, upgrade, source (premier mot de « source add »), triés.
	for _, want := range []string{"install", "source", "upgrade"} {
		trouve := false
		for _, s := range subs {
			if s == want {
				trouve = true
				break
			}
		}
		if !trouve {
			t.Errorf("Subcommands() sans %q, got %v", want, subs)
		}
	}
	for i := 1; i < len(subs); i++ {
		if subs[i] < subs[i-1] {
			t.Errorf("Subcommands() non trié : %v", subs)
			break
		}
	}
}

func TestPluginManagerInstalledOnly(t *testing.T) {
	cfg := validConfig(t)
	cfg.Verbs["uninstall"] = Verb{Native: []string{"remove"}, Pool: "installees"}
	cfg.Verbs["search"] = Verb{Native: []string{"find"}, Pool: "catalogue"}

	m := NewPluginManager(cfg, "/test/dir")

	if !m.InstalledOnly("uninstall") {
		t.Error("InstalledOnly(uninstall) devrait être vrai")
	}
	if m.InstalledOnly("search") {
		t.Error("InstalledOnly(search) devrait être faux")
	}
	if m.InstalledOnly("inconnu") {
		t.Error("InstalledOnly(inconnu) devrait être faux")
	}
}

func TestPluginManagerInsert(t *testing.T) {
	m := NewPluginManager(validConfig(t), "/test/dir")
	if got := m.Insert(nil, "install", "", "mon-paquet"); got != "mon-paquet" {
		t.Errorf("Insert() = %q, want %q", got, "mon-paquet")
	}
}

func TestDiscoverSkipsInvalid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("JIGGER_CACHE_DIR", t.TempDir())

	plugins := filepath.Join(dir, "jigger", "plugins")

	// Descripteur invalide (nom manquant).
	mauvais := filepath.Join(plugins, "mauvais")
	if err := os.MkdirAll(mauvais, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(mauvais, "config.json"),
		[]byte(`{"cmd":"jigger-mauvais","verbs":{"install":{"native":["install"],"pool":"catalogue"}}}`), 0o644)

	// Descripteur valide mais binaire absent → Available() = false.
	valide := filepath.Join(plugins, "valide")
	if err := os.MkdirAll(valide, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(valide, "config.json"), []byte(`{
		"name": "valide",
		"version": "1.0.0",
		"cmd": "jigger-ce-binaire-nexiste-pas",
		"verbs": {"install": {"native": ["install"], "pool": "catalogue"}}
	}`), 0o644)

	if got := Discover(); len(got) != 0 {
		t.Errorf("Discover() a rendu %d plugins, want 0", len(got))
	}
}

// TestDiscoverTrouve : un descripteur valide dont le binaire est posé à côté doit être
// découvert, et son binaire résolu dans le dossier du plugin.
func TestDiscoverTrouve(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("le bit exécutable n'a pas de sens sous Windows")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("JIGGER_CACHE_DIR", t.TempDir())

	p := filepath.Join(dir, "jigger", "plugins", "demo")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(p, "config.json"), []byte(`{
		"name": "demo",
		"version": "1.0.0",
		"cmd": "jigger-demo",
		"verbs": {"list": {"native": ["list", "--json"], "pool": "installees"}}
	}`), 0o644)
	bin := filepath.Join(p, "jigger-demo")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho '[]'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := Discover()
	if len(got) != 1 {
		t.Fatalf("Discover() a rendu %d plugins, want 1", len(got))
	}
	if got[0].Cmd() != "demo" {
		t.Errorf("Cmd() = %q, want %q", got[0].Cmd(), "demo")
	}
	if b, _ := Binaire(got[0]); b != bin {
		t.Errorf("Binaire() = %q, want %q", b, bin)
	}
}

func TestIsPlugin(t *testing.T) {
	if !IsPlugin(&PluginManager{}) {
		t.Error("IsPlugin(&PluginManager{}) devrait être vrai")
	}
	if IsPlugin(nil) {
		t.Error("IsPlugin(nil) devrait être faux")
	}
}

// TestVerbs tient la règle corrigée : c'est pm.Normalise qui décide si une sortie est
// analysée, et non le pool. `install` puise ses candidats dans le catalogue, mais c'est
// une écriture : la relayer est ce qui laisse passer une invite d'authentification.
func TestVerbs(t *testing.T) {
	cfg := Config{
		Name:    "test-plugin",
		Version: "1.0.0",
		Cmd:     "jigger-test",
		Verbs: map[string]Verb{
			"list":      {Native: []string{"list", "--json"}, Pool: "installees"},
			"search":    {Native: []string{"search", "{args}"}, Pool: "catalogue"},
			"outdated":  {Native: []string{"outdated", "--json"}, Pool: "installees"},
			"install":   {Native: []string{"run", "install", "{arg}"}, Pool: "catalogue"},
			"uninstall": {Native: []string{"run", "uninstall", "{arg}"}, Pool: "installees"},
		},
	}
	verbs := (&PluginManager{cfg: cfg}).Verbs()
	if len(verbs) != 5 {
		t.Fatalf("Verbs() a rendu %d verbes, want 5", len(verbs))
	}

	// Verbes normalisés : sortie capturée puis analysée.
	for _, v := range []string{"list", "search", "outdated"} {
		if verbs[pm.Verb(v)].Parse == nil {
			t.Errorf("Verbs()[%q].Parse est nil : un verbe normalisé doit être analysé", v)
		}
	}
	// Verbes relayés : aucune analyse, la sortie va au terminal.
	for _, v := range []string{"install", "uninstall"} {
		if verbs[pm.Verb(v)].Parse != nil {
			t.Errorf("Verbs()[%q].Parse non nil : un verbe relayé ne doit pas être analysé", v)
		}
	}

	// L'argv déclaré est passé tel quel : jigger ne préfixe rien.
	argv := verbs[pm.Verb("install")].Argv([]string{"fd"})
	if len(argv) != 1 || len(argv[0]) != 3 ||
		argv[0][0] != "run" || argv[0][1] != "install" || argv[0][2] != "fd" {
		t.Errorf("Argv() = %v, want [[run install fd]]", argv)
	}

	// Chaque liaison doit être bien formée au sens de pm.
	for v, b := range verbs {
		if err := b.Valid(); err != nil {
			t.Errorf("liaison %q invalide : %v", v, err)
		}
	}
}

// TestLoadDepuisCache : Load ne lit que des fichiers de cache — il est dans le chemin du
// rendu et ne doit lancer aucun sous-processus.
func TestLoadDepuisCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("JIGGER_CACHE_DIR", cache)

	m := NewPluginManager(validConfig(t), "/test/dir")
	if err := pm.Store(m.cle+"-catalog", []string{"alpha\tG", "beta\tG", "gamma\tG"}); err != nil {
		t.Fatal(err)
	}
	if err := pm.Store(m.cle+"-installed", []string{"beta\tG\tmain\torigin"}); err != nil {
		t.Fatal(err)
	}

	cat := m.Load()
	if len(cat.Names) != 3 {
		t.Fatalf("Names = %v, want 3 entrées", cat.Names)
	}
	if cat.Badge("alpha") != "G" {
		t.Errorf("Badge(alpha) = %q, want %q", cat.Badge("alpha"), "G")
	}
	if cat.Version("beta") != "main" {
		t.Errorf("Version(beta) = %q, want %q", cat.Version("beta"), "main")
	}
	installes := cat.InstalledNames()
	if len(installes) != 1 || installes[0] != "beta" {
		t.Errorf("InstalledNames() = %v, want [beta]", installes)
	}
}

// TestWarmEcritLesCaches boucle le protocole : un faux plugin en shell produit le JSON
// attendu, Warm le range en cache, Load le relit. C'est le test qui aurait attrapé les
// deux formats incompatibles du premier jet — Warm écrivait du TSV, Load lisait du JSON.
func TestWarmEcritLesCaches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("faux plugin écrit en sh")
	}
	dir := t.TempDir()
	t.Setenv("JIGGER_CACHE_DIR", t.TempDir())

	script := `#!/bin/sh
case "$1" in
  catalog) echo '{"names":["alpha","beta"],"badges":{"alpha":"G","beta":"G"}}' ;;
  list)    echo '[{"name":"beta","version":"main","kind":"G","source":"origin"}]' ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "jigger-faux"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Name: "faux", Version: "1.0.0", Cmd: "jigger-faux",
		Verbs: map[string]Verb{"list": {Native: []string{"list"}, Pool: "installees"}},
		Warmup: map[string]WarmupCmd{
			"catalog":   {Args: []string{"catalog"}},
			"installed": {Args: []string{"list"}},
		},
	}
	m := NewPluginManager(cfg, dir)
	if err := m.Warm(pm.ScopeAll); err != nil {
		t.Fatalf("Warm() error = %v", err)
	}

	cat := m.Load()
	if len(cat.Names) != 2 {
		t.Fatalf("Names = %v, want 2 entrées", cat.Names)
	}
	if cat.Version("beta") != "main" {
		t.Errorf("Version(beta) = %q, want %q", cat.Version("beta"), "main")
	}
}

// TestWarmRemonteLErreur : un plugin qui échoue ne doit pas écraser le cache précédent
// par du vide, et son erreur doit remonter.
func TestWarmRemonteLErreur(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("faux plugin écrit en sh")
	}
	dir := t.TempDir()
	t.Setenv("JIGGER_CACHE_DIR", t.TempDir())

	script := "#!/bin/sh\necho 'plus de disque' >&2\nexit 3\n"
	if err := os.WriteFile(filepath.Join(dir, "jigger-casse"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Name: "casse", Version: "1.0.0", Cmd: "jigger-casse",
		Verbs:  map[string]Verb{"list": {Native: []string{"list"}, Pool: "installees"}},
		Warmup: map[string]WarmupCmd{"catalog": {Args: []string{"catalog"}}},
	}
	m := NewPluginManager(cfg, dir)

	// Un cache antérieur, qui doit survivre à l'échec.
	if err := pm.Store(m.cle+"-catalog", []string{"ancien\tG"}); err != nil {
		t.Fatal(err)
	}

	if err := m.Warm(pm.ScopeAll); err == nil {
		t.Fatal("Warm() = nil, un plugin en échec doit remonter son erreur")
	}
	if noms := m.Load().Names; len(noms) != 1 || noms[0] != "ancien" {
		t.Errorf("le cache antérieur a été écrasé : %v", noms)
	}
}

func TestRunRemonteStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("faux plugin écrit en sh")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "casse")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'raison précise' >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Run(bin, []string{"catalog"})
	if err == nil {
		t.Fatal("Run() = nil, want une erreur")
	}
	if !contientTexte(err.Error(), "raison précise") {
		t.Errorf("l'erreur n'explique rien : %v", err)
	}
	if !contientTexte(err.Error(), "2") {
		t.Errorf("l'erreur ne dit pas le code de sortie : %v", err)
	}
}

func contientTexte(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestParsePluginOutput(t *testing.T) {
	data := []byte(`[
		{"name": "pkg1", "version": "1.0", "kind": "lib", "source": "git", "available": "1.1"},
		{"name": "pkg2", "version": "2.0"},
		{"version": "3.0"}
	]`)

	pkgs, err := parsePluginOutput(data)
	if err != nil {
		t.Fatalf("parsePluginOutput error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2 (l'entrée sans nom est écartée)", len(pkgs))
	}
	if pkgs[0].Name != "pkg1" || pkgs[0].Version != "1.0" || pkgs[0].Available != "1.1" {
		t.Errorf("pkg1 = %+v", pkgs[0])
	}
	if pkgs[1].Name != "pkg2" || pkgs[1].Version != "2.0" {
		t.Errorf("pkg2 = %+v", pkgs[1])
	}
}

func TestParsePluginOutputEmpty(t *testing.T) {
	pkgs, err := parsePluginOutput([]byte(`[]`))
	if err != nil {
		t.Fatalf("parsePluginOutput([]) error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("got %d packages, want 0", len(pkgs))
	}
}

func TestParsePluginOutputMalformed(t *testing.T) {
	if _, err := parsePluginOutput([]byte(`not json`)); err == nil {
		t.Error("parsePluginOutput(malformed) devrait échouer")
	}
}

func TestUnPluginDeclareSesVerbesExhaustifs(t *testing.T) {
	// Le contrat pm.Exhaustif est ce qui fait taire la completion sur un verbe que le
	// plugin ne declare pas — `git checkout ` ne doit pas proposer de depots (#141). Sans
	// cette declaration, la garde de complete ne s'arme jamais.
	var m pm.Manager = &PluginManager{}
	if !pm.VerbesExhaustifsDe(m) {
		t.Error("un plugin doit declarer ses verbes exhaustifs")
	}
}

// ── ADR-0009 : viviers par verbe et options par verbe ──────────────────

// plantePlugin fabrique un plugin dont le binaire est un script shell qui imprime ce
// qu'on lui dit. C'est le seul moyen d'eprouver un vivier « direct » de bout en bout.
func plantePlugin(t *testing.T, corps string, cfg Config) *PluginManager {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("script shell")
	}
	dir := t.TempDir()
	cfg.Cmd = "faux-pm"
	if err := os.WriteFile(filepath.Join(dir, "faux-pm"), []byte(corps), 0o755); err != nil {
		t.Fatal(err)
	}
	return NewPluginManager(cfg, dir)
}

func configAVivier(regime string) Config {
	return Config{
		Name: "faux", Cmd: "faux-pm",
		Verbs: map[string]Verb{
			"checkout": {Native: []string{"checkout", "{args}"}, Pool: "branches",
				Options: []string{"-b", "--detach"}},
		},
		Pools: map[string]Vivier{"branches": {Regime: regime, Args: []string{"viviers", "branches"}}},
	}
}

func TestDescripteurAccepteUnVivierNomme(t *testing.T) {
	cfg := configAVivier("direct")
	if err := validate(&cfg); err != nil {
		t.Fatalf("validate() = %v, un vivier nomme doit etre accepte", err)
	}
}

func TestDescripteurRefuseUnVivierNonDeclare(t *testing.T) {
	// Un pool qui ne designe ni une valeur historique ni un vivier declare est une faute
	// de frappe silencieuse : le verbe ne proposerait jamais rien.
	cfg := configAVivier("direct")
	cfg.Verbs["checkout"] = Verb{Native: []string{"checkout"}, Pool: "branhces"}
	err := validate(&cfg)
	if err == nil || !strings.Contains(err.Error(), "branhces") {
		t.Fatalf("validate() = %v, doit nommer le vivier inconnu", err)
	}
}

func TestDescripteurRefuseUnRegimeInconnu(t *testing.T) {
	cfg := configAVivier("magique")
	err := validate(&cfg)
	if err == nil || !strings.Contains(err.Error(), "magique") {
		t.Fatalf("validate() = %v, doit nommer le regime inconnu", err)
	}
}

func TestDescripteurGardeLesValeursHistoriques(t *testing.T) {
	// Non-regression : `catalogue`, `installees` et `aucun` restent valides sans qu'un
	// bloc `pools` soit necessaire. C'est un contrat public depuis la 0.16.0.
	cfg := validConfig(t)
	if err := validate(&cfg); err != nil {
		t.Fatalf("validate() = %v sur un descripteur historique", err)
	}
}

func TestOptionsViennentDuVerbe(t *testing.T) {
	// Options() rendait nil : un plugin ne pouvait proposer aucune option, la ou brew en
	// declare par sous-commande. C'est la moitie de ce qui manquait a un helper.
	m := plantePlugin(t, "#!/bin/sh\n", configAVivier("direct"))
	got := m.Options("checkout")
	if len(got) != 2 || got[0] != "-b" {
		t.Errorf("Options(checkout) = %v, attendu [-b --detach]", got)
	}
	if len(m.Options("verbe-sans-options")) != 0 {
		t.Error("Options() d'un verbe inconnu doit etre vide")
	}
}

func TestVivierDirectInterrogeLeBinaire(t *testing.T) {
	// Le contrat : le binaire DU PLUGIN est lance avec les args declares, et rend une
	// ligne par candidat, « nom<TAB>badge » comme le cache du catalogue.
	m := plantePlugin(t, "#!/bin/sh\nprintf 'main\\tlocal\\nfeat/x\\n'\n", configAVivier("direct"))
	cat, ok := m.Candidats("checkout")
	if !ok || cat == nil {
		t.Fatal("Candidats() = false, le vivier direct doit repondre")
	}
	// L'ORDRE DU PLUGIN EST PRESERVE : il l'a choisi, souvent par pertinence
	// (`--sort=-creatordate`), et le retrier jetterait cette information.
	if len(cat.Names) != 2 || cat.Names[0] != "main" || cat.Names[1] != "feat/x" {
		t.Errorf("Names = %v, attendu l'ordre rendu par le plugin", cat.Names)
	}
	if cat.Badge("main") != "local" {
		t.Errorf("Badge(main) = %q, attendu \"local\"", cat.Badge("main"))
	}
}

func TestVivierDirectTourneDansLeRepertoireCourant(t *testing.T) {
	// C'est LA raison d'etre du regime direct : les candidats dependent d'ou l'on est.
	m := plantePlugin(t, "#!/bin/sh\npwd\n", configAVivier("direct"))
	ailleurs := t.TempDir()
	avant, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(avant) })
	if err := os.Chdir(ailleurs); err != nil {
		t.Fatal(err)
	}
	cat, ok := m.Candidats("checkout")
	if !ok || len(cat.Names) != 1 {
		t.Fatalf("Candidats() = %v, %v", cat, ok)
	}
	reel, _ := filepath.EvalSymlinks(ailleurs)
	if vu, _ := filepath.EvalSymlinks(cat.Names[0]); vu != reel {
		t.Errorf("le vivier a tourne dans %q, attendu %q", cat.Names[0], reel)
	}
}

func TestVivierDirectAbandonneAuDelai(t *testing.T) {
	// Un plugin lent ne doit pas tenir le prompt : au dela du delai, rien plutot que
	// d'attendre (doctrine de l'ADR-0006, reprise par l'ADR-0009).
	t.Setenv("JIGGER_DELAI_VIVIER", "50")
	m := plantePlugin(t, "#!/bin/sh\nsleep 5\necho tard\n", configAVivier("direct"))
	debut := time.Now()
	cat, ok := m.Candidats("checkout")
	if ok && cat != nil && len(cat.Names) > 0 {
		t.Errorf("Candidats() = %v, un plugin trop lent ne doit rien rendre", cat.Names)
	}
	if d := time.Since(debut); d > 2*time.Second {
		t.Errorf("Candidats() a attendu %s : le delai n'a pas ete tenu", d)
	}
}

func TestVerbeSansVivierNommeNInterrogeRien(t *testing.T) {
	// Non-regression : un verbe a pool historique ne doit pas declencher de
	// sous-processus a la frappe.
	m := plantePlugin(t, "#!/bin/sh\necho ne-doit-pas-etre-appele\n", configAVivier("direct"))
	if cat, ok := m.Candidats("verbe-inconnu"); ok {
		t.Errorf("Candidats(verbe-inconnu) = %v, attendu false", cat)
	}
}

func TestVivierDirectRendUneColonneDeContexte(t *testing.T) {
	// La colonne de droite du popup est ce qui distingue un helper d'une simple
	// completion : zsh sait completer un nom de branche, il ne dit pas laquelle est en
	// retard ni quand elle a bouge. Le vivier rend donc « nom<TAB>badge<TAB>version »,
	// meme convention que le cache des installes.
	m := plantePlugin(t,
		"#!/bin/sh\nprintf 'main\\t\\t[behind 2] il y a 3 h\\nfeat/x\\t\\thier\\n'\n",
		configAVivier("direct"))
	cat, ok := m.Candidats("checkout")
	if !ok {
		t.Fatal("Candidats() = false")
	}
	if got := cat.Version("main"); got != "[behind 2] il y a 3 h" {
		t.Errorf("Version(main) = %q, attendu le contexte", got)
	}
	if got := cat.Version("feat/x"); got != "hier" {
		t.Errorf("Version(feat/x) = %q", got)
	}
}

func TestVivierDirectSansContexteResteValide(t *testing.T) {
	// Non-regression : un vivier qui ne rend que des noms garde son sens.
	m := plantePlugin(t, "#!/bin/sh\nprintf 'origin\\ngithub\\n'\n", configAVivier("direct"))
	cat, _ := m.Candidats("checkout")
	if len(cat.Names) != 2 || cat.Version("origin") != "" {
		t.Errorf("Names = %v, Version(origin) = %q", cat.Names, cat.Version("origin"))
	}
}
