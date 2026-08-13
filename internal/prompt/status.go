// Package prompt produit l'état de Homebrew affiché dans le prompt : la version de
// brew et le nombre de paquets obsolètes.
//
// Tout est bâti autour d'une contrainte : `brew outdated` coûte de une à cinq
// secondes, ce qui est inconcevable dans le chemin d'un prompt. L'état est donc
// calculé en arrière-plan et déposé dans un fichier de cache d'une seule ligne, que
// le hook zsh relit sans forker le moindre processus.
//
// Les fonctions d'analyse (ParseVersion, ParseOutdated, ParseLine) sont pures et
// exportées : elles se testent sur des sorties `brew` réellement capturées, sans
// lancer de processus.
package prompt

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
)

// ErrVerrouille signale qu'un autre rafraîchissement est déjà en cours. Ce n'est pas
// une panne : dix terminaux ouverts déclenchent dix `precmd`, un seul doit appeler brew.
var ErrVerrouille = errors.New("rafraîchissement déjà en cours")

// peremptionVerrou : au-delà, un verrou est réputé abandonné (processus tué avant
// d'avoir pu le retirer) et n'empêche plus de rafraîchir.
const peremptionVerrou = 5 * time.Minute

// Status est l'état de Homebrew tel qu'il est affiché — et mis en cache.
type Status struct {
	Version  string    // version de brew, sans le suffixe de commits : « 6.0.17 »
	Formulae int       // formulae obsolètes
	Casks    int       // casks obsolètes
	At       time.Time // instant du calcul, pour la péremption du cache
}

// Outdated est le total affiché par le prompt.
func (s Status) Outdated() int { return s.Formulae + s.Casks }

// Runner exécute `brew <args>` et renvoie sa sortie standard. Les tests en fournissent
// un factice ; la production utilise brewReel.
type Runner func(args ...string) ([]byte, error)

// File renvoie le chemin du fichier de cache dans le dossier donné.
func File(dir string) string { return filepath.Join(dir, "status") }

// Dir renvoie le dossier de cache utilisé en production (partagé avec le catalogue).
func Dir() string { return brew.CacheDir() }

// ParseVersion extrait « 6.0.17 » de la sortie de `brew --version`, qui prend deux
// formes selon qu'on est pile sur un tag (« Homebrew 4.5.4 ») ou quelques commits plus
// loin (« Homebrew 6.0.17-36-g6cf9b12 »). Une sortie inattendue donne une chaîne vide :
// le prompt masquera simplement le bloc plutôt que d'afficher n'importe quoi.
func ParseVersion(out string) string {
	ligne, _, _ := strings.Cut(out, "\n")
	champs := strings.Fields(ligne)
	if len(champs) < 2 || champs[0] != "Homebrew" {
		return ""
	}
	v, _, _ := strings.Cut(champs[1], "-")
	if v == "" || v[0] < '0' || v[0] > '9' {
		return ""
	}
	return v
}

