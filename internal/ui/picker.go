// Package ui fournit le sélecteur interactif (Bubble Tea + Lip Gloss) aux couleurs
// de Cocktails : accent teal, icônes distinctes formula/cask, indicateur « installé »,
// rappels de touches ⇥ (insérer) / ↩ (exécuter).
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
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

	// Ligne courante quand le popup n'a pas le clavier : elle reste désignée — c'est
	// elle que ⇥ insère — mais au repos, sur le fond des pastilles plutôt qu'en accent
	// souligné. C'est la convention des listes du système : sélection grisée tant que le
	// contrôle n'a pas le focus, colorée dès qu'il l'a.
	selIdleStyle = lipgloss.NewStyle().
			Foreground(fg).
			Background(sepCl).
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

// itemLigne fait d'un candidat de complétion une Ligne : une clé, une cellule. Le
// sélecteur délègue ainsi filtre, curseur et défilement au cœur commun (liste.go),
// que la vue tabulaire emploie de la même façon.
type itemLigne struct{ complete.Item }

func (i itemLigne) Cle() string        { return i.Name }
func (i itemLigne) Cellules() []string { return []string{i.Name} }

// Model est le sélecteur.
type Model struct {
	title      string
	liste      *Liste
	input      textinput.Model
	executable bool
	quitting   bool // sortie en cours → View() vide pour que Bubble Tea efface le cadre

	// Projections du cœur, rafraîchies par sync(). Elles existent pour deux raisons :
	// le rendu du cadre veut des complete.Item concrets, et les tests du sélecteur —
	// qui font foi pour cette refactorisation — les lisent directement.
	filtered []complete.Item
	cursor   int
	offset   int

	Chosen  *complete.Item // sélection (nil si annulé)
	Execute bool           // ↩ : la commande est à exécuter

	// Keys personnalise le pied du cadre : nil (le cas ordinaire, jigger pick) garde le
	// pied historique (⇥ insérer / ↩ exécuter / ↑↓ naviguer / esc annuler). Le sélecteur
	// de désambiguïsation de la façade (main.trancher) le renseigne pour dire
	// « ↵ choisir / ^G annuler » — ⇥ et ↩ n'ont pas de sens propre là où on choisit un
	// gestionnaire, pas un texte à insérer (spec §3, README).
	Keys []Key
}

// New crée le sélecteur pour un résultat de complétion.
func New(title string, res complete.Result) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = i18n.T("popup.filter")
	// Même raison que `pad` : sans fond explicite, la saisie s'affiche sur celui du terminal.
	ti.TextStyle = base.Foreground(ink)
	ti.PlaceholderStyle = base.Foreground(muted)
	ti.Focus()

	lignes := make([]Ligne, len(res.Items))
	for i, it := range res.Items {
		lignes[i] = itemLigne{it}
	}

	m := Model{
		title:      title,
		liste:      NouvelleListe(lignes, visibleRows),
		executable: res.Executable,
		input:      ti,
	}
	m.sync()
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

// sync recopie l'état du cœur dans les projections du modèle. Un seul endroit le fait,
// pour qu'il n'y ait jamais deux vérités sur ce qui est affiché.
func (m *Model) sync() {
	ls := m.liste.Filtrees()
	out := make([]complete.Item, len(ls))
	for i, l := range ls {
		out[i] = l.(itemLigne).Item
	}
	m.filtered = out
	m.cursor = m.liste.Curseur()
	m.offset = m.liste.Offset()
}

// modeView affiche le mode de filtre courant, et signale un motif qui ne compile pas.
// En mode texte — le comportement historique — rien ne s'affiche : le sélecteur reste
// exactement ce qu'il était pour qui ne se sert pas des expressions rationnelles.
func (m Model) modeView() string {
	if m.liste.Mode() == FiltreSousChaine {
		return ""
	}
	s := filterHint.Render("  [" + i18n.T("table.moderegex") + "]")
	if !m.liste.Valide() {
		s += base.Foreground(amber).Render(" " + i18n.T("table.badregex"))
	}
	return s
}

// applyFilter conserve son nom : les tests du sélecteur l'appellent directement.
func (m *Model) applyFilter() {
	m.liste.Filtrer(m.input.Value())
	m.sync()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+g":
			// ctrl+g : c'est la touche que la désambiguïsation façade documente
			// (« ^G annuler », spec §3, README). Elle annule au même titre qu'esc et
			// ctrl+c partout ailleurs — un alias de plus, rien qui change esc pour le
			// sélecteur ordinaire.
			m.Chosen = nil
			m.quitting = true
			return m, tea.Quit
		// ^R bascule entre texte brut et expression rationnelle. Sans conflit ici : le
		// sélecteur plein écran a le clavier pour lui seul, là où ^R appartient au shell
		// dans le popup vivant (A-11, seconde moitié).
		case "ctrl+r":
			m.liste.BasculerMode()
			m.sync()
			return m, nil
		case "up", "ctrl+p":
			m.liste.Haut()
			m.sync()
			return m, nil
		case "down", "ctrl+n":
			m.liste.Bas()
			m.sync()
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
	if l := m.liste.Courante(); l != nil {
		it := l.(itemLigne).Item
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

	keys := m.Keys
	if keys == nil {
		keys = []Key{{"⇥", i18n.T("popup.insert")}}
		if m.executable {
			keys = append(keys, Key{"↩", i18n.T("popup.execute")})
		}
		// ^R prend la place de « ↑↓ naviguer » : le cadre a une largeur fixe, et un pied
		// qui déborde perd son dernier libellé. Dans une liste, les flèches sont
		// évidentes ; l'existence d'un mode regex ne l'est pas — c'est elle qu'un pied
		// doit enseigner. La ligne de filtre, elle, affiche le mode courant.
		keys = append(keys,
			Key{"^R", i18n.T("table.regex")},
			Key{"esc", i18n.T("popup.cancel")})
	}

	return Frame{
		Title:      m.title,
		Items:      m.filtered,
		Sel:        m.cursor,
		Offset:     m.offset,
		FilterView: filterHint.Render("› ") + m.input.View() + m.modeView(),
		Empty:      i18n.T("popup.empty"),
		Keys:       keys,
		Focused:    true, // le sélecteur plein écran possède le clavier, par construction
	}.Render()
}
