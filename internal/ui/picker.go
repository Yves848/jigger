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
	accent  = lipgloss.Color("#2DD4BF") // teal (accent + ligne courante)
	ink     = lipgloss.Color("#EAFBF7") // texte clair (pastilles de touches)
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
	boxW    = 58
	rowW    = boxW - 2 // largeur d'une ligne : gouttière de 2 colonnes à droite
	nameMax = boxW - 12
	// Les lignes sont désormais jointives (plus d'interligne) : à hauteur de popup
	// constante, on affiche deux fois plus de candidats.
	visibleRows = 12
)

var (
	// Tout style de texte doit porter le fond du panneau : la séquence de reset émise en
	// fin de segment coupe sinon le fond posé par le cadre, et le reste de la ligne
	// (remplissage compris) s'affiche sur le fond du terminal — d'où une bande visible.
	base = lipgloss.NewStyle().Background(panelBg)

	// Le cadre porte le fond ET la largeur : lipgloss remplit uniformément les zones
	// nues (espaces sans fond) avec panelBg, y compris après les resets internes.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			BorderBackground(panelBg).
			Background(panelBg).
			Width(boxW)

	// Ligne courante : pas de cadre, un simple soulignement. Elle garde exactement la
	// géométrie d'une ligne ordinaire (même indentation, même gouttière), donc rien ne
	// se décale quand le curseur bouge. Le soulignement porte aussi sur le remplissage
	// (lipgloss souligne les espaces par défaut) : la règle court sur toute la largeur.
	selStyle = lipgloss.NewStyle().
			Foreground(accent).
			Background(panelBg).
			Bold(true).
			Underline(true).
			Width(rowW)

	// Rappels de touches : pastilles (fond + remplissage). Une vraie bordure coûterait
	// deux lignes de plus au pied du popup pour le même effet.
	pillStyle = lipgloss.NewStyle().Background(sepCl).Foreground(ink).Bold(true).Padding(0, 1)

	promptStyle = base.Foreground(accent).Bold(true)
	titleStyle  = base.Foreground(accent).Bold(true)
	filterHint  = base.Foreground(muted)

	formulaStyle = base.Foreground(amber).Bold(true)
	caskStyle    = base.Foreground(violet).Bold(true)
	bulletStyle  = base.Foreground(muted)
	nameStyle    = base.Foreground(fg)
	verStylePkg  = base.Foreground(muted) // version installée (atténuée)
	dotStyle     = base.Foreground(green)

	hintStyle  = base.Foreground(muted)
	emptyStyle = base.Foreground(muted).Italic(true)
	verStyle   = base.Foreground(muted)
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
	// Même raison que `pad` : sans fond explicite, la saisie s'affiche sur celui du terminal.
	ti.TextStyle = base.Foreground(ink)
	ti.PlaceholderStyle = base.Foreground(muted)
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

	// Version poussée à droite (repère du binaire lancé), sur la même ligne que le titre.
	head := promptStyle.Render("❯") + pad(1) + titleStyle.Render(m.title)
	if Version != "" {
		ver := verStyle.Render("jigger " + Version)
		gap := boxW - lipgloss.Width(head) - lipgloss.Width(ver)
		if gap < 1 {
			gap = 1
		}
		head += pad(gap) + ver
	}
	b.WriteString(head + "\n")
	b.WriteString(filterHint.Render("› ") + m.input.View() + "\n")

	if len(m.filtered) == 0 {
		b.WriteString(pad(2) + emptyStyle.Render("aucun candidat"))
		return boxStyle.Render(b.String())
	}

	// Lignes jointives : la ligne courante se signale par sa couleur et son soulignement,
	// il n'y a plus d'interligne à intercaler. Toutes les lignes faisant une ligne de haut,
	// la hauteur du popup ne dépend pas de la position du curseur.
	end := min(m.offset+visibleRows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(m.filtered[i], i == m.cursor) + "\n")
	}
	b.WriteString("\n") // respiration avant le pied
	b.WriteString(m.footer())
	return boxStyle.Render(b.String())
}

// pad rend n espaces au fond du panneau. Indispensable partout où du remplissage sépare
// deux segments stylés : un espace en texte nu hérite du reset précédent, donc du fond du
// terminal, et trahit une bande plus claire au milieu de la ligne.
func pad(n int) string {
	if n < 1 {
		return ""
	}
	return base.Render(strings.Repeat(" ", n))
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

	// Géométrie commune aux deux états : 2 colonnes d'indentation à gauche, gouttière de
	// 2 colonnes à droite. On calcule le remplissage sur le texte nu, avant tout style.
	leftPlain := "  " + glyph + "  " + name
	gap := rowW - lipgloss.Width(leftPlain) - lipgloss.Width(rightPlain)
	if gap < 1 {
		gap = 1
	}

	// Ligne courante : rendue d'un seul tenant en texte nu, sans les couleurs par segment
	// (leurs séquences de reset interrompraient le soulignement au milieu de la ligne).
	if selected {
		return selStyle.Render(leftPlain + strings.Repeat(" ", gap) + rightPlain)
	}

	color := bulletStyle
	switch it.Badge {
	case "F":
		color = formulaStyle
	case "C":
		color = caskStyle
	}
	left := pad(2) + color.Render(glyph) + pad(2) + nameStyle.Render(name)
	right := ""
	if it.Version != "" {
		right = verStylePkg.Render(it.Version)
	}
	if it.Installed {
		if it.Version != "" {
			right += pad(2)
		}
		right += dotStyle.Render("●")
	}
	return left + pad(gap) + right
}

func (m Model) footer() string {
	sep := pad(2)
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
	return pad(2) + strings.Join(parts, sep)
}