// ParseOutdated compte les paquets obsolètes dans la sortie de
// `brew outdated --json=v2`. Ce format est préféré à `--quiet` : il est plus rapide ici
// et donne la répartition formulae/casks sans travail supplémentaire.
func ParseOutdated(data []byte) (formulae, casks int, err error) {
	var doc struct {
		Formulae []struct{} `json:"formulae"`
		Casks    []struct{} `json:"casks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, 0, err
	}
	return len(doc.Formulae), len(doc.Casks), nil
}

// Line sérialise l'état en une ligne de champs séparés par des tabulations. Ce format
// est choisi pour le consommateur : le hook zsh le découpe avec ${(s.\t.)line}, sans
// lancer ni `cut` ni `awk`.
func (s Status) Line() string {
	return fmt.Sprintf("%s\t%d\t%d\t%d", s.Version, s.Formulae, s.Casks, s.At.Unix())
}

// ParseLine relit une ligne produite par Line. Le second retour est faux dès que la
// ligne n'est pas exactement conforme — un cache corrompu ou d'une version antérieure
// est traité comme absent, jamais comme à moitié valide.
func ParseLine(line string) (Status, bool) {
	champs := strings.Split(strings.TrimRight(line, "\n"), "\t")
	if len(champs) != 4 {
		return Status{}, false
	}
	f, err := strconv.Atoi(champs[1])
	if err != nil {
		return Status{}, false
	}
	c, err := strconv.Atoi(champs[2])
	if err != nil {
		return Status{}, false
	}
	at, err := strconv.ParseInt(champs[3], 10, 64)
	if err != nil {
		return Status{}, false
	}
	return Status{Version: champs[0], Formulae: f, Casks: c, At: time.Unix(at, 0)}, true
}

// Read relit l'état en cache. Cache absent, illisible ou corrompu : ok vaut faux, et
// l'appelant se contente de ne rien afficher.
func Read(dir string) (Status, bool) {
	data, err := os.ReadFile(File(dir))
	if err != nil {
		return Status{}, false
	}
	premiere, _, _ := strings.Cut(string(data), "\n")
	return ParseLine(premiere)
}

// Refresh interroge brew et réécrit le cache. C'est le chemin lent (plusieurs
// secondes) : il n'est jamais appelé que détaché, en arrière-plan.
func Refresh(dir string) (Status, error) { return RefreshWith(dir, brewReel) }

// RefreshWith est Refresh avec un exécuteur injectable (tests).
func RefreshWith(dir string, run Runner) (Status, error) {
	libere, ok := lock(dir)
	if !ok {
		return Status{}, ErrVerrouille
	}
	defer libere()

	// La version d'abord : si brew est injoignable, on renonce avant d'avoir touché
	// au cache, qui garde sa dernière valeur connue.
	brut, err := run("--version")
	if err != nil {
		return Status{}, err
	}
	s := Status{Version: ParseVersion(string(brut))}

	brut, err = run("outdated", "--json=v2")
	if err != nil {
		return Status{}, err
	}
	if s.Formulae, s.Casks, err = ParseOutdated(brut); err != nil {
		return Status{}, err
	}
	s.At = time.Now()

	return s, ecrire(dir, s)
}

// ecrire dépose la ligne de façon atomique : un prompt concurrent lit soit l'ancienne
// version, soit la nouvelle, jamais une ligne tronquée.
func ecrire(dir string, s Status) error {
	tmp, err := os.CreateTemp(dir, ".status-*")
	if err != nil {
		return err
	}
	nom := tmp.Name()
	defer os.Remove(nom) // sans effet après un Rename réussi

	if _, err := tmp.WriteString(s.Line() + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(nom, 0o644); err != nil {
		return err
	}
	return os.Rename(nom, File(dir))
}

func verrou(dir string) string { return File(dir) + ".lock" }

// lock prend le verrou de rafraîchissement. Il renvoie la fonction de libération et un
// booléen : faux si un autre rafraîchissement est déjà en cours.
func lock(dir string) (func(), bool) {
	chemin := verrou(dir)

	f, err := os.OpenFile(chemin, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Verrou déjà pris — sauf s'il traîne depuis assez longtemps pour que son
		// propriétaire soit certainement mort.
		info, serr := os.Stat(chemin)
		if serr != nil || time.Since(info.ModTime()) < peremptionVerrou {
			return nil, false
		}
		if err := os.Remove(chemin); err != nil {
			return nil, false
		}
		if f, err = os.OpenFile(chemin, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err != nil {
			return nil, false
		}
	}
	f.Close()

	return func() { os.Remove(chemin) }, true
}

// brewReel exécute le vrai brew. Sa sortie d'erreur est ignorée : `brew outdated`
// bavarde volontiers (téléchargement du catalogue) sans que ce soit une panne.
func brewReel(args ...string) ([]byte, error) {
	return exec.Command(brew.Path(), args...).Output()
}
