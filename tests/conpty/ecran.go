//go:build windows

package main

import (
	"strconv"
	"strings"
)

// Ecran est un terminal de poche : il interprète le flux du shell et rend l'écran tel
// qu'on le verrait. C'est indispensable pour juger un popup — le flux brut, lui, ne dit
// que ce qui a été écrit, jamais ce qui reste affiché ni où.
//
// Le sous-ensemble couvert est celui que PSReadLine et jigger emploient : positionnement
// du curseur, effacements de ligne et d'écran, mémorisation/restitution du curseur,
// défilement par saut de ligne en bas d'écran.
type Ecran struct {
	lignes     [][]rune
	l, c       int // curseur (0-indexé)
	memL, memC int // curseur mémorisé (DECSC)
	haut, larg int
}

func NewEcran(larg, haut int) *Ecran {
	e := &Ecran{haut: haut, larg: larg}
	e.lignes = make([][]rune, haut)
	for i := range e.lignes {
		e.lignes[i] = blanche(larg)
	}
	return e
}

func blanche(n int) []rune {
	l := make([]rune, n)
	for i := range l {
		l[i] = ' '
	}
	return l
}

// String rend l'écran, sans les espaces de fin.
func (e *Ecran) String() string {
	var b strings.Builder
	for _, l := range e.lignes {
		b.WriteString(strings.TrimRight(string(l), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// Curseur rend la position du curseur, en lignes/colonnes 1-indexées comme un terminal.
func (e *Ecran) Curseur() (ligne, colonne int) { return e.l + 1, e.c + 1 }

func (e *Ecran) Ecrire(flux string) {
	runes := []rune(flux)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == 0x1b && i+1 < len(runes):
			i += e.echappement(runes[i+1:])
		case r == '\r':
			e.c = 0
		case r == '\n':
			e.saut()
		case r == '\b':
			if e.c > 0 {
				e.c--
			}
		case r == 7: // BEL
		case r < 32:
		default:
			e.poser(r)
		}
	}
}

func (e *Ecran) poser(r rune) {
	if e.c >= e.larg {
		e.c = 0
		e.saut()
	}
	e.lignes[e.l][e.c] = r
	e.c++
}

// saut descend d'une ligne, en faisant défiler l'écran quand on est déjà en bas.
func (e *Ecran) saut() {
	if e.l+1 < e.haut {
		e.l++
		return
	}
	e.lignes = append(e.lignes[1:], blanche(e.larg))
}

// echappement traite une séquence commençant après l'ESC, et rend le nombre de runes
// consommées.
func (e *Ecran) echappement(s []rune) int {
	if len(s) == 0 {
		return 0
	}
	switch s[0] {
	case '7':
		e.memL, e.memC = e.l, e.c
		return 1
	case '8':
		e.l, e.c = e.memL, e.memC
		return 1
	case ']': // OSC … BEL|ST : titre de fenêtre, sans effet à l'écran
		for i := 1; i < len(s); i++ {
			if s[i] == 7 {
				return i
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 1
			}
		}
		return len(s) - 1
	case '[':
		return 1 + e.csi(s[1:])
	}
	return 1
}

func (e *Ecran) csi(s []rune) int {
	i := 0
	for i < len(s) && (s[i] < '@' || s[i] > '~') {
		i++
	}
	if i >= len(s) {
		return len(s)
	}
	params, final := string(s[:i]), s[i]
	prive := strings.HasPrefix(params, "?")
	params = strings.TrimPrefix(params, "?")
	n := func(defaut int) int {
		champ, _, _ := strings.Cut(params, ";")
		if v, err := strconv.Atoi(champ); err == nil {
			return v
		}
		return defaut
	}

	if prive {
		return i + 1 // modes privés (curseur visible, souris…) : sans effet ici
	}

	switch final {
	case 'H', 'f':
		ligne, colonne, _ := strings.Cut(params, ";")
		e.l = borne(atoi(ligne, 1)-1, 0, e.haut-1)
		e.c = borne(atoi(colonne, 1)-1, 0, e.larg-1)
	case 'A':
		e.l = borne(e.l-n(1), 0, e.haut-1)
	case 'B':
		e.l = borne(e.l+n(1), 0, e.haut-1)
	case 'C':
		e.c = borne(e.c+n(1), 0, e.larg-1)
	case 'D':
		e.c = borne(e.c-n(1), 0, e.larg-1)
	case 'G':
		e.c = borne(n(1)-1, 0, e.larg-1)
	case 'J': // effacement d'écran
		switch n(0) {
		case 0:
			e.effacerLigne(0)
			for l := e.l + 1; l < e.haut; l++ {
				e.lignes[l] = blanche(e.larg)
			}
		case 1:
			for l := 0; l < e.l; l++ {
				e.lignes[l] = blanche(e.larg)
			}
		case 2:
			for l := range e.lignes {
				e.lignes[l] = blanche(e.larg)
			}
		}
	case 'K': // effacement de ligne
		e.effacerLigne(n(0))
	case 'X': // ECH — efface n caractères sur place, sans bouger le curseur. ConPTY s'en
		// sert pour le remplissage : plutôt que d'écrire quarante espaces, il efface
		// quarante cases puis avance d'autant. Ne pas le traiter laisse traîner à
		// l'écran ce que le cadre précédent y avait mis.
		for k, c := 0, e.c; k < n(1) && c < e.larg; k, c = k+1, c+1 {
			e.lignes[e.l][c] = ' '
		}
	case 'd': // VPA — ligne absolue, colonne inchangée
		e.l = borne(n(1)-1, 0, e.haut-1)
	case 'P': // DCH — supprime n caractères, le reste de la ligne se décale
		nb := borne(n(1), 0, e.larg-e.c)
		copy(e.lignes[e.l][e.c:], e.lignes[e.l][e.c+nb:])
		for c := e.larg - nb; c < e.larg; c++ {
			e.lignes[e.l][c] = ' '
		}
	case '@': // ICH — insère n blancs, le reste de la ligne se décale
		nb := borne(n(1), 0, e.larg-e.c)
		copy(e.lignes[e.l][e.c+nb:], e.lignes[e.l][e.c:])
		for c := e.c; c < e.c+nb; c++ {
			e.lignes[e.l][c] = ' '
		}
	case 'S':
		for k := 0; k < n(1); k++ {
			e.lignes = append(e.lignes[1:], blanche(e.larg))
		}
	}
	return i + 1
}

func (e *Ecran) effacerLigne(mode int) {
	switch mode {
	case 0:
		for c := e.c; c < e.larg; c++ {
			e.lignes[e.l][c] = ' '
		}
	case 1:
		for c := 0; c <= e.c && c < e.larg; c++ {
			e.lignes[e.l][c] = ' '
		}
	case 2:
		e.lignes[e.l] = blanche(e.larg)
	}
}

func atoi(s string, defaut int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaut
}

func borne(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
