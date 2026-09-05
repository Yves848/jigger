package managers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllContientLesTroisCommandesSSH(t *testing.T) {
	vus := map[string]bool{}
	for _, m := range All() {
		vus[m.Cmd()] = true
	}
	for _, c := range []string{"ssh", "scp", "sftp"} {
		if !vus[c] {
			t.Errorf("%q absent de All()", c)
		}
	}
	// Les trois gestionnaires de paquets restent là.
	for _, c := range []string{"brew", "winget", "scoop"} {
		if !vus[c] {
			t.Errorf("%q disparu de All()", c)
		}
	}
}

func TestLesFournisseursSSHNeDeclarentAucunVerbe(t *testing.T) {
	for _, m := range All() {
		switch m.Cmd() {
		case "ssh", "scp", "sftp":
			if len(m.Subcommands()) != 0 {
				t.Errorf("%q déclare des sous-commandes", m.Cmd())
			}
		}
	}
}

// --- #140 : le greffon shell doit connaitre les mots apportes par les plugins ---------

func planterPlugin(t *testing.T, nom string) (cache string) {
	t.Helper()
	cfg, cache := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("JIGGER_CACHE_DIR", cache)
	dir := filepath.Join(cfg, "jigger", "plugins", nom)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// `cmd: sh` : le binaire doit se resoudre pour que le plugin soit « disponible ».
	desc := `{"name":"` + nom + `","cmd":"sh","verbs":{"list":{"native":["list"],"pool":"aucun"}}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(desc), 0o644); err != nil {
		t.Fatal(err)
	}
	return cache
}

func TestWritePluginCommandsEcritLesMotsDecouverts(t *testing.T) {
	// Le defaut de JIGGER_COMMANDS ne peut pas connaitre ces mots : ils dependent de ce
	// qui est installe sur la machine. Le greffon shell les lit dans ce fichier a son
	// chargement, sans lancer de processus.
	planterPlugin(t, "faux")
	if err := WritePluginCommands(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(PluginCommandsPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "faux" {
		t.Errorf("fichier = %q, attendu \"faux\"", got)
	}
}

func TestWritePluginCommandsVideLeFichierQuandPlusAucunPlugin(t *testing.T) {
	// Un plugin desinstalle ne doit pas rester arme dans le shell : le fichier est
	// reecrit a chaque warm, y compris vide.
	cfg, cache := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("JIGGER_CACHE_DIR", cache)
	if err := os.WriteFile(filepath.Join(cache, "plugin-commands"), []byte("perime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WritePluginCommands(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(PluginCommandsPath())
	if strings.TrimSpace(string(b)) != "" {
		t.Errorf("fichier = %q, attendu vide", b)
	}
}
