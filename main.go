// jigger — assistance aux gestionnaires de paquets dans le terminal : complétion
// contextuelle et sélecteur interactif (Bubble Tea), pensés pour être branchés dans zsh
// (Homebrew, pacman, yay) ou PowerShell (winget, scoop).
//
// C'est le premier mot de la ligne qui désigne le gestionnaire : `brew`, `winget`,
// `scoop`, `pacman` ou `yay`. Toutes les sous-commandes ci-dessous s'emploient de la même
// façon quel que soit celui-ci.
//
// Sous-commandes :
//
//	jigger pick "<ligne>"      sélecteur interactif ; imprime la nouvelle ligne.
//	                           Code de sortie : 0 = insérer (⇥), 10 = exécuter (↩),
//	                           2 = annulé (rien imprimé).
//	jigger render --line "…"   popup sans état ni clavier : une ligne de métadonnées
//	                           puis le cadre. C'est ce que le widget du shell appelle à
//	                           chaque frappe pour le popup vivant.
//	jigger complete "<ligne>"  candidats (un par ligne) pour une complétion classique.
//	jigger prompt              état du gestionnaire en cache, pour le bloc oh-my-posh.
//	                           --refresh l'interroge et réécrit le cache (lent, à
//	                           lancer détaché), --wait lui fait attendre le verrou
//	                           plutôt que renoncer ; --path imprime le fichier de cache.
//	jigger warm                reconstitue les catalogues périmés (lent, à lancer
//	                           détaché) ; --all refait tout, --installed les seules
//	                           listes de paquets installés (après une installation).
//	jigger --version           version.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/elevate"
	"gitlab.yg-devworks.com/yves/jigger/internal/facade"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/prompt"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

var version = "0.16.0"

// motsReserves sont les sous-commandes internes de jigger. Tout autre premier mot est un
// verbe de façade.
//
// Contrainte permanente : aucune sous-commande interne future ne peut porter le nom d'un
// verbe canonique. Si « jigger list » devait un jour désigner un usage interne, c'est le
// mot interne qui change — pas le verbe.
var motsReserves = map[string]bool{
	"pick": true, "render": true, "complete": true,
	"prompt": true, "warm": true, "demo": true, "config": true,
}

func main() {
	ui.Version = version // affichée dans l'en-tête du sélecteur (repère du binaire lancé)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "--version", "-v", "version":
		fmt.Println("jigger", version)
		return
	case "--help", "-h", "help":
		usage()
		return
	}

	if !motsReserves[os.Args[1]] {
		os.Exit(runFacade(os.Args[1:]))
	}

	switch os.Args[1] {
	case "pick":
		os.Exit(runPick(arg(2)))
	case "render":
		os.Exit(runRender(os.Args[2:]))
	case "complete":
		runComplete(arg(2))
	case "prompt":
		os.Exit(runPrompt(os.Args[2:]))
	case "config":
		os.Exit(runConfig(os.Args[2:]))
	case "warm":
		os.Exit(runWarm(os.Args[2:]))
	case "demo":
		runDemo()
	}
}

func arg(i int) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return ""
}

func usage() {
	fmt.Fprintln(os.Stderr, i18n.T("cli.usage1"))
	fmt.Fprintln(os.Stderr, i18n.T("cli.usage2"))
	fmt.Fprintln(os.Stderr, i18n.T("cli.usage3"))
}

// optsCLI rassemble les drapeaux que jigger interprète lui-même. Tous les autres mots en
// « -- » sont passés au gestionnaire : `jg install --cask firefox` doit marcher.
type optsCLI struct {
	PM     string
	JSON   bool
	Yes    bool
	Select bool // --select : ouvrir la vue paginée et imprimer les lignes retenues
}

func separerDrapeaux(argv []string) (verbe string, args []string, o optsCLI, err error) {
	if len(argv) == 0 {
		return "", nil, o, errors.New(i18n.T("cli.no_verb"))
	}
	verbe = argv[0]
	for i := 1; i < len(argv); i++ {
		switch argv[i] {
		case "--pm":
			if i+1 >= len(argv) {
				return "", nil, o, errors.New(i18n.T("cli.pm_expects_value"))
			}
			i++
			o.PM = argv[i]
		case "--json":
			o.JSON = true
		case "--yes":
			o.Yes = true
		case "--select":
			o.Select = true
		default:
			args = append(args, argv[i])
		}
	}
	return verbe, args, o, nil
}

