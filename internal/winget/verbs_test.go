package winget

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got, want)
		}
	}
}

func TestTableWingetEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

// winget n'installe qu'un paquet par appel : deux noms font deux invocations.
func TestArgvWingetUnAppelParPaquet(t *testing.T) {
	b := New().Verbs()["install"]
	lignes := b.Argv([]string{"Git.Git", "7zip.7zip"})

	if len(lignes) != 2 {
		t.Fatalf("%d invocations, attendu 2 — winget ne prend qu'un id", len(lignes))
	}
	egal(t, lignes[0], []string{"install", "--id", "Git.Git", "--exact"})
	egal(t, lignes[1], []string{"install", "--id", "7zip.7zip", "--exact"})
}

func TestArgvWinget(t *testing.T) {
	table := New().Verbs()
	egal(t, table["list"].Argv(nil)[0], []string{"list"})
	egal(t, table["outdated"].Argv(nil)[0], []string{"list", "--upgrade-available"})
	egal(t, table["search"].Argv([]string{"git"})[0], []string{"search", "git"})
	egal(t, table["source"].Argv(nil)[0], []string{"source", "list"})
	egal(t, table["source add"].Argv([]string{"x"})[0], []string{"source", "add", "x"})
	egal(t, table["pin"].Argv([]string{"Git.Git"})[0],
		[]string{"pin", "add", "--id", "Git.Git"})
}

// winget n'a ni cleanup ni doctor. Leur absence EST la capacité déclarée — c'est ce qui
// fait dire « aucun gestionnaire disponible ne sait faire ça » en tâche 6.
func TestWingetNeSaitNiCleanupNiDoctor(t *testing.T) {
	table := New().Verbs()
	for _, v := range []pm.Verb{"cleanup", "doctor"} {
		if _, ok := table[v]; ok {
			t.Errorf("verbe %q présent dans la table winget, attendu absent", v)
		}
	}
}
