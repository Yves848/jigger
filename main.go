// jigger — assistance Homebrew pour le terminal : complétion contextuelle et
// sélecteur interactif (Bubble Tea), pensés pour être branchés dans zsh.
//
// Sous-commandes :
//
//	jigger pick "<ligne>"      sélecteur interactif ; imprime la nouvelle ligne.
//	                           Code de sortie : 0 = insérer (⇥), 10 = exécuter (↩),
//	                           2 = annulé (rien imprimé).
//	jigger render --line "…"   popup sans état ni clavier : une ligne de métadonnées
//	                           puis le cadre. C'est ce que le widget zsh appelle à
//	                           chaque frappe pour le popup vivant.
//	jigger complete "<ligne>"  candidats (un par ligne) pour une complétion classique.
//	jigger prompt              état de Homebrew en cache, pour le bloc oh-my-posh.
//	                           --refresh interroge brew et réécrit le cache (lent, à
//	                           lancer détaché), --wait lui fait attendre le verrou
//	                           plutôt que renoncer ; --path imprime le fichier de cache.
//	jigger --version           version.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/prompt"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

var version = "0.4.3"

func main() {
	ui.Version = version // affichée dans l'en-tête du sélecteur (repère du binaire lancé)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
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
	case "demo":
		runDemo()
	case "--version", "-v", "version":
		fmt.Println("jigger", version)
	default:
		usage()
		os.Exit(2)
	}
}

func arg(i int) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return ""
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: jigger pick|complete \"<ligne brew>\" | jigger render --line \"<ligne brew>\" | jigger prompt [--refresh [--wait]|--path]")
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
	res := complete.Complete(line, brew.Load())
	for _, it := range res.Items {
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
	if err := fs.Parse(args); err != nil {
		return 2
	}

	lipgloss.SetColorProfile(colorProfile(*color))

	res := complete.Complete(*line, brew.Load())

	frame := ui.Frame{
		Title: title(res),
		Items: res.Items,
		Rows:  *rows,
		Keys: []ui.Key{
			{Key: "⇥", Label: "insérer"},
			{Key: "^N ^P", Label: "naviguer"},
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
		frame.Empty = "aucun candidat"
	}

	left := *line
	if len(frame.Items) > 0 {
		frame.Sel = min(max(*sel, 0), len(frame.Items)-1)
		frame.Offset = ui.ScrollOffset(frame.Sel, len(frame.Items), *rows)
		left = res.Prefix + insertText(res, frame.Items[frame.Sel].Name)
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

// runPick lance le sélecteur et imprime la nouvelle ligne. Le TUI dessine sur /dev/tty
// pour laisser stdout au résultat (comme fzf).
func runPick(line string) int {
	res := complete.Complete(line, brew.Load())
	if len(res.Items) == 0 {
		return 1
	}

	// Candidat unique : il n'y a rien à choisir. On insère directement, sans ouvrir le
	// popup (ni même le TTY) — comme le fait la complétion zsh sur une correspondance
	// unique.
	if len(res.Items) == 1 {
		fmt.Print(res.Prefix + insertText(res, res.Items[0].Name))
		return 0
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 1
	}
	defer tty.Close()

	// stdout est capturé par le widget → lipgloss croirait à un pipe sans couleur.
	// On fixe le profil couleur d'après le vrai terminal (le TTY).
	lipgloss.SetColorProfile(termenv.NewOutput(tty).Profile)

	model := ui.New(title(res), res)
	// Popup « classique » : rendu inline, SOUS la ligne de commande — qui reste
	// visible. On descend d'une ligne (\r\n) pour que le cadre se dessine dessous (pas
	// d'écran alterné). Bubble Tea gère ensuite son cadre en mouvements *relatifs*
	// (robuste au défilement) et, à la sortie, l'efface lui-même : le modèle rend une
	// vue vide quand `quitting` (cf. picker.View) → EraseScreenBelow. On remonte enfin
	// d'une ligne pour reposer le curseur sur la ligne de commande ; le widget zsh la
	// redessine (reset-prompt).
	fmt.Fprint(tty, "\r\n")
	prog := tea.NewProgram(model, tea.WithInput(tty), tea.WithOutput(tty))
	final, err := prog.Run()
	fmt.Fprint(tty, "\x1b[1A\r") // curseur → début de la ligne de commande
	if err != nil {
		return 1
	}

	m := final.(ui.Model)
	if m.Chosen == nil {
		return 2 // annulé
	}

	fmt.Print(res.Prefix + insertText(res, m.Chosen.Name))
	if m.Execute {
		return 10 // ↩ : commande à exécuter
	}
	return 0 // ⇥ : insérée
}

// insertText renvoie le texte à insérer, avec --cask automatique pour un cask pur
// installé/réinstallé (sauf si --cask/--formula déjà présents).
func insertText(res complete.Result, name string) string {
	needCask := (res.Sub == "install" || res.Sub == "reinstall") &&
		!strings.Contains(res.Prefix, "--cask") && !strings.Contains(res.Prefix, "--formula")
	// NeedsCask est recalculé via le badge : un cask pur porte le badge "C".
	if needCask {
		for _, it := range res.Items {
			if it.Name == name && it.Badge == "C" {
				return "--cask " + name
			}
		}
	}
	return name
}

// runDemo imprime un aperçu statique du sélecteur (pour prévisualiser le rendu sans
// installer le widget). stdout est ici un vrai terminal → couleurs détectées.
func runDemo() {
	lipgloss.SetColorProfile(termenv.TrueColor) // aperçu toujours coloré
	res := complete.Result{
		Executable: true,
		Items: []complete.Item{
			{Name: "git", Badge: "F", Installed: true, Version: "2.55.0"},
			{Name: "gitui", Badge: "F"},
			{Name: "gh", Badge: "F", Installed: true, Version: "2.62.0"},
			{Name: "git-delta", Badge: "F"},
			{Name: "google-chrome", Badge: "C"},
			{Name: "firefox", Badge: "C"},
		},
	}
	fmt.Println(ui.New("brew install", res).View())
}

// title résume le contexte affiché en tête du sélecteur.
func title(res complete.Result) string {
	if res.Sub == "" {
		return "brew"
	}
	return "brew " + res.Sub
}
