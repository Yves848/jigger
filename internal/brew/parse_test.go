package brew

import (
	"os"
	"testing"
)

func TestParseOutdated(t *testing.T) {
	data, err := os.ReadFile("testdata/outdated.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOutdated(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne : le jeu d'essai de la tâche 1 est-il non vide ?")
	}
	for _, r := range rows {
		if r.Name == "" {
			t.Errorf("ligne sans nom : %+v", r)
		}
		if r.Version == "" || r.Available == "" {
			t.Errorf("outdated doit porter les deux versions : %+v", r)
		}
		if r.Kind == "" {
			t.Errorf("ligne sans badge : %+v", r)
		}
	}
}

func TestParseList(t *testing.T) {
	data, err := os.ReadFile("testdata/list-versions.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne")
	}
	for _, r := range rows {
		if r.Name == "" || r.Version == "" {
			t.Errorf("« nom version » attendu sur chaque ligne : %+v", r)
		}
	}
}

// Une sortie vide est un résultat valide — rien n'est périmé —, pas une erreur.
func TestParseOutdadedVide(t *testing.T) {
	rows, err := parseOutdated([]byte(`{"formulae":[],"casks":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu vide", rows)
	}
}
