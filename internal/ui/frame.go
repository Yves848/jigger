package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	// Alias : le paquet de tests du même dossier a déjà un `ansi` à lui.
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Key est un rappel de touche affiché en pastille au pied du popup.
type Key struct{ Key, Label string }

// Frame décrit tout ce qu'il faut pour dessiner le popup, et rien de plus : aucun état,
// aucune notion de clavier. C'est la seule entrée du rendu, partagée par les deux modes
// — le sélecteur interactif (`jigger pick`, qui possède le terminal) et le rendu sans
// état (`jigger render`, appelé par le widget zsh à chaque frappe, où c'est zsh qui
// possède la ligne). Tout ce qui varie entre les deux passe par un champ.
type Frame struct {
	Title      string
	Items      []complete.Item
	Sel        int    // ligne courante ; hors bornes = aucune ligne active
	Offset     int    // premier candidat affiché
	Rows       int    // candidats affichés au plus (0 = visibleRows)
	Width      int    // largeur intérieure du cadre (0 = boxW)
	FilterView string // ligne de filtre déjà rendue (mode pick) ; "" = pas de ligne
	Empty      string // message affiché quand il n'y a aucun candidat
	Keys       []Key  // rappels de touches du pied
	// Focused dit si le popup a le clavier — c'est-à-dire si les flèches lui reviennent
	// plutôt qu'à l'historique du shell. Sans focus, la ligne courante reste marquée (⇥
	// l'insère toujours) mais d'une teinte au repos : c'est le seul indice qui dit à
	// l'utilisateur où ira sa prochaine flèche.
	Focused bool
}

// Largeur minimale en dessous de laquelle le cadre n'a plus de sens (le pied et les
// versions ne tiennent plus). L'appelant est censé ne pas dessiner du tout.
const minWidth = 24

func (f Frame) boxWidth() int {
	w := f.Width
	if w <= 0 {
		w = boxW
	}
	return max(w, minWidth)
}

func (f Frame) rowWidth() int { return f.boxWidth() - 2 }

