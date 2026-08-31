package ui

import (
	"regexp"
	"strings"
)

// Ligne est ce que la liste sait manipuler : une clé sur laquelle filtrer, et des
// cellules à afficher. Un candidat de complétion en a une seule ; un paquet en a
// jusqu'à cinq. La liste ignore tout du reste — badge, gestionnaire, style : c'est
// l'affaire des rendus, qui restent distincts (spec §1).
type Ligne interface {
	Cle() string
	Cellules() []string
}

// ModeFiltre dit comment la saisie est interprétée. Le mode est toujours affiché : on
// ne devine jamais dans lequel on se trouve.
type ModeFiltre int

const (
	// FiltreSousChaine est le mode par défaut, celui que `jigger pick` a toujours eu.
	FiltreSousChaine ModeFiltre = iota
	// FiltreRegex interprète la saisie comme une expression rationnelle.
	FiltreRegex
)

// Liste porte les données, le filtre, le curseur, le défilement et la sélection —
// tout ce qui se comporte, rien de ce qui se dessine. Le sélecteur et la vue tabulaire
// s'appuient dessus pour naviguer de la même façon.
type Liste struct {
	toutes   []Ligne
	filtrees []Ligne
	cochees  map[string]bool

	motif  string
	mode   ModeFiltre
	regex  *regexp.Regexp // nil si le motif ne compile pas, ou en mode sous-chaîne
	valide bool           // le motif courant compile-t-il ? (toujours vrai hors regex)

	curseur int
	offset  int
	hauteur int // lignes visibles à la fois
}

// NouvelleListe crée une liste sur les lignes données. `hauteur` est le nombre de lignes
// visibles simultanément ; une hauteur nulle ou négative est ramenée à 1, sans quoi le
// défilement n'aurait plus de sens.
func NouvelleListe(lignes []Ligne, hauteur int) *Liste {
	if hauteur < 1 {
		hauteur = 1
	}
	l := &Liste{toutes: lignes, cochees: map[string]bool{}, hauteur: hauteur, valide: true}
	l.appliquer()
	return l
}

// Filtrer pose un nouveau motif. En mode regex, un motif qui ne compile pas ne filtre
// rien — toutes les lignes restent visibles et Valide() dit non, pour que le rendu le
// signale plutôt que de laisser croire à une liste vide.
func (l *Liste) Filtrer(motif string) {
	l.motif = motif
	l.appliquer()
}

// BasculerMode passe de la sous-chaîne à la regex et retour, en conservant la saisie.
func (l *Liste) BasculerMode() {
	if l.mode == FiltreSousChaine {
		l.mode = FiltreRegex
	} else {
		l.mode = FiltreSousChaine
	}
	l.appliquer()
}

func (l *Liste) appliquer() {
	q := strings.TrimSpace(l.motif)
	l.regex, l.valide = nil, true

	if q != "" && l.mode == FiltreRegex {
		// (?i) : la casse est ignorée, comme en mode sous-chaîne — basculer de mode ne
		// doit pas changer discrètement la sensibilité à la casse.
		re, err := regexp.Compile("(?i)" + q)
		if err != nil {
			l.valide = false
		} else {
			l.regex = re
		}
	}

	// Toujours reconstruire dans un nouveau tableau : réutiliser le backing de `toutes`
	// le réécrirait au fil des frappes (le défaut que picker.go documente déjà).
	filtrees := make([]Ligne, 0, len(l.toutes))
	switch {
	case q == "" || !l.valide:
		filtrees = append(filtrees, l.toutes...)
	case l.regex != nil:
		for _, li := range l.toutes {
			if l.regex.MatchString(li.Cle()) {
				filtrees = append(filtrees, li)
			}
		}
	default:
		bas := strings.ToLower(q)
		for _, li := range l.toutes {
			if strings.Contains(strings.ToLower(li.Cle()), bas) {
				filtrees = append(filtrees, li)
			}
		}
	}
	l.filtrees = filtrees

	if l.curseur >= len(l.filtrees) {
		l.curseur = max(0, len(l.filtrees)-1)
	}
	l.borner()
}

func (l *Liste) borner() {
	if l.curseur < l.offset {
		l.offset = l.curseur
	}
	if l.curseur >= l.offset+l.hauteur {
		l.offset = l.curseur - l.hauteur + 1
	}
	l.offset = max(l.offset, 0)
}

