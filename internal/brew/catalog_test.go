package brew

import (
	"os"
	"path/filepath"
	"testing"
)

// mkKeg crée <racine>/<paquet>/<version> comme le fait Homebrew dans Cellar/Caskroom.
func mkKeg(t *testing.T, root, name string, versions ...string) {
	t.Helper()
	for _, v := range versions {
		if err := os.MkdirAll(filepath.Join(root, name, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstalledFromDirs(t *testing.T) {
	dir := t.TempDir()
	cellar := filepath.Join(dir, "Cellar")
	caskroom := filepath.Join(dir, "Caskroom")

	mkKeg(t, cellar, "git", "2.55.0")
	mkKeg(t, cellar, "jq", "1.7.1")
	mkKeg(t, caskroom, "firefox", "142.0")

	got := installedFromDirs(cellar, caskroom)
	want := []string{"firefox 142.0", "git 2.55.0", "jq 1.7.1"}
	assertLines(t, got, want)
}

func TestInstalledFromDirsGardeLaVersionLaPlusRecente(t *testing.T) {
	// Plusieurs kegs pour un même paquet (ancienne version non nettoyée) : on annonce
	// la plus récente, celle que brew utilise.
	cellar := filepath.Join(t.TempDir(), "Cellar")
	mkKeg(t, cellar, "node", "22.9.0", "24.1.0", "23.11.0")

	assertLines(t, installedFromDirs(cellar, ""), []string{"node 24.1.0"})
}

func TestInstalledFromDirsIgnoreLeBruit(t *testing.T) {
	dir := t.TempDir()
	cellar := filepath.Join(dir, "Cellar")
	mkKeg(t, cellar, "git", "2.55.0")
	mkKeg(t, cellar, ".DS_Store_dir")                    // caché : ignoré
	mkKeg(t, cellar, "vide")                             // sans keg : ignoré
	os.WriteFile(filepath.Join(cellar, "f"), nil, 0o644) // fichier : ignoré

	assertLines(t, installedFromDirs(cellar, ""), []string{"git 2.55.0"})
}

func TestInstalledFromDirsRepertoiresAbsents(t *testing.T) {
	// Machine sans Caskroom (ou préfixe inattendu) : pas d'erreur, liste vide.
	if got := installedFromDirs("/nexiste/pas/Cellar", "/nexiste/pas/Caskroom"); len(got) != 0 {
		t.Fatalf("attendu vide, obtenu %v", got)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("obtenu %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ligne %d : obtenu %q, attendu %q", i, got[i], want[i])
		}
	}
}
