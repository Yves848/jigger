// Les jeux d'essai de testdata/ sont des **captures réelles** d'un scoop 0.5.3 sous
// Windows, prises comme jigger les reçoit — derrière un tuyau (cf. tests/captures-scoop.ps1).
// Les attendus ci-dessous en découlent : ce sont les paquets réellement installés et les
// buckets réellement configurés sur cette machine-là.
package scoop

import (
	"os"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func lire(t *testing.T, nom string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nom)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParseList(t *testing.T) {
	rows, err := parseList(lire(t, "list.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("%d lignes, attendu 4 : %+v", len(rows), rows)
	}
	attendus := []pm.Package{
		{Name: "7zip", Version: "26.00", Source: "main", Kind: pm.BadgeScoop},
		{Name: "gcc", Version: "15.2.0", Source: "main", Kind: pm.BadgeScoop},
		{Name: "jetbrainsmono-nf", Version: "3.4.0", Source: "nerd-fonts", Kind: pm.BadgeOther},
		{Name: "make", Version: "4.4.1", Source: "main", Kind: pm.BadgeScoop},
	}
	for i, a := range attendus {
		if rows[i] != a {
			t.Errorf("ligne %d = %+v, attendu %+v", i, rows[i], a)
		}
	}
}

// La colonne « Updated » de `scoop list` contient une date **et** une heure, séparées par
// un espace. Un découpage qui traiterait tout espace comme une frontière la couperait en
// deux et décalerait tout ce qui suit.
func TestParseListNeCoupePasLaDateEnDeux(t *testing.T) {
	rows, err := parseList(lire(t, "list.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Source != "main" && r.Source != "nerd-fonts" {
			t.Errorf("source = %q — la colonne Updated a débordé sur Source : %+v", r.Source, r)
		}
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
	rows, err := parseSearch(lire(t, "search.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 74 {
		t.Fatalf("%d lignes, attendu 74 : %+v", len(rows), rows)
	}
	if a := (pm.Package{Name: "biodiff", Version: "1.2.1", Source: "main", Kind: pm.BadgeScoop}); rows[0] != a {
		t.Errorf("première ligne = %+v, attendu %+v", rows[0], a)
	}
	if a := (pm.Package{Name: "wslgit", Version: "1.3.1", Source: "extras", Kind: pm.BadgeOther}); rows[len(rows)-1] != a {
		t.Errorf("dernière ligne = %+v, attendu %+v", rows[len(rows)-1], a)
	}
}

// La régression qui a motivé la réécriture : Format-Table remplit chaque colonne à la
// largeur de sa cellule la plus longue, si bien que la ligne la plus large n'a **qu'un
// seul espace** avant la colonne suivante. Un découpage sur « au moins deux espaces »
// lisait « git-interactive-rebase-tool 2.4.1 » comme un seul champ — et se trompait
// précisément sur la ligne qui donne sa largeur à la colonne, dans chaque tableau.
func TestParseSearchLitLaLigneLaPlusLarge(t *testing.T) {
	rows, err := parseSearch(lire(t, "search.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var trouve *pm.Package
	for i := range rows {
		if rows[i].Name == "git-interactive-rebase-tool" {
			trouve = &rows[i]
		}
	}
	if trouve == nil {
		t.Fatal("« git-interactive-rebase-tool » absent : le nom le plus long n'a pas été isolé de sa version")
	}
	if trouve.Version != "2.4.1" || trouve.Source != "main" {
		t.Errorf("%+v, attendu version 2.4.1 et source main", *trouve)
	}
}

// La colonne « Binaries » est vide pour la plupart des lignes : la ligne s'arrête alors
// après le bucket. Une colonne absente ne doit pas décaler les précédentes.
func TestParseSearchColonneFinaleVide(t *testing.T) {
	rows, err := parseSearch(lire(t, "search.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Source != "main" && r.Source != "extras" {
			t.Errorf("source = %q, attendu un nom de bucket : %+v", r.Source, r)
		}
	}
}

func TestParseSource(t *testing.T) {
	rows, err := parseSource(lire(t, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("%d lignes, attendu 3 : %+v", len(rows), rows)
	}
	attendus := []pm.Package{
		{Name: "main", Source: "https://github.com/ScoopInstaller/Main"},
		{Name: "extras", Source: "https://github.com/ScoopInstaller/Extras"},
		{Name: "nerd-fonts", Source: "https://github.com/matthewjberger/scoop-nerd-fonts"},
	}
	for i, a := range attendus {
		if rows[i] != a {
			t.Errorf("ligne %d = %+v, attendu %+v", i, rows[i], a)
		}
	}
}

// Les en-têtes et la ligne de tirets de PowerShell sont **colorés**, même derrière un
// tuyau ; les données, non. C'est ce qui rendait les trois analyseurs muets : la ligne de
// tirets, entourée d'échappements ANSI, ne ressemblait plus à des tirets, et aucun d'eux
// n'entrait jamais dans le tableau.
func TestAnsiDansLEnteteNEmpechePasDeLire(t *testing.T) {
	colore := []byte("\x1b[32;1mName  \x1b[0m\x1b[32;1m Source\x1b[0m\n" +
		"\x1b[32;1m----  \x1b[0m \x1b[32;1m------\x1b[0m\n" +
		"main   https://example.invalid/main\n")
	rows, err := parseSource(colore)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "main" || rows[0].Source != "https://example.invalid/main" {
		t.Fatalf("%+v, attendu une ligne main → https://example.invalid/main", rows)
	}
}
