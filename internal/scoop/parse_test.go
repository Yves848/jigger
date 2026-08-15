// Les jeux d'essai de testdata/ sont écrits à la main, NON VÉRIFIÉS contre un vrai scoop
// (cf. tâche 1b, parse.go) : ce Mac n'a ni scoop ni PowerShell.
package scoop

import (
	"os"
	"testing"
)

func TestParseList(t *testing.T) {
	data, err := os.ReadFile("testdata/list.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("%d lignes, attendu 4 : %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.Name == "" || r.Version == "" {
			t.Errorf("« nom version » attendu sur chaque ligne : %+v", r)
		}
	}
	if rows[0].Name != "7zip" || rows[0].Version != "23.01" || rows[0].Source != "main" {
		t.Errorf("première ligne = %+v, attendu 7zip/23.01/main", rows[0])
	}
	// « flux » vient du bucket extras : badge autre que le bucket main.
	trouve := false
	for _, r := range rows {
		if r.Name == "flux" {
			trouve = true
			if r.Source != "extras" || r.Kind != "X" {
				t.Errorf("flux = %+v, attendu source extras, badge autre-que-main", r)
			}
		}
	}
	if !trouve {
		t.Fatal("« flux » absent des lignes analysées")
	}
}

func TestParseListIgnoreLeBruit(t *testing.T) {
	bruit := []byte("Installed apps:\n\nName    Version\n----    -------\n")
	rows, err := parseList(bruit)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu vide (aucune application)", rows)
	}
}

func TestParseSearch(t *testing.T) {
	data, err := os.ReadFile("testdata/search.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseSearch(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("%d lignes, attendu 4 : %+v", len(rows), rows)
	}
	if rows[0].Name != "fd" || rows[0].Version != "9.0.0" || rows[0].Source != "main" {
		t.Errorf("première ligne = %+v, attendu fd/9.0.0/main", rows[0])
	}
	dernier := rows[len(rows)-1]
	if dernier.Name != "firealpaca" || dernier.Version != "1.1.4" || dernier.Source != "extras" {
		t.Errorf("dernière ligne = %+v, attendu firealpaca/1.1.4/extras (le « --> includes » ne doit pas polluer la version)", dernier)
	}
}

func TestParseSource(t *testing.T) {
	data, err := os.ReadFile("testdata/source.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseSource(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("%d lignes, attendu 2 : %+v", len(rows), rows)
	}
	if rows[0].Name != "main" || rows[0].Source != "https://github.com/ScoopInstaller/Main" {
		t.Errorf("première ligne = %+v", rows[0])
	}
	if rows[1].Name != "extras" || rows[1].Source != "https://github.com/ScoopInstaller/Extras" {
		t.Errorf("deuxième ligne = %+v", rows[1])
	}
}
