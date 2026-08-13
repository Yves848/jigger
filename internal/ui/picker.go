// Package ui fournit le sélecteur interactif (Bubble Tea + Lip Gloss) aux couleurs
// de Cocktails : accent teal, icônes distinctes formula/cask, indicateur « installé »,
// rappels de touches ⇥ (insérer) / ↩ (exécuter).
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
)

// Palette (vive, dérivée de Cocktails).
var (
	accent  = lipgloss.Color("#2DD4BF") // teal
	accentD = lipgloss.Color("#0E312C") // teal profond (fond sélection)
	ink     = lipgloss.Color("#EAFBF7") // texte clair (sélection)
	fg      = lipgloss.Color("#CBD2DE") // texte normal
	muted   = lipgloss.Color("#7C8598")
	amber   = lipgloss.Color("#F5B841") // formula ◆
	violet  = lipgloss.Color("#B79BFF") // cask ▣
	green   = lipgloss.Color("#4ADE80") // installé
	panelBg = lipgloss.Color("#0C131F")
	sepCl   = lipgloss.Color("#26374C")
)

// Largeur intérieure fixe : chaque ligne est complétée à cette largeur.
const (
	boxW        = 58
	rowW        = boxW - 4 // largeur intérieure d'une ligne (marge + bordure de chaque côté)
	nameMax     = boxW - 12
	visibleRows = 6
)

var (
	// Le cadre porte le fond ET la largeur : lipgloss remplit uniformément les zones
	// nues (espaces sans fond) avec panelBg, y compris après les resets internes.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			BorderBackground(panelBg).
			Background(panelBg).
			Width(boxW)

	// Ligne sélectionnée : une boîte arrondie autonome, détachée du cadre du popup.
	// La marge gauche est peinte au fond du panneau, sinon elle apparaîtrait en noir.
	selBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			BorderBackground(panelBg).
			Background(accentD).
			Foreground(ink).
			Bold(true).
			MarginLeft(1).
			MarginBackground(panelBg).
			Width(rowW)

	// Rappels de touches : pastilles (fond + remplissage). Une vraie bordure coûterait
	// deux lignes de plus au pied du popup pour le même effet.
	pillStyle = lipgloss.NewStyle().Background(sepCl).Foreground(ink).Bold(true).Padding(0, 1)

	promptStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	titleStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	filterHint  = lipgloss.NewStyle().Foreground(muted)

	formulaStyle = lipgloss.NewStyle().Foreground(amber).Bold(true)
	caskStyle    = lipgloss.NewStyle().Foreground(violet).Bold(true)
	bulletStyle  = lipgloss.NewStyle().Foreground(muted)
	nameStyle    = lipgloss.NewStyle().Foreground(fg)
	verStylePkg  = lipgloss.NewStyle().Foreground(muted) // version installée (atténuée)
	dotStyle     = lipgloss.NewStyle().Foreground(green)

	sepStyle   = lipgloss.NewStyle().Foreground(sepCl)
	keyStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(muted)
	emptyStyle = lipgloss.NewStyle().Foreground(muted).Italic(true)
	verStyle   = lipgloss.NewStyle().Foreground(muted)
)

// Version est affichée (discrètement) dans l'en-tête du sélecteur pour lever toute
// ambiguïté sur le binaire réellement lancé. Renseignée par main au démarrage.
var Version = ""

// Model est le sélecteur.
type Model struct {
	title      string
	all        []complete.Item
	filtered   []complete.Item
	input      textinput.Model
	cursor     int
	offset     int
	executable bool
	quitting   bool // sortie en cours → View() vide pour que Bubble Tea efface le cadre

	Chosen  *complete.Item // sélection (nil si annulé)
	Execute bool           // ↩ : la commande est à exécuter
}

// New crée le sélecteur pour un résultat de complétion.
func New(title string, res complete.Result) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "filtrer…"
	ti.Focus()

	m := Model{title: title, all: res.Items, executable: res.Executable, input: ti}
	m.applyFilter()
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	// Toujours reconstruire dans un nouveau tableau : ne JAMAIS réutiliser le backing
	// de m.all (un `m.filtered = m.all` suivi d'un `m.filtered[:0]` réécrirait m.all et
	// dupliquerait les candidats au fil des frappes).
	filtered := make([]complete.Item, 0, len(m.all))
	if q == "" {
		filtered = append(filtered, m.all...)
	} else {
		for _, it := range m.all {
			if strings.Contains(strings.ToLower(it.Name), q) {
				filtered = append(filtered, it)
			}
		}
	}
	m.filtered = filtered
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.clampOffset()
}

