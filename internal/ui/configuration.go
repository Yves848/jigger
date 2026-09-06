package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
)

// LigneConfig est un réglage tel que l'écran l'affiche. Les valeurs sont calculées par
// l'appelant : l'écran ne lit ni l'environnement ni le fichier, et n'écrit rien — il rend
// les modifications, et c'est main qui les enregistre.
// TypeLigne dit quelle sorte de valeur la ligne porte, donc comment la montrer et la
// modifier. Défini ici plutôt qu'importé de internal/config, à dessein : l'écran ne lit ni
// l'environnement ni le fichier (cf. LigneConfig), et c'est cette indépendance qui le rend
// éprouvable sans eux. L'appelant traduit ses propres types vers ceux-ci.
type TypeLigne int

const (
	LigneTexte TypeLigne = iota
	LigneBooleen
	LigneEntier
	LigneDuree
)

type LigneConfig struct {
	Cle         string // clé du fichier, sans le préfixe (« rows »)
	Env         string // nom de la variable d'environnement (« JIGGER_ROWS »)
	Valeur      string
	Provenance  string
	Description string
	Type        TypeLigne
	// ParDefaut : rien n'a été choisi pour cette ligne. L'écran l'estompe, pour que l'œil
	// aille droit aux réglages réellement posés — ils sont trois sur dix-huit, et se
	// noyaient dans autant de « [default] » de même poids visuel.
	ParDefaut bool
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

	// Abandon dit que l'écran a été quitté SANS vouloir enregistrer. L'appelant écrit
	// Modifs et Retraits ; c'est donc à lui de ne rien écrire quand ce drapeau est levé,
	// et non à l'écran de vider ses champs — les effacer priverait un test de ce qu'il
	// vérifie, et masquerait un appelant qui aurait oublié de lire le drapeau.
	Abandon bool
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
	// Quitter SANS enregistrer. « esc » et « ctrl+c » signifient « abandonne » partout
	// ailleurs, et « ctrl+g » est déjà le geste d'annulation du dépôt — il annule une
	// édition en cours quelques lignes plus bas, et referme le popup. Les trois faisaient
	// pourtant enregistrer, avec « q », jusqu'ici : quelqu'un qui modifiait une valeur, se
	// ravisait et frappait ctrl+c obtenait exactement ce qu'il essayait d'éviter.
	case "esc", "ctrl+c", "ctrl+g":
		c.Abandon = true
		c.quitting = true
		return c, tea.Quit

	// Enregistrer et quitter. Une seule touche le fait, et c'est celle que le pied annonce.
	case "q":
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

	// Espace bascule un booléen sur place. C'est la raison d'être des deux modes : en
	// navigation, les touches simples sont libres, et régler un booléen ne devrait pas
	// demander d'ouvrir un champ pour y taper un chiffre puis valider.
	case " ", "space":
		if li := c.courante(); li != nil && li.Type == LigneBooleen {
			v := "0"
			if !vrai(li.Valeur) {
				v = "1"
			}
			c.poser(li, v)
		}
		return c, nil

	// « r » comme remise : la ligne reprend sa valeur par défaut, donc disparaît du
	// fichier. Distinct d'une valeur vide, qui est un choix délibéré.
	case "r":
		if li := c.courante(); li != nil {
			c.Retraits = append(c.Retraits, li.Cle)
			delete(c.Modifs, li.Cle)
			li.Valeur = "—"
			li.Provenance = i18n.T("cfg.from_default")
			li.ParDefaut = true
		}
		return c, nil
	}
	return c, nil
}

// poser inscrit une valeur choisie : même geste pour l'édition et pour la bascule à
// l'espace, donc un seul endroit où se tromper.
func (c *Configuration) poser(li *LigneConfig, v string) {
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
	li.ParDefaut = false
}

// vrai lit un booléen de configuration. « 1 » est la forme écrite dans le fichier ; les
// autres sont acceptées parce qu'un humain les tape.
func vrai(v string) bool {
	switch v {
	case "1", "true", "on", "yes", "oui":
		return true
	}
	return false
}

func (c Configuration) editer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+g":
		c.edition = false
		c.input.Blur()
		return c, nil

	case "enter":
		if li := c.courante(); li != nil {
			c.poser(li, c.input.Value())
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

	l := c.largeurUtile()
	var b strings.Builder
	b.WriteString(titleStyle.Render(caler(i18n.T("cfg.title"), l)) + "\n")
	b.WriteString(base.Render(caler("", l)) + "\n")

	plate := 0
	for _, gr := range c.groupes {
		b.WriteString(base.Foreground(accent).Bold(true).Render(caler(gr.Titre, l)) + "\n")
		if gr.Note != "" {
			b.WriteString(filterHint.Render(caler("  "+gr.Note, l)) + "\n")
		}
		for _, li := range gr.Lignes {
			courante := !li.Fige && plate == c.curseur
			b.WriteString(c.ligne(li, courante, l) + "\n")
			if !li.Fige {
				plate++
			}
		}
		// Séparation REMPLIE, et non un simple saut de ligne : le fond des lignes vient de
		// `base`, donc une ligne vide non calée perce le panneau d'un trou de la couleur du
		// terminal. Le bloc se lit alors comme des rangées flottantes plutôt que comme un
		// panneau continu.
		b.WriteString(base.Render(caler("", l)) + "\n")
	}

	b.WriteString(c.pied())
	return b.String()
}

