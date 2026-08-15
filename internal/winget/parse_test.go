package winget

import (
	"os"
	"testing"
)

// Les en-têtes de winget sont traduits : le parser doit tenir sur un jeu français.
func TestParseUpgradeFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/upgrade-fr.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOutdated(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne extraite du tableau français")
	}
	for _, r := range rows {
		if r.Name == "" {
			t.Errorf("ligne sans identifiant : %+v", r)
		}
		if r.Version == "" || r.Available == "" {
			t.Errorf("outdated doit porter les deux versions : %+v", r)
		}
	}
}

func TestParseListFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/list-fr.txt")
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
}

func TestParseSearchFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/search-fr.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseSearch(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne")
	}
}

// Une barre de progression ou un en-tête de copyright ne doivent pas devenir des paquets.
func TestParseIgnoreLeBruit(t *testing.T) {
	bruit := []byte("   \\\n   |\nAucun paquet installé ne correspond aux critères.\n")
	rows, err := parseList(bruit)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu vide", rows)
	}
}