// Haut et Bas déplacent d'une ligne, sans jamais sortir de la liste filtrée.
func (l *Liste) Haut() {
	if l.curseur > 0 {
		l.curseur--
		l.borner()
	}
}

func (l *Liste) Bas() {
	if l.curseur < len(l.filtrees)-1 {
		l.curseur++
		l.borner()
	}
}

// PageHaut et PageBas déplacent d'un écran, en s'arrêtant aux extrémités.
func (l *Liste) PageHaut() {
	l.curseur = max(0, l.curseur-l.hauteur)
	l.borner()
}

func (l *Liste) PageBas() {
	l.curseur = min(len(l.filtrees)-1, l.curseur+l.hauteur)
	if l.curseur < 0 {
		l.curseur = 0
	}
	l.borner()
}

// Cocher bascule la sélection de la ligne courante. La marque suit la **clé**, pas
// l'indice : une ligne cochée puis masquée par le filtre reste cochée, et se retrouve
// dans le résultat final. C'est le comportement attendu quand on affine sa recherche
// entre deux sélections.
func (l *Liste) Cocher() {
	li := l.Courante()
	if li == nil {
		return
	}
	c := li.Cle()
	if l.cochees[c] {
		delete(l.cochees, c)
	} else {
		l.cochees[c] = true
	}
}

// CocherTout bascule la sélection de **toutes les lignes visibles**, c'est-à-dire de la
// liste filtrée et non du catalogue entier. C'est le geste utile : on filtre, puis on
// prend tout ce qui reste.
//
// La bascule suit la règle du « tout ou rien » : si chaque ligne visible est déjà cochée,
// on les décoche ; sinon on coche celles qui manquent. Une sélection partielle se complète
// donc d'un geste au lieu de s'inverser, ce qui serait imprévisible.
func (l *Liste) CocherTout() {
	if len(l.filtrees) == 0 {
		return
	}
	toutes := true
	for _, li := range l.filtrees {
		if !l.cochees[li.Cle()] {
			toutes = false
			break
		}
	}
	for _, li := range l.filtrees {
		if toutes {
			delete(l.cochees, li.Cle())
		} else {
			l.cochees[li.Cle()] = true
		}
	}
}

// EstCochee dit si la ligne d'indice donné (dans la liste filtrée) est sélectionnée.
func (l *Liste) EstCochee(i int) bool {
	if i < 0 || i >= len(l.filtrees) {
		return false
	}
	return l.cochees[l.filtrees[i].Cle()]
}

// Choix rend les lignes retenues : les cochées si au moins une l'est, sinon la ligne
// courante. Rien ne se perd et rien ne se devine — c'est ce que le pied du cadre annonce.
func (l *Liste) Choix() []Ligne {
	if len(l.cochees) == 0 {
		if li := l.Courante(); li != nil {
			return []Ligne{li}
		}
		return nil
	}
	// Parcourir `toutes` et non `filtrees` : une ligne cochée puis masquée compte.
	var out []Ligne
	for _, li := range l.toutes {
		if l.cochees[li.Cle()] {
			out = append(out, li)
		}
	}
	return out
}

// Courante rend la ligne sous le curseur, ou nil si la liste filtrée est vide.
func (l *Liste) Courante() Ligne {
	if l.curseur < 0 || l.curseur >= len(l.filtrees) {
		return nil
	}
	return l.filtrees[l.curseur]
}

// Visibles rend la tranche affichable et l'indice du curseur dans cette tranche.
func (l *Liste) Visibles() ([]Ligne, int) {
	if len(l.filtrees) == 0 {
		return nil, 0
	}
	debut := min(max(l.offset, 0), len(l.filtrees)-1)
	fin := min(debut+l.hauteur, len(l.filtrees))
	return l.filtrees[debut:fin], l.curseur - debut
}

// Accesseurs de lecture, pour les rendus et les tests.
func (l *Liste) Filtrees() []Ligne { return l.filtrees }
func (l *Liste) Curseur() int      { return l.curseur }
func (l *Liste) Offset() int       { return l.offset }
func (l *Liste) Mode() ModeFiltre  { return l.mode }
func (l *Liste) Valide() bool      { return l.valide }
func (l *Liste) NbCochees() int    { return len(l.cochees) }
func (l *Liste) Total() int        { return len(l.toutes) }
func (l *Liste) Hauteur() int      { return l.hauteur }
func (l *Liste) DefinirHauteur(h int) {
	if h < 1 {
		h = 1
	}
	l.hauteur = h
	l.borner()
}
