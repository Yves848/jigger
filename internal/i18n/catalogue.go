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
}
