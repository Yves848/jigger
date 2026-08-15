package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
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
		for _, focus := range []bool{false, true} {
			f := Frame{Title: "brew install", Items: items(3), Width: w, Rows: 3,
				Focused: focus, Keys: []Key{{"⇥", "insérer"}}}
			for i, line := range strings.Split(f.Render(), "\n") {
				// +2 : les deux colonnes de bordure du cadre.
				if got := lipgloss.Width(line); got != w+2 {
					t.Errorf("largeur %d (focus=%v) : ligne %d fait %d colonnes (attendu %d)",
						w, focus, i, got, w+2)
				}
			}
		}
	}
}

// Une ligne qui déborde est le pire défaut possible : le terminal la replie, le popup
// occupe alors deux fois les lignes annoncées, et tout l'affichage du shell se décale.
// Le cas réel qui l'a révélé : les identifiants des applications détectées hors
// catalogue winget, longs, suivis d'une version à quatre nombres et du point d'installé.
func TestFrameNeDeborderJamais(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	longs := []complete.Item{
		{Name: `ARP\Machine\X64\{226CEF88-E174-49A6-865C-874AB2C5F6D1}`,
			Badge: pm.BadgeOther, Installed: true, Version: "6.4.0.3079"},
		{Name: "AgileBits.1Password", Badge: pm.BadgeWinget, Installed: true, Version: "8.12.32.33"},
		{Name: "court", Badge: pm.BadgeWinget},
	}
	touches := []Key{{"⇥", "insérer"}, {"↑↓", "naviguer"}, {"^G", "fermer"}}

	for _, w := range []int{24, 30, 40, 58, 80} {
		for _, sel := range []int{-1, 0, 1} {
			f := Frame{Title: "winget uninstall", Items: longs, Width: w, Rows: 3,
				Sel: sel, Focused: sel == 1, Keys: touches}
			for i, line := range strings.Split(f.Render(), "\n") {
				if got := lipgloss.Width(line); got != w+2 {
					t.Errorf("largeur %d (sel=%d) : ligne %d fait %d colonnes (attendu %d)",
						w, sel, i, got, w+2)
				}
			}
		}
	}
}

// Le focus dit où ira la prochaine flèche — au popup ou à l'historique du shell. Il n'y
// a aucun autre moyen de le savoir que de le voir : la ligne courante est soulignée
// quand le popup a le clavier, au repos quand il ne l'a pas.
func TestFrameFocusChangeLaLigneCourante(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	souligne := func(f Frame) bool {
		for _, line := range strings.Split(f.Render(), "\n") {
			for _, on := range underlinedColumns(line) {
				if on {
					return true
				}
			}
		}
		return false
	}

	repos := Frame{Title: "winget install", Items: items(3), Sel: 1, Keys: []Key{{"⇥", "insérer"}}}
	actif := repos
	actif.Focused = true

	if !souligne(actif) {
		t.Error("avec le focus, la ligne courante doit être soulignée")
	}
	if souligne(repos) {
		t.Error("sans le focus, aucune ligne ne doit être soulignée")
	}

	// Elle reste tout de même désignée : c'est elle que ⇥ insère.
	aucune := repos
	aucune.Sel = -1
	if repos.Render() == aucune.Render() {
		t.Error("sans le focus, la ligne courante doit rester distinguée des autres")
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

// La colonne PM apparaît selon les données, pas selon un drapeau — même règle que les
// tableaux de sortie.
func TestFrameColonnePMSelonLesDonnees(t *testing.T) {
	avec := Frame{
		Title: "jigger install",
		Items: []pm.Item{
			{Name: "Git.Git", Badge: pm.BadgeWinget, PM: "winget"},
			{Name: "git", Badge: pm.BadgeScoop, PM: "scoop"},
		},
		Rows: 8,
	}.Render()
	if !strings.Contains(avec, "winget") || !strings.Contains(avec, "scoop") {
		t.Errorf("les deux gestionnaires doivent apparaître dans le cadre :\n%s", avec)
	}

	sans := Frame{
		Title: "brew install",
		Items: []pm.Item{{Name: "git", Badge: pm.BadgeFormula}},
		Rows:  8,
	}.Render()
	if strings.Contains(sans, "brew") && !strings.Contains(sans, "brew install") {
		t.Errorf("aucun PM sur les items : rien ne doit s'ajouter :\n%s", sans)
	}
}

// Un seul gestionnaire disponible (macOS, brew seul) : CompleteFacade marque tout de même
// Item.PM sur chaque candidat (cf. internal/complete.CompleteFacade), puisque c'est ce
// champ qui permet à InsertItem de retrouver la correction du bon gestionnaire — mais un
// seul gestionnaire a contribué, la colonne serait du bruit. avecPM doit suivre la même
// règle que facade.plusieursPM (§4) : au moins deux gestionnaires DISTINCTS, pas « au
// moins un item porte un PM ». Avant la correction, `jg render --line "jg install fire"`
// sur cette machine posait une colonne « brew » sur chaque ligne.
func TestFrameColonnePMPasAvecUnSeulGestionnaireQuiContribue(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	out := visible(Frame{
		Title: "jigger install",
		Items: []pm.Item{
			{Name: "fd", Badge: pm.BadgeFormula, PM: "brew"},
			{Name: "fzf", Badge: pm.BadgeFormula, PM: "brew"},
		},
		Rows: 8,
	}.Render())
	if strings.Contains(out, "brew") {
		t.Errorf("un seul gestionnaire a contribué (brew partout) : aucune colonne PM attendue :\n%s", out)
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