func (m *Model) clampOffset() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			m.Chosen = nil
			m.quitting = true
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				m.clampOffset()
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.clampOffset()
			}
			return m, nil
		case "tab":
			m.choose(false)
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.choose(m.executable) // ↩ exécute si la commande est complète
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m *Model) choose(execute bool) {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		it := m.filtered[m.cursor]
		m.Chosen = &it
		m.Execute = execute
	}
}

func (m Model) View() string {
	// À la sortie, on rend une vue vide : Bubble Tea remonte (en mouvements relatifs,
	// donc robuste au défilement) et efface tout le cadre du popup. Évite tout résidu
	// à l'écran quand on annule avec Esc (aucun redraw du buffer côté zsh ne le masque).
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Lignes « normales » : couleur de texte seulement ; le cadre remplit le fond.
	// Version poussée à droite (repère du binaire lancé), sur la même ligne que le titre.
	head := promptStyle.Render("❯") + " " + titleStyle.Render(m.title)
	if Version != "" {
		ver := verStyle.Render("jigger " + Version)
		gap := boxW - lipgloss.Width(head) - lipgloss.Width(ver)
		if gap < 1 {
			gap = 1
		}
		head += strings.Repeat(" ", gap) + ver
	}
	b.WriteString(head + "\n")
	b.WriteString(filterHint.Render("› ") + m.input.View() + "\n")

	if len(m.filtered) == 0 {
		b.WriteString("  " + emptyStyle.Render("aucun candidat"))
		return boxStyle.Render(b.String())
	}

	end := min(m.offset+visibleRows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(m.filtered[i], i == m.cursor) + "\n")
		if i != m.cursor {
			b.WriteString("\n") // interligne ; la sélection le remplace par ses bordures
		}
	}
	b.WriteString(m.footer())
	return boxStyle.Render(b.String())
}

func (m Model) renderRow(it complete.Item, selected bool) string {
	name := it.Name
	if len(name) > nameMax {
		name = name[:nameMax-1] + "…"
	}

	glyph := "•"
	switch it.Badge {
	case "F":
		glyph = "◆"
	case "C":
		glyph = "▣"
	}

	// Partie droite (alignée au bord) : version installée puis point « installé ».
	// On la compose d'abord en texte nu pour calculer le remplissage.
	rightPlain := ""
	if it.Version != "" {
		rightPlain = it.Version
	}
	if it.Installed {
		if rightPlain != "" {
			rightPlain += "  "
		}
		rightPlain += "●"
	}

	// Ligne sélectionnée : une boîte arrondie à elle seule. Elle occupe 3 lignes
	// (bordure, contenu, bordure) là où une ligne ordinaire en occupe 2 (contenu +
	// interligne) — comme il y a toujours exactement une sélection, la hauteur totale
	// du popup ne bouge pas quand le curseur se déplace.
	if selected {
		leftPlain := glyph + "  " + name
		gap := rowW - lipgloss.Width(leftPlain) - lipgloss.Width(rightPlain)
		if gap < 1 {
			gap = 1
		}
		return selBoxStyle.Render(leftPlain + strings.Repeat(" ", gap) + rightPlain)
	}

	color := bulletStyle
	switch it.Badge {
	case "F":
		color = formulaStyle
	case "C":
		color = caskStyle
	}
	// Ligne ordinaire : alignée sur le contenu de la boîte de sélection (2 colonnes
	// d'indentation = marge + bordure), et suivie d'un interligne ajouté par View.
	leftPlain := "  " + glyph + "  " + name
	// On réserve à droite la même gouttière que la boîte de sélection (sa bordure et
	// sa marge), pour que versions et points « installé » s'alignent d'une ligne à l'autre.
	gap := boxW - 2 - lipgloss.Width(leftPlain) - lipgloss.Width(rightPlain)
	if gap < 1 {
		gap = 1
	}
	left := "  " + color.Render(glyph) + "  " + nameStyle.Render(name)
	right := ""
	if it.Version != "" {
		right = verStylePkg.Render(it.Version)
	}
	if it.Installed {
		if it.Version != "" {
			right += "  "
		}
		right += dotStyle.Render("●")
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) footer() string {
	sep := hintStyle.Render("  ")
	pill := func(key, label string) string {
		return pillStyle.Render(key) + hintStyle.Render(" "+label)
	}
	parts := []string{
		pill("⇥", "insérer"),
	}
	if m.executable {
		parts = append(parts, pill("↩", "exécuter"))
	}
	parts = append(parts, pill("↑↓", "naviguer"), pill("esc", "annuler"))
	return "  " + strings.Join(parts, sep)
}
