package pacman

import (
	"bytes"
	"errors"
	"os/exec"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// L'état affiché par le bloc de prompt : la version de pacman, et les deux compteurs
// d'obsolètes. Tout ce fichier est **lent** — réseau compris — et n'est appelé que par
// `jigger prompt --refresh`, détaché, jamais dans le chemin d'un prompt ni d'une frappe.

// Version rend la version de pacman, sans le « v ».
func Version() (string, error) {
	out, err := pm.Run(Path("pacman"), "--version")
	if err != nil {
		return "", err
	}
	return ParseVersion(string(out)), nil
}

// Outdated compte les paquets des dépôts qui ont une mise à jour en attente.
//
// `checkupdates` (pacman-contrib) est préféré quand il est là, et pas seulement par
// commodité : il synchronise une base **privée**, sans toucher à /var/lib/pacman/sync. Un
// `pacman -Sy` lancé pour compter laisserait le système dans l'état « base à jour, paquets
// anciens » — le moyen le plus sûr de casser une installation Arch au prochain install.
// jigger ne lance donc jamais `-Sy` lui-même.
//
// Sans checkupdates, on se rabat sur `pacman -Qu` : instantané, sans réseau, mais il ne dit
// que ce que la dernière synchronisation faite par l'utilisateur sait. C'est exactement la
// réserve que scoop porte déjà — ses buckets datent du dernier `scoop update`.
func Outdated() (int, error) {
	if Present("checkupdates") {
		return compter(Path("checkupdates"))
	}
	return compter(Path("pacman"), "-Qu")
}

// OutdatedAUR compte les paquets AUR à mettre à jour. Zéro sans yay : la machine n'a alors
// pas d'AUR, et le prompt masque ce qu'il ne sait pas plutôt que d'afficher un compteur qui
// n'a aucun sens (même règle que scoop sur une machine sans scoop).
func OutdatedAUR() (int, error) {
	if !Present("yay") {
		return 0, nil
	}
	return compter(Path("yay"), "-Qua")
}

// compter lance une commande qui liste « nom ancienne -> nouvelle » et compte ses lignes.
//
// Les codes de sortie du « rien à faire » diffèrent d'un outil à l'autre — 2 pour
// checkupdates, 1 pour `pacman -Qu` et `yay -Qua` — et aucun n'est une panne. Une sortie
// vide les couvre tous les trois sans avoir à tenir une table de codes par binaire.
func compter(bin string, args ...string) (int, error) {
	out, err := pm.Run(bin, args...)
	if err != nil {
		var sortie *exec.ExitError
		if !errors.As(err, &sortie) || len(bytes.TrimSpace(out)) > 0 {
			return 0, err
		}
		return 0, nil // rien à mettre à jour
	}
	rows, perr := parseOutdated(out)
	if perr != nil {
		return 0, perr
	}
	return len(rows), nil
}
