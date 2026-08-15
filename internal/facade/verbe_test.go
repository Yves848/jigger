package facade

import (
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// Le verbe composé se reconnaît avant le verbe simple : « source add extras » est
// « source add », pas « source » avec un argument.
func TestResoudreVerbeComposeDAbord(t *testing.T) {
	dispo := []pm.Manager{brew.New()}

	v, args, _, err := resoudreVerbe([]string{"source", "add", "extras"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if v != "source add" {
		t.Fatalf("verbe = %q, attendu « source add »", v)
	}
	if len(args) != 1 || args[0] != "extras" {
		t.Fatalf("args = %v, attendu [extras]", args)
	}

	v, args, _, err = resoudreVerbe([]string{"source"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if v != "source" || len(args) != 0 {
		t.Fatalf("verbe = %q, args = %v, attendu « source » sans argument", v, args)
	}
}

func TestResoudreVerbeRendLesGestionnairesCapables(t *testing.T) {
	dispo := []pm.Manager{winget.New(), scoop.New()}

	_, _, capables, err := resoudreVerbe([]string{"install", "fd"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if len(capables) != 2 {
		t.Fatalf("%d gestionnaires capables, attendu 2", len(capables))
	}

	// doctor : scoop sait (checkup), winget non.
	_, _, capables, err = resoudreVerbe([]string{"doctor"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if len(capables) != 1 || capables[0].Cmd() != "scoop" {
		t.Fatalf("capables = %v, attendu [scoop]", capables)
	}
}

// Le message doit nommer qui saurait faire, et sous quel nom : c'est tout l'intérêt du
// modèle de capacités.
func TestVerbeConnuAilleursMaisIndisponible(t *testing.T) {
	dispo := []pm.Manager{winget.New()}
	tous := []pm.Manager{brew.New(), winget.New(), scoop.New()}

	_, _, _, err := resoudreVerbe([]string{"doctor"}, dispo, tous)
	if err == nil {
		t.Fatal("attendu une erreur : winget seul ne sait pas doctor")
	}
	msg := err.Error()
	for _, attendu := range []string{"doctor", "scoop", "checkup"} {
		if !strings.Contains(msg, attendu) {
			t.Errorf("le message ne contient pas %q : %s", attendu, msg)
		}
	}
}

func TestVerbeInconnuDeTous(t *testing.T) {
	tous := []pm.Manager{brew.New(), winget.New(), scoop.New()}

	_, _, _, err := resoudreVerbe([]string{"teleporter"}, tous, tous)
	if err == nil {
		t.Fatal("attendu une erreur pour un verbe qui n'existe nulle part")
	}
	if !strings.Contains(err.Error(), "teleporter") {
		t.Errorf("le message doit nommer le verbe : %s", err.Error())
	}
}

func TestLigneVide(t *testing.T) {
	tous := []pm.Manager{brew.New()}
	if _, _, _, err := resoudreVerbe(nil, tous, tous); err == nil {
		t.Fatal("attendu une erreur sur une ligne vide")
	}
}
