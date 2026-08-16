package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
)

// ligneTableau est une ligne de tableau : des cellules déjà calculées. La première
// cellule est le nom du paquet — c'est elle qui sert de clé au filtre et à la sélection.
//
// La vue ne connaît ni pm.Package ni la façade : elle reçoit des colonnes toutes faites
// (facade.Colonnes), ce qui lui évite d'avoir un avis sur ce qu'est un paquet.
type ligneTableau struct{ cellules []string }

func (l ligneTableau) Cle() string {
	if len(l.cellules) == 0 {
		return ""
	}
	return l.cellules[0]
}

func (l ligneTableau) Cellules() []string { return l.cellules }

// Hauteur et largeur par défaut, tant que le terminal n'a pas annoncé les siennes.
const (
	tableauLignesDefaut  = 20
	tableauLargeurDefaut = 100
	// Chrome du cadre : titre, en-tête de colonnes, séparateur, ligne de filtre, pied.
	tableauChrome = 7
)

// Tableau est la vue paginée des verbes tabulaires. Elle lit, elle filtre, elle coche —
// elle n'exécute rien (spec, non-buts).
type Tableau struct {
	titre    string
	entete   []string
	liste    *Liste
	input    textinput.Model
	largeurs []int // largeur de chaque colonne, calculée sur TOUTES les lignes
	largeur  int
	quitting bool
	annule   bool

	// Choisis porte le résultat à la sortie : les lignes cochées, ou la ligne courante
	// si aucune ne l'est. Vide si l'utilisateur a annulé.
	Choisis []string
}

// NouveauTableau crée la vue pour un en-tête et des cellules déjà calculés.
func NouveauTableau(titre string, entete []string, cellules [][]string) Tableau {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = i18n.T("popup.filter")
	ti.TextStyle = base.Foreground(ink)
	ti.PlaceholderStyle = base.Foreground(muted)
	ti.Focus()

	lignes := make([]Ligne, len(cellules))
	for i, c := range cellules {
		lignes[i] = ligneTableau{c}
	}

	t := Tableau{
		titre:   titre,
		entete:  entete,
		liste:   NouvelleListe(lignes, tableauLignesDefaut),
		input:   ti,
		largeur: tableauLargeurDefaut,
	}
	t.mesurer(cellules)
	return t
}

// mesurer fixe la largeur de chaque colonne sur l'ensemble des lignes, et non sur les
// seules lignes visibles : sans cela, les colonnes sauteraient à chaque défilement.
func (t *Tableau) mesurer(cellules [][]string) {
	t.largeurs = make([]int, len(t.entete))
	for i, e := range t.entete {
		t.largeurs[i] = len([]rune(e))
	}
	for _, ligne := range cellules {
		for i, c := range ligne {
			if i < len(t.largeurs) {
				if n := len([]rune(c)); n > t.largeurs[i] {
					t.largeurs[i] = n
				}
			}
		}
	}
}

func (t Tableau) Init() tea.Cmd { return textinput.Blink }

func (t Tableau) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.largeur = msg.Width
		t.liste.DefinirHauteur(msg.Height - tableauChrome)
		return t, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "ctrl+g":
			t.annule = true
			t.Choisis = nil
			t.quitting = true
			return t, tea.Quit

		case "enter":
			for _, l := range t.liste.Choix() {
				t.Choisis = append(t.Choisis, l.Cle())
			}
			t.quitting = true
			return t, tea.Quit

		// ⇥ coche : le champ de filtre a le focus en permanence, donc Espace est une
		// lettre et ne peut pas servir de raccourci (spec §3).
		case "tab":
			t.liste.Cocher()
			return t, nil

		case "ctrl+r":
			t.liste.BasculerMode()
			return t, nil

		case "up", "ctrl+p":
			t.liste.Haut()
			return t, nil
		case "down", "ctrl+n":
			t.liste.Bas()
			return t, nil
		case "pgup", "ctrl+b":
			t.liste.PageHaut()
			return t, nil
		case "pgdown", "ctrl+f":
			t.liste.PageBas()
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	t.liste.Filtrer(t.input.Value())
	return t, cmd
}

// Annule dit si la vue a été quittée sans rien retenir.
func (t Tableau) Annule() bool { return t.annule }

func (t Tableau) View() string {
	if t.quitting {
		return ""
	}

	var b strings.Builder

	// Titre, et compteur de lignes retenues.
	tete := titleStyle.Render(t.titre)
	if n := t.liste.NbCochees(); n > 0 {
		tete += filterHint.Render("   " + fmt.Sprintf(i18n.T("table.selected"), n))
	}
	b.WriteString(tete + "\n")

	// Ligne de filtre, avec le mode courant — on ne devine jamais dans lequel on est.
	mode := i18n.T("table.modetexte")
	if t.liste.Mode() == FiltreRegex {
		mode = i18n.T("table.moderegex")
	}
	filtre := filterHint.Render("› ") + t.input.View() + filterHint.Render("  ["+mode+"]")
	if !t.liste.Valide() {
		filtre += base.Foreground(amber).Render("  " + i18n.T("table.badregex"))
	}
	b.WriteString(filtre + "\n\n")

	// En-tête de colonnes, puis les lignes visibles.
	b.WriteString(base.Foreground(muted).Bold(true).Render(t.ligne(t.entete, false)) + "\n")

	visibles, sel := t.liste.Visibles()
	if len(visibles) == 0 {
		b.WriteString(emptyStyle.Render(i18n.T("popup.empty")) + "\n")
	}
	debut := t.liste.Offset()
	for i, l := range visibles {
		marque := "  "
		if t.liste.EstCochee(debut + i) {
			marque = dotStyle.Render("● ")
		}
		texte := t.ligne(l.Cellules(), true)
		if i == sel {
			b.WriteString(marque + selStyle.Render(texte) + "\n")
		} else {
			b.WriteString(marque + nameStyle.Render(texte) + "\n")
		}
	}

	b.WriteString("\n" + t.pied())
	return b.String()
}

// ligne pose les cellules à largeur fixe. `tronque` distingue les lignes de données de
// l'en-tête, qui ne doit jamais perdre de caractères.
func (t Tableau) ligne(cellules []string, tronque bool) string {
	var b strings.Builder
	for i, c := range cellules {
		if i >= len(t.largeurs) {
			break
		}
		if i == len(cellules)-1 {
			b.WriteString(c)
		} else {
			fmt.Fprintf(&b, "%-*s  ", t.largeurs[i], c)
		}
	}
	s := b.String()
	if tronque && t.largeur > 4 && len([]rune(s)) > t.largeur-4 {
		s = string([]rune(s)[:t.largeur-5]) + "…"
	}
	return s
}

func (t Tableau) pied() string {
	keys := []Key{
		{"⇥", i18n.T("table.toggle")},
		{"↵", i18n.T("table.confirm")},
		{"↑↓", i18n.T("popup.navigate")},
		{"^R", i18n.T("table.regex")},
		{"^G", i18n.T("popup.cancel")},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, pillStyle.Render(k.Key)+hintStyle.Render(" "+k.Label))
	}
	return lipgloss.NewStyle().Background(panelBg).Render(strings.Join(parts, hintStyle.Render("   ")))
}
