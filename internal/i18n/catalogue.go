// Le catalogue des chaînes visibles, une ligne par clé, toutes les langues côte à côte.
//
//	{EN, FR}
//
// L'anglais est l'original : il n'est jamais vide. Le français peut l'être — le repli joue.
// Les clés sont préfixées par la surface où la chaîne s'affiche (popup, facade, cli), pour
// qu'une relecture de traduction soit possible sans lire le code.
package i18n

var catalogue = map[string][nbLangues]string{
	// ── Popup ─────────────────────────────────────────────────────────────────────────
	"popup.insert":   {"insert", "insérer"},
	"popup.execute":  {"execute", "exécuter"},
	"popup.navigate": {"navigate", "naviguer"},
	"popup.browse":   {"browse", "parcourir"},
	"popup.close":    {"close", "fermer"},
	"popup.cancel":   {"cancel", "annuler"},
	"popup.choose":   {"choose", "choisir"},
	"popup.filter":   {"filter…", "filtrer…"},
	"popup.empty":    {"no match", "aucun candidat"},
	// %d est le nombre de paquets du catalogue.
	"popup.filter_hint":    {"type to filter… (%d packages)", "tapez pour filtrer… (%d paquets)"},
	"popup.catalog_brew":   {"building the Homebrew catalog…", "catalogue Homebrew en préparation…"},
	"popup.catalog_winget": {"building the winget catalog…", "catalogue winget en préparation…"},

	// ── Façade ────────────────────────────────────────────────────────────────────────
	"facade.no_verb": {
		"jigger: no verb. Try « jg install <package> » or « jg outdated »",
		"jigger : aucun verbe. Essaie « jg install <paquet> » ou « jg outdated »",
	},
	"facade.unknown_verb": {
		"jigger: « %s » — unknown verb. « jg ⇥ » lists what jigger can do",
		"jigger : « %s » — verbe inconnu. « jg ⇥ » liste ce que jigger sait faire",
	},
	"facade.nobody_can": {
		"jigger: « %s » — no available manager can do that.\n        %s, but it is not installed",
		"jigger : « %s » — aucun gestionnaire disponible ne sait faire ça.\n        %s, mais n'est pas installé",
	},
	"facade.unknown_pm": {
		"jigger: --pm %s — unknown manager. Known: %s",
		"jigger : --pm %s — gestionnaire inconnu de jigger. Connus : %s",
	},
	"facade.pm_unavailable": {
		"jigger: --pm %s — manager unavailable for this verb. Available: %s",
		"jigger : --pm %s — gestionnaire indisponible pour ce verbe. Disponibles : %s",
	},
	"facade.unknown_name": {"jigger: « %s » — unknown to %s", "jigger : « %s » — inconnu de %s"},
	"facade.near":         {"\n        Close: ", "\n        Proche : "},
	"facade.too_recent": {
		"\n        If the package is too recent for the catalog: jg … --pm %s %s",
		"\n        Si le paquet est trop récent pour le catalogue : jg … --pm %s %s",
	},
	"facade.ambiguous": {
		"jigger: « %s » — known to several managers:\n",
		"jigger : « %s » — connu de plusieurs gestionnaires :\n",
	},
	"facade.choose_pm":  {"        Choose with --pm <manager>", "        Choisis avec --pm <gestionnaire>"},
	"facade.failed":     {"jigger (%s): failed\n", "jigger (%s) : échec\n"},
	"facade.unreadable": {"jigger (%s): unreadable output — %v\n", "jigger (%s) : sortie illisible — %v\n"},
	"facade.no_parser": {
		"jigger (%s): verb %q normalized without a declared parser — output ignored for safety\n",
		"jigger (%s) : verbe %q normalisé sans analyseur déclaré — sortie ignorée par sécurité\n",
	},
	"facade.nothing": {"nothing to report\n", "rien à signaler\n"},

	// Ajoutées en relecture : la note d'un catalogue en construction (cat.Note, déjà
	// traduite) et l'erreur brute d'un gestionnaire (%v) fuyaient la ponctuation
	// française même en anglais — de même que le séparateur et les fragments qui
	// composent la liste « X le sait (Y) » de facade.nobody_can.
	"facade.note":           {"jigger: %s", "jigger : %s"},
	"facade.knows_it":       {"%s knows it", "%s le sait"},
	"facade.knows_it_as":    {"%s knows it (%s)", "%s le sait (%s)"},
	"facade.manager_error":  {"jigger (%s): %v\n", "jigger (%s) : %v\n"},
	"facade.list_separator": {"; ", " ; "},

	// ── En-têtes de tableaux ──────────────────────────────────────────────────────────
	// Traduits sans risque : --json sert ceux qui analysent.
	"table.package":   {"PACKAGE", "PAQUET"},
	"table.current":   {"CURRENT", "ACTUEL"},
	"table.available": {"AVAILABLE", "DISPO"},
	"table.source":    {"SOURCE", "SOURCE"},
	"table.pm":        {"PM", "PM"},
}
