// Package complete analyse une ligne de commande et propose des candidats contextuels :
// sous-commandes, options (--…) ou noms de paquets selon la position, et — pour
// uninstall, upgrade… — seulement les paquets installés.
//
// Rien ici ne connaît Homebrew, winget ou scoop : c'est le premier mot de la ligne qui
// désigne le gestionnaire (cf. internal/managers), lequel fournit ses sous-commandes,
// ses options et son catalogue.
package complete

import (
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Item est un candidat de complétion. C'est le type du gestionnaire : le popup affiche
// tel quel ce que celui-ci a produit.
type Item = pm.Item

// Result décrit le contexte de complétion et les candidats.
type Result struct {
	Prefix     string // texte avant le mot courant (pour reconstruire la ligne)
	Word       string // mot en cours de complétion
	Cmd        string // gestionnaire : « brew », « winget », « scoop »
	Sub        string // sous-commande courante (« install », « uninstall »…)
	Items      []Item
	Executable bool   // contexte paquet → accepter complète une commande exécutable
	Note       string // message à afficher à la place de « aucun candidat »

	mgr pm.Manager
	cat *pm.Catalog
}

// Insert rend le texte à insérer pour un candidat : le nom, éventuellement corrigé par
// le gestionnaire (`--cask` de brew, qualification par bucket de scoop).
func (r Result) Insert(name string) string {
	if r.mgr == nil || r.cat == nil {
		return name
	}
	return r.mgr.Insert(r.cat, r.Sub, r.Prefix, name)
}

// Title résume le contexte, en tête du popup : « winget install ».
func (r Result) Title() string {
	if r.Sub == "" {
		return r.Cmd
	}
	return r.Cmd + " " + r.Sub
}

// Complete calcule le contexte et les candidats pour la ligne donnée, en interrogeant
// le gestionnaire qu'elle nomme.
func Complete(line string) Result {
	m := managers.Detect(line)
	return CompleteWith(line, m, m.Load())
}

// CompleteWith est Complete sur un gestionnaire et un catalogue donnés (tests, et tout
// appelant qui a déjà chargé le catalogue).
func CompleteWith(line string, m pm.Manager, cat *pm.Catalog) Result {
	var prefix, word string
	if i := strings.LastIndex(line, " "); i < 0 {
		word = line
	} else {
		prefix, word = line[:i+1], line[i+1:]
	}

	before := strings.Fields(strings.TrimSpace(prefix))
	if len(before) > 0 && strings.EqualFold(motCommande(before[0]), m.Cmd()) {
		before = before[1:]
	}
	sub := ""
	if len(before) > 0 {
		sub = strings.ToLower(before[0])
	}

	firstWord := len(before) == 0
	isOption := strings.HasPrefix(word, "-")
	isPackage := !isOption && !firstWord

	res := Result{
		Prefix: prefix, Word: word, Cmd: m.Cmd(), Sub: sub,
		Executable: isPackage, mgr: m, cat: cat,
	}
	lw := strings.ToLower(word)

	switch {
	case isOption:
		for _, o := range m.Options(sub) {
			if strings.HasPrefix(strings.ToLower(o), lw) {
				res.Items = append(res.Items, Item{Name: o})
			}
		}
	case firstWord:
		for _, s := range m.Subcommands() {
			if strings.HasPrefix(s, lw) {
				res.Items = append(res.Items, Item{Name: s})
			}
		}
	default: // paquet
		pool := cat.Names
		if m.InstalledOnly(sub) {
			pool = cat.InstalledNames()
		}
		for _, n := range pool {
			if strings.HasPrefix(strings.ToLower(n), lw) {
				res.Items = append(res.Items, Item{
					Name:      n,
					Badge:     cat.Badge(n),
					Installed: cat.Installed[n],
					Version:   cat.Version(n),
				})
			}
		}
		if len(res.Items) == 0 {
			res.Note = cat.Note
		}
	}
	return res
}

// motCommande ramène un mot à un nom de commande : sans chemin ni extension. Une ligne
// peut très bien commencer par `C:\Users\…\scoop\shims\scoop.exe`.
func motCommande(s string) string {
	s = strings.Trim(s, `"'`)
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	return strings.TrimSuffix(strings.ToLower(s), ".exe")
}
