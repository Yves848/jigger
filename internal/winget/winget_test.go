package winget

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Les jeux d'essai sont des sorties winget réellement capturées, en français : c'est le
// cas qui compte. winget n'a pas de sortie machine, et ses en-têtes sont traduits — un
// parseur qui ne marcherait qu'en anglais ne marcherait chez personne.
func fixture(t *testing.T, nom string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", nom))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCatalogIDs(t *testing.T) {
	ids := CatalogIDs(fixture(t, "search-fr.txt"))

	if len(ids) != 10 {
		t.Fatalf("attendu 10 identifiants, obtenu %d : %v", len(ids), ids)
	}
	for _, veut := range []string{"Git.Git", "Microsoft.Git", "AndreasWascher.RepoZ"} {
		if !slices.Contains(ids, veut) {
			t.Errorf("%q absent de %v", veut, ids)
		}
	}
	// La colonne « Correspondance » (« Tag: git ») ne doit pas déborder sur l'identifiant.
	for _, id := range ids {
		if len(id) == 0 || id[0] == ' ' {
			t.Errorf("identifiant mal découpé : %q", id)
		}
	}
}

func TestInstalledLines(t *testing.T) {
	lignes := InstalledLines(fixture(t, "list-fr.txt"))

	if len(lignes) != 18 {
		t.Fatalf("attendu 18 paquets, obtenu %d : %v", len(lignes), lignes)
	}

	veut := map[string]string{
		// Un paquet épinglé : winget préfixe sa version d'un « > » qui n'en fait pas partie.
		"AgileBits.1Password": "8.12.32.33",
		"7zip.7zip":           "26.01",
		// Un identifiant avec des espaces (application détectée hors catalogue) : le
		// découpage en colonnes le garde entier, là où un découpage aux espaces le
		// couperait en quatre.
		`ARP\Machine\X86\Canon IJ Printer Assistant Tool`: "1.20.1.51",
	}
	trouve := map[string]string{}
	for _, l := range lignes {
		id, version, _ := cut(l)
		trouve[id] = version
	}
	for id, version := range veut {
		if trouve[id] != version {
			t.Errorf("%q : version %q, attendue %q", id, trouve[id], version)
		}
	}
}

// TestCountOutdated : la sortie contient, sous le tableau, deux phrases de résumé dont
// l'une est assez longue pour couvrir toutes les colonnes. Les compter comme des
// paquets serait le bogue le plus facile à écrire ici.
func TestCountOutdated(t *testing.T) {
	if got := CountOutdated(fixture(t, "upgrade-fr.txt")); got != 48 {
		t.Fatalf("CountOutdated = %d, attendu 48 (le tableau en compte 48, suivi de deux lignes de résumé)", got)
	}
}

func TestCountOutdatedSansTableau(t *testing.T) {
	// Rien à mettre à niveau : winget n'imprime aucun tableau.
	if got := CountOutdated([]byte("Aucun package installé trouvé correspondant aux critères d'entrée.\r\n")); got != 0 {
		t.Fatalf("CountOutdated = %d, attendu 0", got)
	}
}

func TestParseVersion(t *testing.T) {
	cas := map[string]string{
		"v1.29.280\r\n":              "1.29.280",
		"v1.11.400-preview\r\n":      "1.11.400",
		"1.29.280\n":                 "1.29.280",
		"":                           "",
		"winget : commande inconnue": "",
	}
	for out, want := range cas {
		if got := ParseVersion(out); got != want {
			t.Errorf("ParseVersion(%q) = %q, attendu %q", out, got, want)
		}
	}
}

func TestParseTableIgnoreLesSequencesANSI(t *testing.T) {
	out := []byte("\x1b[32;1mNom       ID          Version\x1b[0m\r\n" +
		"------------------------------\r\n" +
		"Git       Git.Git     2.55.0\r\n")
	rows := ParseTable(out)
	if len(rows) != 1 || rows[0].Col(colID) != "Git.Git" {
		t.Fatalf("découpage inattendu : %v", rows)
	}
}

func TestInsertProtegeLesEspaces(t *testing.T) {
	m := New()
	if got := m.Insert(nil, "install", "winget install ", "Git.Git"); got != "Git.Git" {
		t.Errorf("Insert = %q, attendu « Git.Git »", got)
	}
	if got := m.Insert(nil, "uninstall", "winget uninstall ", `ARP\X86\Canon IJ`); got != `"ARP\X86\Canon IJ"` {
		t.Errorf("Insert = %q, attendu la forme protégée", got)
	}
}

// cut découpe « id<TAB>version ».
func cut(l string) (id, version string, ok bool) {
	for i := range l {
		if l[i] == '\t' {
			return l[:i], l[i+1:], true
		}
	}
	return l, "", false
}
