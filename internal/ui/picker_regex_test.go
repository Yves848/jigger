package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
)

func selecteurDeTest(noms ...string) Model {
	items := make([]complete.Item, len(noms))
	for i, n := range noms {
		items[i] = complete.Item{Name: n}
	}
	return New("test", complete.Result{Items: items})
}

func frappe(m Model, r rune) Model {
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return nm.(Model)
}

func ctrlR(m Model) Model {
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	return nm.(Model)
}

// ^R bascule le sélecteur en expression rationnelle. Le moteur venait d'A-10 ; ce qui
// manquait, c'était la touche.
func TestSelecteurBasculeEnRegex(t *testing.T) {
	m := selecteurDeTest("node.js", "nodeXjs", "wget")

	for _, r := range "node.js" {
		m = frappe(m, r)
	}
	if len(m.filtered) != 1 {
		t.Fatalf("en texte brut, %d candidats — le point ne doit pas être un joker", len(m.filtered))
	}

	m = ctrlR(m)
	if len(m.filtered) != 2 {
		t.Fatalf("en regex, %d candidats — le point devrait valoir joker", len(m.filtered))
	}

	m = ctrlR(m)
	if len(m.filtered) != 1 {
		t.Fatalf("retour en texte brut : %d candidats", len(m.filtered))
	}
}

// Le mode ne se devine pas : il s'affiche. Et rien ne s'affiche en mode texte, pour que
// le sélecteur reste identique à ce qu'il a toujours été pour qui l'ignore.
func TestModeAfficheSeulementEnRegex(t *testing.T) {
	m := selecteurDeTest("git", "gitui")

	if got := visible(m.View()); strings.Contains(got, "regex]") {
		t.Error("le mode texte ne doit rien afficher sur la ligne de filtre")
	}

	m = ctrlR(m)
	if got := visible(m.View()); !strings.Contains(got, "regex]") {
		t.Errorf("le mode regex doit s'afficher :\n%s", got)
	}
}

// Un motif fautif laisse la liste entière et le dit — le vider laisserait croire
// qu'aucun paquet ne correspond.
func TestMotifInvalideSignaleEtNeVidePas(t *testing.T) {
	m := selecteurDeTest("c++", "gcc", "clang")
	m = ctrlR(m)
	for _, r := range "c++" {
		m = frappe(m, r)
	}
	if len(m.filtered) != 3 {
		t.Fatalf("%d candidats, attendu les 3 — un motif invalide ne filtre pas", len(m.filtered))
	}
	if got := visible(m.View()); !strings.Contains(got, "invalid") && !strings.Contains(got, "invalide") {
		t.Errorf("le motif invalide doit être signalé :\n%s", got)
	}
}

// La touche est annoncée dans le pied : sans ça, personne ne la découvrira.
func TestPiedAnnonceLaTouche(t *testing.T) {
	m := selecteurDeTest("git")
	if got := visible(m.View()); !strings.Contains(got, " ^R ") {
		t.Errorf("le pied doit annoncer ^R :\n%s", got)
	}
}

// Le sélecteur de désambiguïsation garde son pied à lui : on y choisit un gestionnaire
// parmi deux ou trois, où un mode regex n'apprendrait rien.
func TestPiedPersonnaliseInchange(t *testing.T) {
	m := selecteurDeTest("brew", "scoop")
	m.Keys = []Key{{"↵", "choisir"}, {"^G", "annuler"}}
	if got := visible(m.View()); strings.Contains(got, "^R") {
		t.Errorf("le pied personnalisé ne doit pas gagner ^R :\n%s", got)
	}
}
