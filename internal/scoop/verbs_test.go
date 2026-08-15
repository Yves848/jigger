package scoop

import "testing"

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

func TestTableScoopEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

func TestArgvScoop(t *testing.T) {
	table := New().Verbs()
	egal(t, table["install"].Argv([]string{"fd", "7zip"})[0],
		[]string{"install", "fd", "7zip"})
	egal(t, table["list"].Argv(nil)[0], []string{"list"})
	egal(t, table["source"].Argv(nil)[0], []string{"bucket", "list"})
	egal(t, table["source add"].Argv([]string{"extras"})[0],
		[]string{"bucket", "add", "extras"})
	egal(t, table["source rm"].Argv([]string{"extras"})[0],
		[]string{"bucket", "rm", "extras"})
	egal(t, table["pin"].Argv([]string{"fd"})[0], []string{"hold", "fd"})
	egal(t, table["unpin"].Argv([]string{"fd"})[0], []string{"unhold", "fd"})
	egal(t, table["doctor"].Argv(nil)[0], []string{"checkup"})
}

// `scoop update` sans argument met à jour scoop lui-même et les buckets, pas les
// applications : le verbe upgrade sans nom doit donc viser « * ».
func TestUpgradeScoopSansNomViseTout(t *testing.T) {
	got := New().Verbs()["upgrade"].Argv(nil)
	if len(got) != 1 {
		t.Fatalf("%d invocations, attendu 1", len(got))
	}
	egal(t, got[0], []string{"update", "*"})
}

// outdated ne lance pas scoop : la réponse se lit sur le disque. C'est ce que Direct
// exprime, et c'est ce qui rend `jg outdated` instantané côté scoop.
func TestOutdatedScoopEstDirect(t *testing.T) {
	b := New().Verbs()["outdated"]
	if b.Direct == nil {
		t.Fatal("outdated doit passer par Direct, pas par un sous-processus")
	}
	if b.Native != nil || b.Parse != nil {
		t.Errorf("outdated : Direct exclut Native et Parse (%v / %v)", b.Native, b.Parse)
	}
}
