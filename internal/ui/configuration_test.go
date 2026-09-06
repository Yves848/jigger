package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
)

// `visible()`, qui retire les séquences de style, vient de picker_test.go : les assertions
// de ce fichier portent, comme les siennes, sur ce que l'œil verrait.

// configDeTest rend un écran d'une seule ligne modifiable, suffisant pour éprouver les
// touches : ce qui est testé ici est la sortie, pas la mise en page.
func configDeTest() Configuration {
	return NouvelleConfiguration([]GroupeConfig{{
		Titre: "Réglages",
		Lignes: []LigneConfig{
			{Cle: "rows", Env: "JIGGER_ROWS", Valeur: "8", Provenance: "défaut"},
		},
	}})
}

func configTouche(c Configuration, msg tea.KeyMsg) Configuration {
	m, _ := c.Update(msg)
	return m.(Configuration)
}

var (
	cfgEchap  = tea.KeyMsg{Type: tea.KeyEsc}
	cfgCtrlC  = tea.KeyMsg{Type: tea.KeyCtrlC}
	cfgCtrlG  = tea.KeyMsg{Type: tea.KeyCtrlG}
	cfgLettre = func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
)

// Les trois touches d'abandon lèvent le drapeau, « q » ne le lève pas. C'est toute la
// distinction que l'appelant exploite pour n'écrire qu'à bon escient.
func TestQuitterSansEnregistrer(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		msg     tea.KeyMsg
		abandon bool
	}{
		{"esc abandonne", cfgEchap, true},
		{"ctrl+c abandonne", cfgCtrlC, true},
		{"ctrl+g abandonne", cfgCtrlG, true},
		{"q enregistre", cfgLettre('q'), false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			c := configTouche(configDeTest(), cas.msg)
			if c.Abandon != cas.abandon {
				t.Errorf("Abandon = %v, attendu %v", c.Abandon, cas.abandon)
			}
		})
	}
}

// L'écran ne vide PAS ses modifications en abandonnant : c'est l'appelant qui décide de ne
// pas les écrire. Sans cette garantie, un appelant qui oublierait de lire Abandon
// passerait inaperçu — il écrirait un ensemble vide au lieu d'écrire à tort, et le défaut
// ne se verrait qu'à l'usage.
func TestAbandonNeVidePasLesModifications(t *testing.T) {
	c := configDeTest()
	c.Modifs["rows"] = "12"

	c = configTouche(c, cfgEchap)

	if !c.Abandon {
		t.Fatal("esc devait lever Abandon")
	}
	if c.Modifs["rows"] != "12" {
		t.Errorf("les modifications ont été vidées par l'écran : %v", c.Modifs)
	}
}

// Le piège de cette modification : esc a DEUX sens. En édition il annule la saisie et rend
// la main à la navigation — il ne doit surtout pas quitter l'écran.
func TestEnEditionEchapAnnuleSansQuitter(t *testing.T) {
	c := configTouche(configDeTest(), tea.KeyMsg{Type: tea.KeyEnter}) // entre en édition
	if !c.edition {
		t.Fatal("↵ devait entrer en édition")
	}

	c = configTouche(c, cfgEchap)

	if c.edition {
		t.Error("esc devait sortir de l'édition")
	}
	if c.Abandon {
		t.Error("esc en édition a quitté l'écran : il ne devait qu'annuler la saisie")
	}
}

// Une touche qui jette du travail ne se devine pas : le pied doit l'annoncer. Comparé au
// catalogue et non à une chaîne en dur — c'est la concordance qui est l'exigence, pas la
// valeur, et le pied change de langue avec le reste.
func TestLePiedAnnonceLaSortieSansEnregistrer(t *testing.T) {
	c := configDeTest()

	pied := c.pied()
	if !strings.Contains(pied, i18n.T("cfg.quit_discard")) {
		t.Errorf("le pied de navigation n'annonce pas l'abandon : %q", pied)
	}
	if !strings.Contains(pied, i18n.T("cfg.quit_save")) {
		t.Errorf("le pied de navigation n'annonce plus l'enregistrement : %q", pied)
	}

	// En édition, esc annule la saisie et ne quitte pas : y annoncer « abandonner »
	// désignerait la même touche pour deux gestes différents.
	c = configTouche(c, tea.KeyMsg{Type: tea.KeyEnter})
	if pied := c.pied(); strings.Contains(pied, i18n.T("cfg.quit_discard")) {
		t.Errorf("le pied d'édition annonce l'abandon de l'écran : %q", pied)
	}
}

