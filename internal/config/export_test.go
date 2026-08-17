package config

import (
	"os/exec"
	"strings"
	"testing"
)

// Les valeurs hostiles : chacune a déjà cassé quelque chose quelque part.
var valeursHostiles = []struct{ nom, valeur string }{
	{"simple", "12"},
	{"espaces", "^ "},
	{"apostrophe", "l'apostrophe"},        // le défaut qui a tronqué des messages
	{"guillemets", `des "guillemets"`},    //
	{"dollar", "$HOME et ${PATH}"},        // ne doit jamais être développé
	{"backtick", "des `backticks`"},       // ni exécuté
	{"substitution", "$(echo compromis)"}, // surtout pas
	{"accents", "éèçàù"},                  // le catalogue en contient
	{"chevrons", "a > b | c & d"},         // redirections et tubes
	{"antislash", `un\antislash`},         //
	{"tout", `'"$(x)` + "`y`" + ` z\ é`},  //
}

// exporteEtRelit fait passer une valeur par l'export, puis par le shell, et rend ce que le
// shell a réellement mis dans la variable. C'est une EXÉCUTION : relire le code
// d'échappement ne prouverait rien — c'est précisément ainsi que le projet s'est fait
// prendre par une apostrophe non échappée.
func exporteEtRelit(t *testing.T, sh Shell, valeur string) (string, bool) {
	t.Helper()

	f := Nouveau()
	f.Poser("key", valeur)
	code := Export(f, sh)

	var cmd *exec.Cmd
	switch sh {
	case PowerShell:
		if _, err := exec.LookPath("pwsh"); err != nil {
			return "", false
		}
		cmd = exec.Command("pwsh", "-NoProfile", "-Command", code+"\n[Console]::Out.Write($env:JIGGER_KEY)")
	default:
		if _, err := exec.LookPath("zsh"); err != nil {
			return "", false
		}
		cmd = exec.Command("zsh", "-c", code+"\nprintf %s \"$JIGGER_KEY\"")
	}
	// Un environnement nu : sinon un JIGGER_KEY hérité fausserait le résultat, et
	// l'export sauterait la ligne (la valeur venant alors de l'environnement).
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/local/bin:/opt/homebrew/bin"}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("le shell a refusé ce que l'export a produit : %v\n--- code émis ---\n%s", err, code)
	}
	return string(out), true
}

func TestExportSurvitAZsh(t *testing.T) {
	for _, c := range valeursHostiles {
		t.Run(c.nom, func(t *testing.T) {
			got, ok := exporteEtRelit(t, Zsh, c.valeur)
			if !ok {
				t.Skip("zsh absent")
			}
			if got != c.valeur {
				t.Fatalf("zsh a lu %q, la valeur était %q", got, c.valeur)
			}
		})
	}
}

func TestExportSurvitAPowerShell(t *testing.T) {
	for _, c := range valeursHostiles {
		t.Run(c.nom, func(t *testing.T) {
			got, ok := exporteEtRelit(t, PowerShell, c.valeur)
			if !ok {
				t.Skip("pwsh absent")
			}
			if got != c.valeur {
				t.Fatalf("pwsh a lu %q, la valeur était %q", got, c.valeur)
			}
		})
	}
}

// L'export ne dicte que ce que le fichier a fixé : ni les défauts, que le greffon connaît
// déjà, ni ce qui vient de l'environnement, qui y est déjà.
func TestExportNEmetQueLeFichier(t *testing.T) {
	f := Nouveau()
	f.Poser("rows", "12")
	code := Export(f, Zsh)

	if !strings.Contains(code, "JIGGER_ROWS='12'") {
		t.Errorf("le réglage du fichier doit être dicté :\n%s", code)
	}
	if strings.Contains(code, "JIGGER_MIN_COLUMNS") {
		t.Errorf("un défaut n'a pas à être dicté :\n%s", code)
	}
	if strings.Contains(code, "JIGGER_PAGER") {
		t.Errorf("un réglage que le greffon ne lit pas n'a rien à faire dans l'export :\n%s", code)
	}
}

// Ce qui vient de l'environnement n'est pas réémis : sinon `JIGGER_ROWS=3 zsh` serait
// écrasé par le fichier au chargement du greffon — l'inverse de la préséance annoncée.
func TestExportNEcrasePasLEnvironnement(t *testing.T) {
	t.Setenv("JIGGER_ROWS", "3")
	f := Nouveau()
	f.Poser("rows", "12")
	if code := Export(f, Zsh); strings.Contains(code, "JIGGER_ROWS") {
		t.Errorf("l'environnement l'emporte, donc rien à dicter :\n%s", code)
	}
}
