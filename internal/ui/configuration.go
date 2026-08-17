package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
)

// LigneConfig est un réglage tel que l'écran l'affiche. Les valeurs sont calculées par
// l'appelant : l'écran ne lit ni l'environnement ni le fichier, et n'écrit rien — il rend
// les modifications, et c'est main qui les enregistre.
type LigneConfig struct {
	Cle         string // clé du fichier, sans le préfixe (« rows »)
	Env         string // nom de la variable d'environnement (« JIGGER_ROWS »)
	Valeur      string
	Provenance  string
	Description string
	// Fige marque les lignes que jigger observe sans les posséder — $SCOOP, les
	// gestionnaires détectés. Les proposer à la modification serait mentir.
	Fige bool
}

// GroupeConfig rassemble des réglages de même nature. La note explique une fois, sur le
// groupe, ce qui vaudrait sinon d'être répété à chaque ligne — « prend effet au prochain
// shell », par exemple.
type GroupeConfig struct {
	Titre  string
	Note   string
	Lignes []LigneConfig
}

// Configuration est l'écran de réglage.
//
// Deux modes, et c'est ce qui rend ses raccourcis confortables : en **navigation**, aucun
// champ ne détient le clavier, donc les lettres simples sont libres. En **édition**, le
// champ prend tout, et seuls ↵ et esc en sortent. La pénurie de touches qui contraint le
// popup (A-19) n'existe pas ici.
type Configuration struct {
	groupes  []GroupeConfig
	plates   []indexLigne // les lignes modifiables, à plat, dans l'ordre d'affichage
	curseur  int
	edition  bool
	input    textinput.Model
	largeur  int
	quitting bool

	// Modifs porte les valeurs changées, prêtes à être écrites. Retraits porte les clés
	// remises à leur défaut, donc à supprimer du fichier.
	Modifs   map[string]string
	Retraits []string
}

type indexLigne struct{ groupe, ligne int }

// La ligne courante de l'écran : même accent que le sélecteur, sans sa largeur imposée.
var configSelStyle = lipgloss.NewStyle().Foreground(accent).Background(panelBg).Bold(true)

// NouvelleConfiguration crée l'écran.
func NouvelleConfiguration(groupes []GroupeConfig) Configuration {
	ti := textinput.New()
	ti.Prompt = ""
	ti.TextStyle = base.Foreground(ink)
	ti.PlaceholderStyle = base.Foreground(muted)

	c := Configuration{
		groupes: groupes,
		input:   ti,
		largeur: 100,
		Modifs:  map[string]string{},
	}
	for g, gr := range groupes {
		for l, li := range gr.Lignes {
			if !li.Fige {
				c.plates = append(c.plates, indexLigne{g, l})
			}
		}
	}
	return c
}

func (c Configuration) Init() tea.Cmd { return nil }

// courante rend la ligne sous le curseur, ou nil si l'écran n'a rien de modifiable.
func (c *Configuration) courante() *LigneConfig {
	if c.curseur < 0 || c.curseur >= len(c.plates) {
		return nil
	}
	i := c.plates[c.curseur]
	return &c.groupes[i.groupe].Lignes[i.ligne]
}

func (c Configuration) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.largeur = msg.Width
		return c, nil

	case tea.KeyMsg:
		if c.edition {
			return c.editer(msg)
		}
		return c.naviguer(msg)
	}
	return c, nil
}

// naviguer : aucun champ n'a le clavier, donc les lettres simples servent de raccourcis.
func (c Configuration) naviguer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c", "ctrl+g", "q":
		c.quitting = true
		return c, tea.Quit

	case "up", "ctrl+p", "k":
		if c.curseur > 0 {
			c.curseur--
		}
		return c, nil

	case "down", "ctrl+n", "j":
		if c.curseur < len(c.plates)-1 {
			c.curseur++
		}
		return c, nil

	case "enter":
		li := c.courante()
		if li == nil {
			return c, nil
		}
		c.edition = true
		c.input.SetValue(li.Valeur)
		c.input.CursorEnd()
		c.input.Focus()
		return c, textinput.Blink

	// « r » comme remise : la ligne reprend sa valeur par défaut, donc disparaît du
	// fichier. Distinct d'une valeur vide, qui est un choix délibéré.
	case "r":
		if li := c.courante(); li != nil {
			c.Retraits = append(c.Retraits, li.Cle)
			delete(c.Modifs, li.Cle)
			li.Valeur = "—"
			li.Provenance = i18n.T("cfg.from_default")
		}
		return c, nil
	}
	return c, nil
}

func (c Configuration) editer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+g":
		c.edition = false
		c.input.Blur()
		return c, nil

	case "enter":
		if li := c.courante(); li != nil {
			v := c.input.Value()
			c.Modifs[li.Cle] = v
			// Une modification retire la ligne des remises : le dernier geste gagne.
			for i, cle := range c.Retraits {
				if cle == li.Cle {
					c.Retraits = append(c.Retraits[:i], c.Retraits[i+1:]...)
					break
				}
			}
			li.Valeur = v
			if v == "" {
				li.Valeur = "—"
			}
			li.Provenance = i18n.T("cfg.from_file")
		}
		c.edition = false
		c.input.Blur()
		return c, nil
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

func (c Configuration) View() string {
	if c.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("cfg.title")) + "\n\n")

	plate := 0
	for _, gr := range c.groupes {
		b.WriteString(base.Foreground(accent).Bold(true).Render(gr.Titre) + "\n")
		if gr.Note != "" {
			b.WriteString(filterHint.Render("  "+gr.Note) + "\n")
		}
		for _, li := range gr.Lignes {
			courante := !li.Fige && plate == c.curseur
			b.WriteString(c.ligne(li, courante) + "\n")
			if !li.Fige {
				plate++
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(c.pied())
	return b.String()
}

func (c Configuration) ligne(li LigneConfig, courante bool) string {
	valeur := li.Valeur
	if courante && c.edition {
		valeur = c.input.View()
	}

	nom := fmt.Sprintf("  %-22s", li.Env)
	corps := fmt.Sprintf("%-24s %-14s %s", valeur, "["+li.Provenance+"]", li.Description)

	switch {
	case li.Fige:
		return base.Foreground(muted).Render(nom + corps)
	case courante:
		// Pas selStyle : il porte la largeur fixe du popup (56 colonnes), qui replierait
		// la ligne sur un écran plein. Ici c'est l'écran qui donne la largeur.
		return configSelStyle.Render(nom + corps)
	default:
		return nameStyle.Render(nom + corps)
	}
}

func (c Configuration) pied() string {
	var keys []Key
	if c.edition {
		keys = []Key{
			{"↵", i18n.T("table.confirm")},
			{"esc", i18n.T("popup.cancel")},
		}
	} else {
		keys = []Key{
			{"↵", i18n.T("cfg.edit")},
			{"r", i18n.T("cfg.reset")},
			{"↑↓", i18n.T("popup.navigate")},
			{"q", i18n.T("cfg.quit_save")},
		}
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, pillStyle.Render(k.Key)+hintStyle.Render(" "+k.Label))
	}
	return lipgloss.NewStyle().Background(panelBg).
		Render(strings.Join(parts, hintStyle.Render("   ")))
}
