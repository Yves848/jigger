package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
)

func items(n int) []complete.Item {
	out := make([]complete.Item, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, complete.Item{Name: fmt.Sprintf("p%03d", i), Badge: "F"})
	}
	return out
}

// Le popup vivant est dessiné dans un terminal dont la largeur n'est pas la nôtre :
// toutes les lignes doivent tenir dans la largeur demandée, sans quoi le terminal
// replie et le compte de lignes à effacer devient faux.
func TestFrameRespecteLaLargeurDemandee(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	for _, w := range []int{30, 40, 58, 80} {
		f := Frame{Title: "brew install", Items: items(3), Width: w, Rows: 3,
			Keys: []Key{{"⇥", "insérer"}}}
		for i, line := range strings.Split(f.Render(), "\n") {
			// +2 : les deux colonnes de bordure du cadre.
			if got := lipgloss.Width(line); got != w+2 {
				t.Errorf("largeur %d : ligne %d fait %d colonnes (attendu %d)", w, i, got, w+2)
			}
		}
	}
}

// Le widget zsh réduit le nombre de lignes quand le terminal est court : la frame doit
// alors compter exactement ce qui a été demandé.
func TestFrameRowsLimiteLesCandidats(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	f := Frame{Title: "brew install", Items: items(20), Rows: 4, Keys: []Key{{"⇥", "insérer"}}}
	shown := 0
	for _, it := range items(20) {
		if strings.Contains(visible(f.Render()), it.Name) {
			shown++
		}
	}
	if shown != 4 {
		t.Errorf("%d candidats affichés, attendu 4", shown)
	}
}

// `jigger render` est appelé avec un index venu de zsh : rien ne garantit qu'il soit
// dans les bornes (liste rétrécie par la frappe précédente). Ça ne doit pas paniquer,
// et aucune ligne ne doit être marquée courante.
func TestFrameSelHorsBornes(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	for _, sel := range []int{-1, 3, 99} {
		f := Frame{Title: "brew install", Items: items(3), Sel: sel, Keys: []Key{{"⇥", "insérer"}}}
		out := f.Render()
		if sel >= 0 && sel < 3 {
			continue
		}
		for _, line := range strings.Split(out, "\n") {
			for _, on := range underlinedColumns(line) {
				if on {
					t.Fatalf("sel=%d : une ligne est marquée courante alors que l'index est hors bornes", sel)
				}
			}
		}
	}
}

// Contexte paquet avec un mot vide : ~7000 candidats n'apprendraient rien, le widget
// n'envoie aucun item et fait afficher une invite à la place.
func TestFrameMessageQuandAucunItem(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := visible(Frame{Title: "brew install", Empty: "tapez pour filtrer…"}.Render())
	if !strings.Contains(out, "tapez pour filtrer…") {
		t.Errorf("message d'invite absent : %q", out)
	}
}

// ScrollOffset remplace la mémoire de défilement du sélecteur : le seul état conservé
// entre deux frappes est l'index, il faut donc recalculer la fenêtre à chaque rendu.
func TestScrollOffset(t *testing.T) {
	cases := []struct{ sel, count, rows, want int }{
		{0, 100, 10, 0},   // début de liste
		{9, 100, 10, 0},   // dernière ligne visible sans défiler
		{10, 100, 10, 1},  // premier défilement
		{99, 100, 10, 90}, // fin de liste : la fenêtre s'arrête au dernier écran
		{5, 3, 10, 0},     // moins de candidats que de lignes
		{0, 0, 10, 0},     // liste vide
	}
	for _, c := range cases {
		if got := ScrollOffset(c.sel, c.count, c.rows); got != c.want {
			t.Errorf("ScrollOffset(%d, %d, %d) = %d, attendu %d", c.sel, c.count, c.rows, got, c.want)
		}
	}
}
