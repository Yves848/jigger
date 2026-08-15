package brew

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// argv1 rend l'unique ligne d'arguments d'un verbe, et échoue s'il y en a plusieurs.
func argv1(t *testing.T, v pm.Verb, args []string) []string {
	t.Helper()
	b, ok := New().Verbs()[v]
	if !ok {
		t.Fatalf("verbe %q absent de la table brew", v)
	}
	lignes := b.Argv(args)
	if len(lignes) != 1 {
		t.Fatalf("verbe %q : %d invocations, attendu 1", v, len(lignes))
	}
	return lignes[0]
}

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

func TestTableBrewEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

func TestArgvBrew(t *testing.T) {
	egal(t, argv1(t, "install", []string{"fd", "git"}), []string{"install", "fd", "git"})
	egal(t, argv1(t, "uninstall", []string{"fd"}), []string{"uninstall", "fd"})
	egal(t, argv1(t, "outdated", nil), []string{"outdated", "--json=v2"})
	egal(t, argv1(t, "list", nil), []string{"list", "--versions"})
	egal(t, argv1(t, "info", []string{"fd"}), []string{"info", "fd"})
	egal(t, argv1(t, "pin", []string{"fd"}), []string{"pin", "fd"})
	egal(t, argv1(t, "doctor", nil), []string{"doctor"})
}

// tap et untap sont deux mots sans rapport : c'est le cas qui justifie Build.
func TestArgvBrewSource(t *testing.T) {
	egal(t, argv1(t, "source", nil), []string{"tap"})
	egal(t, argv1(t, "source add", []string{"homebrew/cask-fonts"}),
		[]string{"tap", "homebrew/cask-fonts"})
	egal(t, argv1(t, "source rm", []string{"homebrew/cask-fonts"}),
		[]string{"untap", "homebrew/cask-fonts"})
}

func TestPoolBrew(t *testing.T) {
	cas := map[pm.Verb]pm.Pool{
		"install":   pm.PoolCatalogue,
		"search":    pm.PoolCatalogue,
		"uninstall": pm.PoolInstalles,
		"upgrade":   pm.PoolInstalles,
		"pin":       pm.PoolInstalles,
		"outdated":  pm.PoolAucun,
		"doctor":    pm.PoolAucun,
	}
	table := New().Verbs()
	for v, want := range cas {
		if got := table[v].Pool; got != want {
			t.Errorf("verbe %q : Pool = %v, attendu %v", v, got, want)
		}
	}
}

// winget n'a ni cleanup ni doctor : la capacité se lit dans l'absence de clé. brew, lui,
// doit les avoir.
func TestBrewSaitFaireCleanupEtDoctor(t *testing.T) {
	table := New().Verbs()
	for _, v := range []pm.Verb{"cleanup", "doctor"} {
		if _, ok := table[v]; !ok {
			t.Errorf("verbe %q absent de la table brew", v)
		}
	}
}
