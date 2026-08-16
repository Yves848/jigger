package i18n

import "testing"

// L'anglais est l'original : une entrée vide n'aurait rien sur quoi se replier.
func TestAucuneEntreeAnglaiseVide(t *testing.T) {
	for cle, trad := range catalogue {
		if trad[EN] == "" {
			t.Errorf("%s : l'anglais est vide", cle)
		}
	}
}

// Les clés du popup existent et sont traduites dans les deux langues.
func TestClesDuPopup(t *testing.T) {
	for _, cle := range []string{
		"popup.insert", "popup.execute", "popup.navigate", "popup.browse",
		"popup.close", "popup.cancel", "popup.choose", "popup.filter",
		"popup.empty", "popup.filter_hint", "popup.catalog_brew", "popup.catalog_winget",
	} {
		trad, ok := catalogue[cle]
		if !ok {
			t.Errorf("%s : absente du catalogue", cle)
			continue
		}
		if trad[FR] == "" {
			t.Errorf("%s : le français est vide", cle)
		}
	}
}

// Les clés de la façade et des en-têtes de tableaux existent et sont traduites dans les
// deux langues — ce sont les chaînes les plus lues par un utilisateur qui se trompe.
func TestClesDeLaFacade(t *testing.T) {
	for _, cle := range []string{
		"facade.no_verb", "facade.unknown_verb", "facade.nobody_can",
		"facade.unknown_pm", "facade.pm_unavailable", "facade.unknown_name",
		"facade.near", "facade.too_recent", "facade.ambiguous", "facade.ambiguous_title",
		"facade.choose_pm", "facade.failed", "facade.unreadable", "facade.no_parser",
		"facade.nothing", "table.package", "table.current", "table.available",
		"table.source", "table.pm",
	} {
		trad, ok := catalogue[cle]
		if !ok {
			t.Errorf("%s : absente du catalogue", cle)
			continue
		}
		if trad[FR] == "" {
			t.Errorf("%s : le français est vide", cle)
		}
	}
}

// Les clés de la ligne de commande (usage, drapeaux, derniers messages de main.go)
// existent dans le catalogue.
func TestClesDuCli(t *testing.T) {
	for _, cle := range []string{
		"cli.usage1", "cli.usage2", "cli.usage3",
		"cli.flag_all", "cli.flag_installed", "cli.flag_refresh", "cli.flag_wait",
		"cli.flag_path", "cli.flag_line", "cli.flag_sel", "cli.flag_cols",
		"cli.flag_rows", "cli.flag_color", "cli.flag_focus", "cli.warm_failed",
		"cli.prompt_failed",
	} {
		if _, ok := catalogue[cle]; !ok {
			t.Errorf("%s : absente du catalogue", cle)
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
