package pacman

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Les sorties de ce fichier sont copiées telles quelles depuis la machine de
// développement (Arch / Omarchy, pacman 7.1.0, yay 13.0.1) le 2 septembre 2026.

func TestParseList(t *testing.T) {
	out := []byte(`abseil-cpp 20260817.0-1
acl 2.4.0-1
gpu-screen-recorder 6.0.1-1
`)
	rows, err := parseList(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("%d lignes, attendu 3", len(rows))
	}
	if rows[2].Name != "gpu-screen-recorder" || rows[2].Version != "6.0.1-1" {
		t.Errorf("ligne 3 : %+v", rows[2])
	}
}

func TestParseOutdated(t *testing.T) {
	out := []byte(`linux 6.10.1.arch1-1 -> 6.10.2.arch1-1
zsh 5.9.2-1 -> 5.9.3-1
firefox 141.0-1 -> 142.0-1 [ignored]
`)
	rows, err := parseOutdated(out)
	if err != nil {
		t.Fatal(err)
	}
	// firefox est tenu par IgnorePkg : il ne bougera pas au prochain -Syu, donc le
	// compter serait annoncer une mise à jour qui n'arrivera pas.
	if len(rows) != 2 {
		t.Fatalf("%d lignes, attendu 2 (firefox est ignoré) : %+v", len(rows), rows)
	}
	if rows[0].Name != "linux" || rows[0].Version != "6.10.1.arch1-1" || rows[0].Available != "6.10.2.arch1-1" {
		t.Errorf("ligne 1 : %+v", rows[0])
	}
}

func TestParseOutdatedIgnoreCeQuiNEstPasUneMiseAJour(t *testing.T) {
	rows, err := parseOutdated([]byte("acl 2.4.0-1\n:: rien à faire\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("attendu aucune ligne, obtenu %+v", rows)
	}
}

func TestParseSearch(t *testing.T) {
	// La description est indentée : c'est le seul critère fiable pour l'écarter, car son
	// texte peut contenir n'importe quoi — barre oblique comprise.
	out := []byte(`extra/zsh 5.9.2-1 [installed]
    A very advanced and programmable command interpreter (shell) for UNIX
omarchy/yay 13.0.1-1 (4.5 MiB 11.3 MiB) (Installed)
    Yet another yogurt. Pacman wrapper and AUR helper written in go.
aur/yay-git 13.0.1.r0.gabc-1 (+1234 5.67%) (Installed)
    Yet another yogurt — dernier commit / GNU-Linux
`)
	rows, err := parseSearch(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("%d résultats, attendu 3 : %+v", len(rows), rows)
	}
	if rows[0].Name != "zsh" || rows[0].Source != "extra" || rows[0].Kind != pm.BadgeRepo {
		t.Errorf("zsh : %+v", rows[0])
	}
	if rows[1].Name != "yay" || rows[1].Available != "13.0.1-1" {
		t.Errorf("yay : %+v", rows[1])
	}
	// Le préfixe « aur/ » donne le badge sans travail supplémentaire.
	if rows[2].Name != "yay-git" || rows[2].Kind != pm.BadgeAUR {
		t.Errorf("yay-git : %+v", rows[2])
	}
}

func TestParseSearchSansResultat(t *testing.T) {
	rows, err := parseSearch(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("attendu aucun résultat, obtenu %+v", rows)
	}
}

func TestParseVersion(t *testing.T) {
	// La vraie sortie de `pacman --version` : un dessin ASCII, puis la version.
	out := ` .--.                  Pacman v7.1.0 - libalpm v16.0.1
/ _.-' .-.  .-.  .-.   Copyright (C) 2006-2025 Pacman Development Team
\  '-. '-'  '-'  '-'   Copyright (C) 2002-2006 Judd Vinet
 '--'
                       This program may be freely redistributed under
                       the terms of the GNU General Public License.
`
	if v := ParseVersion(out); v != "7.1.0" {
		t.Errorf("ParseVersion = %q, attendu « 7.1.0 »", v)
	}
	// Sortie inattendue : chaîne vide, et le prompt masque le bloc plutôt que d'afficher
	// n'importe quoi.
	if v := ParseVersion("commande introuvable"); v != "" {
		t.Errorf("ParseVersion = %q, attendu vide", v)
	}
}
