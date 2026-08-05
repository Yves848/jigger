package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
)

var truecolorFg = regexp.MustCompile(`38;2;\d+;\d+;\d+`)

// Vérifie que le rendu émet bien les icônes distinctes et de la couleur (TrueColor),
// pour éviter une régression vers un affichage « fade ». (lipgloss arrondissant les
// teintes, on compte le nombre de couleurs plutôt que des RGB exacts.)
func TestRenderHasIconsAndColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	res := complete.Result{
		Executable: true,
		Items: []complete.Item{
			{Name: "git", Badge: "F", Installed: true, Version: "2.55.0"},
			{Name: "firefox", Badge: "C"},
		},
	}
	out := New("brew install", res).View()

	if !strings.Contains(out, "◆") {
		t.Error("icône formula (◆) absente")
	}
	if !strings.Contains(out, "▣") {
		t.Error("icône cask (▣) absente")
	}
	if !strings.Contains(out, "2.55.0") {
		t.Error("version installée (2.55.0) absente du rendu")
	}

	distinct := map[string]bool{}
	for _, m := range truecolorFg.FindAllString(out, -1) {
		distinct[m] = true
	}
	if len(distinct) < 5 {
		t.Errorf("rendu peu coloré : %d teintes distinctes (attendu ≥ 5)", len(distinct))
	}
	if !strings.Contains(out, "48;2;") {
		t.Error("fond de sélection (couleur d'arrière-plan) absent")
	}
}

// Régression : filtrer progressivement une longue liste ne doit jamais dupliquer de
// candidats. (Bug d'aliasing : `m.filtered = m.all` puis `m.filtered[:0]` réécrivait
// le début du tableau sous-jacent de `m.all`, laissant les originaux à leur place et
// donc en double — d'où les « powershell / powershell@preview » empilés.)
func TestProgressiveFilterNoDuplicates(t *testing.T) {
	// Liste triée où les candidats « pow… » sont situés loin dans le tableau.
	items := []complete.Item{}
	for _, n := range []string{"aa", "ab", "ac", "ad", "ae", "af", "ag", "ah"} {
		items = append(items, complete.Item{Name: n, Badge: "F"})
	}
	items = append(items,
		complete.Item{Name: "powershell", Badge: "F", Installed: true},
		complete.Item{Name: "powershell@preview", Badge: "C"},
	)

	m := New("brew search", complete.Result{Items: items})

	for _, q := range []string{"p", "po", "pow", "powe", "power", "powers"} {
		m.input.SetValue(q)
		m.applyFilter()

		seen := map[string]int{}
		for _, it := range m.filtered {
			seen[it.Name]++
		}
		for name, n := range seen {
			if n > 1 {
				t.Fatalf("filtre %q : candidat %q présent %d fois (attendu 1)", q, name, n)
			}
		}
	}

	if len(m.filtered) != 2 {
		t.Fatalf("filtre final « powers » : attendu 2 candidats, obtenu %d (%v)", len(m.filtered), names(m.filtered))
	}
}

func names(items []complete.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}
