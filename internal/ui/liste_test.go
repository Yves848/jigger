package ui

import (
	"reflect"
	"testing"
)

// ligneTest est la plus petite Ligne possible : une clé, une cellule.
type ligneTest struct{ nom string }

func (l ligneTest) Cle() string        { return l.nom }
func (l ligneTest) Cellules() []string { return []string{l.nom} }

func lignes(noms ...string) []Ligne {
	out := make([]Ligne, len(noms))
	for i, n := range noms {
		out[i] = ligneTest{n}
	}
	return out
}

func cles(ls []Ligne) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = l.Cle()
	}
	return out
}

func TestFiltreSousChaineIgnoreLaCasse(t *testing.T) {
	l := NouvelleListe(lignes("Git", "gitui", "wget", "node"), 10)
	l.Filtrer("GIT")
	if got := cles(l.Filtrees()); !reflect.DeepEqual(got, []string{"Git", "gitui"}) {
		t.Fatalf("obtenu %v", got)
	}
}

// Le point est un caractère ordinaire tant qu'on n'a pas basculé : c'est toute la
// différence entre les deux modes, et le piège que la bascule explicite évite.
func TestSousChaineNeTraitePasLePointCommeUnJoker(t *testing.T) {
	l := NouvelleListe(lignes("node.js", "nodeXjs"), 10)
	l.Filtrer("node.js")
	if got := cles(l.Filtrees()); !reflect.DeepEqual(got, []string{"node.js"}) {
		t.Fatalf("obtenu %v, attendu le seul node.js", got)
	}
}

func TestRegexFiltreVraiment(t *testing.T) {
	l := NouvelleListe(lignes("node.js", "nodeXjs", "wget"), 10)
	l.BasculerMode()
	l.Filtrer("node.js")
	if got := cles(l.Filtrees()); !reflect.DeepEqual(got, []string{"node.js", "nodeXjs"}) {
		t.Fatalf("obtenu %v, attendu les deux", got)
	}
	if l.Mode() != FiltreRegex {
		t.Error("le mode devrait être regex")
	}
}

// Un motif fautif ne vide pas la liste : il la laisse entière et se signale. Vider
// l'affichage laisserait croire qu'aucun paquet ne correspond.
func TestRegexInvalideNeVidePasLaListe(t *testing.T) {
	l := NouvelleListe(lignes("c++", "gcc", "clang"), 10)
	l.BasculerMode()
	l.Filtrer("c++")
	if l.Valide() {
		t.Error("« c++ » n'est pas une regex valide : Valide() devrait dire non")
	}
	if got := len(l.Filtrees()); got != 3 {
		t.Fatalf("%d lignes affichées, attendu les 3", got)
	}
}

// Le même « c++ » en sous-chaîne — le mode par défaut — trouve bien le paquet.
func TestCPlusPlusTrouvableEnSousChaine(t *testing.T) {
	l := NouvelleListe(lignes("c++", "gcc"), 10)
	l.Filtrer("c++")
	if got := cles(l.Filtrees()); !reflect.DeepEqual(got, []string{"c++"}) {
		t.Fatalf("obtenu %v", got)
	}
}

func TestBasculerConserveLaSaisie(t *testing.T) {
	l := NouvelleListe(lignes("git", "gitui"), 10)
	l.Filtrer("git")
	l.BasculerMode()
	if got := len(l.Filtrees()); got != 2 {
		t.Fatalf("%d lignes après bascule, attendu 2", got)
	}
	l.BasculerMode()
	if l.Mode() != FiltreSousChaine {
		t.Error("deux bascules devraient ramener en sous-chaîne")
	}
}

func TestCurseurEtDefilement(t *testing.T) {
	l := NouvelleListe(lignes("a", "b", "c", "d", "e"), 2)
	for i := 0; i < 4; i++ {
		l.Bas()
	}
	if l.Curseur() != 4 {
		t.Fatalf("curseur = %d, attendu 4", l.Curseur())
	}
	if l.Offset() != 3 {
		t.Fatalf("offset = %d, attendu 3 (hauteur 2)", l.Offset())
	}
	vis, sel := l.Visibles()
	if got := cles(vis); !reflect.DeepEqual(got, []string{"d", "e"}) || sel != 1 {
		t.Fatalf("visibles %v, sel %d", got, sel)
	}
}

