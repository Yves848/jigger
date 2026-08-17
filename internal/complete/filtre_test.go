package complete

import "testing"

func TestFiltrePrefixe(t *testing.T) {
	f := NouveauFiltre("fire", false)
	for _, cas := range []struct {
		nom    string
		retenu bool
	}{
		{"firefox", true},
		{"FireAlpaca", true}, // la casse est ignorée
		{"campfire", false},  // préfixe, pas sous-chaîne
	} {
		if got := f.Correspond(cas.nom); got != cas.retenu {
			t.Errorf("%q : %v, attendu %v", cas.nom, got, cas.retenu)
		}
	}
}

// Le point n'est un joker qu'en mode regex — c'est toute la différence entre les deux.
func TestFiltrePointSelonLeMode(t *testing.T) {
	if NouveauFiltre("node.js", false).Correspond("nodeXjs") {
		t.Error("en préfixe, le point est un caractère ordinaire")
	}
	if !NouveauFiltre("node.js", true).Correspond("nodeXjs") {
		t.Error("en regex, le point vaut joker")
	}
}

// Une regex peut ancrer, ce qu'un préfixe ne sait pas faire : c'est l'intérêt du mode.
func TestFiltreRegexAncree(t *testing.T) {
	f := NouveauFiltre("^fire.*x$", true)
	if !f.Correspond("firefox") {
		t.Error("firefox devrait correspondre")
	}
	if f.Correspond("firefoxpwa") {
		t.Error("firefoxpwa ne devrait pas : le motif est ancré en fin")
	}
}

// Un motif fautif ne retient rien — le cadre dira « aucun candidat » plutôt que de
// déverser 16 000 entrées parce qu'une parenthèse manque. C'est l'inverse du choix fait
// dans le sélecteur plein écran, et c'est délibéré : ici le motif EST le mot de la ligne.
func TestFiltreMotifFautif(t *testing.T) {
	f := NouveauFiltre("c++", true)
	if !f.Fautif {
		t.Fatal("« c++ » n'est pas une regex valide")
	}
	if f.Correspond("c++") {
		t.Error("un motif fautif ne retient rien")
	}
}

// Le même « c++ » en mode préfixe — le mode par défaut — trouve bien le paquet.
func TestFiltreCPlusPlusEnPrefixe(t *testing.T) {
	if !NouveauFiltre("c++", false).Correspond("c++") {
		t.Error("en préfixe, « c++ » est du texte")
	}
}

// Mot vide : tout passe, dans les deux modes. C'est le cas du popup juste après
// « brew install », où le cadre invite à taper une lettre.
func TestFiltreMotVide(t *testing.T) {
	for _, regex := range []bool{false, true} {
		f := NouveauFiltre("", regex)
		if !f.Vide || !f.Correspond("n'importe quoi") {
			t.Errorf("regex=%v : un mot vide doit tout retenir", regex)
		}
	}
}
