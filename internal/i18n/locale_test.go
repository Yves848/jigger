package i18n

import "testing"

// Une locale que jigger ne sait pas traduire doit donner l'anglais — pas une page
// blanche, et pas la langue de la machine qui lance les tests.
//
// La version précédente de ce test comparait la langue rendue à « ni EN ni FR » : une
// condition insatisfiable, resoudre() ne rendant jamais rien d'autre. Elle ne vidait par
// ailleurs ni LC_MESSAGES ni LANG, si bien qu'un ja_JP.UTF-8 posé dans LC_ALL laissait un
// LANG=fr_FR.UTF-8 hérité de l'environnement décider à sa place. On pose donc les quatre
// variables, et on assère la valeur attendue.
func TestLocaleExotiqueDonneAnglais(t *testing.T) {
	t.Cleanup(Recharger)

	// Sous Windows, resoudre() consulte encore la culture du système quand aucune variable
	// ne tranche (langue_windows.go) : sur une machine réglée en français, « ja_JP.UTF-8 »
	// y donnerait FR — et ce serait juste. Ce test se passait donc de tourner là où cette
	// culture existe, c'est-à-dire sur la seule plateforme où elle décide. Il la pose
	// désormais lui-même : les sept cas valent sur les trois plateformes.
	avecCultureSysteme(t, "")

	for _, cas := range []struct {
		lcAll    string
		attendu  Langue
		pourquoi string
	}{
		{"ja_JP.UTF-8", EN, "une langue que jigger ne traduit pas"},
		{"C", EN, "la locale POSIX minimale"},
		{"POSIX", EN, "son autre nom"},
		{"", EN, "aucune locale du tout"},
		{"en_GB", EN, "un anglais régional"},
		{"fr_FR.UTF-8", FR, "le français, pour que le test ne passe pas par accident"},
		{"fr-CA", FR, "un français régional, séparé par un tiret"},
	} {
		t.Setenv("JIGGER_LANG", "")
		t.Setenv("LC_ALL", cas.lcAll)
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANG", "")
		Recharger()

		if l := Courante(); l != cas.attendu {
			t.Errorf("LC_ALL=%q (%s) : langue = %d, attendu %d", cas.lcAll, cas.pourquoi, l, cas.attendu)
		}
	}
}
