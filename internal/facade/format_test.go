package facade

import (
	"encoding/json"
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func lignes() []pm.Package {
	return []pm.Package{
		{Name: "fd", Version: "10.1.0", Available: "10.2.0", PM: "scoop"},
		{Name: "Git.Git", Version: "2.54.0", Available: "2.55.0", PM: "winget"},
	}
}

// La colonne PM n'apparaît que si plus d'un gestionnaire a contribué : sur macOS, où seul
// brew répond, elle serait du bruit.
func TestColonnePMSeulementSiPlusieurs(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()

	avec := Formater("outdated", lignes(), false)
	if !strings.Contains(avec, "PM") {
		t.Errorf("deux gestionnaires : la colonne PM est attendue\n%s", avec)
	}
	if !strings.Contains(avec, "scoop") || !strings.Contains(avec, "winget") {
		t.Errorf("les deux gestionnaires doivent apparaître\n%s", avec)
	}

	seul := Formater("outdated", lignes()[:1], false)
	if strings.Contains(seul, "PM") {
		t.Errorf("un seul gestionnaire : pas de colonne PM\n%s", seul)
	}
}

func TestColonnesAlignees(t *testing.T) {
	out := Formater("outdated", lignes(), false)
	var largeurs []int
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		largeurs = append(largeurs, strings.Index(l, strings.Fields(l)[1]))
	}
	for i := 1; i < len(largeurs); i++ {
		if largeurs[i] != largeurs[0] {
			t.Fatalf("colonnes désalignées :\n%s", out)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	out := Formater("outdated", lignes(), true)

	var relu []pm.Package
	if err := json.Unmarshal([]byte(out), &relu); err != nil {
		t.Fatalf("sortie JSON illisible : %v\n%s", err, out)
	}
	if len(relu) != 2 || relu[0].Name != "fd" {
		t.Fatalf("relu = %+v", relu)
	}
}

// `list` n'a pas de version disponible : la colonne DISPO n'a rien à faire là.
func TestListNAPasDeColonneDispo(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()

	rows := []pm.Package{{Name: "fd", Version: "10.2.0", PM: "scoop"}}
	out := Formater("list", rows, false)
	if strings.Contains(out, "DISPO") {
		t.Errorf("colonne DISPO inattendue pour list :\n%s", out)
	}
}

// jg source et jg search n'ont pas de version installée : ACTUEL n'a rien de plus à faire
// là que DISPO ou SOURCE quand personne ne le porte — la même règle adaptative doit
// s'appliquer aux quatre colonnes, pas à trois sur quatre. Avant la correction, ACTUEL
// était systématique : chaque ligne finissait par les espaces de remplissage de PAQUET,
// puisque la colonne ACTUEL restait là, vide, en bout de ligne.
func TestSourceNAPasDeColonneActuelNiEspacesResiduels(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()

	rows := []pm.Package{{Name: "homebrew/cask-fonts", PM: "brew"}}
	out := Formater("source", rows, false)
	if strings.Contains(out, "ACTUEL") {
		t.Errorf("colonne ACTUEL inattendue pour source (aucune ligne n'a de version) :\n%q", out)
	}
	for _, ligne := range strings.Split(out, "\n") {
		if ligne != strings.TrimRight(ligne, " ") {
			t.Errorf("ligne avec espaces résiduels en fin : %q", ligne)
		}
	}
}

func TestAucuneLigne(t *testing.T) {
	if out := Formater("outdated", nil, false); strings.TrimSpace(out) == "" {
		t.Error("une liste vide doit dire quelque chose, pas rien")
	}
	if out := Formater("outdated", nil, true); strings.TrimSpace(out) != "[]" {
		t.Errorf("JSON vide = %q, attendu « [] »", out)
	}
}
