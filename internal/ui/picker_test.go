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

// lipgloss découpe le remplissage d'une pastille en séquences ANSI distinctes : pour
// juger de ce que l'utilisateur voit, il faut retirer les échappements.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visible(s string) string { return ansi.ReplaceAllString(s, "") }

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
	// Sur la ligne courante, lipgloss emballe chaque caractère dans sa propre séquence
	// (conséquence du soulignement des espaces) : on cherche donc dans le texte visible.
	shown := visible(out)

	if !strings.Contains(shown, "◆") {
		t.Error("icône formula (◆) absente")
	}
	if !strings.Contains(shown, "▣") {
		t.Error("icône cask (▣) absente")
	}
	if !strings.Contains(shown, "2.55.0") {
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
		t.Error("fond du panneau (couleur d'arrière-plan) absent")
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

// underlinedColumns rejoue les séquences SGR d'une ligne pour dire, colonne par colonne,
// si l'attribut « souligné » est actif. C'est le seul moyen honnête de vérifier que la
// règle court bien sous le remplissage et pas seulement sous le texte.
func underlinedColumns(line string) []bool {
	var cols []bool
	on := false
	for len(line) > 0 {
		if loc := ansi.FindStringIndex(line); loc != nil && loc[0] == 0 {
			for _, p := range strings.Split(line[2:loc[1]-1], ";") {
				switch p {
				case "4":
					on = true
				case "24", "0", "":
					on = false
				}
			}
			line = line[loc[1]:]
			continue
		}
		r := []rune(line)[0]
		for w := lipgloss.Width(string(r)); w > 0; w-- {
			cols = append(cols, on)
		}
		line = line[len(string(r)):]
	}
	return cols
}

// La ligne courante ne porte plus de boîte : seul le cadre du popup dessine des coins,
// et la sélection se signale par un soulignement qui court sur toute la largeur.
func TestSelectedRowIsUnderlinedNotBoxed(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := New("brew install", complete.Result{Items: []complete.Item{
		{Name: "wget", Badge: "F", Installed: true, Version: "1.25.0"},
		{Name: "wgetpaste", Badge: "F"},
	}}).View()

	if got := strings.Count(out, "╭"); got != 1 {
		t.Errorf("coin haut-gauche vu %d fois (attendu 1 : le cadre du popup, sans boîte de sélection)", got)
	}

	var row string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(visible(line), "wget ") || strings.HasSuffix(strings.TrimRight(visible(line), " │"), "wget") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("ligne sélectionnée (wget) introuvable dans le rendu")
	}

	cols := underlinedColumns(row)
	first, last := -1, -1
	for i, u := range cols {
		if u {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatalf("aucun soulignement sur la ligne courante : %q", visible(row))
	}
	if span := last - first + 1; span != rowW {
		t.Errorf("soulignement large de %d colonnes, attendu %d (il doit courir sous le remplissage)", span, rowW)
	}
}

// La hauteur ne doit pas changer selon la ligne sélectionnée : sinon le popup
// « saute » sous le prompt à chaque déplacement du curseur.
func TestHeightIsStableAcrossSelection(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	items := []complete.Item{}
	for _, n := range []string{"aa", "ab", "ac", "ad"} {
		items = append(items, complete.Item{Name: n, Badge: "F"})
	}
	m := New("brew install", complete.Result{Items: items})

	first := lipgloss.Height(m.View())
	for cur := 1; cur < len(items); cur++ {
		m.cursor = cur
		if h := lipgloss.Height(m.View()); h != first {
			t.Fatalf("curseur %d : hauteur %d, attendu %d (le popup saute)", cur, h, first)
		}
	}
}

// Les rappels de touches sont des pastilles : la touche est entourée d'espaces de
// remplissage, ce qui la détache visuellement de son libellé.
func TestFooterKeysArePills(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := New("brew install", complete.Result{
		Executable: true,
		Items:      []complete.Item{{Name: "wget", Badge: "F"}},
	}).View()

	shown := visible(out)
	for _, key := range []string{" ⇥ ", " ↩ ", " esc "} {
		if !strings.Contains(shown, key) {
			t.Errorf("touche %q non rendue en pastille (espaces de remplissage absents)", key)
		}
	}
}

// Le sélecteur aéré ne montre que quelques candidats : au-delà, on défile.
func TestShowsAtMostSixCandidates(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	items := []complete.Item{}
	for _, n := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		items = append(items, complete.Item{Name: n, Badge: "F"})
	}
	// Texte visible : la ligne courante est fragmentée caractère par caractère par le
	// soulignement, son nom ne serait pas trouvé dans le rendu brut.
	out := visible(New("brew install", complete.Result{Items: items}).View())

	shown := 0
	for _, it := range items {
		if strings.Contains(out, it.Name) {
			shown++
		}
	}
	if shown != 6 {
		t.Errorf("%d candidats affichés, attendu 6", shown)
	}
}
