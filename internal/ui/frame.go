package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

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

func (f Frame) rowWidth() int  { return f.boxWidth() - 2 }
func (f Frame) nameWidth() int { return f.boxWidth() - 12 }

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
	head := promptStyle.Render("❯") + pad(1) + titleStyle.Render(f.Title)
	if Version != "" {
		ver := verStyle.Render("jigger " + Version)
		gap := f.boxWidth() - lipgloss.Width(head) - lipgloss.Width(ver)
		if gap < 1 {
			gap = 1
		}
		head += pad(gap) + ver
	}
	b.WriteString(head + "\n")
	if f.FilterView != "" {
		b.WriteString(f.FilterView + "\n")
	}

	box := boxStyle.Width(f.boxWidth())
	if len(f.Items) == 0 {
		b.WriteString(pad(2) + emptyStyle.Render(f.Empty))
		return box.Render(b.String())
	}

	// Lignes jointives : la ligne courante se signale par sa couleur et son soulignement,
	// il n'y a plus d'interligne à intercaler. Toutes les lignes faisant une ligne de haut,
	// la hauteur du popup ne dépend pas de la position du curseur.
	offset := min(max(f.Offset, 0), len(f.Items)-1)
	end := min(offset+f.rows(), len(f.Items))
	for i := offset; i < end; i++ {
		b.WriteString(f.renderRow(f.Items[i], i == f.Sel) + "\n")
	}
	b.WriteString("\n") // respiration avant le pied
	b.WriteString(f.footer())
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

func (f Frame) renderRow(it complete.Item, selected bool) string {
	name := it.Name
	if nameMax := f.nameWidth(); len(name) > nameMax {
		name = name[:nameMax-1] + "…"
	}

	glyph := glyphe(it.Badge)

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
	gap := f.rowWidth() - lipgloss.Width(leftPlain) - lipgloss.Width(rightPlain)
	if gap < 1 {
		gap = 1
	}

	// Ligne courante : rendue d'un seul tenant en texte nu, sans les couleurs par segment
	// (leurs séquences de reset interrompraient le soulignement au milieu de la ligne).
	if selected {
		return selStyle.Width(f.rowWidth()).Render(leftPlain + strings.Repeat(" ", gap) + rightPlain)
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
