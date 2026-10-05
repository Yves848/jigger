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

// Palette (vive, dérivée de Cocktails), déclinée pour chaque profil de terminal.
//
// Les index 256 et 16 couleurs sont choisis, pas déduits : termenv projette un hex sur
// l'index le plus proche, et en 16 couleurs cette projection trahissait le sens — l'ambre
// devenait du rouge vif (91), et l'encre des touches tombait sur le cyan de leur propre
// fond (96 sur 46), identiques dans Catppuccin : des touches invisibles. En 16 couleurs, le
// cadre s'en remet donc au thème de l'utilisateur : pas de fond de panneau, texte à la
// couleur par défaut, et des index de sens (3 jaune, 5 magenta, 2 vert, 6 cyan). Un index
// vide ("") ne pose aucune couleur.
var (
	accent    = lipgloss.CompleteColor{TrueColor: "#2DD4BF", ANSI256: "43", ANSI: "6"}  // teal : bordure au focus, ligne courante
	accentBas = lipgloss.CompleteColor{TrueColor: "#0F766E", ANSI256: "30", ANSI: "6"}  // teal éteint : bordure au repos
	ink       = lipgloss.CompleteColor{TrueColor: "#EAFBF7", ANSI256: "195", ANSI: ""}  // texte clair (touches, nom courant)
	fg        = lipgloss.CompleteColor{TrueColor: "#CBD2DE", ANSI256: "188", ANSI: ""}  // texte normal
	muted     = lipgloss.CompleteColor{TrueColor: "#7C8598", ANSI256: "102", ANSI: ""}  // secondaire (cf. estompe)
	amber     = lipgloss.CompleteColor{TrueColor: "#F5B841", ANSI256: "215", ANSI: "3"} // formula ◆
	violet    = lipgloss.CompleteColor{TrueColor: "#B79BFF", ANSI256: "141", ANSI: "5"} // cask ▣
	green     = lipgloss.CompleteColor{TrueColor: "#4ADE80", ANSI256: "78", ANSI: "2"}  // installé
	panelBg   = lipgloss.CompleteColor{TrueColor: "#0C131F", ANSI256: "233", ANSI: ""}
	sepCl     = lipgloss.CompleteColor{TrueColor: "#26374C", ANSI256: "236", ANSI: ""} // bande de la ligne courante au repos
	bandeVive = lipgloss.CompleteColor{TrueColor: "#134E4A", ANSI256: "236", ANSI: ""} // bande de la ligne courante au focus
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

	// Ligne courante de la vue tabulaire : un simple soulignement. Elle garde exactement
	// la géométrie d'une ligne ordinaire, donc rien ne se décale quand le curseur bouge.
	// (Le popup, lui, compose sa ligne courante segment par segment : cf. Frame.renderRow.)
	selStyle = lipgloss.NewStyle().
			Foreground(accent).
			Background(panelBg).
			Bold(true).
			Underline(true).
			Width(rowW)

	// Rappels de touches : la touche en gras, sans fond. Les anciennes pastilles étaient
	// les blocs les plus contrastés du cadre, pour l'information qui change le moins.
	keyStyle = base.Foreground(ink).Bold(true)

	titleStyle = base.Foreground(accent).Bold(true)
	filterHint = base.Foreground(muted)

	formulaStyle = base.Foreground(amber).Bold(true)
	caskStyle    = base.Foreground(violet).Bold(true)
	bulletStyle  = base.Foreground(muted)
	nameStyle    = base.Foreground(fg)
	verStylePkg  = base.Foreground(muted) // version installée (atténuée)
	dotStyle     = base.Foreground(green)

	hintStyle  = base.Foreground(muted)
	emptyStyle = base.Foreground(muted).Italic(true)
)

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

	// SansFiltre retire la ligne de saisie. Un filtre a du sens sur des candidats ; il
	// n'en a aucun sur une question par oui ou non — il invite à taper là où il n'y a
	// rien à chercher, et masquer la ligne sans couper la saisie filtrerait à l'insu de
	// celui qui répond. C'est la confirmation d'élévation qui le pose (ADR-0004).
	SansFiltre bool
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

	// Sans ligne de filtre, la frappe ne va nulle part : filtrer une liste dont on ne
	// montre pas le motif reviendrait à faire disparaître des choix sans dire pourquoi.
	if m.SansFiltre {
		return m, nil
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

	filtre := ""
	if !m.SansFiltre {
		filtre = filterHint.Render("› ") + m.input.View() + m.modeView()
	}

	return Frame{
		Title:      m.title,
		Items:      m.filtered,
		Sel:        m.cursor,
		Offset:     m.offset,
		FilterView: filtre,
		Empty:      i18n.T("popup.empty"),
		Keys:       keys,
		Focused:    true, // le sélecteur plein écran possède le clavier, par construction
	}.Render()
}
