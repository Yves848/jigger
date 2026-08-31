package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Les six mots réservés ne doivent jamais devenir des verbes de façade. Contrainte
// permanente : aucune sous-commande interne future ne peut porter le nom d'un verbe
// canonique (cf. spec §1).
func TestMotsReserves(t *testing.T) {
	attendus := []string{"pick", "render", "complete", "prompt", "warm", "demo"}
	for _, m := range attendus {
		if !motsReserves[m] {
			t.Errorf("« %s » doit être réservé", m)
		}
	}
	// Un verbe de la façade ne doit surtout pas y figurer.
	for _, v := range []string{"install", "list", "outdated", "search", "info"} {
		if motsReserves[v] {
			t.Errorf("« %s » est un verbe de façade, il ne peut pas être réservé", v)
		}
	}
}

func TestSeparerDrapeaux(t *testing.T) {
	verbe, args, o, err := separerDrapeaux(
		[]string{"install", "--pm", "scoop", "fd", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if verbe != "install" {
		t.Fatalf("verbe = %q", verbe)
	}
	if len(args) != 1 || args[0] != "fd" {
		t.Fatalf("args = %v, attendu [fd]", args)
	}
	if o.PM != "scoop" {
		t.Fatalf("PM = %q, attendu scoop", o.PM)
	}
	if !o.Yes {
		t.Fatal("--yes non pris en compte")
	}
}

func TestSeparerDrapeauxJSON(t *testing.T) {
	_, _, o, err := separerDrapeaux([]string{"outdated", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.JSON {
		t.Fatal("--json non pris en compte")
	}
}

// Un drapeau destiné au gestionnaire ne doit pas être avalé par jigger.
func TestDrapeauxInconnusPassentAuGestionnaire(t *testing.T) {
	_, args, _, err := separerDrapeaux([]string{"install", "--cask", "firefox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "--cask" || args[1] != "firefox" {
		t.Fatalf("args = %v, attendu [--cask firefox]", args)
	}
}

func TestPMSansValeur(t *testing.T) {
	if _, _, _, err := separerDrapeaux([]string{"install", "--pm"}); err == nil {
		t.Fatal("attendu une erreur : --pm sans valeur")
	}
}

// capturerStdout rend ce que f a imprimé sur la sortie standard.
func capturerStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ancien := os.Stdout
	os.Stdout = w
	f()
	os.Stdout = ancien
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// TestRenderSeTaitSansConfigurationSSH vérifie le protocole du silence, et non seulement
// le drapeau qui le porte : une sortie d'UNE SEULE ligne. C'est ce que les deux greffons
// traitent comme « rien à afficher » — `_jigger_fetch` exige deux lignes, `Get-JiggerFrame`
// aussi — et qui les fait effacer le cadre resté à l'écran. Émettre un cadre vide, comme
// avant, faisait clignoter une boîte « aucun candidat » sous chaque frappe d'une ligne ssh
// sur une machine neuve.
func TestRenderSeTaitSansConfigurationSSH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir() sous Windows

	sortie := capturerStdout(t, func() {
		runRender([]string{"--line", "ssh serv", "--color", "never"})
	})
	lignes := strings.Split(strings.TrimRight(sortie, "\n"), "\n")
	if len(lignes) != 1 {
		t.Fatalf("render a émis %d lignes, attendu la seule ligne de métadonnées :\n%s", len(lignes), sortie)
	}
	if !strings.HasPrefix(lignes[0], "count=0\t") {
		t.Errorf("métadonnées = %q", lignes[0])
	}
}

// Le pendant : dès que la configuration existe, le cadre revient. Sans lui, faire taire
// jigger en toutes circonstances passerait pour une correction.
func TestRenderDessineUnCadreQuandLaConfigurationExiste(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".ssh", "config")
	if err := os.WriteFile(cfg, []byte("Host serveur\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	sortie := capturerStdout(t, func() {
		runRender([]string{"--line", "ssh serv", "--color", "never"})
	})
	lignes := strings.Split(strings.TrimRight(sortie, "\n"), "\n")
	if len(lignes) < 2 {
		t.Fatalf("render n'a émis que %d ligne(s), attendu un cadre :\n%s", len(lignes), sortie)
	}
	if !strings.HasPrefix(lignes[0], "count=1\t") {
		t.Errorf("métadonnées = %q, attendu un candidat", lignes[0])
	}
}
