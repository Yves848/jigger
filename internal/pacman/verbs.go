package pacman

import (
	"bytes"
	"errors"
	"os/exec"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// yayPresent est une variable de paquet, et non un appel direct : la table des verbes
// dépend de l'environnement (cf. ADR-0007), et les tests doivent pouvoir éprouver les deux
// branches sans dépendre de la machine qui les fait tourner.
var yayPresent = func() bool { return Present("yay") }

// Verbs déclare ce que chacun des deux fournisseurs sait faire — ADR-0007, « pacman lit,
// yay pilote ».
//
// yay, quand il est là, est le seul des deux à piloter. Deux tables identiques déclarées en
// même temps feraient lister **deux fois** les mêmes paquets — Router lance tous les
// gestionnaires capables d'un verbe sans nom à résoudre — et rendraient `jg install fd`
// ambigu entre deux portes sur la même base alpm.
//
// Sans yay, pacman déclare les quatre verbes de **lecture**, ceux qui n'exigent pas root.
// Les verbes mutants restent absents : pacman ne sait pas les rendre sans élévation, et
// jigger n'élève rien (ADR-0004). Un verbe absent est un verbe qu'on ne sait pas rendre —
// c'est tout ce que le modèle de capacités demande.
func (m Manager) Verbs() map[pm.Verb]pm.Binding {
	if m.cmd == "yay" {
		return verbesYay()
	}
	if yayPresent() {
		return nil
	}
	return verbesLecture("pacman")
}

// verbesLecture rend les quatre verbes que pacman comme yay servent sans rien changer à la
// machine. Le binaire est passé parce que les liaisons de lecture prennent elles-mêmes le
// sous-processus en charge (cf. lire).
func verbesLecture(bin string) map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"list":     {Direct: lire(bin, parseList, "-Q"), Pool: pm.PoolAucun},
		"outdated": {Direct: lire(bin, parseOutdated, "-Qu"), Pool: pm.PoolAucun},
		// search prend une requête, pas un nom à résoudre au catalogue : PoolAucun, comme
		// chez brew, sans quoi jigger refuserait de chercher un mot qui n'est justement
		// pas encore un nom connu.
		"search": {Direct: lire(bin, parseSearch, "-Ss"), Pool: pm.PoolAucun},
		"info":   {Native: []string{"-Si", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
	}
}

func verbesYay() map[pm.Verb]pm.Binding {
	v := verbesLecture("yay")
	// yay -Qu couvre dépôts ET AUR, contrairement à pacman -Qu : c'est tout l'intérêt de
	// le laisser piloter.
	v["install"] = pm.Binding{Native: []string{"-S", pm.MarqueurTous}, Pool: pm.PoolCatalogue}
	v["uninstall"] = pm.Binding{Native: []string{"-Rns", pm.MarqueurTous}, Pool: pm.PoolInstalles}
	v["upgrade"] = pm.Binding{Native: []string{"-Syu", pm.MarqueurTous}, Pool: pm.PoolInstalles}
	v["cleanup"] = pm.Binding{Native: []string{"-Sc"}, Pool: pm.PoolAucun}
	return v
}

// lire prend en charge le sous-processus d'un verbe de lecture, au lieu de le laisser à la
// façade, pour une seule raison : **pacman rend 1 quand il ne trouve rien**. `pacman -Qu`
// sur une machine à jour, `pacman -Ss <mot introuvable>` : code 1, sortie vide. La façade
// lit ce code comme une panne et écrit « erreur du gestionnaire » là où il n'y a qu'une
// bonne nouvelle.
//
// pm.Binding n'a pas de champ pour dire « ce code-là n'en est pas un », et lui en ajouter un
// pour un seul gestionnaire serait payer cher une particularité. La liaison assume donc son
// code de sortie elle-même — et **seulement** la signature exacte du « rien trouvé » : code
// 1 et sortie vide. Tout autre échec reste un échec.
//
// C'est aussi pourquoi ces liaisons remplissent PM : la branche Direct de la façade rend les
// lignes telles quelles, là où la branche Parse les estampille au passage.
func lire(bin string, parse pm.Parser, args ...string) func([]string) ([]pm.Package, error) {
	return func(extra []string) ([]pm.Package, error) {
		out, err := pm.Run(Path(bin), append(append([]string{}, args...), extra...)...)
		if err != nil {
			var sortie *exec.ExitError
			if !errors.As(err, &sortie) || sortie.ExitCode() != 1 || len(bytes.TrimSpace(out)) > 0 {
				return nil, err
			}
			return nil, nil // rien trouvé : ce n'est pas une panne
		}
		rows, perr := parse(out)
		if perr != nil {
			return nil, perr
		}
		for i := range rows {
			rows[i].PM = bin
		}
		return rows, nil
	}
}