// runFacade déroule le pipeline de la spec §3 : résoudre le verbe, résoudre la cible,
// exécuter, formater.
func runFacade(argv []string) int {
	premier, reste, o, err := separerDrapeaux(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	ligne := append([]string{premier}, reste...)
	verbe, args, capables, err := facade.ResoudreVerbe(ligne)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	cats := map[string]*pm.Catalog{}
	for _, m := range capables {
		cats[m.Cmd()] = m.Load()
	}

	// resolu accumule les désambiguïsations obtenues au fil des noms : chaque nom ambigu
	// tranché par l'utilisateur ne lie que lui, jamais les autres noms de la même ligne
	// (spec §3 — `jg install git fd` route fd vers scoop sans qu'on ait à le dire, même
	// si git, ambigu, vient d'être tranché pour winget). --pm (o.PM), lui, continue de
	// s'appliquer à toute la ligne : c'est le rôle qu'il a toujours eu.
	resolu := map[string]string{}
	var cibles []facade.Cible
	for {
		var amb *facade.Ambiguite
		cibles, amb, err = facade.Router(verbe, args, o.PM, resolu, capables, cats)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if amb == nil {
			break
		}
		choisi, ok := trancher(amb)
		if !ok {
			return 2
		}
		resolu[amb.Nom] = choisi
	}

	res := facade.ExecuterAvec(verbe, cibles, facade.Opts{JSON: o.JSON, Yes: o.Yes})
	if res.Rejeu != nil {
		return elever(res.Rejeu, res.Code)
	}
	if len(res.Rows) > 0 || facade.Normalise(verbe) {
		if !afficherPagine(verbe, res.Rows, o) {
			fmt.Print(facade.Formater(verbe, res.Rows, o.JSON))
		}
	}
	return res.Code
}

// elever traite un échec de privilèges constaté par la façade (ADR-0004). Il ne s'appelle
// qu'après coup : la commande a déjà tourné, relayée comme d'habitude, et elle a échoué en
// nommant sa cause. Rien n'a été intercepté, rien n'est élevé sans un oui explicite.
//
// Le code rendu est celui du gestionnaire dans tous les cas où l'utilisateur n'a pas
// rejoué — refuser une élévation ne doit pas changer ce que le shell voit de l'échec
// d'origine.
func elever(r *facade.Rejeu, code int) int {
	ligne := r.Cmd + " " + strings.Join(r.Argv, " ")

	// Le contre-cas : élever serait exactement le contraire de ce qu'il faut faire. On le
	// dit, et on ne propose rien.
	if r.Droits == pm.DroitsInterdits {
		fmt.Fprint(os.Stderr, i18n.Tf("elev.forbidden", r.Cmd))
		return code
	}

	fmt.Fprint(os.Stderr, i18n.Tf("elev.required", r.Cmd))

	// Pas de terminal (un tube, un script, une tâche planifiée) ou pas d'élévation
	// possible sur cette plateforme : on imprime la ligne exacte et on s'arrête. Un
	// pipeline ne doit jamais se bloquer sur une invite.
	if !elevate.Possible() || !demanderElevation(r) {
		fmt.Fprint(os.Stderr, i18n.Tf("elev.manual", ligne))
		return code
	}

	nouveau, err := elevate.Rejouer(r.Cmd, r.Argv)
	if errors.Is(err, elevate.ErrRefuse) {
		// L'invite du système a été refusée. C'est une réponse, pas une panne.
		fmt.Fprint(os.Stderr, i18n.T("elev.declined"))
		return code
	}
	if err != nil {
		fmt.Fprint(os.Stderr, i18n.Tf("facade.manager_error", r.Cmd, err))
		return code
	}
	return nouveau
}

// demanderElevation pose la question sur le terminal, et rend false dès qu'il n'y en a
// pas. Même forme que trancher() — il n'y a pas lieu d'avoir deux façons de poser une
// question — et même dégradation : sans terminal, aucune invite.
//
// « Annuler » est la ligne ouverte par défaut : la touche la plus facile à frapper ne doit
// pas être celle qui élève.
func demanderElevation(r *facade.Rejeu) bool {
	tty, err := openTTY()
	if err != nil {
		return false
	}
	defer tty.Close()

	oui := i18n.T("elev.retry_window")
	if elevate.Prevue() == elevate.VoieSudo {
		oui = i18n.T("elev.retry_sudo")
	}

	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)
	model := ui.New(i18n.T("elev.title"), complete.Result{Items: []complete.Item{
		{Name: i18n.T("popup.cancel")},
		{Name: oui},
	}})
	model.Keys = []ui.Key{
		{Key: "↵", Label: i18n.T("popup.choose")},
		{Key: "^G", Label: i18n.T("popup.cancel")},
	}
	// Pas de ligne de filtre : on répond par oui ou par non, il n'y a rien à chercher.
	model.SansFiltre = true

	fmt.Fprint(tty.Out, "\r\n")
	prog := tea.NewProgram(model, tea.WithInput(tty.In), tea.WithOutput(tty.Out))
	final, err := prog.Run()
	fmt.Fprint(tty.Out, "\x1b[1A\r")
	if err != nil {
		return false
	}
	m := final.(ui.Model)
	return m.Chosen != nil && m.Chosen.Name == oui
}

