package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ── reconnaissance des URL ─────────────────────────────────────────────

// TestEstURL tient la décision qui protège l'utilisateur : ce qui n'est pas une URL doit
// passer par le registre. Le premier jet fabriquait « https://github.com/<nom>.git » à
// partir de n'importe quel mot — donc clonait n'importe quoi.
func TestEstURL(t *testing.T) {
	urls := []string{
		"https://gitlab.yg-devworks.com/yves/jigger.git",
		"http://exemple.test/depot",
		"ssh://git@exemple.test/yves/jigger.git",
		"file:///tmp/amont/demo.git",
		"git@github.com:Yves848/omarchy.git",
	}
	for _, u := range urls {
		if !estURL(u) {
			t.Errorf("estURL(%q) = false, want true", u)
		}
	}

	pasDesURL := []string{"jigger", "mon-depot", "omarchy-mac", "a.b.c", ""}
	for _, s := range pasDesURL {
		if estURL(s) {
			t.Errorf("estURL(%q) = true, want false", s)
		}
	}
}

func TestNomDepuisURL(t *testing.T) {
	tests := map[string]string{
		"https://gitlab.yg-devworks.com/yves/jigger.git": "jigger",
		"https://github.com/macarchy/omarchy-mac":        "omarchy-mac",
		"git@github.com:Yves848/omarchy.git":             "omarchy",
		"file:///tmp/amont/demo.git":                     "demo",
		"https://exemple.test/yves/projet/":              "projet",
	}
	for url, want := range tests {
		if got := nomDepuisURL(url); got != want {
			t.Errorf("nomDepuisURL(%q) = %q, want %q", url, got, want)
		}
	}
}

// TestResoudreRefuseDeDeviner : sans URL ni entrée de registre, install doit se taire
// plutôt que de construire une adresse au jugé.
func TestResoudreRefuseDeDeviner(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := resoudre("un-depot-inconnu"); err == nil {
		t.Fatal("resoudre() = nil, un nom inconnu ne doit pas produire d'URL")
	}
}

func TestResoudreDepuisRegistre(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dossier := filepath.Join(cfg, "jigger", "plugins", "git")
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		t.Fatal(err)
	}
	écrire(t, filepath.Join(dossier, "depots.json"),
		`{"mien":"https://exemple.test/yves/mien.git"}`)

	url, nom, err := resoudre("mien")
	if err != nil {
		t.Fatalf("resoudre() error = %v", err)
	}
	if url != "https://exemple.test/yves/mien.git" || nom != "mien" {
		t.Errorf("resoudre() = (%q, %q)", url, nom)
	}
}

// TestRegistreDepotsPrimeConnus : le fichier écrit à la main doit l'emporter sur
// l'origine retenue, faute de quoi l'utilisateur ne pourrait pas corriger une URL périmée.
func TestRegistreDepotsPrimeConnus(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dossier := filepath.Join(cfg, "jigger", "plugins", "git")
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		t.Fatal(err)
	}
	écrire(t, filepath.Join(dossier, "depots.json"), `{"a":"https://neuf.test/a.git"}`)
	écrire(t, filepath.Join(dossier, "connus.json"),
		`{"a":"https://vieux.test/a.git","b":"https://vieux.test/b.git"}`)

	reg := registre()
	if reg["a"] != "https://neuf.test/a.git" {
		t.Errorf("depots.json devrait primer : a = %q", reg["a"])
	}
	if reg["b"] != "https://vieux.test/b.git" {
		t.Errorf("connus.json devrait compléter : b = %q", reg["b"])
	}
}

// TestRetenir : les origines vues au catalogue sont notées, pour qu'un dépôt supprimé
// puisse être recloné plus tard.
func TestRetenir(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)

	retenir([]depot{
		{nom: "avec", origine: "https://exemple.test/avec.git"},
		{nom: "sans", origine: ""}, // dépôt purement local : rien à retenir
	})

	data, err := os.ReadFile(cheminConnus())
	if err != nil {
		t.Fatalf("connus.json non écrit : %v", err)
	}
	var table map[string]string
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if table["avec"] != "https://exemple.test/avec.git" {
		t.Errorf("origine non retenue : %v", table)
	}
	if _, ok := table["sans"]; ok {
		t.Error("un dépôt sans origine ne doit pas être retenu")
	}
}

// ── arguments ──────────────────────────────────────────────────────────

func TestSansOptions(t *testing.T) {
	got := sansOptions([]string{"--force", "demo", "--json", "autre"})
	if len(got) != 2 || got[0] != "demo" || got[1] != "autre" {
		t.Errorf("sansOptions() = %v, want [demo autre]", got)
	}
}

func TestALOption(t *testing.T) {
	args := []string{"--force", "demo"}
	if !aLOption(args, "--force") {
		t.Error("aLOption(--force) = false")
	}
	if aLOption(args, "--json") {
		t.Error("aLOption(--json) = true")
	}
}

