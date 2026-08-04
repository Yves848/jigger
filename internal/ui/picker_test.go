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
			{Name: "git", Badge: "F", Installed: true},
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
