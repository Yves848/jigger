package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func ecrireConfig(t *testing.T, contenu string) {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JIGGER_CONFIG", chemin)
	Recharger()
	t.Cleanup(Recharger)
}

func TestDureeLueDepuisLeFichier(t *testing.T) {
	Declarer(Reglage{Cle: "essai_ttl", CleI18n: "cfg.ttl", Type: TypeDuree, Defaut: "24h"})
	ecrireConfig(t, "essai_ttl = 1h30m\n")

	if got := Duree("essai_ttl", 24*time.Hour); got != 90*time.Minute {
		t.Fatalf("%v, attendu 1h30m", got)
	}
}

func TestDureeDefautQuandRienNEstRegle(t *testing.T) {
	Declarer(Reglage{Cle: "essai_ttl2", CleI18n: "cfg.ttl", Type: TypeDuree, Defaut: "24h"})
	ecrireConfig(t, "")

	if got := Duree("essai_ttl2", 24*time.Hour); got != 24*time.Hour {
		t.Fatalf("%v, attendu le défaut", got)
	}
}

// Une durée illisible ne fait pas échouer : le défaut s'applique. Refuser de charger un
// catalogue à cause d'une faute de frappe serait une punition disproportionnée — d'autant
// que cette valeur est sur le chemin de la frappe.
func TestDureeIllisibleRetombeSurLeDefaut(t *testing.T) {
	Declarer(Reglage{Cle: "essai_ttl3", CleI18n: "cfg.ttl", Type: TypeDuree, Defaut: "24h"})
	for _, mauvaise := range []string{"vingt-quatre heures", "24", "-5h", ""} {
		ecrireConfig(t, "essai_ttl3 = "+mauvaise+"\n")
		if got := Duree("essai_ttl3", 24*time.Hour); got != 24*time.Hour {
			t.Errorf("%q → %v, attendu le défaut", mauvaise, got)
		}
	}
}

// L'environnement l'emporte, ici aussi : la préséance est la même partout.
func TestDureeEnvironnementLEmporte(t *testing.T) {
	Declarer(Reglage{Cle: "essai_ttl4", CleI18n: "cfg.ttl", Type: TypeDuree, Defaut: "24h"})
	ecrireConfig(t, "essai_ttl4 = 1h\n")
	t.Setenv("JIGGER_ESSAI_TTL4", "5m")

	if got := Duree("essai_ttl4", 24*time.Hour); got != 5*time.Minute {
		t.Fatalf("%v, attendu 5m — l'environnement l'emporte", got)
	}
}

// Une clé non déclarée n'a pas de valeur : c'est le défaut, sans quoi une faute de frappe
// dans le fichier passerait pour un réglage.
func TestDureeCleNonDeclaree(t *testing.T) {
	ecrireConfig(t, "inconnue = 1h\n")
	if got := Duree("inconnue", 24*time.Hour); got != 24*time.Hour {
		t.Fatalf("%v, attendu le défaut", got)
	}
}
