package i18n

import "testing"

// Une locale exotique ne doit ni planter ni changer autre chose que la langue.
func TestLocaleExotiqueNeCassePas(t *testing.T) {
	for _, valeur := range []string{"", "C", "POSIX", "ja_JP.UTF-8", "fr", "en_GB", "fr-CA"} {
		t.Setenv("JIGGER_LANG", "")
		t.Setenv("LC_ALL", valeur)
		Recharger()
		if l := Courante(); l != EN && l != FR {
			t.Fatalf("LC_ALL=%q donne une langue hors bornes", valeur)
		}
	}
}
