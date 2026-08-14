package pm

import (
	"os"
	"testing"
	"time"
)

func TestCatalogSortIgnoreLaCasse(t *testing.T) {
	cat := NewCatalog()
	for _, n := range []string{"Microsoft.PowerShell", "mailspring", "zoxide", "Git.Git", "7zip"} {
		cat.Add(n, BadgeWinget)
	}
	cat.Sort()

	want := []string{"7zip", "Git.Git", "mailspring", "Microsoft.PowerShell", "zoxide"}
	for i := range want {
		if cat.Names[i] != want[i] {
			t.Fatalf("tri = %v, attendu %v", cat.Names, want)
		}
	}
}

func TestLessFold(t *testing.T) {
	cas := []struct {
		a, b string
		want bool
	}{
		{"mailspring", "Microsoft.X", true}, // « ai » avant « ic », quelle que soit la casse
		{"Microsoft.X", "mailspring", false},
		{"git", "gitui", true}, // préfixe : le plus court d'abord
		{"gitui", "git", false},
		{"Git", "git", true}, // à la casse près : un ordre stable, quel qu'il soit
		{"git", "git", false},
	}
	for _, c := range cas {
		if got := LessFold(c.a, c.b); got != c.want {
			t.Errorf("LessFold(%q, %q) = %v, attendu %v", c.a, c.b, got, c.want)
		}
	}
}

// Un nom déjà déclaré garde son premier badge : les gestionnaires listent leurs sources
// par ordre de priorité — chez brew, un nom porté à la fois par une formula et un cask
// reste une formula, et ne réclame donc pas `--cask`.
func TestAddGardeLePremierBadge(t *testing.T) {
	cat := NewCatalog()
	cat.Add("docker", BadgeFormula)
	cat.Add("docker", BadgeCask)

	if got := cat.Badge("docker"); got != BadgeFormula {
		t.Errorf("badge = %q, attendu %q", got, BadgeFormula)
	}
	if len(cat.Names) != 1 {
		t.Errorf("nom dupliqué : %v", cat.Names)
	}
}

func TestMarkInstalledDeclareLesInconnus(t *testing.T) {
	cat := NewCatalog()
	cat.Add("Git.Git", BadgeWinget)
	cat.MarkInstalled("Git.Git", "2.55.0", BadgeWinget)
	// Une application installée hors catalogue : winget sait la désinstaller, il faut
	// donc pouvoir la nommer.
	cat.MarkInstalled(`ARP\Machine\X86\Canon`, "1.02", BadgeOther)

	if !cat.Installed["Git.Git"] || cat.Version("Git.Git") != "2.55.0" {
		t.Errorf("installé mal noté : %v / %v", cat.Installed, cat.Versions)
	}
	if got := cat.Badge(`ARP\Machine\X86\Canon`); got != BadgeOther {
		t.Errorf("badge hors catalogue = %q, attendu %q", got, BadgeOther)
	}
	if got := cat.InstalledNames(); len(got) != 2 {
		t.Errorf("installés = %v, attendu 2", got)
	}
}

func TestStoreEtCached(t *testing.T) {
	t.Setenv("JIGGER_CACHE_DIR", t.TempDir())

	if _, frais := Cached("absent", time.Hour); frais {
		t.Error("un cache absent ne peut pas être frais")
	}

	lignes := []string{"Git.Git\t2.55.0", "7zip.7zip\t26.01"}
	if err := Store("essai", lignes); err != nil {
		t.Fatal(err)
	}

	relu, frais := Cached("essai", time.Hour)
	if !frais {
		t.Error("un cache qu'on vient d'écrire doit être frais")
	}
	if len(relu) != 2 || relu[0] != lignes[0] || relu[1] != lignes[1] {
		t.Fatalf("relecture = %v, attendu %v", relu, lignes)
	}

	// Périmé, mais rendu tout de même : compléter sur le catalogue d'hier vaut mieux que
	// sur rien — à l'appelant de demander un réchauffement.
	if relu, frais := Cached("essai", 0); frais || len(relu) != 2 {
		t.Errorf("cache périmé = %v (frais=%v), attendu les lignes et frais=false", relu, frais)
	}
}

func TestStoreNeLaissePasDeFichierTemporaire(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIGGER_CACHE_DIR", dir)

	if err := Store("essai", []string{"a"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "essai" {
			t.Errorf("résidu dans le cache : %s", e.Name())
		}
	}
}

func TestLock(t *testing.T) {
	chemin := t.TempDir() + string(os.PathSeparator) + "essai.lock"

	libere, ok := Lock(chemin)
	if !ok {
		t.Fatal("premier verrou refusé")
	}
	if _, ok := Lock(chemin); ok {
		t.Error("le verrou devrait être exclusif")
	}
	libere()
	if _, ok := Lock(chemin); !ok {
		t.Error("le verrou libéré doit pouvoir être repris")
	}

	// Un `jigger warm` tué laisse son verrou derrière lui : au-delà de la péremption, il
	// ne doit plus bloquer pour toujours.
	vieux := time.Now().Add(-PeremptionVerrou - time.Minute)
	if err := os.Chtimes(chemin, vieux, vieux); err != nil {
		t.Fatal(err)
	}
	if _, ok := Lock(chemin); !ok {
		t.Error("un verrou périmé doit être ignoré")
	}
}