func TestCorrespond(t *testing.T) {
	if !correspond("jigger", nil) {
		t.Error("sans motif, tout doit correspondre")
	}
	if !correspond("scoop-jigger", []string{"JIG"}) {
		t.Error("la casse ne doit pas compter")
	}
	if correspond("omarchy", []string{"jig"}) {
		t.Error("correspond() = true sur un motif absent")
	}
}

func TestRacinesDepuisEnv(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	t.Setenv("JIGGER_GIT_ROOTS", a+string(os.PathListSeparator)+b)
	got := racines()
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("racines() = %v, want [%s %s]", got, a, b)
	}
}

// ── parcours du disque ─────────────────────────────────────────────────

// TestParcourirNeDescendPasDansUnDepot : un dépôt est une feuille. Sans cette règle,
// chaque sous-module et chaque dossier de travail remonterait comme un paquet distinct.
func TestParcourirNeDescendPasDansUnDepot(t *testing.T) {
	racine := t.TempDir()
	// racine/projet/.git  et  racine/projet/interne/.git (un sous-module)
	interne := filepath.Join(racine, "projet", "interne")
	if err := os.MkdirAll(filepath.Join(interne, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(racine, "projet", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// racine/client/autre/.git — un dépôt au deuxième niveau
	if err := os.MkdirAll(filepath.Join(racine, "client", "autre", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := parcourir(racine, profondeurMax)
	if len(got) != 2 {
		t.Fatalf("parcourir() = %v, want 2 dépôts", got)
	}
	for _, c := range got {
		if strings.Contains(c, "interne") {
			t.Errorf("parcourir() est descendu dans un dépôt : %s", c)
		}
	}
}

func TestParcourirRacineAbsente(t *testing.T) {
	if got := parcourir(filepath.Join(t.TempDir(), "nexiste-pas"), 2); got != nil {
		t.Errorf("parcourir() = %v, une racine absente n'est pas une erreur", got)
	}
}

// ── garde de suppression ───────────────────────────────────────────────

// TestTravailEnPeril est le test qui compte : uninstall supprime un dossier pour de bon.
// Un clone se refait, des modifications non validées ou des commits non poussés, non.
func TestTravailEnPeril(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dépôts git fabriqués en shell")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git absent")
	}

	base := t.TempDir()
	amont := filepath.Join(base, "amont.git")
	run(t, "git", "init", "-q", "--bare", amont)
	run(t, "git", "-C", amont, "symbolic-ref", "HEAD", "refs/heads/main")

	clone := filepath.Join(base, "clone")
	run(t, "git", "init", "-q", "-b", "main", clone)
	écrire(t, filepath.Join(clone, "f.txt"), "un\n")
	run(t, "git", "-C", clone, "add", ".")
	run(t, "git", "-C", clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "un")
	run(t, "git", "-C", clone, "remote", "add", "origin", amont)
	run(t, "git", "-C", clone, "push", "-q", "-u", "origin", "main")

	d := decrire(clone)
	if d.origine == "" {
		t.Fatal("le clone devrait avoir une origine")
	}
	if raison := travailEnPeril(d); raison != "" {
		t.Errorf("dépôt propre et poussé : travailEnPeril() = %q, want \"\"", raison)
	}

	// Une modification non validée bloque.
	écrire(t, filepath.Join(clone, "f.txt"), "un\ndeux\n")
	if raison := travailEnPeril(d); raison == "" {
		t.Error("travail non validé : travailEnPeril() = \"\", want une raison")
	}

	// Validée mais non poussée : bloque encore.
	run(t, "git", "-C", clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qam", "deux")
	if raison := travailEnPeril(d); raison == "" {
		t.Error("commit non poussé : travailEnPeril() = \"\", want une raison")
	}

	// Poussée : plus rien à perdre.
	run(t, "git", "-C", clone, "push", "-q", "origin", "main")
	if raison := travailEnPeril(decrire(clone)); raison != "" {
		t.Errorf("après push : travailEnPeril() = %q, want \"\"", raison)
	}
}

// TestTravailEnPerilSansDistant : un dépôt qui n'existe qu'ici ne se reclone pas.
func TestTravailEnPerilSansDistant(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git absent")
	}
	clone := t.TempDir()
	run(t, "git", "init", "-q", clone)
	écrire(t, filepath.Join(clone, "f.txt"), "un\n")
	run(t, "git", "-C", clone, "add", ".")
	run(t, "git", "-C", clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "un")

	d := decrire(clone)
	if d.badge() != badgeLocal {
		t.Errorf("badge() = %q, un dépôt sans distant doit porter %q", d.badge(), badgeLocal)
	}
	if travailEnPeril(d) == "" {
		t.Error("un dépôt sans distant ne doit pas se supprimer sans --force")
	}
}

// ── utilitaires de test ────────────────────────────────────────────────

func run(t *testing.T, nom string, args ...string) {
	t.Helper()
	c := exec.Command(nom, args...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%s %s : %v\n%s", nom, strings.Join(args, " "), err, out)
	}
}

func écrire(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}
