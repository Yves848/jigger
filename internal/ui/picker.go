// Package ui fournit le sélecteur interactif (Bubble Tea + Lip Gloss) aux couleurs
// de Cocktails : accent teal, badges F/C, indicateur « installé », rappels de touches
// ⇥ (insérer) / ↩ (exécuter).
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
)

// Palette (identique à l'app Cocktails).
var (
	accent   = lipgloss.Color("#2DD4BF")
	fg       = lipgloss.Color("#ECEEF5")
	muted    = lipgloss.Color("#8A93A6")
	green    = lipgloss.Color("#43C07A")
	violet   = lipgloss.Color("#A78BFA")
	selBg    = lipgloss.Color("#0E2A26")
	borderCl = lipgloss.Color("#2DD4BF")
)

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCl).
			Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Foreground(accent).Bold(true)
	promptStyle   = lipgloss.NewStyle().Foreground(accent)
	rowStyle      = lipgloss.NewStyle().Foreground(fg)
	selRowStyle   = lipgloss.NewStyle().Foreground(accent).Background(selBg).Bold(true)
	barStyle      = lipgloss.NewStyle().Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	badgeFStyle   = lipgloss.NewStyle().Foreground(muted)
	badgeCStyle   = lipgloss.NewStyle().Foreground(violet)
	installedDot  = lipgloss.NewStyle().Foreground(green).Render("●")
	footerStyle   = lipgloss.NewStyle().Foreground(muted)
	hintKeyStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	hintTextStyle = lipgloss.NewStyle().Foreground(muted)
)

const visibleRows = 12

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

	m := Model{
		title:      title,
		all:        res.Items,
		executable: res.Executable,
		input:      ti,
	}
	m.applyFilter()
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		m.filtered = m.all
	} else {
		m.filtered = m.filtered[:0]
		for _, it := range m.all {
			if strings.Contains(strings.ToLower(it.Name), q) {
				m.filtered = append(m.filtered, it)
			}
		}
	}
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
		case "esc", "ctrl-c", "ctrl+c":
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

	b.WriteString(promptStyle.Render("❯ ") + titleStyle.Render(m.title) + "\n")
	b.WriteString(mutedStyle.Render("/ ") + m.input.View() + "\n")

	if len(m.filtered) == 0 {
		b.WriteString(mutedStyle.Render("aucun candidat"))
		return boxStyle.Render(b.String())
	}

	end := min(m.offset+visibleRows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(m.filtered[i], i == m.cursor))
		b.WriteByte('\n')
	}

	b.WriteString(m.footer())
	return boxStyle.Render(b.String())
}

func (m Model) renderRow(it complete.Item, selected bool) string {
	bar := "  "
	if selected {
		bar = barStyle.Render("▎ ")
	}

	var badge string
	switch it.Badge {
	case "F":
		badge = badgeFStyle.Render("F ")
	case "C":
		badge = badgeCStyle.Render("C ")
	default:
		badge = "  "
	}

	name := it.Name
	if selected {
		name = selRowStyle.Render(name)
	} else {
		name = rowStyle.Render(name)
	}

	dot := " "
	if it.Installed {
		dot = installedDot
	}

	return bar + badge + name + " " + dot
}

func (m Model) footer() string {
	sep := footerStyle.Render("  ·  ")
	parts := []string{
		hintKeyStyle.Render("↑↓") + " " + hintTextStyle.Render("naviguer"),
		hintKeyStyle.Render("⇥") + " " + hintTextStyle.Render("insérer"),
	}
	if m.executable {
		parts = append(parts, hintKeyStyle.Render("↩")+" "+hintTextStyle.Render("exécuter"))
	}
	parts = append(parts, hintKeyStyle.Render("esc")+" "+hintTextStyle.Render("annuler"))
	return "\n" + strings.Join(parts, sep)
}
