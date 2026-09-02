package pacman

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// avecYay force la présence (ou l'absence) de yay le temps d'un test. La table des verbes
// dépend de l'environnement (ADR-0007) : les deux branches doivent être éprouvées, quelle
// que soit la machine qui fait tourner les tests.
func avecYay(t *testing.T, present bool) {
	t.Helper()
	ancien := yayPresent
	yayPresent = func() bool { return present }
	t.Cleanup(func() { yayPresent = ancien })
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

// Les deux branches de la table doivent être bien formées, pas seulement celle de la
// machine de test.
func TestTablesPacmanBienFormees(t *testing.T) {
	for _, present := range []bool{false, true} {
		avecYay(t, present)
		for _, m := range []Manager{New("pacman"), New("yay")} {
			for v, b := range m.Verbs() {
				if err := b.Valid(); err != nil {
					t.Errorf("yay présent=%v, %s, verbe %q : %v", present, m.Cmd(), v, err)
				}
			}
		}
	}
}

// Le cœur de l'ADR-0007 : avec yay sur la machine, pacman ne déclare plus rien. Sans quoi
// `jg list` listerait deux fois les mêmes paquets et `jg install fd` serait ambigu entre
// deux portes sur la même base alpm.
func TestPacmanSeTaitQuandYayEstLa(t *testing.T) {
	avecYay(t, true)
	if v := New("pacman").Verbs(); len(v) != 0 {
		t.Fatalf("pacman déclare %d verbes alors que yay est là : %v", len(v), v)
	}
	if len(New("yay").Verbs()) == 0 {
		t.Fatal("yay doit piloter")
	}
}

func TestPacmanSeulDeclareLaLecture(t *testing.T) {
	avecYay(t, false)
	table := New("pacman").Verbs()

	for _, v := range []pm.Verb{"list", "outdated", "search", "info"} {
		if _, ok := table[v]; !ok {
			t.Errorf("verbe de lecture %q absent de la table pacman", v)
		}
	}
	// Les verbes mutants exigent root, et jigger n'élève rien (ADR-0004) : un verbe qu'on
	// ne sait pas rendre ne se déclare pas.
	for _, v := range []pm.Verb{"install", "uninstall", "upgrade", "cleanup"} {
		if _, ok := table[v]; ok {
			t.Errorf("verbe mutant %q déclaré par pacman", v)
		}
	}
}

func TestYayPiloteTout(t *testing.T) {
	table := New("yay").Verbs()
	for _, v := range []pm.Verb{"install", "uninstall", "upgrade", "list", "outdated", "search", "info", "cleanup"} {
		if _, ok := table[v]; !ok {
			t.Errorf("verbe %q absent de la table yay", v)
		}
	}
	// Ce qui n'existe pas sous Arch reste absent, et c'est le modèle de capacités qui
	// parle : un dépôt s'ajoute en éditant /etc/pacman.conf, IgnorePkg est une ligne de
	// configuration, et pacman n'a pas d'équivalent de `brew doctor`.
	for _, v := range []pm.Verb{"source", "source add", "source rm", "pin", "unpin", "doctor"} {
		if _, ok := table[v]; ok {
			t.Errorf("verbe %q déclaré alors qu'il n'a pas de sens sous Arch", v)
		}
	}
}

func TestArgvDesVerbesMutants(t *testing.T) {
	table := New("yay").Verbs()
	cas := []struct {
		verbe pm.Verb
		args  []string
		want  []string
	}{
		{"install", []string{"fd", "ripgrep"}, []string{"-S", "fd", "ripgrep"}},
		{"uninstall", []string{"fd"}, []string{"-Rns", "fd"}},
		{"upgrade", nil, []string{"-Syu"}},
		{"cleanup", nil, []string{"-Sc"}},
		{"info", []string{"fd"}, []string{"-Si", "fd"}},
	}
	for _, c := range cas {
		lignes := table[c.verbe].Argv(c.args)
		if len(lignes) != 1 {
			t.Fatalf("verbe %q : %d invocations, attendu 1", c.verbe, len(lignes))
		}
		egal(t, lignes[0], c.want)
	}
}

// Les pools disent où la façade puise ses candidats. `search` prend une requête, pas un nom
// à résoudre : le router en PoolCatalogue refuserait de chercher un mot qui n'est justement
// pas encore un nom connu (même raison que chez brew).
func TestPools(t *testing.T) {
	table := New("yay").Verbs()
	cas := map[pm.Verb]pm.Pool{
		"install":   pm.PoolCatalogue,
		"info":      pm.PoolCatalogue,
		"uninstall": pm.PoolInstalles,
		"upgrade":   pm.PoolInstalles,
		"list":      pm.PoolAucun,
		"outdated":  pm.PoolAucun,
		"search":    pm.PoolAucun,
		"cleanup":   pm.PoolAucun,
	}
	for v, want := range cas {
		if got := table[v].Pool; got != want {
			t.Errorf("verbe %q : pool %v, attendu %v", v, got, want)
		}
	}
}
