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
	"popup.empty":    {"no matches", "aucun candidat"},
	// %d est le nombre de paquets du catalogue.
	"popup.filter_hint":    {"type to filter… (%d packages)", "tapez pour filtrer… (%d paquets)"},
	"popup.catalog_brew":   {"building the Homebrew catalog…", "catalogue Homebrew en préparation…"},
	"popup.catalog_winget": {"building the winget catalog…", "catalogue winget en préparation…"},
	// Titre du popup de désambiguïsation ouvert par trancher() (main.go) : fuite
	// assemblée trouvée hors des lignes citées par le brief — un fmt.Sprintf qui
	// fabriquait ce titre en français, y compris à JIGGER_LANG=en. Préfixée popup. (et
	// non facade.) : elle s'affiche dans le cadre, comme popup.insert ou
	// popup.filter_hint — facade.ambiguous et facade.choose_pm, eux, sortent en Fprintf
	// brut sur stderr, une surface différente.
	"popup.ambiguous_title": {"%s: %d managers", "%s : %d gestionnaires"},

	// ── Façade ────────────────────────────────────────────────────────────────────────
	"facade.no_verb": {
		"jigger: no verb. Try \"jg install <package>\" or \"jg outdated\"",
		"jigger : aucun verbe. Essaie « jg install <paquet> » ou « jg outdated »",
	},
	"facade.unknown_verb": {
		"jigger: \"%s\" — unknown verb. \"jg ⇥\" lists what jigger can do",
		"jigger : « %s » — verbe inconnu. « jg ⇥ » liste ce que jigger sait faire",
	},
	"facade.nobody_can": {
		"jigger: \"%s\" — no available manager can do that.\n        %s, but it is not installed",
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
	"facade.unknown_name": {"jigger: \"%s\" — unknown to %s", "jigger : « %s » — inconnu de %s"},
	"facade.near":         {"\n        Close: ", "\n        Proche : "},
	"facade.too_recent": {
		"\n        If the package is too recent for the catalog: jg … --pm %s %s",
		"\n        Si le paquet est trop récent pour le catalogue : jg … --pm %s %s",
	},
	"facade.ambiguous": {
		"jigger: \"%s\" — known to several managers:\n",
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
	// Le joint entre deux noms de gestionnaires dans « inconnu de winget et scoop »
	// (nomInconnu, routage.go). Il était écrit en dur en français : sur une machine
	// Windows, la phrase anglaise sortait « unknown to winget et scoop ».
	"facade.name_separator": {" and ", " et "},

	// ── En-têtes de tableaux ──────────────────────────────────────────────────────────
	// Traduits sans risque : --json sert ceux qui analysent.
	"table.package":   {"PACKAGE", "PAQUET"},
	"table.current":   {"CURRENT", "ACTUEL"},
	"table.available": {"AVAILABLE", "DISPO"},
	"table.source":    {"SOURCE", "SOURCE"},
	"table.pm":        {"PM", "PM"},

	// ── Vue paginée ───────────────────────────────────────────────────────────────────
	// Les touches sont annoncées dans le pied du cadre : on ne devine ni ce qu'elles
	// font, ni dans quel mode de filtre on se trouve.
	"table.toggle": {"select", "cocher"},
	// « tout » plutôt que « tout cocher » : le pied du cadre a une largeur à tenir, et
	// la touche est à côté de son libellé.
	"table.toggleall": {"all", "tout"},
	"table.confirm": {"confirm", "valider"},
	"table.page":    {"page", "page"},
	"table.regex":   {"regex", "regex"},
	// Le mode courant, affiché en permanence : « texte » est le mode par défaut, celui
	// que le sélecteur a toujours eu.
	"table.modetexte": {"text", "texte"},
	"table.moderegex": {"regex", "regex"},
	// %d est le nombre de lignes cochées.
	"table.selected": {"%d selected", "%d cochées"},
	// Un motif qui ne compile pas laisse la liste entière plutôt que de la vider : sans
	// ce message, on croirait qu'aucun paquet ne correspond.
	"table.badregex": {"invalid pattern", "motif invalide"},
	// --select hors terminal : le dire plutôt que de retomber en silence sur la table
	// brute, ce qui laisserait croire que le drapeau a été pris en compte.
	"cli.select_needs_tty": {
		"--select needs a terminal to draw on; printing the plain table instead",
		"--select a besoin d'un terminal pour s'afficher ; table brute à la place",
	},

	// ── Ligne de commande ─────────────────────────────────────────────────────────────
	"cli.usage1": {
		"usage: jigger <verb> [--pm <manager>] [--json] [--yes] [--select] [arguments…]",
		"usage: jigger <verbe> [--pm <gestionnaire>] [--json] [--yes] [--select] [arguments…]",
	},
	"cli.usage2": {
		"       jigger pick|complete \"<line>\" | jigger render --line \"<line>\"",
		"       jigger pick|complete \"<ligne>\" | jigger render --line \"<ligne>\"",
	},
	"cli.usage3": {
		"       jigger prompt [--refresh [--wait]|--path] | jigger warm [--all|--installed]",
		"       jigger prompt [--refresh [--wait]|--path] | jigger warm [--all|--installed]",
	},
	"cli.flag_all":       {"rebuild everything, even what is still fresh", "refait tout, même ce qui est encore frais"},
	"cli.flag_installed": {"rebuild only the installed-package lists", "refait les seules listes de paquets installés"},
	// « interroge brew » depuis toujours ; faux depuis la façade multi-gestionnaires, qui
	// interroge aussi bien winget et scoop. Aide de drapeau : hors du banc de
	// non-régression, le français peut donc être corrigé.
	"cli.flag_refresh":  {"query the manager and rewrite the cache (slow)", "interroge le gestionnaire et réécrit le cache (lent)"},
	"cli.flag_wait":     {"with --refresh: wait for the lock instead of giving up", "avec --refresh : attend le verrou au lieu de renoncer"},
	"cli.flag_path":     {"print the cache file path", "imprime le chemin du fichier de cache"},
	"cli.flag_export":      {"print the lines the shell plugin evaluates", "imprime les lignes que le greffon évalue"},
	"cli.flag_shell":       {"which shell the export targets: zsh or powershell", "shell visé par l'export : zsh ou powershell"},
	"cli.flag_config_path": {"print the configuration file path", "imprime le chemin du fichier de configuration"},
	"cli.flag_list":        {"list every setting, its value and where it comes from", "liste chaque réglage, sa valeur et sa provenance"},

	// ── Configuration ─────────────────────────────────────────────────────────────────
	// La provenance n'est pas décorative : sans elle, l'écran montrerait une valeur du
	// fichier pendant que la machine en applique une autre, venue de l'environnement.
	"cfg.from_env":     {"environment", "environnement"},
	"cfg.from_file":    {"file", "fichier"},
	"cfg.from_default": {"default", "défaut"},
	"cfg.unreadable":   {"jigger: unreadable configuration — %v\n", "jigger : configuration illisible — %v\n"},

	"cfg.pager":       {"paged view for long listings", "vue paginée pour les longues listes"},
	"cfg.lang":        {"message language: en or fr", "langue des messages : en ou fr"},
	"cfg.cache_dir":   {"where the catalogs are cached", "où les catalogues sont mis en cache"},
	"cfg.live":        {"popup that follows your typing", "popup qui suit la frappe"},
	"cfg.rows":        {"candidates shown at once", "candidats affichés à la fois"},
	"cfg.key":         {"insertion key", "touche d'insertion"},
	"cfg.keys_extra":  {"keys relayed beyond printable ASCII", "touches relayées en plus de l'ASCII imprimable"},
	"cfg.commands":    {"commands that trigger the popup", "commandes qui déclenchent le popup"},
	"cfg.min_columns": {"below this width, the popup stays away", "en dessous de cette largeur, le popup s'abstient"},
	"cfg.prompt":      {"package-manager block in the prompt", "bloc gestionnaire dans l'invite"},
	"cfg.prompt_ttl":  {"seconds before the prompt block is refreshed", "secondes avant rafraîchissement du bloc d'invite"},
	"cfg.bin":         {"which jigger binary the plugin calls", "binaire jigger appelé par le greffon"},
	"cfg.ttl":         {"how long the catalog stays fresh", "durée de validité du catalogue"},

	"cfg.title":      {"jigger — settings", "jigger — réglages"},
	"cfg.edit":       {"edit", "modifier"},
	"cfg.reset":      {"reset", "remise à zéro"},
	"cfg.quit_save":  {"save and quit", "enregistrer et quitter"},
	"cfg.now":        {"takes effect immediately", "prend effet tout de suite"},
	"cfg.next_shell": {"takes effect in your next shell", "prend effet au prochain shell"},
	"cfg.readonly":   {"what jigger sees — read only", "ce que jigger voit — en lecture seule"},
	"cfg.grp_now":    {"Settings", "Réglages"},
	"cfg.grp_shell":  {"Shell plugin", "Greffon du shell"},
	"cfg.grp_seen":   {"Your installation", "Ton installation"},
	// %s est le chemin du fichier.
	"cfg.written": {"written to %s\n", "écrit dans %s\n"},
	"cli.flag_regex":    {"filter package names as a regular expression", "filtre les noms de paquets en expression rationnelle"},
	"cli.flag_line":     {"line to complete (up to the cursor)", "ligne à compléter (jusqu'au curseur)"},
	"cli.flag_sel":      {"index of the current candidate", "index du candidat courant"},
	"cli.flag_cols":     {"terminal width", "largeur du terminal"},
	"cli.flag_rows":     {"number of candidates shown", "nombre de candidats affichés"},
	"cli.flag_color":    {"color profile: auto|never|16|256|truecolor", "profil couleur : auto|never|16|256|truecolor"},
	"cli.flag_focus":    {"the popup owns the keyboard: arrows go to it", "le popup a le clavier : les flèches lui reviennent"},
	"cli.warm_failed":   {"jigger warm (%s): %v\n", "jigger warm (%s) : %v\n"},
	"cli.prompt_failed": {"jigger prompt --refresh:", "jigger prompt --refresh :"},

	// Les deux erreurs du paquet prompt (internal/prompt/status.go) : elles ressortent
	// telles quelles derrière cli.prompt_failed, et donnaient jusqu'ici la phrase hybride
	// « jigger prompt --refresh: rafraîchissement déjà en cours ». cli.no_winget_no_scoop
	// porte son %w — c'est fmt.Errorf qui l'interprète, pour garder l'enveloppement.
	"cli.refresh_locked":     {"a refresh is already running", "rafraîchissement déjà en cours"},
	"cli.no_winget_no_scoop": {"neither winget nor scoop: %w", "ni winget ni scoop : %w"},

	// Erreurs de separerDrapeaux — relevées en relecture : « les derniers messages de
	// main.go » de l'intitulé de la tâche 5 les couvre, le brief avait juste omis de les
	// citer.
	"cli.no_verb":          {"no verb", "aucun verbe"},
	"cli.pm_expects_value": {"jigger: --pm expects a manager name", "jigger : --pm attend un nom de gestionnaire"},
}
