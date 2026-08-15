package scoop

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// arbre monte une installation scoop de poche : des buckets de manifestes, et des
// applications installées sous apps/<nom>/<version> avec leur lien `current` — que les
// tests remplacent par un vrai dossier, faute de pouvoir créer une jonction partout.
func arbre(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	manifeste(t, root, "main", "7zip", "26.02")
	manifeste(t, root, "main", "git", "2.54.0")
	manifeste(t, root, "main", "flux", "2.8.8")
	manifeste(t, root, "main", "yazi", "26.5.6")
	manifeste(t, root, "extras", "flux", "4.0.0") // le même nom dans deux buckets
	manifeste(t, root, "extras", "vscode", "1.99")

	installe(t, root, "7zip", "26.00", "main", false)  // une version de retard
	installe(t, root, "git", "2.54.0", "main", false)  // à jour
	installe(t, root, "yazi", "26.1.22", "main", true) // en retard, mais tenue
	installe(t, root, "maison", "1.0", "", false)      // installée depuis une URL

	return root
}

func manifeste(t *testing.T, root, bucket, nom, version string) {
	t.Helper()
	dir := filepath.Join(root, "buckets", bucket, "bucket")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ecrire(t, filepath.Join(dir, nom+".json"), `{"version":"`+version+`","description":"…"}`)
}

func installe(t *testing.T, root, nom, version, bucket string, tenue bool) {
	t.Helper()
	current := filepath.Join(root, "apps", nom, "current")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "apps", nom, version), 0o755); err != nil {
		t.Fatal(err)
	}
	ecrire(t, filepath.Join(current, "manifest.json"), `{"version":"`+version+`"}`)
	install := `{"architecture":"64bit"`
	if bucket != "" {
		install += `,"bucket":"` + bucket + `"`
	}
	if tenue {
		install += `,"hold":true`
	}
	ecrire(t, filepath.Join(current, "install.json"), install+"}")
}

func ecrire(t *testing.T, path, contenu string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFrom(t *testing.T) {
	cat := LoadFrom(arbre(t), "")

	for _, nom := range []string{"7zip", "flux", "git", "vscode", "yazi"} {
		if !slices.Contains(cat.Names, nom) {
			t.Errorf("%q absent du catalogue %v", nom, cat.Names)
		}
	}
	for i := 1; i < len(cat.Names); i++ {
		if !pm.LessFold(cat.Names[i-1], cat.Names[i]) {
			t.Errorf("catalogue non trié en %q / %q : %v", cat.Names[i-1], cat.Names[i], cat.Names)
			break
		}
	}

	// Le bucket officiel et les autres se distinguent, comme formulae et casks.
	if got := cat.Badge("git"); got != pm.BadgeScoop {
		t.Errorf("badge de git = %q, attendu %q", got, pm.BadgeScoop)
	}
	if got := cat.Badge("vscode"); got != pm.BadgeOther {
		t.Errorf("badge de vscode = %q, attendu %q", got, pm.BadgeOther)
	}

	// Version installée : lue dans le manifeste copié sous `current`.
	if got := cat.Version("7zip"); got != "26.00" {
		t.Errorf("version de 7zip = %q, attendue « 26.00 »", got)
	}
	if !cat.Installed["7zip"] || cat.Installed["vscode"] {
		t.Errorf("installés mal repérés : %v", cat.Installed)
	}
	// Une application installée hors de tout bucket reste proposée : c'est elle qu'il
	// faudra nommer pour la désinstaller.
	if !cat.Installed["maison"] {
		t.Error("une application installée depuis une URL doit rester au catalogue")
	}
}

func TestLoadFromQualifieLesNomsAmbigus(t *testing.T) {
	cat := LoadFrom(arbre(t), "")

	if got := cat.Qualified["flux"]; got != "main/flux" {
		t.Errorf("qualification de flux = %q, attendue « main/flux »", got)
	}
	if _, ambigu := cat.Qualified["git"]; ambigu {
		t.Error("git n'est que dans un bucket : rien à qualifier")
	}

	m := New()
	if got := m.Insert(cat, "install", "scoop install ", "flux"); got != "main/flux" {
		t.Errorf("Insert = %q, attendu « main/flux »", got)
	}
	// L'utilisateur a déjà nommé un bucket : on ne touche à rien.
	if got := m.Insert(cat, "install", "scoop install extras/", "flux"); got != "flux" {
		t.Errorf("Insert = %q, attendu « flux »", got)
	}
}

func TestOutdatedIn(t *testing.T) {
	// 7zip est en retard (26.00 vs 26.02) ; git est à jour ; yazi est en retard mais
	// tenue par `scoop hold` ; maison n'a pas de manifeste de référence.
	n, err := OutdatedIn(arbre(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("OutdatedIn = %d, attendu 1", n)
	}
}

func TestOutdatedInRacineAbsente(t *testing.T) {
	n, err := OutdatedIn(filepath.Join(t.TempDir(), "nexiste-pas"), "")
	if err != nil || n != 0 {
		t.Fatalf("OutdatedIn = %d, %v ; attendu 0, nil", n, err)
	}
}

// Les buckets d'avant 2022 déposent leurs manifestes à la racine, sans sous-dossier
// `bucket/`. jigger doit encore les lire : personne ne met à jour un bucket privé.
func TestBucketSansSousDossier(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "buckets", "ancien")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ecrire(t, filepath.Join(dir, "vieux-truc.json"), `{"version":"1.0"}`)

	cat := LoadFrom(root, "")
	if !slices.Contains(cat.Names, "vieux-truc") {
		t.Fatalf("manifeste à la racine du bucket ignoré : %v", cat.Names)
	}
}