// afficherPagine ouvre la vue paginée si les conditions sont réunies, et dit si elle
// s'en est chargée. Un « false » renvoie l'appelant à la table brute — le comportement
// par défaut, celui dont dépendent les scripts.
//
// La vue se dessine sur le terminal (/dev/tty), jamais sur la sortie standard : c'est ce
// qui rend `jg install $(jg search fd --select)` possible, la sélection partant seule
// dans le tube.
func afficherPagine(verbe pm.Verb, rows []pm.Package, o optsCLI) bool {
	ctx := ui.Contexte{
		EstTTY:   sortieEstTerminal(),
		Hauteur:  hauteurEcran(),
		NbLignes: len(rows),
		EnJSON:   o.JSON,
		Force:    o.Select,
		Pager:    os.Getenv("JIGGER_PAGER"),
	}
	if !ui.DoitPaginer(ctx) {
		// --select sans terminal n'a nulle part où dessiner : le dire, plutôt que de
		// retomber silencieusement sur la table brute et laisser croire que le drapeau
		// a été pris en compte.
		if o.Select && !o.JSON && !ctx.EstTTY {
			fmt.Fprintln(os.Stderr, i18n.T("cli.select_needs_tty"))
		}
		return false
	}

	tty, err := openTTY()
	if err != nil {
		return false
	}
	defer tty.Close()

	entete, cellules := facade.Colonnes(rows)
	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)

	modele := ui.NouveauTableau(string(verbe), entete, cellules)
	prog := tea.NewProgram(modele,
		tea.WithInput(tty.In), tea.WithOutput(tty.Out), tea.WithAltScreen())
	final, err := prog.Run()
	if err != nil {
		return false
	}

	// Les noms retenus partent sur la sortie standard, un par ligne : de quoi les
	// enchaîner avec xargs ou une substitution de commande.
	if t, ok := final.(ui.Tableau); ok && !t.Annule() {
		for _, nom := range t.Choisis {
			fmt.Println(nom)
		}
	}
	return true
}

// hauteurEcran rend la hauteur du terminal en lignes, ou 0 si elle est inconnue —
// auquel cas la décision d'armer choisit de paginer plutôt que de deviner.
func hauteurEcran() int {
	if _, h, err := term.GetSize(os.Stdout.Fd()); err == nil {
		return h
	}
	// La sortie peut être redirigée alors qu'un terminal existe (`jg list --select | …`) :
	// on interroge alors le terminal lui-même.
	if tty, err := openTTY(); err == nil {
		defer tty.Close()
		if _, h, err := term.GetSize(tty.Out.Fd()); err == nil {
			return h
		}
	}
	return 0
}

