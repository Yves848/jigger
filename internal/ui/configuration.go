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
	// Choix propose des valeurs ; Ferme dit si la liste est exhaustive. Aide dit le format
	// attendu, pour un champ qu'on saisit. Cf. config.Reglage, qui les déclare.
	Choix []string
	Ferme bool
	Aide  string
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
	groupes []GroupeConfig
	plates  []indexLigne // les lignes modifiables, à plat, dans l'ordre d'affichage
	curseur int
	edition bool
	// choix : la saisie en cours se fait dans une liste fermée, donc au curseur et non au
	// clavier. idxChoix sert aux deux modes — il désigne la valeur retenue dans une liste
	// fermée, et la proposition suivante dans un combo.
	choix    bool
	idxChoix int
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
		c.idxChoix = indexDe(li.Choix, brut(li.Valeur))

		// Liste FERMÉE : la valeur se prend dans la liste. Ouvrir un champ libre inviterait
		// à écrire ce qui sera refusé — « francais » pour la langue, par exemple — sans
		// jamais dire pourquoi.
		if li.Ferme && len(li.Choix) > 0 {
			c.choix = true
			return c, nil
		}
		c.choix = false
		c.input.SetValue(brut(li.Valeur))
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
// affiche rend une valeur telle qu'on la montre : le vide se voit, sinon la ligne aurait
// l'air tronquée.
func affiche(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// brut fait l'inverse : « — » est ce que l'écran MONTRE d'une valeur vide, pas ce qu'il
// faut chercher dans les choix ni réinjecter dans un champ de saisie. Les confondre mettait
// littéralement « — » dans le champ au moment d'éditer une valeur absente.
func brut(v string) string {
	if v == "—" {
		return ""
	}
	return v
}

func indexDe(choix []string, v string) int {
	for i, c := range choix {
		if c == v {
			return i
		}
	}
	return 0
}

func choixA(choix []string, i int, repli string) string {
	if i < 0 || i >= len(choix) {
		return repli
	}
	return choix[i]
}

func vrai(v string) bool {
	switch v {
	case "1", "true", "on", "yes", "oui":
		return true
	}
	return false
}

func (c Configuration) editer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	li := c.courante()

	switch msg.String() {
	case "esc", "ctrl+g":
		c.edition, c.choix = false, false
		c.input.Blur()
		return c, nil

	case "enter":
		if li != nil {
			if c.choix {
				c.poser(li, choixA(li.Choix, c.idxChoix, brut(li.Valeur)))
			} else {
				c.poser(li, c.input.Value())
			}
		}
		c.edition, c.choix = false, false
		c.input.Blur()
		return c, nil
	}

	// Parcourir les propositions. Les mêmes touches dans les deux modes, et c'est ce qui
	// rend le geste apprenable : dans une liste fermée elles CHOISISSENT, dans un combo
	// elles remplissent le champ — qu'on reste libre de corriger ensuite.
	if li != nil && len(li.Choix) > 0 {
		var pas int
		switch msg.String() {
		case "left", "shift+tab":
			pas = -1
		case "right", "tab":
			pas = 1
		}
		if pas != 0 {
			c.idxChoix = (c.idxChoix + pas + len(li.Choix)) % len(li.Choix)
			if !c.choix {
				c.input.SetValue(li.Choix[c.idxChoix])
				c.input.CursorEnd()
			}
			return c, nil
		}
	}

	// Liste fermée : rien d'autre ne s'écrit. Laisser filer les frappes vers un champ que
	// l'écran n'affiche pas donnerait une saisie invisible.
	if c.choix {
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
			if courante && c.edition {
				for _, aide := range c.panneauAide(li, l) {
					b.WriteString(aide + "\n")
				}
			}
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

// panneauAide rend les lignes d'assistance montrées SOUS la ligne en cours de saisie.
//
// Sous la ligne et non ailleurs : l'aide sert au moment où l'on tape, et un encart posé en
// pied de l'écran obligerait l'œil à faire l'aller-retour à chaque frappe. Elle n'apparaît
// que pendant la saisie, donc elle ne coûte rien au reste du temps.
func (c Configuration) panneauAide(li LigneConfig, largeur int) []string {
	var out []string
	marge := strings.Repeat(" ", colNom)

	if len(li.Choix) > 0 {
		var rendu, brutLigne strings.Builder
		rendu.WriteString(base.Render(marge))
		brutLigne.WriteString(marge)
		for i, v := range li.Choix {
			t := " " + affiche(v) + " "
			brutLigne.WriteString(t)
			if i == c.idxChoix {
				rendu.WriteString(configSelStyle.Render(t))
			} else {
				rendu.WriteString(base.Foreground(ink).Render(t))
			}
		}
		// Ce que les touches font ici, dit à l'endroit où on les cherche — et avec le MÊME
		// mot que le pied. Une liste fermée « se choisit » ; un combo « se propose », et
		// c'est là qu'il faut dire qu'on peut aussi écrire.
		note := "   " + i18n.T("cfg.choose")
		if !li.Ferme {
			note = "   " + i18n.T("cfg.suggest") + " · " + i18n.T("cfg.free")
		}
		brutLigne.WriteString(note)
		rendu.WriteString(hintStyle.Render(note))
		rendu.WriteString(base.Render(caler("", largeur-lipgloss.Width(brutLigne.String()))))
		out = append(out, rendu.String())
	}

	if li.Aide != "" {
		out = append(out, filterHint.Render(caler(marge+tronquer(li.Aide, largeur-colNom), largeur)))
	}
	return out
}

func (c Configuration) ligne(li LigneConfig, courante bool, largeur int) string {
	valeur := valeurAffichee(li)
	if courante && c.edition {
		if c.choix {
			// Les chevrons disent « ça se parcourt » là où un champ dirait « ça se tape ».
			valeur = "‹ " + affiche(choixA(li.Choix, c.idxChoix, brut(li.Valeur))) + " ›"
		} else {
			valeur = c.input.View()
		}
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
		keys = []Key{}
		// Annoncer le parcours seulement là où il existe : promettre des propositions
		// devant un champ qui n'en a pas serait pire que se taire.
		if li := c.courante(); li != nil && len(li.Choix) > 0 {
			libelle := i18n.T("cfg.suggest")
			if li.Ferme {
				libelle = i18n.T("cfg.choose")
			}
			keys = append(keys, Key{"←→", libelle})
		}
		keys = append(keys,
			Key{"↵", i18n.T("table.confirm")},
			Key{"esc", i18n.T("popup.cancel")},
		)
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
