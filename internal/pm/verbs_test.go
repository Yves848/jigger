package pm

import "testing"

func TestArgvDeveloppeTousLesArguments(t *testing.T) {
	b := Binding{Native: []string{"install", "{args}"}}
	got := b.Argv([]string{"fd", "git"})

	if len(got) != 1 {
		t.Fatalf("{args} doit produire une seule invocation, obtenu %v", got)
	}
	want := []string{"install", "fd", "git"}
	if len(got[0]) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got[0], want)
	}
	for i := range want {
		if got[0][i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got[0], want)
		}
	}
}

func TestArgvUnAppelParArgument(t *testing.T) {
	b := Binding{Native: []string{"install", "--id", "{arg}", "--exact"}}
	got := b.Argv([]string{"Git.Git", "7zip.7zip"})

	if len(got) != 2 {
		t.Fatalf("{arg} doit produire une invocation par argument, obtenu %v", got)
	}
	if got[0][2] != "Git.Git" || got[1][2] != "7zip.7zip" {
		t.Fatalf("argv = %v", got)
	}
	if got[0][3] != "--exact" {
		t.Fatalf("le suffixe du gabarit est perdu : %v", got[0])
	}
}

func TestArgvSansMarqueur(t *testing.T) {
	b := Binding{Native: []string{"outdated", "--json=v2"}}
	got := b.Argv(nil)

	if len(got) != 1 || len(got[0]) != 2 || got[0][0] != "outdated" {
		t.Fatalf("argv = %v, attendu une invocation [outdated --json=v2]", got)
	}
}

func TestArgvPrefereBuild(t *testing.T) {
	b := Binding{Build: func(args []string) []string {
		return append([]string{"untap"}, args...)
	}}
	got := b.Argv([]string{"extras"})

	if len(got) != 1 || got[0][0] != "untap" || got[0][1] != "extras" {
		t.Fatalf("argv = %v, attendu [[untap extras]]", got)
	}
}

func TestValidRefuseDeuxFaconsDAgir(t *testing.T) {
	cas := []struct {
		nom string
		b   Binding
		ok  bool
	}{
		{"native seul", Binding{Native: []string{"install"}}, true},
		{"build seul", Binding{Build: func([]string) []string { return nil }}, true},
		{"direct seul", Binding{Direct: func([]string) ([]Package, error) { return nil, nil }}, true},
		{"aucun", Binding{}, false},
		{"native + build", Binding{
			Native: []string{"install"},
			Build:  func([]string) []string { return nil },
		}, false},
		{"direct + parse", Binding{
			Direct: func([]string) ([]Package, error) { return nil, nil },
			Parse:  func([]byte) ([]Package, error) { return nil, nil },
		}, false},
	}
	for _, c := range cas {
		err := c.b.Valid()
		if (err == nil) != c.ok {
			t.Errorf("%s : Valid() = %v, attendu ok=%v", c.nom, err, c.ok)
		}
	}
}

func TestNomNatif(t *testing.T) {
	if got := (Binding{Native: []string{"checkup"}}).NomNatif(); got != "checkup" {
		t.Errorf("NomNatif = %q, attendu « checkup »", got)
	}
	if got := (Binding{Build: func([]string) []string { return nil }}).NomNatif(); got != "" {
		t.Errorf("NomNatif d'une liaison calculée = %q, attendu vide", got)
	}
}