// sortieEstTerminal dit si la sortie standard est un terminal, sans dépendance
// supplémentaire : un périphérique caractère, par opposition à un fichier ou un tube.
func sortieEstTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// trancher ouvre le sélecteur sur les candidats d'un nom ambigu. Ce n'est pas un nouvel
// écran : c'est le popup, avec un autre titre et d'autres touches de pied.
//
// Hors terminal — pipe, script, CI — il n'y a personne pour choisir : on échoue en
// listant les candidats et en rappelant --pm. Jamais de choix automatique.
func trancher(amb *facade.Ambiguite) (string, bool) {
	items := make([]complete.Item, 0, len(amb.Candidats))
	for _, c := range amb.Candidats {
		items = append(items, complete.Item{
			Name:  c.Mgr.Cmd(),
			Badge: c.Badge,
			PM:    c.Mgr.Cmd(),
		})
	}

	tty, err := openTTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("facade.ambiguous"), amb.Nom)
		for _, c := range amb.Candidats {
			fmt.Fprintf(os.Stderr, "        %s\n", c.Mgr.Cmd())
		}
		fmt.Fprintln(os.Stderr, i18n.T("facade.choose_pm"))
		return "", false
	}
	defer tty.Close()

	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)
	titre := i18n.Tf("popup.ambiguous_title", amb.Nom, len(amb.Candidats))
	model := ui.New(titre, complete.Result{Executable: true, Items: items})
	// Pied propre à la désambiguïsation : on choisit un gestionnaire, on n'insère ni
	// n'exécute une commande — ⇥/↩ n'y ont pas de sens (spec §3, README).
	model.Keys = []ui.Key{{Key: "↵", Label: i18n.T("popup.choose")}, {Key: "^G", Label: i18n.T("popup.cancel")}}

	fmt.Fprint(tty.Out, "\r\n")
	prog := tea.NewProgram(model, tea.WithInput(tty.In), tea.WithOutput(tty.Out))
	final, err := prog.Run()
	fmt.Fprint(tty.Out, "\x1b[1A\r")
	if err != nil {
		return "", false
	}

	m := final.(ui.Model)
	if m.Chosen == nil {
		return "", false // annulé
	}
	return m.Chosen.Name, true
}

// runWarm reconstitue les catalogues mis en cache. C'est le chemin lent — plusieurs
// secondes pour winget —, lancé détaché : ni le rendu du popup ni le prompt n'attendent
// jamais après lui, ils se contentent du cache précédent jusqu'à ce qu'il soit refait.
//
// Un verrou évite qu'une rafale de frappes — ou dix onglets ouverts d'un coup — ne lance
// dix réchauffements concurrents.
func runWarm(args []string) int {
	fs := flag.NewFlagSet("warm", flag.ContinueOnError)
	tout := fs.Bool("all", false, i18n.T("cli.flag_all"))
	installes := fs.Bool("installed", false, i18n.T("cli.flag_installed"))
	if err := fs.Parse(args); err != nil {
		return 2
	}

	scope := pm.ScopeStale
	switch {
	case *tout:
		scope = pm.ScopeAll
	case *installes:
		scope = pm.ScopeInstalled
	}

	libere, ok := pm.Lock(pm.WarmLock())
	if !ok {
		return 0 // un autre réchauffement est en cours : il fait déjà le travail
	}
	defer libere()

	code := 0
	for _, m := range managers.Available() {
		if err := m.Warm(scope); err != nil {
			fmt.Fprint(os.Stderr, i18n.Tf("cli.warm_failed", m.Cmd(), err))
			code = 1
		}
	}
	return code
}

