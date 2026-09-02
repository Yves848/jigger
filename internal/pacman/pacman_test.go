package pacman

import (
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func contientMot(liste []string, mot string) bool {
	for _, m := range liste {
		if m == mot {
			return true
		}
	}
	return false
}

// Le point de grammaire du module : les opérations sont annoncées des DEUX côtés, parce
// qu'elles commencent par un tiret et que le moteur route là-dessus. Si l'une des deux
// listes divergeait, « pacman ⇥ » et « pacman -⇥ » ne proposeraient pas la même chose.
func TestOperationsAnnonceesDesDeuxCotes(t *testing.T) {
	for _, cmd := range []string{"pacman", "yay"} {
		m := New(cmd)
		subs, ops := m.Subcommands(), m.Options("")
		if len(subs) != len(ops) {
			t.Fatalf("%s : %d sous-commandes, %d options sans opération", cmd, len(subs), len(ops))
		}
		for i := range subs {
			if subs[i] != ops[i] {
				t.Fatalf("%s : divergence en %d — %q / %q", cmd, i, subs[i], ops[i])
			}
		}
	}
}

func TestYayAjouteSesOperations(t *testing.T) {
	if contientMot(New("pacman").Subcommands(), "-Y") {
		t.Error("-Y est une opération de yay, pas de pacman")
	}
	if !contientMot(New("yay").Subcommands(), "-Y") {
		t.Error("-Y manque aux opérations de yay")
	}
	// yay accepte tout pacman.
	for _, op := range operations {
		if !contientMot(New("yay").Subcommands(), op) {
			t.Errorf("yay devrait accepter %q", op)
		}
	}
}

// complete minuscule la sous-commande avant de la passer : les tables doivent donc être
// indexées en minuscules, sinon « pacman -Rns ⇥ » ne trouverait ni ses options ni son
// vivier d'installés.
func TestTablesIndexeesEnMinuscules(t *testing.T) {
	for op := range optionsParOp {
		if op != strings.ToLower(op) {
			t.Errorf("optionsParOp : clé %q non minusculée", op)
		}
	}
	for op := range optionsYayParOp {
		if op != strings.ToLower(op) {
			t.Errorf("optionsYayParOp : clé %q non minusculée", op)
		}
	}
	for op := range installedOnly {
		if op != strings.ToLower(op) {
			t.Errorf("installedOnly : clé %q non minusculée", op)
		}
	}
}

func TestInstalledOnly(t *testing.T) {
	m := New("pacman")
	// Les familles -R et -Q interrogent la base locale.
	for _, op := range []string{"-R", "-Rns", "-Rs", "-Q", "-Qu", "-Qi", "-Qdt"} {
		if !m.InstalledOnly(op) {
			t.Errorf("%s devrait ne proposer que les installés", op)
		}
	}
	// La famille -S puise au catalogue.
	for _, op := range []string{"-S", "-Syu", "-Ss", "-Si", "-U"} {
		if m.InstalledOnly(op) {
			t.Errorf("%s devrait puiser au catalogue", op)
		}
	}
	// -Qo prend un chemin de fichier : ni l'un ni l'autre vivier n'a de sens, et le
	// catalogue est le moindre mal.
	if m.InstalledOnly("-Qo") {
		t.Error("-Qo prend un chemin, pas un paquet installé")
	}
}

func TestOptionsParOperation(t *testing.T) {
	m := New("pacman")
	if !contientMot(m.Options("-S"), "--needed") {
		t.Error("--needed manque à -S")
	}
	if !contientMot(m.Options("-Rns"), "--cascade") {
		t.Error("--cascade manque à -Rns")
	}
	// Les communes s'ajoutent partout, y compris sur une opération sans table propre.
	if !contientMot(m.Options("-Ss"), "--quiet") {
		t.Error("--quiet manque aux options communes de -Ss")
	}
	// --aur est de yay, pas de pacman.
	if contientMot(m.Options("-S"), "--aur") {
		t.Error("pacman ne connaît pas --aur")
	}
	if !contientMot(New("yay").Options("-S"), "--aur") {
		t.Error("--aur manque à yay -S")
	}
}

func TestInsertQualifieLesNomsPartagesPourYay(t *testing.T) {
	cat := NewCatalog([]string{"extra rustup 1.28.1-1"}, []string{"rustup", "caffeine"}, nil)

	yay, pac := New("yay"), New("pacman")

	if got := yay.Insert(cat, "-S", "", "rustup"); got != "extra/rustup" {
		t.Errorf("yay -S rustup : inséré %q, attendu « extra/rustup »", got)
	}
	if got := yay.Insert(cat, "install", "", "rustup"); got != "extra/rustup" {
		t.Errorf("verbe de façade : inséré %q, attendu « extra/rustup »", got)
	}
	// Un nom AUR unique s'insère nu.
	if got := yay.Insert(cat, "-S", "", "caffeine"); got != "caffeine" {
		t.Errorf("caffeine : inséré %q, attendu nu", got)
	}
	// Hors de la famille -S, il n'y a plus rien à départager : le paquet est installé.
	if got := yay.Insert(cat, "-Rns", "", "rustup"); got != "rustup" {
		t.Errorf("yay -Rns rustup : inséré %q, attendu nu", got)
	}
	// pacman tranche par priorité de dépôt, sans erreur ni question : rien à corriger.
	if got := pac.Insert(cat, "-S", "", "rustup"); got != "rustup" {
		t.Errorf("pacman -S rustup : inséré %q, attendu nu", got)
	}
}

// Le module doit satisfaire les deux contrats, pour les deux mots de commande.
func TestContrats(t *testing.T) {
	var _ pm.Manager = New("pacman")
	var _ pm.Bindings = New("yay")
}
