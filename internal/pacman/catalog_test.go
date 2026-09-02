package pacman

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func TestCoupeNomVersion(t *testing.T) {
	cas := []struct{ entree, nom, version string }{
		{"acl-2.4.0-1", "acl", "2.4.0-1"},
		// Le nom porte deux tirets : c'est le cas que toute heuristique « premier tiret »
		// casse, et que le découpage par les DEUX DERNIERS tirets règle exactement — Arch
		// interdit le tiret dans pkgver comme dans pkgrel.
		{"gpu-screen-recorder-6.0.1-1", "gpu-screen-recorder", "6.0.1-1"},
		{"abseil-cpp-20260817.0-1", "abseil-cpp", "20260817.0-1"},
		{"linux-firmware-nvidia-20260815.abc-2", "linux-firmware-nvidia", "20260815.abc-2"},
		// Entrées qui ne sont pas des paquets.
		{"ALPM_DB_VERSION", "", ""},
		{"sans-release", "", ""},
		{"", "", ""},
	}
	for _, c := range cas {
		nom, version := CoupeNomVersion(c.entree)
		if nom != c.nom || version != c.version {
			t.Errorf("CoupeNomVersion(%q) = %q, %q — attendu %q, %q", c.entree, nom, version, c.nom, c.version)
		}
	}
}

func TestCoupeSync(t *testing.T) {
	cas := []struct{ ligne, depot, nom, version string }{
		{"core acl 2.4.0-1 [installed]", "core", "acl", "2.4.0-1"},
		{"omarchy gpu-screen-recorder 5.12.3-2 [installed: 6.0.1-1]", "omarchy", "gpu-screen-recorder", "5.12.3-2"},
		{"extra zsh 5.9.2-1", "extra", "zsh", "5.9.2-1"},
		{"deux champs", "", "", ""},
	}
	for _, c := range cas {
		depot, nom, version := CoupeSync(c.ligne)
		if depot != c.depot || nom != c.nom || version != c.version {
			t.Errorf("CoupeSync(%q) = %q, %q, %q", c.ligne, depot, nom, version)
		}
	}
}

