// Package complete analyse une ligne de commande brew et propose des candidats
// contextuels : sous-commandes, options (--…) ou noms de paquets selon la position,
// et — pour uninstall/reinstall… — seulement les paquets installés.
package complete

import (
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
)

// Item est un candidat de complétion.
type Item struct {
	Name      string
	Badge     string // "F" (formula), "C" (cask), "" (sous-commande / option)
	Installed bool
	Version   string // version installée (vide si non installé/inconnue)
}

// Result décrit le contexte de complétion et les candidats.
type Result struct {
	Prefix     string // texte avant le mot courant (pour reconstruire la ligne)
	Word       string // mot en cours de complétion
	Sub        string // sous-commande courante (« install », « uninstall »…)
	Items      []Item
	Executable bool // contexte paquet → accepter complète une commande exécutable
}

// Sous-commandes proposées pour le premier mot.
var subcommands = []string{
	"install", "uninstall", "reinstall", "upgrade", "info", "search", "list", "deps",
	"uses", "pin", "unpin", "tap", "untap", "outdated", "cleanup", "autoremove", "doctor",
	"services", "link", "unlink", "home", "desc", "leaves", "missing", "update",
}

// Options communes à toutes les sous-commandes.
var commonOptions = []string{"--help", "--verbose", "--debug", "--quiet"}

// Options spécifiques par sous-commande.
var optionsBySub = map[string][]string{
	"install":   {"--cask", "--formula", "--force", "--HEAD", "--build-from-source", "--no-quarantine", "--dry-run"},
	"reinstall": {"--cask", "--formula", "--force", "--no-quarantine"},
	"uninstall": {"--cask", "--formula", "--force", "--zap", "--ignore-dependencies"},
	"upgrade":   {"--cask", "--formula", "--greedy", "--dry-run", "--force"},
	"list":      {"--versions", "--cask", "--formula", "--pinned", "--full-name", "-1"},
	"outdated":  {"--cask", "--formula", "--greedy", "--verbose", "--json"},
	"info":      {"--json", "--cask", "--formula", "--github"},
	"deps":      {"--tree", "--installed", "--cask", "--formula", "--include-build"},
	"uses":      {"--installed", "--recursive", "--cask", "--formula"},
	"search":    {"--cask", "--formula", "--desc"},
	"cleanup":   {"--prune", "--dry-run", "-s"},
	"services":  {"--all"},
}

// Sous-commandes dont les arguments sont des paquets déjà installés.
var installedOnly = map[string]bool{
	"uninstall": true, "remove": true, "rm": true, "reinstall": true, "upgrade": true,
	"pin": true, "unpin": true, "link": true, "unlink": true, "uses": true,
}

func optionsFor(sub string) []string {
	if specific, ok := optionsBySub[sub]; ok {
		return append(append([]string{}, specific...), commonOptions...)
	}
	return commonOptions
}

// Complete calcule le contexte et les candidats pour la ligne donnée.
func Complete(line string, cat *brew.Catalog) Result {
	var prefix, word string
	if i := strings.LastIndex(line, " "); i < 0 {
		word = line
	} else {
		prefix, word = line[:i+1], line[i+1:]
	}

	before := strings.Fields(strings.TrimSpace(prefix))
	if len(before) > 0 && strings.EqualFold(before[0], "brew") {
		before = before[1:]
	}
	sub := ""
	if len(before) > 0 {
		sub = strings.ToLower(before[0])
	}

	firstWord := len(before) == 0
	isOption := strings.HasPrefix(word, "-")
	isPackage := !isOption && !firstWord

	res := Result{Prefix: prefix, Word: word, Sub: sub, Executable: isPackage}
	lw := strings.ToLower(word)

	switch {
	case isOption:
		for _, o := range optionsFor(sub) {
			if strings.HasPrefix(strings.ToLower(o), lw) {
				res.Items = append(res.Items, Item{Name: o})
			}
		}
	case firstWord:
		for _, s := range subcommands {
			if strings.HasPrefix(s, lw) {
				res.Items = append(res.Items, Item{Name: s})
			}
		}
	default: // paquet
		var pool []string
		if installedOnly[sub] {
			for n := range cat.Installed {
				pool = append(pool, n)
			}
		} else {
			pool = append(pool, cat.Formulae...)
			pool = append(pool, cat.Casks...)
		}
		sort.Strings(pool)
		for _, n := range pool {
			if strings.HasPrefix(strings.ToLower(n), lw) {
				res.Items = append(res.Items, Item{Name: n, Badge: badge(cat, n), Installed: cat.Installed[n], Version: cat.Version(n)})
			}
		}
	}
	return res
}

func badge(cat *brew.Catalog, name string) string {
	if cat.IsCask(name) && !cat.IsFormula(name) {
		return "C"
	}
	return "F"
}

// NeedsCask indique s'il faut ajouter --cask pour installer ce paquet (cask pur).
func NeedsCask(cat *brew.Catalog, name string) bool {
	return cat.IsCask(name) && !cat.IsFormula(name)
}
