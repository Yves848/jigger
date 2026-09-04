// Package managers rassemble les gestionnaires de paquets connus de jigger et désigne
// celui dont parle une ligne de commande.
//
// jigger n'est pas « un outil Homebrew » ou « un outil winget » : c'est le premier mot
// de la ligne qui décide. Sur une même machine, `winget install …` et `scoop install …`
// ouvrent donc le même popup, avec chacun ses paquets, ses sous-commandes et ses
// options.
package managers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/pacman"
	"gitlab.yg-devworks.com/yves/jigger/internal/plugin"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/ssh"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// All rend les gestionnaires reconnus, quelle que soit la plateforme : c'est la
// présence du mot dans la ligne qui compte, pas celle du binaire. Un `brew` tapé sous
// Windows (WSL, Git Bash…) reste ainsi complété — au pire sur un catalogue vide.
//
// Les gestionnaires natifs sont placés en premier : ils ont priorité sur les plugins
// lorsqu'ils portent le même nom (un plugin ne peut pas écraser brew, winget…).
func All() []pm.Manager {
	out := []pm.Manager{
		brew.New(), winget.New(), scoop.New(),
		// pacman et yay : deux portes sur la même base alpm, donc deux fournisseurs qui
		// partagent tout sauf le catalogue AUR et la table des verbes (ADR-0007).
		pacman.New("pacman"), pacman.New("yay"),
		// Trois fournisseurs plutôt qu'un à trois noms : Manager.Cmd() ne rend qu'un
		// mot, et l'élargir obligerait brew, winget et scoop à répondre à une question
		// qu'ils ne se posent pas. Ils partagent implémentation et catalogue.
		ssh.New("ssh"), ssh.New("scp"), ssh.New("sftp"),
	}
	// Plugins tiers découverts dynamiquement. Un plugin qui reprend le mot d'un
	// gestionnaire natif est écarté : laisser deux « brew » dans la liste ferait exécuter
	// la ligne deux fois, et le routage n'a aucun moyen de les départager.
	natifs := make(map[string]bool, len(out))
	for _, m := range out {
		natifs[m.Cmd()] = true
	}
	for _, p := range plugin.Discover() {
		if natifs[p.Cmd()] {
			fmt.Fprintf(os.Stderr, "jigger : le plugin %q est ignoré, ce nom est déjà celui d'un gestionnaire natif\n", p.Cmd())
			continue
		}
		out = append(out, p)
	}
	return out
}

// Available rend les gestionnaires réellement installés sur la machine. C'est la liste
// que `jigger warm` parcourt : inutile d'aller demander son catalogue à un brew qui
// n'existe pas.
func Available() []pm.Manager {
	var dispo []pm.Manager
	for _, m := range All() {
		if m.Available() {
			dispo = append(dispo, m)
		}
	}
	return dispo
}

// For rend le gestionnaire d'un mot de commande.
func For(cmd string) (pm.Manager, bool) {
	cmd = mot(cmd)
	for _, m := range All() {
		if m.Cmd() == cmd {
			return m, true
		}
	}
	return nil, false
}

// Default rend le gestionnaire retenu quand la ligne ne nomme personne — c'est le cas
// de `jigger complete "install fire"`, appelé sans le mot de commande.
func Default() pm.Manager {
	switch {
	case runtime.GOOS == "windows":
		return winget.New()
	case runtime.GOOS == "linux" && pacman.Present("pacman"):
		return pacman.New("pacman")
	}
	return brew.New()
}

// Detect rend le gestionnaire dont parle une ligne.
func Detect(line string) pm.Manager {
	premier, _, _ := strings.Cut(strings.TrimSpace(line), " ")
	if m, ok := For(premier); ok {
		return m
	}
	return Default()
}

// mot ramène un premier mot à un nom de commande : sans chemin, sans extension, en
// minuscules. `C:\Users\…\scoop\shims\scoop.exe` est bien scoop.
func mot(s string) string {
	s = strings.Trim(s, `"'`)
	s = filepath.Base(filepath.FromSlash(s))
	s = strings.TrimSuffix(strings.ToLower(s), ".exe")
	return s
}

// Tables rend, pour chaque verbe, les gestionnaires qui savent le rendre. C'est le modèle
// de capacités vu depuis le verbe : la clé existe si au moins un gestionnaire la déclare.
func Tables(mgrs []pm.Manager) map[pm.Verb][]pm.Manager {
	out := map[pm.Verb][]pm.Manager{}
	for _, m := range mgrs {
		b, ok := m.(pm.Bindings)
		if !ok {
			continue // on sait le compléter sans savoir le piloter
		}
		for v := range b.Verbs() {
			out[v] = append(out[v], m)
		}
	}
	return out
}

// Vocabulaire rend les verbes de premier niveau proposés par les gestionnaires donnés,
// triés et dédupliqués. « source add » y figure comme « source » : le popup complète le
// premier mot, puis le second.
func Vocabulaire(mgrs []pm.Manager) []string {
	vu := map[string]bool{}
	for v := range Tables(mgrs) {
		premier, _, _ := strings.Cut(string(v), " ")
		vu[premier] = true
	}
	out := make([]string, 0, len(vu))
	for v := range vu {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
