package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	// Alias : le paquet de tests du même dossier a déjà un `ansi` à lui.
	xansi "github.com/charmbracelet/x/ansi"

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

// avecPM dit si la colonne PM doit apparaître : présente dès qu'un item du cadre en
// porte un, absente sinon. C'est la façade seule qui remplit ce champ (cf. pm.Item.PM) —
// le chemin natif (`brew install ⇥`) n'a rien à désambiguïser et n'en porte jamais.
func (f Frame) avecPM() bool {
	for _, it := range f.Items {
		if it.PM != "" {
			return true
		}
	}
	return false
}

func (f Frame) rows() int {
	if f.Rows <= 0 {
		return visibleRows
	}
	return f.Rows
}

// Render dessine le popup complet (cadre compris).
func (f Frame) Render() string {
	var b strings.Builder

	// Version poussée à droite (repère du binaire lancé), sur la même ligne que le titre.
	// Elle passe à la trappe si le titre occupe déjà toute la largeur : c'est un repère,
	// pas une information.
	head := promptStyle.Render("❯") + pad(1) + titleStyle.Render(tronquer(f.Title, f.rowWidth()))
	if Version != "" {
		ver := verStyle.Render("jigger " + Version)
		if gap := f.boxWidth() - lipgloss.Width(head) - lipgloss.Width(ver); gap >= 1 {
			head += pad(gap) + ver
		}
	}
	b.WriteString(clip(head, f.boxWidth()) + "\n")
	if f.FilterView != "" {
		b.WriteString(clip(f.FilterView, f.boxWidth()) + "\n")
	}

	box := boxStyle.Width(f.boxWidth())
	if len(f.Items) == 0 {
		b.WriteString(clip(pad(2)+emptyStyle.Render(f.Empty), f.boxWidth()))
		return box.Render(b.String())
	}

	// Lignes jointives : la ligne courante se signale par sa couleur et son soulignement,
	// il n'y a plus d'interligne à intercaler. Toutes les lignes faisant une ligne de haut,
	// la hauteur du popup ne dépend pas de la position du curseur.
	// La colonne PM ne s'ajoute que si au moins un item la porte — comme les tableaux de
	// sortie (cf. facade.Formater) : une colonne toujours vide n'apprend rien, et
	// `brew install ⇥`, où aucun item n'a de PM, doit rendre au pixel près comme avant
	// cette colonne.
	avecPM := f.avecPM()

	offset := min(max(f.Offset, 0), len(f.Items)-1)
	end := min(offset+f.rows(), len(f.Items))
	for i := offset; i < end; i++ {
		b.WriteString(clip(f.renderRow(f.Items[i], i == f.Sel, avecPM), f.boxWidth()) + "\n")
	}
	b.WriteString("\n") // respiration avant le pied
	b.WriteString(clip(f.footer(), f.boxWidth()))
	return box.Render(b.String())
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

	// Ligne courante : rendue d'un seul tenant en texte nu, sans les couleurs par segment
	// (leurs séquences de reset interrompraient le soulignement au milieu de la ligne).
	if selected {
		style := selIdleStyle
		if f.Focused {
			style = selStyle
		}
		return style.Width(f.rowWidth()).Render(leftPlain + strings.Repeat(" ", gap) + rightPlain)
	}

	left := pad(2) + couleur(it.Badge).Render(glyph) + pad(2) + nameStyle.Render(name)
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
	if avecPM {
		if right != "" {
			right += pad(2)
		}
		// Même identité visuelle que le glyphe de la ligne : c'est la seule palette de
		// badges du popup, partagée avec les tableaux de sortie et le bloc oh-my-posh
		// (spec §5).
		right += couleur(it.Badge).Render(it.PM)
	}
	return left + pad(gap) + right
}

// Chaque gestionnaire range ses paquets en deux classes (cf. pm) : la classe ordinaire
// — formula, paquet du catalogue winget, bucket main — porte le losange ambré, l'autre
// le carré violet. Une sous-commande ou une option, qui n'appartient à aucune, garde la
// puce discrète.
func glyphe(badge string) string {
	switch badge {
	case pm.BadgeFormula, pm.BadgeWinget, pm.BadgeScoop:
		return "◆"
	case pm.BadgeCask, pm.BadgeOther:
		return "▣"
	}
	return "•"
}

func couleur(badge string) lipgloss.Style {
	switch badge {
	case pm.BadgeFormula, pm.BadgeWinget, pm.BadgeScoop:
		return formulaStyle
	case pm.BadgeCask, pm.BadgeOther:
		return caskStyle
	}
	return bulletStyle
}

func (f Frame) footer() string {
	parts := make([]string, 0, len(f.Keys))
	for _, k := range f.Keys {
		parts = append(parts, pillStyle.Render(k.Key)+hintStyle.Render(" "+k.Label))
	}
	return pad(2) + strings.Join(parts, pad(2))
}
