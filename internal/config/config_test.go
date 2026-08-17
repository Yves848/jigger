package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// La préséance, exhaustivement : c'est le cœur de l'ADR-0003, et la seule implémentation.
func TestResoudre(t *testing.T) {
	valeur := func(s string) *string { return &s }
	cas := []struct {
		nom     string
		env     string
		fichier *string
		defaut  string
		attendu string
		prov    Provenance
	}{
		{"rien nulle part", "", nil, "8", "8", DuDefaut},
		{"le fichier seul", "", valeur("12"), "8", "12", DuFichier},
		{"l'environnement seul", "3", nil, "8", "3", DeLEnvironnement},
		{"l'environnement l'emporte sur le fichier", "3", valeur("12"), "8", "3", DeLEnvironnement},

		// Une variable vide compte comme absente : c'est la convention du projet, et ce
		// qui permet de neutraliser un réglage le temps d'une commande.
		{"environnement vide = absent", "", valeur("12"), "8", "12", DuFichier},

		// Une valeur vide DANS LE FICHIER, elle, a été écrite exprès. Elle l'emporte donc
		// sur le défaut — sans quoi on ne pourrait jamais vider un réglage.
		{"valeur vide dans le fichier", "", valeur(""), "8", "", DuFichier},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			v, p := Resoudre(c.env, c.fichier, c.defaut)
			if v != c.attendu || p != c.prov {
				t.Fatalf("(%q, %v), attendu (%q, %v)", v, p, c.attendu, c.prov)
			}
		})
	}
}

func TestAnalyser(t *testing.T) {
	src := `# un commentaire
rows = 12

  key   =   ^
vide =
sans signe egal
# lang = fr   ← commenté, donc absent
`
	f, err := Analyser(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if v := f.Valeur("rows"); v == nil || *v != "12" {
		t.Errorf("rows = %v", v)
	}
	if v := f.Valeur("key"); v == nil || *v != "^" {
		t.Errorf("key = %v — les espaces autour doivent tomber", v)
	}
	if v := f.Valeur("vide"); v == nil || *v != "" {
		t.Errorf("vide = %v — une clé sans valeur existe et vaut « »", v)
	}
	if f.Valeur("lang") != nil {
		t.Error("une ligne commentée ne doit pas être lue")
	}
	if f.Valeur("sans signe egal") != nil {
		t.Error("une ligne sans « = » est ignorée, pas devinée")
	}
}

// Une valeur dont l'espace compte doit traverser l'aller-retour intacte. « ^ » — c'est
// Ctrl-Espace, et c'est une valeur documentée de JIGGER_KEY : la rogner rendrait la touche
// inopérante, silencieusement.
func TestEspaceSignificatifPreserve(t *testing.T) {
	t.Setenv("JIGGER_CONFIG", filepath.Join(t.TempDir(), "config"))
	f := Nouveau()
	for _, v := range []string{"^ ", " avant", "des \"guillemets\"", ""} {
		f.Poser("essai", v)
		if err := f.Ecrire(); err != nil {
			t.Fatal(err)
		}
		relu, err := Charger()
		if err != nil {
			t.Fatal(err)
		}
		if got := relu.Valeur("essai"); got == nil || *got != v {
			t.Errorf("%q relu %v", v, got)
		}
	}
}

// Aller-retour : écrire, relire, retrouver la même chose.
func TestAllerRetour(t *testing.T) {
	t.Setenv("JIGGER_CONFIG", filepath.Join(t.TempDir(), "config"))

	f := Nouveau()
	f.Poser("rows", "12")
	f.Poser("lang", "fr")
	f.Poser("key", "^ ")
	if err := f.Ecrire(); err != nil {
		t.Fatal(err)
	}

	relu, err := Charger()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ cle, val string }{{"rows", "12"}, {"lang", "fr"}, {"key", "^ "}} {
		if v := relu.Valeur(c.cle); v == nil || *v != c.val {
			t.Errorf("%s = %v, attendu %q", c.cle, v, c.val)
		}
	}
	// L'ordre d'écriture est conservé : relire puis réécrire ne mélange pas le fichier.
	if got := strings.Join(relu.Cles(), ","); got != "rows,lang,key" {
		t.Errorf("ordre = %q", got)
	}
}

// Un fichier absent est le cas ordinaire, pas une erreur.
func TestFichierAbsent(t *testing.T) {
	t.Setenv("JIGGER_CONFIG", filepath.Join(t.TempDir(), "jamais-ecrit"))
	f, err := Charger()
	if err != nil {
		t.Fatalf("un fichier absent ne doit pas être une erreur : %v", err)
	}
	if len(f.Cles()) != 0 {
		t.Errorf("attendu vide, obtenu %v", f.Cles())
	}
}

func TestRetirer(t *testing.T) {
	f := Nouveau()
	f.Poser("a", "1")
	f.Poser("b", "2")
	f.Retirer("a")
	if f.Valeur("a") != nil {
		t.Error("la clé retirée doit disparaître, pas devenir vide")
	}
	if got := strings.Join(f.Cles(), ","); got != "b" {
		t.Errorf("ordre après retrait = %q", got)
	}
}

// Le nom de la variable d'environnement se déduit de la clé : une seule vérité.
func TestEnvDeduit(t *testing.T) {
	for _, c := range []struct{ cle, env string }{
		{"rows", "JIGGER_ROWS"},
		{"min_columns", "JIGGER_MIN_COLUMNS"},
		{"bin", "JIGGER_BIN"},
	} {
		if got := (Reglage{Cle: c.cle}).Env(); got != c.env {
			t.Errorf("%q → %q, attendu %q", c.cle, got, c.env)
		}
	}
}

// Les réglages déclarés doivent correspondre aux variables que le code lit réellement.
// Une déclaration qui invente une clé serait pire qu'une absence : l'écran proposerait un
// réglage sans effet.
func TestDeclarationsCoherentes(t *testing.T) {
	vues := map[string]bool{}
	for _, r := range Declares {
		if vues[r.Cle] {
			t.Errorf("%q déclaré deux fois", r.Cle)
		}
		vues[r.Cle] = true
		if r.CleI18n == "" {
			t.Errorf("%q n'a pas de libellé traduit", r.Cle)
		}
	}
	for _, attendu := range []string{"live", "rows", "key", "lang", "pager", "bin"} {
		if !vues[attendu] {
			t.Errorf("%q devrait être déclaré", attendu)
		}
	}
}
