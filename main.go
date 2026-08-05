// jigger — assistance Homebrew pour le terminal : complétion contextuelle et
// sélecteur interactif (Bubble Tea), pensés pour être branchés dans zsh.
//
// Sous-commandes :
//
//	jigger pick "<ligne>"      sélecteur interactif ; imprime la nouvelle ligne.
//	                           Code de sortie : 0 = insérer (⇥), 10 = exécuter (↩),
//	                           2 = annulé (rien imprimé).
//	jigger complete "<ligne>"  candidats (un par ligne) pour une complétion classique.
//	jigger --version           version.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "pick":
		os.Exit(runPick(arg(2)))
	case "complete":
		runComplete(arg(2))
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
	fmt.Fprintln(os.Stderr, "usage: jigger pick|complete \"<ligne brew>\"")
}

// runComplete imprime les candidats (complétion non interactive).
func runComplete(line string) {
	res := complete.Complete(line, brew.Load())
	for _, it := range res.Items {
		fmt.Println(it.Name)
	}
}

// runPick lance le sélecteur et imprime la nouvelle ligne. Le TUI dessine sur /dev/tty
// pour laisser stdout au résultat (comme fzf).
func runPick(line string) int {
	res := complete.Complete(line, brew.Load())
	if len(res.Items) == 0 {
		return 1
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
	// Popup « classique » : rendu inline, sous la ligne de commande (pas d'écran
	// alterné). Bubble Tea laisse sa dernière image affichée en sortie ; on l'efface
	// nous-mêmes juste après (cf. clearInline).
	prog := tea.NewProgram(model, tea.WithInput(tty), tea.WithOutput(tty))
	final, err := prog.Run()
	if err != nil {
		return 1
	}

	m := final.(ui.Model)
	// Effacement manuel du popup inline, quel que soit le choix (insérer, exécuter,
	// annuler) : on repart d'un terminal propre, le widget zsh redessine le prompt.
	clearInline(tty, strings.Count(m.View(), "\n")+1)

	if m.Chosen == nil {
		return 2 // annulé
	}

	fmt.Print(res.Prefix + insertText(res, m.Chosen.Name))
	if m.Execute {
		return 10 // ↩ : commande à exécuter
	}
	return 0 // ⇥ : insérée
}

// clearInline efface le popup rendu en inline. En sortie, Bubble Tea (renderer
// standard) a déjà vidé la dernière ligne du cadre et laissé le curseur en colonne 0
// dessus ; les height-1 lignes au-dessus restent affichées. On remonte donc jusqu'au
// haut du cadre puis on efface jusqu'au bas de l'écran.
func clearInline(w *os.File, height int) {
	if height > 1 {
		fmt.Fprintf(w, "\x1b[%dA", height-1) // curseur vers le haut du cadre
	}
	fmt.Fprint(w, "\x1b[J") // efface du curseur jusqu'au bas de l'écran
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
			{Name: "git", Badge: "F", Installed: true},
			{Name: "gitui", Badge: "F"},
			{Name: "gh", Badge: "F", Installed: true},
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
