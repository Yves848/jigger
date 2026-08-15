// jigger — assistance aux gestionnaires de paquets dans le terminal : complétion
// contextuelle et sélecteur interactif (Bubble Tea), pensés pour être branchés dans zsh
// (Homebrew) ou PowerShell (winget, scoop).
//
// C'est le premier mot de la ligne qui désigne le gestionnaire : `brew`, `winget` ou
// `scoop`. Toutes les sous-commandes ci-dessous s'emploient de la même façon quel que
// soit celui-ci.
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
	"flag"
	"fmt"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/facade"
	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/prompt"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

var version = "0.8.0"

// motsReserves sont les sous-commandes internes de jigger. Tout autre premier mot est un
// verbe de façade.
//
// Contrainte permanente : aucune sous-commande interne future ne peut porter le nom d'un
// verbe canonique. Si « jigger list » devait un jour désigner un usage interne, c'est le
// mot interne qui change — pas le verbe.
var motsReserves = map[string]bool{
	"pick": true, "render": true, "complete": true,
	"prompt": true, "warm": true, "demo": true,
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
	fmt.Fprintln(os.Stderr, "usage: jigger <verbe> [--pm <gestionnaire>] [--json] [--yes] [arguments…]")
	fmt.Fprintln(os.Stderr, "       jigger pick|complete \"<ligne>\" | jigger render --line \"<ligne>\"")
	fmt.Fprintln(os.Stderr, "       jigger prompt [--refresh [--wait]|--path] | jigger warm [--all|--installed]")
}

// optsCLI rassemble les drapeaux que jigger interprète lui-même. Tous les autres mots en
// « -- » sont passés au gestionnaire : `jg install --cask firefox` doit marcher.
type optsCLI struct {
	PM   string
	JSON bool
	Yes  bool
}

func separerDrapeaux(argv []string) (verbe string, args []string, o optsCLI, err error) {
	if len(argv) == 0 {
		return "", nil, o, fmt.Errorf("aucun verbe")
	}
	verbe = argv[0]
	for i := 1; i < len(argv); i++ {
		switch argv[i] {
		case "--pm":
			if i+1 >= len(argv) {
				return "", nil, o, fmt.Errorf("jigger : --pm attend un nom de gestionnaire")
			}
			i++
			o.PM = argv[i]
		case "--json":
			o.JSON = true
		case "--yes":
			o.Yes = true
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

	rows, code := facade.Executer(verbe, cibles, facade.Opts{JSON: o.JSON, Yes: o.Yes})
	if len(rows) > 0 || facade.Normalise(verbe) {
		fmt.Print(facade.Formater(verbe, rows, o.JSON))
	}
	return code
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
		fmt.Fprintf(os.Stderr, "jigger : « %s » — connu de plusieurs gestionnaires :\n", amb.Nom)
		for _, c := range amb.Candidats {
			fmt.Fprintf(os.Stderr, "        %s\n", c.Mgr.Cmd())
		}
		fmt.Fprintln(os.Stderr, "        Choisis avec --pm <gestionnaire>")
		return "", false
	}
	defer tty.Close()

	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)
	titre := fmt.Sprintf("%s : %d gestionnaires", amb.Nom, len(amb.Candidats))
	model := ui.New(titre, complete.Result{Executable: true, Items: items})
	// Pied propre à la désambiguïsation : on choisit un gestionnaire, on n'insère ni
	// n'exécute une commande — ⇥/↩ n'y ont pas de sens (spec §3, README).
	model.Keys = []ui.Key{{Key: "↵", Label: "choisir"}, {Key: "^G", Label: "annuler"}}

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
	tout := fs.Bool("all", false, "refait tout, même ce qui est encore frais")
	installes := fs.Bool("installed", false, "refait les seules listes de paquets installés")
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
			fmt.Fprintf(os.Stderr, "jigger warm (%s) : %v\n", m.Cmd(), err)
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
	refresh := fs.Bool("refresh", false, "interroge brew et réécrit le cache (lent)")
	wait := fs.Bool("wait", false, "avec --refresh : attend le verrou au lieu de renoncer")
	path := fs.Bool("path", false, "imprime le chemin du fichier de cache")
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
			fmt.Fprintln(os.Stderr, "jigger prompt --refresh :", err)
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
	line := fs.String("line", "", "ligne à compléter (jusqu'au curseur)")
	sel := fs.Int("sel", 0, "index du candidat courant")
	cols := fs.Int("cols", 0, "largeur du terminal")
	rows := fs.Int("rows", 8, "nombre de candidats affichés")
	color := fs.String("color", "auto", "profil couleur : auto|never|16|256|truecolor")
	focus := fs.Bool("focus", false, "le popup a le clavier : les flèches lui reviennent")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	lipgloss.SetColorProfile(colorProfile(*color))

	res := complete.Complete(*line)

	// Le pied dit où ira la prochaine flèche : sans le focus, ↓ sert à entrer dans la
	// liste (↑ reste l'historique) ; avec, les deux parcourent les candidats.
	navigation := ui.Key{Key: "↓", Label: "parcourir"}
	if *focus {
		navigation = ui.Key{Key: "↑↓", Label: "naviguer"}
	}

	frame := ui.Frame{
		Title:   res.Title(),
		Items:   res.Items,
		Rows:    *rows,
		Focused: *focus,
		Keys: []ui.Key{
			{Key: "⇥", Label: "insérer"},
			navigation,
			{Key: "^G", Label: "fermer"},
		},
	}
	if *cols > 0 {
		frame.Width = min(58, *cols-2)
	}

	// Contexte paquet, mot vide : on invite à filtrer plutôt que d'égrener le catalogue.
	if res.Executable && res.Word == "" && len(res.Items) > tropDeCandidats {
		frame.Items = nil
		frame.Empty = fmt.Sprintf("tapez pour filtrer… (%d paquets)", len(res.Items))
	} else if len(res.Items) == 0 {
		// Le gestionnaire a parfois mieux à dire qu'« aucun candidat » — un catalogue
		// winget encore en cours de constitution, par exemple.
		frame.Empty = "aucun candidat"
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

	titre := "brew install"
	items := []complete.Item{
		{Name: "git", Badge: pm.BadgeFormula, Installed: true, Version: "2.55.0"},
		{Name: "gitui", Badge: pm.BadgeFormula},
		{Name: "gh", Badge: pm.BadgeFormula, Installed: true, Version: "2.62.0"},
		{Name: "git-delta", Badge: pm.BadgeFormula},
		{Name: "google-chrome", Badge: pm.BadgeCask},
		{Name: "firefox", Badge: pm.BadgeCask},
	}
	if runtime.GOOS == "windows" {
		titre = "winget install"
		items = []complete.Item{
			{Name: "Git.Git", Badge: pm.BadgeWinget, Installed: true, Version: "2.55.0"},
			{Name: "GitHub.cli", Badge: pm.BadgeWinget, Installed: true, Version: "2.62.0"},
			{Name: "GitHub.GitHubDesktop", Badge: pm.BadgeWinget},
			{Name: "dandavison.delta", Badge: pm.BadgeWinget},
			{Name: "Google.Chrome", Badge: pm.BadgeWinget},
			{Name: "Canon.PrinterDriver", Badge: pm.BadgeOther, Installed: true, Version: "1.02"},
		}
	}

	fmt.Println(ui.New(titre, complete.Result{Executable: true, Items: items}).View())
}
