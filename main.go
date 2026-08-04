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

	model := ui.New(title(res), res)
	prog := tea.NewProgram(model, tea.WithInput(tty), tea.WithOutput(tty))
	final, err := prog.Run()
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

// title résume le contexte affiché en tête du sélecteur.
func title(res complete.Result) string {
	if res.Sub == "" {
		return "brew"
	}
	return "brew " + res.Sub
}

