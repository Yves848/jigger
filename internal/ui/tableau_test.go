package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func tableauDeTest(noms ...string) Tableau {
	cellules := make([][]string, len(noms))
	for i, n := range noms {
		cellules[i] = []string{n, "1.0"}
	}
	return NouveauTableau("list", []string{"PAQUET", "ACTUEL"}, cellules)
}

func touche(t Tableau, k tea.KeyType) Tableau {
	nt, _ := t.Update(tea.KeyMsg{Type: k})
	return nt.(Tableau)
}

func saisir(t Tableau, s string) Tableau {
	for _, r := range s {
		nt, _ := t.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		t = nt.(Tableau)
	}
	return t
}

// ^A coche tout ce qui est visible. « Visible » et non « tout le catalogue » : c'est le
// geste utile — on filtre, puis on prend ce qui reste.
func TestToutCocherPorteSurLeFiltre(t *testing.T) {
	tb := tableauDeTest("git", "gitui", "wget")
	tb = saisir(tb, "git")
	tb = touche(tb, tea.KeyCtrlA)

	if n := tb.liste.NbCochees(); n != 2 {
		t.Fatalf("%d lignes cochées, attendu les 2 que le filtre laisse", n)
	}

	tb = touche(tb, tea.KeyEnter)
	if len(tb.Choisis) != 2 {
		t.Fatalf("%v — attendu git et gitui", tb.Choisis)
	}
	for _, n := range tb.Choisis {
		if !strings.HasPrefix(n, "git") {
			t.Errorf("wget n'aurait pas dû être retenu : %v", tb.Choisis)
		}
	}
}

// Tout coché → ^A décoche. Sélection partielle → ^A complète, au lieu d'inverser :
// inverser serait imprévisible.
func TestToutCocherEstUneBasculeTouOuRien(t *testing.T) {
	tb := tableauDeTest("a", "b", "c")

	tb = touche(tb, tea.KeyCtrlA)
	if n := tb.liste.NbCochees(); n != 3 {
		t.Fatalf("%d cochées après ^A, attendu 3", n)
	}
	tb = touche(tb, tea.KeyCtrlA)
	if n := tb.liste.NbCochees(); n != 0 {
		t.Fatalf("%d cochées après le second ^A, attendu 0", n)
	}

	tb = touche(tb, tea.KeyTab) // une seule ligne cochée
	tb = touche(tb, tea.KeyCtrlA)
	if n := tb.liste.NbCochees(); n != 3 {
		t.Fatalf("%d cochées — une sélection partielle doit se compléter, pas s'inverser", n)
	}
}

// Le défaut qu'A-19 corrige : ^B et ^F appartiennent au champ de saisie. Les intercepter
// pour les pages lui volait le déplacement du curseur.
func TestCtrlBEtCtrlFReviennentAuChamp(t *testing.T) {
	tb := tableauDeTest("a", "b", "c", "d", "e", "f", "g", "h")
	tb.liste.DefinirHauteur(2)

	avant := tb.liste.Curseur()
	tb = touche(tb, tea.KeyCtrlF)
	if tb.liste.Curseur() != avant {
		t.Errorf("^F ne doit plus faire défiler : curseur %d → %d", avant, tb.liste.Curseur())
	}
	tb = touche(tb, tea.KeyCtrlB)
	if tb.liste.Curseur() != avant {
		t.Errorf("^B ne doit plus faire défiler : curseur %d", tb.liste.Curseur())
	}

	// Les pages restent accessibles, sur leurs touches à elles.
	tb = touche(tb, tea.KeyPgDown)
	if tb.liste.Curseur() == avant {
		t.Error("PgSuiv doit toujours faire défiler")
	}
}

// Le pied annonce les deux gestes de sélection : sans ça, personne ne les découvrira.
func TestPiedAnnonceLesDeuxSelections(t *testing.T) {
	tb := tableauDeTest("a", "b")
	pied := visible(tb.pied())
	for _, k := range []string{"⇥", "^A", "^R"} {
		if !strings.Contains(pied, k) {
			t.Errorf("touche %q absente du pied : %s", k, pied)
		}
	}
}

// ^R bascule aussi ici, et le mode s'affiche.
func TestTableauBasculeEnRegex(t *testing.T) {
	tb := tableauDeTest("node.js", "nodeXjs")
	tb = saisir(tb, "node.js")
	if len(tb.liste.Filtrees()) != 1 {
		t.Fatalf("%d lignes en texte brut, attendu 1", len(tb.liste.Filtrees()))
	}
	tb = touche(tb, tea.KeyCtrlR)
	if len(tb.liste.Filtrees()) != 2 {
		t.Fatalf("%d lignes en regex, attendu 2", len(tb.liste.Filtrees()))
	}
	if !strings.Contains(visible(tb.View()), "regex]") {
		t.Error("le mode regex doit s'afficher")
	}
}