// runPrompt imprime l'état de Homebrew en cache — « version<TAB>formulae<TAB>casks<TAB>epoch » —
// pour le bloc oh-my-posh. Sans cache, il n'imprime rien : le prompt masque alors le bloc.
//
// Cette commande sort **toujours** en 0 quand elle est appelée pour afficher : un prompt
// n'a pas à signaler que brew est indisponible. Seul --refresh, qui n'est lancé que
// détaché, remonte ses échecs (utile en débogage).
func runPrompt(args []string) int {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, i18n.T("cli.flag_refresh"))
	wait := fs.Bool("wait", false, i18n.T("cli.flag_wait"))
	path := fs.Bool("path", false, i18n.T("cli.flag_path"))
	if err := fs.Parse(args); err != nil {
		return 2
	}

	dir := prompt.Dir()
	if *path {
		fmt.Println(prompt.File(dir))
		return 0
	}

	if *refresh {
		rafraichir := prompt.Refresh
		if *wait {
			// Le cache est connu faux (une commande brew mutante vient de tourner) :
			// on prend son tour dans la file plutôt que de renoncer.
			rafraichir = prompt.RefreshWait
		}
		s, err := rafraichir(dir)
		if err != nil {
			// Verrou pris par un autre shell, ou brew injoignable : le cache précédent
			// reste en place et sera réessayé au prochain prompt périmé.
			fmt.Fprintln(os.Stderr, i18n.T("cli.prompt_failed"), err)
			return 1
		}
		fmt.Println(s.Line())
		return 0
	}

	if s, ok := prompt.Read(dir); ok {
		fmt.Println(s.Line())
	}
	return 0
}

// runComplete imprime les candidats (complétion non interactive).
func runComplete(line string) {
	for _, it := range complete.Complete(line).Items {
		fmt.Println(it.Name)
	}
}

// tropDeCandidats : au-delà, une liste non filtrée n'apprend rien (les ~7 000 formulae
// commencent par « 0ad », « 0xtools »…). On affiche alors une invite à filtrer. En
// dessous — les paquets installés, typiquement quelques centaines — la liste complète
// reste utile à parcourir.
const tropDeCandidats = 300

// runRender imprime, sans état ni clavier, une ligne de métadonnées puis le popup. C'est
// le mode appelé par le widget zsh à chaque frappe : tout l'état (l'index sélectionné)
// vit côté shell et revient par --sel.
//
// La ligne de métadonnées est faite de champs `clé=valeur` séparés par des tabulations,
// `left=` toujours en dernier — c'est le seul champ qui peut contenir des espaces, le
// shell le récupère donc comme « tout ce qui suit ».
func runRender(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	line := fs.String("line", "", i18n.T("cli.flag_line"))
	sel := fs.Int("sel", 0, i18n.T("cli.flag_sel"))
	cols := fs.Int("cols", 0, i18n.T("cli.flag_cols"))
	rows := fs.Int("rows", 8, i18n.T("cli.flag_rows"))
	color := fs.String("color", "auto", i18n.T("cli.flag_color"))
	focus := fs.Bool("focus", false, i18n.T("cli.flag_focus"))
	regex := fs.Bool("regex", false, i18n.T("cli.flag_regex"))
	if err := fs.Parse(args); err != nil {
		return 2
	}

	lipgloss.SetColorProfile(colorProfile(*color))

	res := complete.CompleteAvec(*line, *regex)

	// Un fournisseur qui se tait ne fait dessiner aucun cadre : on n'émet que la ligne
	// de métadonnées. C'est le protocole que les deux greffons connaissent déjà — une
	// sortie d'une seule ligne vaut « rien à afficher », et ils effacent ce qui restait
	// (`_jigger_fetch` de jigger.plugin.zsh, `Get-JiggerFrame` de jigger.psm1). Sans
	// cela, une machine sans ~/.ssh/config voyait une boîte « aucun candidat »
	// apparaître sous chaque frappe d'une ligne ssh, scp ou sftp.
	if res.Silencieux {
		fmt.Printf("count=0\tsel=-1\texec=%s\tleft=%s\n", boolField(false), *line)
		return 0
	}

	// Le pied dit où ira la prochaine flèche : sans le focus, ↓ sert à entrer dans la
	// liste (↑ reste l'historique) ; avec, les deux parcourent les candidats.
	navigation := ui.Key{Key: "↓", Label: i18n.T("popup.browse")}
	if *focus {
		navigation = ui.Key{Key: "↑↓", Label: i18n.T("popup.navigate")}
	}

	// Le mode s'affiche dans le titre, jamais dans le pied : le cadre a une largeur fixe
	// et le pied y perdrait son dernier libellé. En mode préfixe — le cas ordinaire — le
	// titre est exactement celui d'avant, donc le rendu du popup ne bouge pas d'un octet.
	titre := res.Title()
	if *regex {
		titre += " [" + i18n.T("table.moderegex") + "]"
	}

	frame := ui.Frame{
		Title:   titre,
		Items:   res.Items,
		Rows:    *rows,
		Focused: *focus,
		// Quatre pastilles, contre trois avant : ⏎ pose le candidat *et* exécute, ce que
		// ⇥ ne fait pas — deux gestes distincts, donc deux libellés, et le même
		// vocabulaire que le sélecteur plein écran. Les quatre tiennent dans la largeur
		// du cadre ; l'ordre est celui du clipping, le dernier étant celui qu'un
		// terminal étroit perdra d'abord.
		Keys: []ui.Key{
			{Key: "⇥", Label: i18n.T("popup.insert")},
			{Key: "↩", Label: i18n.T("popup.execute")},
			navigation,
			{Key: "^G", Label: i18n.T("popup.close")},
		},
	}
	if *cols > 0 {
		frame.Width = min(58, *cols-2)
	}

	// Contexte paquet, mot vide : on invite à filtrer plutôt que d'égrener le catalogue.
	if res.Executable && res.Word == "" && len(res.Items) > tropDeCandidats {
		frame.Items = nil
		frame.Empty = i18n.Tf("popup.filter_hint", len(res.Items))
	} else if len(res.Items) == 0 {
		// Le gestionnaire a parfois mieux à dire qu'« aucun candidat » — un catalogue
		// winget encore en cours de constitution, par exemple.
		frame.Empty = i18n.T("popup.empty")
		if res.Note != "" {
			frame.Empty = res.Note
		}
	}

	left := *line
	if len(frame.Items) > 0 {
		frame.Sel = min(max(*sel, 0), len(frame.Items)-1)
		frame.Offset = ui.ScrollOffset(frame.Sel, len(frame.Items), *rows)
		left = res.Prefix + res.InsertItem(frame.Items[frame.Sel])
	} else {
		frame.Sel = -1
	}

	fmt.Printf("count=%d\tsel=%d\texec=%s\tleft=%s\n",
		len(frame.Items), frame.Sel, boolField(res.Executable), left)
	fmt.Println(frame.Render())
	return 0
}

