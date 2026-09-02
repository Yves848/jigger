package pacman

import (
	"bufio"
	"bytes"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Les trois analyseurs du module. Fonctions pures : elles ne lancent rien, et se testent
// sur des sorties réellement capturées.
//
// pacman n'a pas de sortie machine — pas de JSON, pas de `--porcelain` — mais il n'en a pas
// besoin : ses colonnes sont séparées par des espaces simples, sans largeur fixe, sans
// en-tête, et sans ANSI dès que la sortie n'est pas un terminal. C'est l'inverse exact de
// la leçon scoop, et c'est ce qui rend ce module bon marché.

// parseList lit « nom version-release », une ligne par paquet (`pacman -Q`).
func parseList(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		champs := strings.Fields(ligne)
		if len(champs) < 2 {
			continue
		}
		rows = append(rows, pm.Package{Name: champs[0], Version: champs[1], Kind: pm.BadgeRepo})
	}
	return rows, nil
}

// parseOutdated lit « nom ancienne -> nouvelle » (`pacman -Qu`, `yay -Qua`).
//
// Les paquets tenus par IgnorePkg sont annoncés par la même ligne suivie de « [ignored] » :
// ils ne bougeront pas au prochain -Syu, donc les compter serait annoncer une mise à jour
// qui n'arrivera pas — c'est la règle que scoop applique déjà à `scoop hold`.
func parseOutdated(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		if strings.Contains(ligne, "[ignored]") {
			continue
		}
		champs := strings.Fields(ligne)
		if len(champs) < 4 || champs[2] != "->" {
			continue
		}
		rows = append(rows, pm.Package{
			Name:      champs[0],
			Version:   champs[1],
			Available: champs[3],
			Kind:      pm.BadgeRepo,
		})
	}
	return rows, nil
}

// parseSearch lit la sortie de `pacman -Ss` et de `yay -Ss`, qui donnent deux lignes par
// résultat :
//
//	extra/zsh 5.9.2-1 [installed]
//	    A very advanced and programmable command interpreter (shell) for UNIX
//	aur/yay-git 13.0.1.r0-1 (+1234 5.67%) (Installed)
//	    Yet another yogurt…
//
// La description est **indentée**, et c'est le seul critère fiable pour l'écarter : son
// texte peut contenir n'importe quoi, barre oblique comprise. Elle impose de scanner les
// octets bruts plutôt que de passer par pm.SplitLines, qui déshabille les lignes de leurs
// espaces — et effacerait précisément le signal qu'on lit.
//
// Le préfixe « dépôt/ » du premier champ donne la provenance sans travail
// supplémentaire, et « aur/ » distingue l'AUR des dépôts binaires.
func parseSearch(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		ligne := sc.Text()
		if ligne == "" || ligne[0] == ' ' || ligne[0] == '\t' {
			continue // description, ou ligne vide entre deux résultats
		}
		champs := strings.Fields(ligne)
		if len(champs) < 2 || !strings.Contains(champs[0], "/") {
			continue
		}
		depot, nom, _ := strings.Cut(champs[0], "/")
		badge := pm.BadgeRepo
		if depot == "aur" {
			badge = pm.BadgeAUR
		}
		rows = append(rows, pm.Package{
			Name:      nom,
			Available: champs[1],
			Kind:      badge,
			Source:    depot,
		})
	}
	return rows, sc.Err()
}

// ParseVersion extrait « 7.1.0 » de la sortie de `pacman --version`, dont la première
// ligne utile est « Pacman v7.1.0 - libalpm v16.0.1 » — précédée d'un dessin ASCII.
//
// Une sortie inattendue donne une chaîne vide : le prompt masque alors le bloc plutôt que
// d'afficher n'importe quoi (même contrat que prompt.ParseVersion pour brew).
func ParseVersion(out string) string {
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		for _, mot := range strings.Fields(sc.Text()) {
			if len(mot) < 2 || mot[0] != 'v' || mot[1] < '0' || mot[1] > '9' {
				continue
			}
			return mot[1:] // « v7.1.0 » → « 7.1.0 »
		}
	}
	return ""
}
