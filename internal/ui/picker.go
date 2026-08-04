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
	accentD = lipgloss.Color("#0B3A34") // teal profond (fond sélection)
	ink     = lipgloss.Color("#EAFBF7") // texte clair
	fg      = lipgloss.Color("#D3D8E3") // texte normal
	muted   = lipgloss.Color("#7C8598")
	amber   = lipgloss.Color("#F5B841") // formula ◆
	violet  = lipgloss.Color("#B79BFF") // cask ▣
	green   = lipgloss.Color("#4ADE80") // installé
	panelBg = lipgloss.Color("#0C131F")
)

const rowWidth = 46

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Background(panelBg).
			Padding(0, 1)
	promptStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	titleStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	filterStyle = lipgloss.NewStyle().Foreground(muted)

	// Styles rendus À L'AFFICHAGE (pas à l'init) : le profil couleur n'est connu
	// qu'au lancement (fixé depuis le TTY dans main), sinon le rendu serait « fade ».
	formulaStyle = lipgloss.NewStyle().Foreground(amber).Bold(true)
	caskStyle    = lipgloss.NewStyle().Foreground(violet).Bold(true)
	dotStyle     = lipgloss.NewStyle().Foreground(green)

	nameStyle   = lipgloss.NewStyle().Foreground(fg)
	selStyle    = lipgloss.NewStyle().Background(accentD).Foreground(ink).Bold(true)
	selBarStyle = lipgloss.NewStyle().Foreground(accent).Background(accentD).Bold(true)

	keyStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(muted)
	emptyStyle = lipgloss.NewStyle().Foreground(muted).Italic(true)
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

	m := Model{title: title, all: res.Items, executable: res.Executable, input: ti}
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

	b.WriteString(promptStyle.Render("❯") + " " + titleStyle.Render(m.title) + "\n")
	b.WriteString(filterStyle.Render("› ") + m.input.View() + "\n")

	if len(m.filtered) == 0 {
		b.WriteString(emptyStyle.Render("aucun candidat"))
		return boxStyle.Render(b.String())
	}

	end := min(m.offset+visibleRows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(m.filtered[i], i == m.cursor) + "\n")
	}
	b.WriteString(m.footer())
	return boxStyle.Render(b.String())
}

func (m Model) renderRow(it complete.Item, selected bool) string {
	icon := "•"
	switch it.Badge {
	case "F":
		icon = formulaStyle.Render("◆")
	case "C":
		icon = caskStyle.Render("▣")
	}

	dotCell := " "
	if it.Installed {
		dotCell = dotStyle.Render("●")
	}

	// corps : icône + nom, complété jusqu'à laisser la place au point « installé ».
	name := nameStyle.Render(it.Name)
	body := icon + "  " + name
	if w := lipgloss.Width(body); w < rowWidth-1 {
		body += strings.Repeat(" ", rowWidth-1-w)
	}
	body += dotCell

	if selected {
		return selBarStyle.Render("▌") + selStyle.Render(" "+body+" ")
	}
	return "  " + body
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
	return hintStyle.Render(strings.Repeat("─", rowWidth+1)) + "\n" + strings.Join(parts, sep)
}