// mkPaquet crée <base>/<nom>-<version> comme le fait alpm dans /var/lib/pacman/local.
func mkPaquet(t *testing.T, base string, entrees ...string) {
	t.Helper()
	for _, e := range entrees {
		if err := os.MkdirAll(filepath.Join(base, e), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallesDe(t *testing.T) {
	base := t.TempDir()
	mkPaquet(t, base, "acl-2.4.0-1", "gpu-screen-recorder-6.0.1-1", "zsh-5.9.2-1")
	// ALPM_DB_VERSION est la seule entrée non répertoire de la vraie base.
	if err := os.WriteFile(filepath.Join(base, "ALPM_DB_VERSION"), []byte("9\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := InstallesDe(base)
	want := []string{"acl 2.4.0-1", "gpu-screen-recorder 6.0.1-1", "zsh 5.9.2-1"}
	assertLignes(t, got, want)
}

func TestInstallesDeDossierAbsent(t *testing.T) {
	// Machine sans pacman : pas d'erreur, liste vide.
	if got := InstallesDe("/nexiste/pas/local"); len(got) != 0 {
		t.Fatalf("attendu vide, obtenu %v", got)
	}
}

func TestNewCatalogBadges(t *testing.T) {
	sync := []string{"core acl 2.4.0-1 [installed]", "extra zsh 5.9.2-1"}
	aur := []string{"google-chrome", "caffeine"}
	cat := NewCatalog(sync, aur, []string{"acl 2.4.0-1", "google-chrome 141.0-1"})

	if b := cat.Badge("zsh"); b != pm.BadgeRepo {
		t.Errorf("zsh : badge %q, attendu %q", b, pm.BadgeRepo)
	}
	if b := cat.Badge("google-chrome"); b != pm.BadgeAUR {
		t.Errorf("google-chrome : badge %q, attendu %q", b, pm.BadgeAUR)
	}
	if !cat.Installed["acl"] || cat.Version("acl") != "2.4.0-1" {
		t.Errorf("acl devait être installé en 2.4.0-1")
	}
	if cat.Installed["zsh"] {
		t.Errorf("zsh n'est pas installé")
	}
}

// Un paquet installé qu'aucune des deux listes ne connaît — construit localement par
// `pacman -U`, ou retiré de l'AUR depuis — entre au catalogue dans la classe « autre ».
func TestNewCatalogInstalleHorsCatalogue(t *testing.T) {
	cat := NewCatalog([]string{"core acl 2.4.0-1"}, nil, []string{"mon-paquet-maison 1.0-1"})
	if b := cat.Badge("mon-paquet-maison"); b != pm.BadgeAUR {
		t.Errorf("badge %q, attendu %q", b, pm.BadgeAUR)
	}
	if !cat.Installed["mon-paquet-maison"] {
		t.Error("le paquet devait être marqué installé")
	}
}

// Un nom que portent à la fois un dépôt et l'AUR reste UNE entrée, celle du dépôt — et
// laisse de quoi la qualifier, sans quoi `yay -S <nom>` ouvre son menu « repo ou AUR ? ».
func TestNewCatalogNomPartageResteAuDepot(t *testing.T) {
	cat := NewCatalog([]string{"extra rustup 1.28.1-1"}, []string{"rustup", "yay-git"}, nil)

	if b := cat.Badge("rustup"); b != pm.BadgeRepo {
		t.Errorf("rustup : badge %q, attendu %q (le dépôt gagne)", b, pm.BadgeRepo)
	}
	if q := cat.Qualified["rustup"]; q != "extra/rustup" {
		t.Errorf("qualification %q, attendu « extra/rustup »", q)
	}
	// Un nom AUR unique n'a rien à qualifier.
	if q := cat.Qualified["yay-git"]; q != "" {
		t.Errorf("yay-git ne devait pas être qualifié, obtenu %q", q)
	}
	// Et il ne doit pas figurer deux fois au catalogue.
	n := 0
	for _, nom := range cat.Names {
		if nom == "rustup" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("rustup figure %d fois au catalogue, attendu 1", n)
	}
}

func assertLignes(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("obtenu %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ligne %d : obtenu %q, attendu %q", i, got[i], want[i])
		}
	}
}

// Le format intermédiaire, celui que `warm` dépose et que la frappe relit. Les trois formes
// de ligne y sont, et l'ordre est déjà celui du popup — Relire n'a plus rien à trier.
func TestFusionner(t *testing.T) {
	lignes := Fusionner(
		[]string{
			"extra zsh 5.9.2-1",
			"extra rustup 1.28.1-1",
			"core acl 2.4.0-1",
			"multilib rustup 1.28.1-1", // même nom dans deux dépôts : le premier gagne
		},
		[]string{"rustup", "caffeine", "caffeine"}, // doublon dans la source AUR
	)
	want := []string{
		"acl\tcore",
		"caffeine",
		"rustup\textra\t+", // les deux le portent
		"zsh\textra",
	}
	assertLignes(t, lignes, want)
}

func TestFusionnerTriSansEgardALaCasse(t *testing.T) {
	lignes := Fusionner([]string{"extra Zebra 1-1", "extra alpha 1-1"}, []string{"Beta"})
	assertLignes(t, lignes, []string{"alpha\textra", "Beta", "Zebra\textra"})
}

// Relire ne déduplique pas — Fusionner l'a fait. Le contrat vaut d'être gardé : un nom qui
// entrerait deux fois dans Names ferait un doublon visible dans le popup.
func TestFusionnerPuisRelire(t *testing.T) {
	cat := NewCatalog(
		[]string{"extra rustup 1.28.1-1", "core acl 2.4.0-1"},
		[]string{"rustup", "caffeine"},
		[]string{"acl 2.4.0-1", "caffeine 1.2-1"},
	)
	assertLignes(t, cat.Names, []string{"acl", "caffeine", "rustup"})
	if cat.Badge("acl") != pm.BadgeRepo || cat.Badge("caffeine") != pm.BadgeAUR {
		t.Errorf("badges : acl=%q caffeine=%q", cat.Badge("acl"), cat.Badge("caffeine"))
	}
	if cat.Qualified["rustup"] != "extra/rustup" {
		t.Errorf("qualification %q", cat.Qualified["rustup"])
	}
	if !cat.Installed["acl"] || !cat.Installed["caffeine"] || cat.Installed["rustup"] {
		t.Errorf("installés : %v", cat.Installed)
	}
}

// Un paquet installé que le catalogue ne connaît pas rouvre le tri : il doit se retrouver
// à sa place, pas à la fin.
func TestRelireInstalleInconnuResteTrie(t *testing.T) {
	cat := Relire(
		[]string{"acl\tcore", "zsh\textra"},
		[]string{"mon-paquet 1.0-1"},
	)
	assertLignes(t, cat.Names, []string{"acl", "mon-paquet", "zsh"})
	if cat.Badge("mon-paquet") != pm.BadgeAUR {
		t.Errorf("badge %q, attendu %q", cat.Badge("mon-paquet"), pm.BadgeAUR)
	}
}
