package i18n

import "testing"

// Toute entrée du catalogue a ses deux langues. L'anglais est l'original : une entrée
// vide n'aurait rien sur quoi se replier. Le français, lui, peut techniquement l'être —
// le repli joue —, mais une chaîne française manquante est un oubli de traduction, pas un
// choix : elle sortirait en anglais au milieu d'une phrase française.
//
// Ce test balaie le catalogue au lieu de recopier des listes de clés : les trois listes
// qu'il remplace (popup, façade, cli) avaient déjà laissé filer les cinq clés ajoutées en
// cours de route — facade.note, knows_it, knows_it_as, manager_error, list_separator. Une
// boucle sur `catalogue` est exhaustive par construction : une clé nouvelle est couverte
// le jour où elle est écrite, sans qu'on ait à y penser.
//
// Les exceptions sont nommées plutôt que tues par une absence de liste : ces trois clés
// ont volontairement la même chaîne des deux côtés. cli.usage3 ne contient que des noms
// d'options (--refresh, --wait, --path, --all, --installed) ; table.source et table.pm
// sont deux en-têtes que le français écrit pareil.
func TestCatalogueComplet(t *testing.T) {
	memeChaineDansLesDeuxLangues := map[string]bool{
		"cli.usage3":   true,
		"table.source": true,
		"table.pm":     true,
	}

	for cle, trad := range catalogue {
		if trad[EN] == "" {
			t.Errorf("%s : l'anglais est vide", cle)
		}
		if trad[FR] == "" {
			t.Errorf("%s : le français est vide", cle)
		}
		if memeChaineDansLesDeuxLangues[cle] {
			if trad[EN] != trad[FR] {
				t.Errorf("%s : donnée pour identique dans les deux langues, mais %q ≠ %q", cle, trad[EN], trad[FR])
			}
			continue
		}
		if trad[EN] == trad[FR] {
			t.Errorf("%s : anglais et français identiques (%q) — traduction oubliée ?", cle, trad[EN])
		}
	}
}

// Les en-têtes de tableaux sont traduits — c'est sans risque parce que --json existe.
func TestEntetesTraduits(t *testing.T) {
	t.Setenv("JIGGER_LANG", "en")
	Recharger()
	if got := T("table.package"); got != "PACKAGE" {
		t.Fatalf("en-tête anglais : %q", got)
	}
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	if got := T("table.package"); got != "PAQUET" {
		t.Fatalf("en-tête français : %q", got)
	}
}

// Le français d'aujourd'hui, mot pour mot : c'est lui que le banc de la tâche 2 compare.
func TestLibellesFrancaisInchanges(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	Recharger()
	for cle, attendu := range map[string]string{
		"popup.insert":   "insérer",
		"popup.execute":  "exécuter",
		"popup.navigate": "naviguer",
		"popup.browse":   "parcourir",
		"popup.close":    "fermer",
		"popup.cancel":   "annuler",
		"popup.choose":   "choisir",
		"popup.filter":   "filtrer…",
		"popup.empty":    "aucun candidat",
	} {
		if got := T(cle); got != attendu {
			t.Errorf("%s : %q, attendu %q", cle, got, attendu)
		}
	}
}