// tronquer ramène un texte à `largeur` colonnes, ellipse comprise. Le compte est fait en
// colonnes d'affichage, pas en octets : un nom accentué n'occupe pas la place que sa
// longueur laisse croire.
func tronquer(s string, largeur int) string {
	if largeur < 1 {
		return ""
	}
	if lipgloss.Width(s) <= largeur {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > largeur {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// clip est le garde-fou : aucune ligne ne doit dépasser la largeur du cadre. Au-delà, le
// terminal la replierait — et le popup occuperait plus de lignes que celles que le
// greffon a réservées sous le prompt.
func clip(s string, largeur int) string { return xansi.Truncate(s, largeur, "") }

// avecPM dit si la colonne PM doit apparaître : seulement si plus d'un gestionnaire a
// contribué aux items du cadre — même règle que facade.Formater (§4), via pm.PlusieursPM.
// C'est la façade seule qui remplit Item.PM (cf. pm.Item.PM) : le chemin natif
// (`brew install ⇥`) n'a rien à désambiguïser et n'en porte jamais, donc un seul
// gestionnaire disponible (macOS, brew seul) ne doit pas plus faire apparaître la colonne
// que le chemin natif ne le ferait — sans quoi `jg render --line "jg install fire"`
// poserait un « brew » sur chaque ligne, du bruit pur.
func (f Frame) avecPM() bool {
	pms := make([]string, len(f.Items))
	for i, it := range f.Items {
		pms[i] = it.PM
	}
	return pm.PlusieursPM(pms)
}

func (f Frame) rows() int {
	if f.Rows <= 0 {
		return visibleRows
	}
	return f.Rows
}

// Render dessine le popup complet (cadre compris).
//
// Le titre et les touches vivent dans les bordures : `╭─ brew install ──── 3/18 ─╮` en
// haut, `╰─ ⇥ insérer  ↩ exécuter … ─╯` en bas. Le décor tient ainsi en deux lignes
// au lieu de cinq — en-tête, respiration et pied ne prennent plus rien à la liste, et le
// popup qui reparaît à chaque frappe en est d'autant moins encombrant. Les greffons
// (jigger.plugin.zsh, jigger.psm1) réservent ces deux lignes : les deux chiffres vont
// ensemble.
func (f Frame) Render() string {
	w := f.boxWidth()
	var lignes []string
	if f.FilterView != "" {
		lignes = append(lignes, clip(f.FilterView, w))
	}

	if len(f.Items) == 0 {
		lignes = append(lignes, clip(pad(2)+estompe(emptyStyle).Render(f.Empty), w))
		return f.encadrer(lignes, "", "")
	}

	// Lignes jointives, toutes d'une ligne de haut : la hauteur du popup ne dépend pas de
	// la position du curseur.
	// La colonne PM ne s'ajoute que si au moins un item la porte — comme les tableaux de
	// sortie (cf. facade.Formater) : une colonne toujours vide n'apprend rien.
	avecPM := f.avecPM()

	offset := min(max(f.Offset, 0), len(f.Items)-1)
	end := min(offset+f.rows(), len(f.Items))
	for i := offset; i < end; i++ {
		lignes = append(lignes, clip(f.renderRow(f.Items[i], i == f.Sel, avecPM), w))
	}
	return f.encadrer(lignes, f.compteur(), f.footer(w-5))
}

// encadrer pose les bordures autour des lignes : titre et `droite` dans celle du haut,
// `pied` dans celle du bas. La bordure s'allume (teal vif) quand le popup a le clavier, et
// reste éteinte au repos — le cadre ne réclame rien tant qu'on ne le prend pas en main.
func (f Frame) encadrer(lignes []string, droite, pied string) string {
	w := f.boxWidth()

	// Ce que le titre peut occuper : la ligne, moins « ╭─ » et « ─╮ », les espaces qui
	// l'entourent, la partie droite et au moins un trait. Le compteur l'emporte sur le
	// titre : le titre redit la ligne tapée juste au-dessus, le compteur est la seule
	// chose que le cadre sache et que la ligne ne dise pas.
	place := w + 2 - 4 - 2 - 1
	if droite != "" {
		place -= lipgloss.Width(droite) + 2
	}
	titre := ""
	if t := tronquer(f.Title, place); t != "" {
		titre = nameStyle.Render(t)
	}

	cotes := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true).
		BorderForeground(f.teinte()).
		BorderBackground(panelBg).
		Background(panelBg).
		Width(w)

	return clip(f.bordure("╭", "╮", titre, droite), w+2) + "\n" +
		cotes.Render(strings.Join(lignes, "\n")) + "\n" +
		clip(f.bordure("╰", "╯", pied, ""), w+2)
}

func (f Frame) teinte() lipgloss.TerminalColor {
	if f.Focused {
		return accent
	}
	return accentBas
}

// bordure compose une bordure horizontale qui porte du texte : `╭─ gauche ─── droite ─╮`.
// Les textes arrivent déjà stylés ; c'est à l'appelant de les avoir fait tenir.
func (f Frame) bordure(coinG, coinD, gauche, droite string) string {
	trait := base.Foreground(f.teinte())
	g, d := trait.Render(coinG+"─"), trait.Render("─"+coinD)
	if gauche != "" {
		g += pad(1) + gauche + pad(1)
	}
	if droite != "" {
		d = pad(1) + droite + pad(1) + d
	}
	reste := f.boxWidth() + 2 - lipgloss.Width(g) - lipgloss.Width(d)
	return g + trait.Render(strings.Repeat("─", max(reste, 0))) + d
}

// compteur dit où l'on est et combien il y en a : « 3/18 ». C'est la seule trace des
// candidats que la fenêtre ne montre pas — 271 paquets installés, 8 lignes visibles.
func (f Frame) compteur() string {
	total := estompe(hintStyle).Render("/" + strconv.Itoa(len(f.Items)))
	if f.Sel < 0 || f.Sel >= len(f.Items) {
		return estompe(hintStyle).Render(strconv.Itoa(len(f.Items)))
	}
	return nameStyle.Render(strconv.Itoa(f.Sel+1)) + total
}

// estompe rend un style secondaire. En 24 bits et en 256 couleurs, la teinte muted y
// suffit. En 16 couleurs, il n'y a pas d'index « secondaire » fiable : le 8 tombe sous
// 1,4:1 dans Catppuccin et se confond avec le fond dans Solarized. On demande alors au
// terminal d'atténuer sa propre couleur de texte, ce qui marche sur thème clair comme
// sombre.
func estompe(s lipgloss.Style) lipgloss.Style {
	if lipgloss.ColorProfile() == termenv.ANSI {
		return s.Faint(true)
	}
	return s
}

// ScrollOffset place la fenêtre de défilement pour que `sel` soit visible, sans mémoire
// d'un appel à l'autre : c'est ce dont a besoin `jigger render`, où le seul état conservé
// entre deux frappes est l'index sélectionné, côté zsh.
func ScrollOffset(sel, count, rows int) int {
	if rows <= 0 {
		rows = visibleRows
	}
	if sel < rows || count <= rows {
		return 0
	}
	return min(sel-rows+1, count-rows)
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

func (f Frame) renderRow(it complete.Item, selected bool, avecPM bool) string {
	glyph := glyphe(it.Badge)

	// Partie droite (alignée au bord) : version installée, point « installé », puis PM
	// (façade seulement, cf. avecPM). On la compose d'abord en texte nu pour calculer le
	// remplissage.
	// Le contexte cède avant le nom, et jamais l'inverse. Depuis l'ADR-0009 cette colonne
	// n'est plus une version de gestionnaire natif — toujours courte — mais ce qu'un
	// DESCRIPTEUR TIERS a bien voulu y mettre : la date d'une branche, le sujet d'un
	// commit, ce qu'on veut. Sans borne, une chaîne plus large que le cadre chassait le
	// nom entièrement (#156) — or le nom est ce qui sera inséré ; le contexte n'est
	// qu'une aide à choisir. La moitié de la ligne lui suffit, et `tronquer` ne touche
	// rien qui tienne déjà.
	version := tronquer(it.Version, f.rowWidth()/2)

	rightPlain := ""
	if version != "" {
		rightPlain = version
	}
	if it.Installed {
		if rightPlain != "" {
			rightPlain += "  "
		}
		rightPlain += "●"
	}
	if avecPM {
		if rightPlain != "" {
			rightPlain += "  "
		}
		rightPlain += it.PM
	}

	// Ce qui reste au nom : la ligne moins l'indentation, le glyphe, ses deux espaces, la
	// partie droite et l'écart minimal. Le calculer d'après la partie droite *réelle* est
	// tout l'enjeu : un identifiant long suivi d'une version longue
	// (« ARP\Machine\X64\{226CEF88…  6.4.0.3079  ● ») débordait la ligne, le terminal la
	// repliait, et le popup occupait deux fois les lignes annoncées — de quoi décaler
	// tout l'affichage du shell.
	name := tronquer(it.Name, f.rowWidth()-5-lipgloss.Width(rightPlain)-1)

	// Géométrie commune aux deux états : 2 colonnes d'indentation à gauche, gouttière de
	// 2 colonnes à droite. On calcule le remplissage sur le texte nu, avant tout style.
	leftPlain := "  " + glyph + "  " + name
	gap := f.rowWidth() - lipgloss.Width(leftPlain) - lipgloss.Width(rightPlain)
	if gap < 1 {
		gap = 1
	}

	// La ligne courante est une bande d'un bord à l'autre, marquée d'un « ▌ » à la place du
	// premier espace : la géométrie ne bouge pas, et le marqueur désigne la ligne même là où
	// aucune couleur ne passe (`--color never`). Elle est composée segment par segment comme
	// les autres, sur le fond de la bande : le glyphe y garde sa couleur, alors que c'est
	// justement sur cette ligne que le type compte — ⇥ sur un cask insère `--cask nom`.
	// Au repos, bande ardoise et marqueur éteint ; au focus, bande teal, marqueur vif et nom
	// en gras : c'est la convention des listes du système, sélection grisée tant que le
	// contrôle n'a pas le focus, colorée dès qu'il l'a.
	fond, nom, ver := base, nameStyle, estompe(verStylePkg)
	marque := pad(1)
	if selected {
		fond = lipgloss.NewStyle().Background(sepCl)
		nom, ver = fond.Foreground(fg), fond.Foreground(fg)
		marque = fond.Foreground(muted).Render("▌")
		if f.Focused {
			fond = lipgloss.NewStyle().Background(bandeVive)
			nom, ver = fond.Foreground(ink).Bold(true), fond.Foreground(fg)
			marque = fond.Foreground(accent).Render("▌")
		}
	}
	sur := func(s lipgloss.Style) lipgloss.Style { return s.Background(fond.GetBackground()) }
	esp := func(n int) string { return fond.Render(strings.Repeat(" ", max(n, 0))) }

	left := marque + esp(1) + sur(couleur(it.Badge)).Render(glyph) + esp(2) + nom.Render(name)
	right := ""
	if version != "" {
		right = ver.Render(version)
	}
	if it.Installed {
		if version != "" {
			right += esp(2)
		}
		right += sur(dotStyle).Render("●")
	}
	if avecPM {
		if right != "" {
			right += esp(2)
		}
		// Même identité visuelle que le glyphe de la ligne : c'est la seule palette de
		// badges du popup, partagée avec les tableaux de sortie et le bloc oh-my-posh
		// (spec §5).
		right += sur(couleur(it.Badge)).Render(it.PM)
	}
	return left + esp(gap) + right + esp(2)
}

// Chaque gestionnaire range ses paquets en deux classes (cf. pm) : la classe ordinaire
// — formula, paquet du catalogue winget, bucket main — porte le losange ambré, l'autre
// le carré violet. Une sous-commande ou une option, qui n'appartient à aucune, garde la
// puce discrète.
func glyphe(badge string) string {
	switch badge {
	case pm.BadgeFormula, pm.BadgeWinget, pm.BadgeScoop, pm.BadgeRepo:
		return "◆"
	case pm.BadgeCask, pm.BadgeOther, pm.BadgeAUR:
		return "▣"
	}
	return "•"
}

func couleur(badge string) lipgloss.Style {
	switch badge {
	case pm.BadgeFormula, pm.BadgeWinget, pm.BadgeScoop, pm.BadgeRepo:
		return formulaStyle
	case pm.BadgeCask, pm.BadgeOther, pm.BadgeAUR:
		return caskStyle
	}
	return bulletStyle
}

// footer rend les rappels de touches qui tiennent dans `largeur`, par éléments entiers :
// une coupe au milieu laissait un « ^ » orphelin à 48 colonnes, et c'était la dernière
// touche — la sortie — qui partait la première. Trois tenues, de la plus bavarde à la plus
// sèche ; la première qui tient gagne. La dernière touche garde son libellé le plus
// longtemps : savoir fermer est ce qu'on doit pouvoir lire en dernier.
func (f Frame) footer(largeur int) string {
	tenues := []func(i int) bool{
		func(int) bool { return true },
		func(i int) bool { return i == len(f.Keys)-1 },
		func(int) bool { return false },
	}
	pied := ""
	for _, libelle := range tenues {
		parts := make([]string, 0, len(f.Keys))
		for i, k := range f.Keys {
			p := keyStyle.Render(k.Key)
			if libelle(i) {
				p += estompe(hintStyle).Render(" " + k.Label)
			}
			parts = append(parts, p)
		}
		if pied = strings.Join(parts, pad(2)); lipgloss.Width(pied) <= largeur {
			break
		}
	}
	return clip(pied, max(largeur, 0))
}
