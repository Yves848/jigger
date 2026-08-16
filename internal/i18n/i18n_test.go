package i18n

import "testing"

// Le catalogue de test remplace le vrai le temps d'un test, et le rend ensuite.
func avecCatalogue(t *testing.T, table map[string][nbLangues]string) {
	t.Helper()
	ancien := catalogue
	catalogue = table
	t.Cleanup(func() { catalogue = ancien })
}

func TestTRendLaLangueCourante(t *testing.T) {
	avecCatalogue(t, map[string][nbLangues]string{
		"essai.bonjour": {"hello", "bonjour"},
	})
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	if got := T("essai.bonjour"); got != "bonjour" {
		t.Fatalf("fr : %q, attendu « bonjour »", got)
	}
	t.Setenv("JIGGER_LANG", "en")
	Recharger()
	if got := T("essai.bonjour"); got != "hello" {
		t.Fatalf("en : %q, attendu « hello »", got)
	}
}

// Une clé oubliée doit se voir à l'œil plutôt que laisser un trou.
func TestCleAbsenteRendueTelleQuelle(t *testing.T) {
	avecCatalogue(t, map[string][nbLangues]string{})
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	if got := T("popup.inconnue"); got != "popup.inconnue" {
		t.Fatalf("%q, attendu la clé elle-même", got)
	}
}

// C'est ce repli qui permettra de livrer une langue partiellement traduite.
func TestTraductionVideSeReplieSurAnglais(t *testing.T) {
	avecCatalogue(t, map[string][nbLangues]string{
		"essai.partiel": {"only english", ""},
	})
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	if got := T("essai.partiel"); got != "only english" {
		t.Fatalf("%q, attendu le repli anglais", got)
	}
}

func TestTfFormate(t *testing.T) {
	avecCatalogue(t, map[string][nbLangues]string{
		"essai.compte": {"%d packages", "%d paquets"},
	})
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	if got := Tf("essai.compte", 3); got != "3 paquets" {
		t.Fatalf("%q, attendu « 3 paquets »", got)
	}
}

// L'ordre de résolution : JIGGER_LANG l'emporte sur LANG.
func TestJiggerLangPrimeSurLang(t *testing.T) {
	t.Setenv("LANG", "fr_FR.UTF-8")
	t.Setenv("JIGGER_LANG", "en")
	Recharger()
	if Courante() != EN {
		t.Fatalf("JIGGER_LANG doit primer sur LANG")
	}
}

func TestLangDonneLaLangue(t *testing.T) {
	t.Setenv("JIGGER_LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "fr_FR.UTF-8")
	Recharger()
	if Courante() != FR {
		t.Fatalf("LANG=fr_FR.UTF-8 doit donner le français")
	}
}

// Une langue que jigger ne connaît pas n'est pas une erreur : elle retombe sur l'anglais.
func TestLangueInconnueRetombeSurAnglais(t *testing.T) {
	t.Setenv("JIGGER_LANG", "ja")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	Recharger()
	if Courante() != EN {
		t.Fatalf("une langue inconnue doit retomber sur l'anglais")
	}
}