// colorProfile traduit --color. La sortie de `render` est toujours capturée par le
// widget : lipgloss n'a donc aucun moyen de deviner ce que vaut le terminal, et rendrait
// tout en gris. C'est le shell — qui, lui, connaît $COLORTERM et $TERM — qui tranche ;
// « auto » se rabat sur stderr, resté attaché au terminal dans une substitution.
func colorProfile(name string) termenv.Profile {
	switch name {
	case "never":
		return termenv.Ascii
	case "16":
		return termenv.ANSI
	case "256":
		return termenv.ANSI256
	case "truecolor":
		return termenv.TrueColor
	default:
		return termenv.NewOutput(os.Stderr).Profile
	}
}

func boolField(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// runPick lance le sélecteur et imprime la nouvelle ligne. Le TUI dessine sur le
// terminal (/dev/tty, CONOUT$ sous Windows) pour laisser stdout au résultat (comme fzf).
func runPick(line string) int {
	res := complete.Complete(line)
	if len(res.Items) == 0 {
		return 1
	}

	// Candidat unique : il n'y a rien à choisir. On insère directement, sans ouvrir le
	// popup (ni même le TTY) — comme le fait la complétion zsh sur une correspondance
	// unique.
	if len(res.Items) == 1 {
		fmt.Print(res.Prefix + res.InsertItem(res.Items[0]))
		return 0
	}

	tty, err := openTTY()
	if err != nil {
		return 1
	}
	defer tty.Close()

	// stdout est capturé par le widget → lipgloss croirait à un pipe sans couleur.
	// On fixe le profil couleur d'après le vrai terminal (le TTY).
	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)

	model := ui.New(res.Title(), res)
	// Popup « classique » : rendu inline, SOUS la ligne de commande — qui reste
	// visible. On descend d'une ligne (\r\n) pour que le cadre se dessine dessous (pas
	// d'écran alterné). Bubble Tea gère ensuite son cadre en mouvements *relatifs*
	// (robuste au défilement) et, à la sortie, l'efface lui-même : le modèle rend une
	// vue vide quand `quitting` (cf. picker.View) → EraseScreenBelow. On remonte enfin
	// d'une ligne pour reposer le curseur sur la ligne de commande ; le widget zsh la
	// redessine (reset-prompt).
	fmt.Fprint(tty.Out, "\r\n")
	prog := tea.NewProgram(model, tea.WithInput(tty.In), tea.WithOutput(tty.Out))
	final, err := prog.Run()
	fmt.Fprint(tty.Out, "\x1b[1A\r") // curseur → début de la ligne de commande
	if err != nil {
		return 1
	}

	m := final.(ui.Model)
	if m.Chosen == nil {
		return 2 // annulé
	}

	fmt.Print(res.Prefix + res.InsertItem(*m.Chosen))
	if m.Execute {
		return 10 // ↩ : commande à exécuter
	}
	return 0 // ⇥ : insérée
}