// Largeurs des trois premières colonnes. Fixes, donc les valeurs et les provenances
// s'alignent d'un groupe à l'autre — c'est ce qui fait lire le bloc comme un tableau.
const (
	colNom    = 24
	colValeur = 22
	colProv   = 12
)

// largeurUtile donne la largeur COMMUNE à toutes les lignes.
//
// Elle n'est pas cosmétique. Les styles de cet écran portent un fond (cf. base) : sans
// calage, le fond de chaque ligne s'arrête où finit son texte, et le bloc a des bords
// droits en dents de scie. C'est ce qui donnait à l'écran son aspect inachevé.
func (c Configuration) largeurUtile() int {
	max := 0
	for _, gr := range c.groupes {
		if l := lipgloss.Width(gr.Titre); l > max {
			max = l
		}
		for _, li := range gr.Lignes {
			if l := colNom + colValeur + colProv + lipgloss.Width(li.Description); l > max {
				max = l
			}
		}
	}
	// Le terminal a le dernier mot : déborder replierait chaque ligne en deux.
	if c.largeur > 0 && max > c.largeur {
		max = c.largeur
	}
	return max
}

// caler complète s jusqu'à n colonnes.
//
// lipgloss.Width et non len : la valeur en cours d'édition porte des séquences ANSI, et les
// libellés portent des accents — len compterait les unes et les autres comme des colonnes,
// et le calage serait faux là où il compte le plus.
func caler(s string, n int) string {
	if l := lipgloss.Width(s); l < n {
		return s + strings.Repeat(" ", n-l)
	}
	return s
}

// valeurAffichee : un booléen se montre coché, pas en « 0 » ou « 1 ».
//
// Le glyphe se lit d'un coup d'œil, ne demande aucune traduction, et dit du même coup que
// la ligne se bascule — ce qu'un chiffre ne disait pas.
func valeurAffichee(li LigneConfig) string {
	if li.Type != LigneBooleen {
		return li.Valeur
	}
	if vrai(li.Valeur) {
		return "[✓]"
	}
	return "[ ]"
}

func (c Configuration) ligne(li LigneConfig, courante bool, largeur int) string {
	valeur := valeurAffichee(li)
	if courante && c.edition {
		valeur = c.input.View()
	}

	nom := caler("  "+li.Env, colNom)
	val := caler(valeur, colValeur)
	prov := caler("["+li.Provenance+"]", colProv)
	desc := li.Description

	// La ligne courante et les lignes figées se rendent d'un seul tenant : un fond découpé
	// en segments de couleurs différentes se lirait comme trois cellules côte à côte, pas
	// comme une ligne choisie.
	// tronquer AVANT de caler, sans quoi une description plus longue que la place restante
	// déborde la largeur commune et ramène les bords en dents de scie qu'on vient de
	// supprimer — caler complète, il ne raccourcit pas. C'est le test de largeur qui l'a
	// montré, pas la relecture.
	if li.Fige {
		return base.Foreground(muted).Render(caler(tronquer(nom+val+prov+desc, largeur), largeur))
	}
	if courante {
		// Pas selStyle : il porte la largeur fixe du popup (56 colonnes), qui replierait
		// la ligne sur un écran plein. Ici c'est l'écran qui donne la largeur.
		return configSelStyle.Render(caler(tronquer(nom+val+prov+desc, largeur), largeur))
	}

	// Hors sélection, les trois colonnes n'ont pas le même poids. Le nom et la valeur
	// portent l'information ; la provenance par défaut et la description ne sont là que si
	// on les cherche. Estomper ce qui est au défaut est ce qui fait ressortir les réglages
	// réellement posés — ils sont une poignée, et se noyaient dans autant de mentions de
	// même intensité.
	styleProv := base.Foreground(muted)
	if !li.ParDefaut {
		styleProv = base.Foreground(accent)
	}
	reste := largeur - colNom - colValeur - colProv
	if reste < 0 {
		reste = 0
	}
	return nameStyle.Render(nom+val) +
		styleProv.Render(prov) +
		hintStyle.Render(caler(tronquer(desc, reste), reste))
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
		}
		// L'espace n'est annoncé que sur une ligne où il fait quelque chose : promettre une
		// bascule devant un champ de texte serait pire que se taire.
		if li := c.courante(); li != nil && li.Type == LigneBooleen {
			keys = append(keys, Key{"espace", i18n.T("cfg.toggle")})
		}
		keys = append(keys,
			Key{"r", i18n.T("cfg.reset")},
			Key{"↑↓", i18n.T("popup.navigate")},
			Key{"esc", i18n.T("cfg.quit_discard")},
			Key{"q", i18n.T("cfg.quit_save")},
		)
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, pillStyle.Render(k.Key)+hintStyle.Render(" "+k.Label))
	}
	return lipgloss.NewStyle().Background(panelBg).
		Render(strings.Join(parts, hintStyle.Render("   ")))
}