func TestBornesDuCurseur(t *testing.T) {
	l := NouvelleListe(lignes("a", "b"), 5)
	l.Haut() // déjà en haut
	if l.Curseur() != 0 {
		t.Fatal("le curseur ne doit pas passer sous zéro")
	}
	l.Bas()
	l.Bas()
	l.Bas() // déjà en bas
	if l.Curseur() != 1 {
		t.Fatalf("curseur = %d, attendu 1", l.Curseur())
	}
}

func TestPages(t *testing.T) {
	l := NouvelleListe(lignes("a", "b", "c", "d", "e", "f", "g"), 3)
	l.PageBas()
	if l.Curseur() != 3 {
		t.Fatalf("après une page : %d, attendu 3", l.Curseur())
	}
	l.PageBas()
	if l.Curseur() != 6 {
		t.Fatalf("après deux pages : %d, attendu 6 (dernière ligne)", l.Curseur())
	}
	l.PageBas() // au-delà de la fin
	if l.Curseur() != 6 {
		t.Fatalf("la page ne doit pas dépasser la fin : %d", l.Curseur())
	}
	l.PageHaut()
	if l.Curseur() != 3 {
		t.Fatalf("retour d'une page : %d, attendu 3", l.Curseur())
	}
}

func TestListeVide(t *testing.T) {
	l := NouvelleListe(nil, 5)
	if l.Courante() != nil {
		t.Error("aucune ligne courante sur une liste vide")
	}
	l.Bas()
	l.PageBas()
	l.Cocher() // ne doit pas paniquer
	if vis, _ := l.Visibles(); vis != nil {
		t.Error("rien à afficher")
	}
	if l.Choix() != nil {
		t.Error("aucun choix possible")
	}
}

// Sans rien cocher, valider retient la ligne courante — c'est ce que le pied annonce.
func TestChoixSansCoche(t *testing.T) {
	l := NouvelleListe(lignes("a", "b", "c"), 5)
	l.Bas()
	if got := cles(l.Choix()); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("obtenu %v", got)
	}
}

func TestCocherPuisValider(t *testing.T) {
	l := NouvelleListe(lignes("a", "b", "c"), 5)
	l.Cocher() // a
	l.Bas()
	l.Bas()
	l.Cocher() // c
	if got := cles(l.Choix()); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("obtenu %v", got)
	}
	if l.NbCochees() != 2 {
		t.Errorf("%d cochées", l.NbCochees())
	}
}

func TestRecocherDecoche(t *testing.T) {
	l := NouvelleListe(lignes("a", "b"), 5)
	l.Cocher()
	l.Cocher()
	if l.NbCochees() != 0 {
		t.Fatal("cocher deux fois doit décocher")
	}
}

// Le point qui rend la sélection utilisable : on coche, on affine le filtre, on coche
// encore — et rien de ce qu'on avait coché ne s'est perdu en route.
func TestLigneCocheePuisMasqueeResteRetenue(t *testing.T) {
	l := NouvelleListe(lignes("firefox", "wget", "git"), 5)
	l.Cocher() // firefox
	l.Filtrer("git")
	if got := len(l.Filtrees()); got != 1 {
		t.Fatalf("%d lignes filtrées, attendu 1", got)
	}
	l.Cocher() // git
	if got := cles(l.Choix()); !reflect.DeepEqual(got, []string{"firefox", "git"}) {
		t.Fatalf("obtenu %v — une ligne cochée puis masquée doit rester retenue", got)
	}
}

// Le curseur ne doit jamais désigner une ligne qui n'existe plus après un filtrage.
func TestFiltrerRecaleLeCurseur(t *testing.T) {
	l := NouvelleListe(lignes("a", "b", "c"), 5)
	l.Bas()
	l.Bas() // curseur sur c
	l.Filtrer("a")
	if l.Curseur() != 0 {
		t.Fatalf("curseur = %d, attendu 0", l.Curseur())
	}
	if l.Courante().Cle() != "a" {
		t.Fatalf("ligne courante = %q", l.Courante().Cle())
	}
}

func TestHauteurMinimale(t *testing.T) {
	l := NouvelleListe(lignes("a", "b"), 0)
	if l.Hauteur() != 1 {
		t.Fatalf("hauteur = %d, attendu 1", l.Hauteur())
	}
	l.DefinirHauteur(-3)
	if l.Hauteur() != 1 {
		t.Fatalf("hauteur = %d après -3, attendu 1", l.Hauteur())
	}
}
