package complete

import (
	"fmt"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
)

func testCatalog() *brew.Catalog {
	return brew.NewCatalog(
		[]string{"git", "wget", "node", "ripgrep", "firefly"},        // formulae
		[]string{"firefox", "visual-studio-code", "firefly-desktop"}, // casks
		[]string{"git", "ripgrep"},                                   // installés
	)
}

func names(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}

func TestFirstWord_CompletesSubcommands(t *testing.T) {
	res := Complete("inst", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "install" {
		t.Fatalf("attendu [install], obtenu %v", got)
	}
	if res.Executable {
		t.Fatal("une sous-commande ne doit pas être « exécutable »")
	}
}

func TestOptionCompletion(t *testing.T) {
	res := Complete("list --vers", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "--versions" {
		t.Fatalf("attendu [--versions], obtenu %v", got)
	}
}

func TestInstall_CompletesAllPackages(t *testing.T) {
	res := Complete("install fire", testCatalog())
	got := names(res.Items)
	// firefly (formula) + firefox, firefly-desktop (casks)
	want := map[string]bool{"firefly": true, "firefox": true, "firefly-desktop": true}
	if len(got) != len(want) {
		t.Fatalf("attendu %d candidats, obtenu %v", len(want), got)
	}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("candidat inattendu %q dans %v", n, got)
		}
	}
	if !res.Executable {
		t.Fatal("un contexte paquet doit être « exécutable »")
	}
}

func TestUninstall_CompletesInstalledOnly(t *testing.T) {
	res := Complete("uninstall ", testCatalog())
	got := names(res.Items)
	if len(got) != 2 {
		t.Fatalf("attendu 2 installés, obtenu %v", got)
	}
	for _, n := range got {
		if n != "git" && n != "ripgrep" {
			t.Fatalf("non installé %q proposé pour uninstall: %v", n, got)
		}
	}
}

func TestBadgeAndCaskDetection(t *testing.T) {
	cat := testCatalog()
	res := Complete("install firefox", cat)
	if len(res.Items) != 1 || res.Items[0].Badge != "C" {
		t.Fatalf("firefox devrait porter le badge C, obtenu %v", res.Items)
	}
	if !NeedsCask(cat, "firefox") {
		t.Fatal("firefox (cask pur) devrait nécessiter --cask")
	}
	if NeedsCask(cat, "git") {
		t.Fatal("git (formula) ne devrait pas nécessiter --cask")
	}
}

func TestStripsLeadingBrew(t *testing.T) {
	res := Complete("brew uninstall gi", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "git" {
		t.Fatalf("attendu [git], obtenu %v", got)
	}
}

// BenchmarkComplete garde un œil sur le filtrage : le popup vivant appelle Complete à
// chaque frappe, sur un catalogue de plusieurs milliers de noms. Budget indicatif :
// quelques ms au plus (le reste du budget part dans le démarrage du binaire).
func BenchmarkComplete(b *testing.B) {
	formulae := make([]string, 0, 8000)
	casks := make([]string, 0, 2000)
	for i := range 8000 {
		formulae = append(formulae, fmt.Sprintf("formula-%04d", i))
	}
	for i := range 2000 {
		casks = append(casks, fmt.Sprintf("cask-%04d", i))
	}
	cat := brew.NewCatalog(formulae, casks, formulae[:200])

	b.ResetTimer()
	for b.Loop() {
		Complete("brew install form", cat)
	}
}
