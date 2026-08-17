package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// backgroundColumns dit, colonne par colonne, si un fond (SGR 48) est actif — le pendant
// de underlinedColumns pour traquer les bandes laissées par un reset au milieu de ligne.
func backgroundColumns(line string) []bool {
	var cols []bool
	on := false
	for len(line) > 0 {
		if loc := ansi.FindStringIndex(line); loc != nil && loc[0] == 0 {
			params := strings.Split(line[2:loc[1]-1], ";")
			for i := 0; i < len(params); i++ {
				switch params[i] {
				case "48":
					on = true
					i = len(params) // le reste de la séquence décrit la couleur
				case "49", "0", "":
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

// Régression visuelle : un segment stylé émet un reset qui coupe le fond posé par le
// cadre ; tout ce qui suit en texte nu s'affichait alors sur le fond du terminal, d'où
// une bande plus claire au milieu des lignes non sélectionnées. Chaque colonne d'une
// ligne de candidat doit porter un fond.
func TestRowsHaveNoBackgroundGap(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := New("brew install", complete.Result{Items: []complete.Item{
		{Name: "git", Badge: "F", Installed: true, Version: "2.55.0"},
		{Name: "git-absorb", Badge: "F"},
		{Name: "firefox", Badge: "C", Installed: true},
	}}).View()

	for _, name := range []string{"git-absorb", "firefox"} {
		var row string
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(visible(line), name) {
				row = line
				break
			}
		}
		if row == "" {
			t.Fatalf("ligne %q introuvable", name)
		}
		for col, hasBg := range backgroundColumns(row) {
			if !hasBg {
				t.Errorf("ligne %q : colonne %d sans fond (bande visible) — %q", name, col, visible(row))
				break
			}
		}
	}
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

// Le sélecteur de désambiguïsation de la façade (main.trancher) réutilise ce Model avec
// un autre titre et d'autres touches de pied — « ↵ choisir / ^G annuler », comme le
// documentent la spec §3 et le README, à la place de « ⇥ insérer / ↩ exécuter / ↑↓
// naviguer / esc annuler » du sélecteur ordinaire. Sans un moyen de les fournir, le
// sélecteur de désambiguïsation ne pouvait afficher que le pied du sélecteur ordinaire —
// trompeur, puisque ⇥/↩ n'y ont pas de sens particulier et qu'esc, pas ^G, l'annulait.
func TestFooterEstPersonnalisable(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := New("git : 2 gestionnaires", complete.Result{
		Executable: true,
		Items:      []complete.Item{{Name: "winget", Badge: "W"}},
	})
	m.Keys = []Key{{"↵", "choisir"}, {"^G", "annuler"}}

	shown := visible(m.View())
	if !strings.Contains(shown, "↵") || !strings.Contains(shown, "choisir") {
		t.Errorf("touche personnalisée ↵ choisir absente du rendu :\n%s", shown)
	}
	if !strings.Contains(shown, "^G") || !strings.Contains(shown, "annuler") {
		t.Errorf("touche personnalisée ^G annuler absente du rendu :\n%s", shown)
	}
	if strings.Contains(shown, "insérer") || strings.Contains(shown, "exécuter") {
		t.Errorf("le pied par défaut ne doit plus apparaître une fois Keys fourni :\n%s", shown)
	}
}

// Sans Keys, le sélecteur ordinaire (jigger pick) rend exactement comme avant : le pied
// par défaut ne doit pas bouger d'un caractère (v0.7.0).
func TestFooterParDefautInchangeSansKeys(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := New("brew install", complete.Result{
		Executable: true,
		Items:      []complete.Item{{Name: "wget", Badge: "F"}},
	}).View()

	shown := visible(out)
	for _, key := range []string{" ⇥ ", " ↩ ", " esc "} {
		if !strings.Contains(shown, key) {
			t.Errorf("touche %q absente du pied par défaut :\n%s", key, shown)
		}
	}
}

// ^G annule le sélecteur au même titre qu'esc et ctrl+c : c'est la touche que la
// désambiguïsation façade documente (spec §3, README), il faut donc qu'elle fasse
// réellement quelque chose, pas seulement qu'elle s'affiche au pied.
func TestCtrlGAnnule(t *testing.T) {
	m := New("git : 2 gestionnaires", complete.Result{
		Executable: true,
		Items:      []complete.Item{{Name: "winget", Badge: "W"}},
	})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	mm := updated.(Model)
	if mm.Chosen != nil {
		t.Error("^G doit annuler : Chosen doit rester nil")
	}
	if cmd == nil {
		t.Error("^G doit quitter le programme (tea.Quit)")
	}
}

// Le sélecteur plafonne à visibleRows candidats : au-delà, on défile. (Nommer des
// paquets « p01 »… évite qu'un nom soit sous-chaîne d'un autre — « p1 » dans « p10 » —
// ce qui gonflerait le compte.)
func TestShowsAtMostVisibleRowsCandidates(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	items := []complete.Item{}
	for i := 1; i <= visibleRows+4; i++ {
		items = append(items, complete.Item{Name: fmt.Sprintf("p%02d", i), Badge: "F"})
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
	if shown != visibleRows {
		t.Errorf("%d candidats affichés, attendu %d", shown, visibleRows)
	}
}

// SansFiltre : une question par oui ou non n'a rien à filtrer. La ligne de saisie
// disparaît, et — c'est l'autre moitié, celle qui compte — la frappe ne filtre plus rien.
// La masquer en laissant la saisie active ferait disparaître des choix sans que rien ne
// dise pourquoi : le pire des deux mondes.
func TestSansFiltreRetireLaLigneDeSaisie(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	res := complete.Result{Items: []complete.Item{{Name: "annuler"}, {Name: "relancer"}}}

	avec := visible(New("Relancer ?", res).View())
	if !strings.Contains(avec, "›") {
		t.Fatal("le sélecteur ordinaire doit garder sa ligne de filtre")
	}

	m := New("Relancer ?", res)
	m.SansFiltre = true
	if shown := visible(m.View()); strings.Contains(shown, "›") {
		t.Errorf("ligne de filtre présente malgré SansFiltre :\n%s", shown)
	}
}

func TestSansFiltreIgnoreLaFrappe(t *testing.T) {
	res := complete.Result{Items: []complete.Item{{Name: "annuler"}, {Name: "relancer"}}}

	m := New("Relancer ?", res)
	m.SansFiltre = true
	suite, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	after := suite.(Model)

	if len(after.filtered) != 2 {
		t.Fatalf("%d candidats après une frappe, attendu 2 : la saisie ne doit plus filtrer",
			len(after.filtered))
	}
}

// Les touches qui comptent restent prises : sans elles, la question serait sans réponse.
func TestSansFiltreGardeLaNavigationEtLeChoix(t *testing.T) {
	res := complete.Result{Items: []complete.Item{{Name: "annuler"}, {Name: "relancer"}}}

	m := New("Relancer ?", res)
	m.SansFiltre = true
	bas, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	choisi, _ := bas.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})

	final := choisi.(Model)
	if final.Chosen == nil || final.Chosen.Name != "relancer" {
		t.Fatalf("choix = %v, attendu « relancer »", final.Chosen)
	}
}