// runDemo imprime un aperçu statique du sélecteur (pour prévisualiser le rendu sans
// installer le widget), avec le gestionnaire de la plateforme. stdout est ici un vrai
// terminal → couleurs détectées.
func runDemo() {
	lipgloss.SetColorProfile(termenv.TrueColor) // aperçu toujours coloré

	titre, items := apercu(managers.Default().Cmd())
	fmt.Println(ui.New(titre, complete.Result{Executable: true, Items: items}).View())
}

// apercu rend la ligne de titre et les candidats d'un aperçu, pour un mot de commande.
//
// Le mot vient de managers.Default() — le gestionnaire que la façade retiendrait pour une
// ligne qui ne nomme personne —, et non d'un test sur runtime.GOOS. La distinction n'est
// pas cosmétique : un `runtime.GOOS == "windows"` en `else` de brew faisait annoncer
// « brew install », et lister des formules Homebrew, sur une machine Arch qui n'a pas de
// brew. Le module pacman a rendu cet aperçu faux sans jamais le toucher. Demander la règle
// là où elle est écrite est aussi ce qui fera suivre l'aperçu au prochain gestionnaire,
// sans qu'on ait à y repenser.
//
// Découpée de runDemo pour être éprouvable sur les trois branches depuis n'importe quelle
// machine : la sortie de runDemo, elle, dépend de ce qui est installé.
func apercu(cmd string) (string, []complete.Item) {
	switch cmd {
	case "winget":
		return "winget install", []complete.Item{
			{Name: "Git.Git", Badge: pm.BadgeWinget, Installed: true, Version: "2.55.0"},
			{Name: "GitHub.cli", Badge: pm.BadgeWinget, Installed: true, Version: "2.62.0"},
			{Name: "GitHub.GitHubDesktop", Badge: pm.BadgeWinget},
			{Name: "dandavison.delta", Badge: pm.BadgeWinget},
			{Name: "Google.Chrome", Badge: pm.BadgeWinget},
			{Name: "Canon.PrinterDriver", Badge: pm.BadgeOther, Installed: true, Version: "1.02"},
		}
	case "pacman", "yay":
		// « -S » et non « install » : chez pacman l'opération est un drapeau, et un aperçu
		// qui montrerait une autre grammaire que celle du popup mentirait deux fois.
		// Versions au format alpm, avec leur numéro de version du paquet (pkgrel).
		return cmd + " -S", []complete.Item{
			{Name: "git", Badge: pm.BadgeRepo, Installed: true, Version: "2.55.0-1"},
			{Name: "gitui", Badge: pm.BadgeRepo},
			{Name: "github-cli", Badge: pm.BadgeRepo, Installed: true, Version: "2.62.0-1"},
			{Name: "git-delta", Badge: pm.BadgeRepo},
			{Name: "google-chrome", Badge: pm.BadgeAUR},
			{Name: "visual-studio-code-bin", Badge: pm.BadgeAUR},
		}
	}
	return "brew install", []complete.Item{
		{Name: "git", Badge: pm.BadgeFormula, Installed: true, Version: "2.55.0"},
		{Name: "gitui", Badge: pm.BadgeFormula},
		{Name: "gh", Badge: pm.BadgeFormula, Installed: true, Version: "2.62.0"},
		{Name: "git-delta", Badge: pm.BadgeFormula},
		{Name: "google-chrome", Badge: pm.BadgeCask},
		{Name: "firefox", Badge: pm.BadgeCask},
	}
}