// ecranRiche : un écran qui mélange les cas — un booléen, un texte, une ligne figée, et des
// descriptions de longueurs très différentes. C'est cette disparité qui produisait les bords
// en dents de scie.
func ecranRiche() Configuration {
	return NouvelleConfiguration([]GroupeConfig{{
		Titre: "Réglages",
		Note:  "prend effet tout de suite",
		Lignes: []LigneConfig{
			{Cle: "pager", Env: "JIGGER_PAGER", Valeur: "1", Provenance: "défaut",
				Description: "vue paginée", Type: LigneBooleen, ParDefaut: true},
			{Cle: "rows", Env: "JIGGER_ROWS", Valeur: "8", Provenance: "fichier",
				Description: "une description nettement plus longue que les autres, pour voir",
				Type:        LigneEntier},
			{Env: "HOMEBREW_PREFIX", Valeur: "—", Provenance: "env", Fige: true},
		},
	}})
}

// Le défaut visuel corrigé : les styles de cet écran portent un fond, donc une ligne non
// calée s'arrête où finit son texte et le bloc a des bords en dents de scie.
func TestToutesLesLignesOntLaMemeLargeur(t *testing.T) {
	vue := ecranRiche().View()

	var largeurs = map[int][]string{}
	for _, l := range strings.Split(vue, "\n") {
		if strings.TrimSpace(visible(l)) == "" {
			continue // lignes vides de séparation, et le pied qui a sa propre largeur
		}
		largeurs[lipgloss.Width(l)] = append(largeurs[lipgloss.Width(l)], visible(l))
	}
	// Le pied a sa propre largeur : on tolère deux valeurs au plus, pas davantage.
	if len(largeurs) > 2 {
		for l, exemples := range largeurs {
			t.Errorf("largeur %d : %q", l, exemples[0])
		}
		t.Fatalf("%d largeurs différentes : le bloc a des bords en dents de scie", len(largeurs))
	}
}

// Un booléen se montre coché. Le chiffre ne disait pas qu'on pouvait le basculer.
func TestUnBooleenSAfficheCoche(t *testing.T) {
	vue := visible(ecranRiche().View())
	if !strings.Contains(vue, "[✓]") {
		t.Errorf("un booléen à 1 devait s'afficher coché :\n%s", vue)
	}
	if strings.Contains(vue, "JIGGER_PAGER            1 ") {
		t.Error("le booléen s'affiche encore en chiffre")
	}
}

// Espace bascule un booléen sans entrer en édition, et ne touche pas à une ligne qui n'en
// est pas un — promettre une bascule là où il n'y en a pas serait pire que rien.
func TestEspaceBasculeUnBooleenEtLuiSeul(t *testing.T) {
	espace := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}

	c := configTouche(ecranRiche(), espace) // curseur sur pager, booléen à 1
	if c.edition {
		t.Error("la bascule ne doit pas entrer en édition")
	}
	if c.Modifs["pager"] != "0" {
		t.Errorf("pager = %q, attendu \"0\"", c.Modifs["pager"])
	}
	if c = configTouche(c, espace); c.Modifs["pager"] != "1" {
		t.Errorf("seconde bascule : pager = %q, attendu \"1\"", c.Modifs["pager"])
	}

	// Ligne suivante : un entier, que l'espace ne doit pas toucher.
	c = configTouche(c, tea.KeyMsg{Type: tea.KeyDown})
	c = configTouche(c, espace)
	if _, touche := c.Modifs["rows"]; touche {
		t.Error("l'espace a modifié une ligne qui n'est pas un booléen")
	}
}
