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
	boxW        = 52
	nameMax     = boxW - 8
	visibleRows = 12
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

	// Ligne sélectionnée : un seul style qui remplit toute la largeur en teal.
	selRowStyle = lipgloss.NewStyle().Background(accentD).Foreground(ink).Bold(true).Width(boxW)

	promptStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	titleStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	filterHint  = lipgloss.NewStyle().Foreground(muted)

	formulaStyle = lipgloss.NewStyle().Foreground(amber).Bold(true)
	caskStyle    = lipgloss.NewStyle().Foreground(violet).Bold(true)
	bulletStyle  = lipgloss.NewStyle().Foreground(muted)
	nameStyle    = lipgloss.NewStyle().Foreground(fg)
	dotStyle     = lipgloss.NewStyle().Foreground(green)

	sepStyle   = lipgloss.NewStyle().Foreground(sepCl)
	keyStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(muted)
	emptyStyle = lipgloss.NewStyle().Foreground(muted).Italic(true)
)

// Model est le sélecteur.
type Model struct {
	title      string
	all        []complete.Item
	filtered   []complete.Item
	input      textinput.Model
	cursor     int
	offset     int
	executable bool

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
			return m, tea.Quit
		case "enter":
			m.choose(m.executable) // ↩ exécute si la commande est complète
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
	var b strings.Builder

	// Lignes « normales » : couleur de texte seulement ; le cadre remplit le fond.
	b.WriteString(promptStyle.Render("❯") + " " + titleStyle.Render(m.title) + "\n")
	b.WriteString(filterHint.Render("› ") + m.input.View() + "\n")

	if len(m.filtered) == 0 {
		b.WriteString("  " + emptyStyle.Render("aucun candidat"))
		return boxStyle.Render(b.String())
	}

	end := min(m.offset+visibleRows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(m.filtered[i], i == m.cursor) + "\n")
	}
	b.WriteString(sepStyle.Render(strings.Repeat("─", boxW)) + "\n")
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

	// Ligne sélectionnée : un seul style Background+Width (motif fiable qui remplit
	// toute la largeur en teal). Le texte passe en clair ; la forme ◆/▣ reste.
	if selected {
		text := "▌ " + glyph + "  " + name
		if it.Installed {
			text += "  ●"
		}
		return selRowStyle.Render(text)
	}

	color := bulletStyle
	switch it.Badge {
	case "F":
		color = formulaStyle
	case "C":
		color = caskStyle
	}
	row := "  " + color.Render(glyph) + "  " + nameStyle.Render(name)
	if it.Installed {
		row += "  " + dotStyle.Render("●")
	}
	return row
}

func (m Model) footer() string {
	sep := hintStyle.Render("   ")
	parts := []string{
		keyStyle.Render("↑↓") + hintStyle.Render(" naviguer"),
		keyStyle.Render("⇥") + hintStyle.Render(" insérer"),
	}
	if m.executable {
		parts = append(parts, keyStyle.Render("↩")+hintStyle.Render(" exécuter"))
	}
	parts = append(parts, keyStyle.Render("esc")+hintStyle.Render(" annuler"))
	return strings.Join(parts, sep)
}
