package main

import (
	"strings"
	"testing"
)

func etat(tete string, tags map[string]string) Etat {
	return Etat{Tete: tete, Tags: tags}
}

func TestDeuxDepotsIdentiquesNeDonnentAucunEcart(t *testing.T) {
	a := etat("abc1234", map[string]string{"v0.12.0": "aaa", "v0.11.0": "bbb"})
	b := etat("abc1234", map[string]string{"v0.11.0": "bbb", "v0.12.0": "aaa"})

	if ecarts := Comparer(a, b); len(ecarts) != 0 {
		t.Fatalf("aucun écart attendu, obtenu %v", ecarts)
	}
}

func TestTeteDivergenteEstSignalee(t *testing.T) {
	a := etat("neuf", nil)
	b := etat("vieux", nil)

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "main" || ecarts[0].GitLab != "neuf" || ecarts[0].GitHub != "vieux" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

// Le cas qui a réellement mordu : le miroir figé garde une tête ancienne et n'a jamais reçu
// les tags des versions publiées depuis.
func TestTagAbsentDuMiroirEstSignale(t *testing.T) {
	a := etat("meme", map[string]string{"v0.11.0": "bbb", "v0.12.0": "aaa"})
	b := etat("meme", map[string]string{"v0.11.0": "bbb"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "tag v0.12.0" || ecarts[0].GitHub != "" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

// Un tag que seul le miroir porte compte aussi : le miroir est en mode push sans
// « keep divergent refs », donc GitHub ne devrait jamais rien avoir en propre.
func TestTagPresentSeulementSurLeMiroirEstSignale(t *testing.T) {
	a := etat("meme", nil)
	b := etat("meme", map[string]string{"v9.9.9": "zzz"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "tag v9.9.9" || ecarts[0].GitLab != "" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

func TestTagDeplaceEstSignale(t *testing.T) {
	a := etat("meme", map[string]string{"v0.12.0": "refait"})
	b := etat("meme", map[string]string{"v0.12.0": "ancien"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 || ecarts[0].Quoi != "tag v0.12.0" {
		t.Fatalf("un écart de tag attendu, obtenu %v", ecarts)
	}
}

// L'ordre est stable : le message d'une issue rouverte trois jours plus tard doit pouvoir
// se comparer à l'œil au précédent.
func TestLOrdreEstStable(t *testing.T) {
	a := etat("neuf", map[string]string{"v0.12.0": "a", "v0.10.0": "c", "v0.11.0": "b"})
	b := etat("vieux", nil)

	ecarts := Comparer(a, b)
	attendu := []string{"main", "tag v0.10.0", "tag v0.11.0", "tag v0.12.0"}
	if len(ecarts) != len(attendu) {
		t.Fatalf("%d écarts attendus, obtenu %v", len(attendu), ecarts)
	}
	for i, quoi := range attendu {
		if ecarts[i].Quoi != quoi {
			t.Fatalf("écart %d : %q attendu, obtenu %q", i, quoi, ecarts[i].Quoi)
		}
	}
}

func TestResumeDitCeQuiManqueEtOuIlManque(t *testing.T) {
	ecarts := []Ecart{
		{Quoi: "main", GitLab: "fadaf2d", GitHub: "7974687"},
		{Quoi: "tag v0.12.0", GitLab: "2c32bd1", GitHub: ""},
	}

	r := Resume(ecarts)
	for _, attendu := range []string{"main", "fadaf2d", "7974687", "tag v0.12.0", "absent"} {
		if !strings.Contains(r, attendu) {
			t.Fatalf("le résumé ne mentionne pas %q :\n%s", attendu, r)
		}
	}
}
