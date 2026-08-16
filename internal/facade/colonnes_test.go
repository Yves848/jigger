package facade

import (
	"reflect"
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Le test croisé de la spec §4 : l'en-tête de la table brute et celui que reçoit la vue
// paginée viennent de la même fonction. Sans lui, les deux divergeront le jour où l'une
// des deux évoluera — et personne ne s'en apercevra avant un utilisateur.
func TestColonnesEtTableBruteSAccordent(t *testing.T) {
	rows := []pm.Package{
		{Name: "fd", Version: "10.2.0", Available: "10.3.0", PM: "brew"},
		{Name: "ripgrep", Version: "14.1.1", PM: "brew"},
	}

	entete, cellules := Colonnes(rows)
	brute := Formater(pm.Verb("outdated"), rows, false)

	premiere := strings.SplitN(brute, "\n", 2)[0]
	for _, col := range entete {
		if !strings.Contains(premiere, col) {
			t.Errorf("la colonne %q est absente de l'en-tête de la table brute : %q", col, premiere)
		}
	}
	if len(cellules) != len(rows) {
		t.Fatalf("%d lignes de cellules pour %d paquets", len(cellules), len(rows))
	}
	for i, c := range cellules {
		if len(c) != len(entete) {
			t.Errorf("ligne %d : %d cellules pour %d colonnes", i, len(c), len(entete))
		}
	}
}

// La règle adaptative, éprouvée sur les deux cas qui la motivent : une colonne toujours
// vide n'apparaît pas.
func TestColonnesAdaptatives(t *testing.T) {
	// search : aucune version, aucune source, un seul gestionnaire → une seule colonne.
	entete, _ := Colonnes([]pm.Package{{Name: "fd", PM: "brew"}, {Name: "fdupes", PM: "brew"}})
	if len(entete) != 1 {
		t.Errorf("search : %v, attendu la seule colonne du nom", entete)
	}

	// outdated multi-gestionnaires : nom, actuel, dispo, PM.
	entete, _ = Colonnes([]pm.Package{
		{Name: "fd", Version: "1", Available: "2", PM: "brew"},
		{Name: "jq", Version: "1", Available: "2", PM: "scoop"},
	})
	if len(entete) != 4 {
		t.Errorf("outdated multi-PM : %v, attendu 4 colonnes", entete)
	}
}

// Une ligne sans cellule ne doit pas faire tomber le calcul des largeurs côté vue.
func TestColonnesSansLignes(t *testing.T) {
	entete, cellules := Colonnes(nil)
	if !reflect.DeepEqual(cellules, [][]string{}) {
		t.Errorf("cellules = %v, attendu vide", cellules)
	}
	if len(entete) != 1 {
		t.Errorf("entete = %v, attendu la seule colonne du nom", entete)
	}
}
