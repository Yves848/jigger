// Package prompt produit l'état du gestionnaire de paquets affiché dans le prompt : sa
// version et le nombre de paquets obsolètes.
//
// Tout est bâti autour d'une contrainte : compter les paquets obsolètes coûte de une à
// cinq secondes — `brew outdated` comme `winget list --upgrade-available` —, ce qui est
// inconcevable dans le chemin d'un prompt. L'état est donc calculé en arrière-plan et
// déposé dans un fichier de cache d'une seule ligne, que le hook du shell relit sans
// lancer le moindre processus.
//
// Les fonctions d'analyse (ParseVersion, ParseOutdated, ParseLine) sont pures et
// exportées : elles se testent sur des sorties réellement capturées, sans lancer de
// processus.
package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pacman"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// ErrVerrouille signale qu'un autre rafraîchissement est déjà en cours. Ce n'est pas
// une panne : dix terminaux ouverts déclenchent dix `precmd`, un seul doit appeler brew.
//
// Elle ressort à l'utilisateur (main.go, `jigger prompt --refresh`) et doit donc parler
// sa langue — mais elle reste une sentinelle comparée par errors.Is. D'où un type sans
// champ, donc comparable, dont seul le message est traduit : errors.New(i18n.T(…))
// aurait figé le sien dans la langue résolue au chargement du paquet, ce que les tests
// (Recharger) auraient pris en défaut.
type errVerrouille struct{}

func (errVerrouille) Error() string { return i18n.T("cli.refresh_locked") }

// ErrVerrouille est la valeur unique de ce type : `errors.Is(err, ErrVerrouille)`
// compare deux interfaces égales, exactement comme avant.
var ErrVerrouille error = errVerrouille{}

// peremptionVerrou : au-delà, un verrou est réputé abandonné (processus tué avant
// d'avoir pu le retirer) et n'empêche plus de rafraîchir.
const peremptionVerrou = pm.PeremptionVerrou

// AttenteVerrou plafonne l'attente du verrou par un rafraîchissement forcé, et
// intervalleVerrou espace ses tentatives.
const (
	AttenteVerrou    = 20 * time.Second
	intervalleVerrou = 100 * time.Millisecond
)

// Status est l'état affiché — et mis en cache. Les deux compteurs sont ceux du prompt ;
// ce qui les sépare dépend de la plateforme : formulae et casks obsolètes pour
// Homebrew, paquets winget et scoop obsolètes sous Windows, dépôts et AUR sous pacman. Le
// format du cache, lui, ne change pas : le hook du shell découpe la même ligne des trois
// côtés.
type Status struct {
	Version   string    // version du gestionnaire, sans suffixe : « 6.0.17 », « 1.29.280 »
	Primary   int       // obsolètes du premier compteur (formulae / winget / dépôts)
	Secondary int       // obsolètes du second (casks / scoop / AUR)
	At        time.Time // instant du calcul, pour la péremption du cache
}

// Outdated est le total affiché par le prompt.
func (s Status) Outdated() int { return s.Primary + s.Secondary }

// Runner exécute `brew <args>` et renvoie sa sortie standard. Les tests en fournissent
// un factice ; la production utilise brewReel.
type Runner func(args ...string) ([]byte, error)

// Sonde interroge le ou les gestionnaires de la machine et rend l'état à afficher.
// C'est le seul endroit qui change d'une plateforme à l'autre : le verrou, l'écriture
// atomique et la péremption sont communs.
type Sonde func() (Status, error)

// File renvoie le chemin du fichier de cache dans le dossier donné.
func File(dir string) string { return filepath.Join(dir, "status") }

// Dir renvoie le dossier de cache utilisé en production (partagé avec les catalogues).
func Dir() string { return pm.CacheDir() }

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
// lancer ni `cut` ni `awk` ; PowerShell fait de même avec `-split "\t"`.
func (s Status) Line() string {
	return fmt.Sprintf("%s\t%d\t%d\t%d", s.Version, s.Primary, s.Secondary, s.At.Unix())
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
	return Status{Version: champs[0], Primary: f, Secondary: c, At: time.Unix(at, 0)}, true
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

// Refresh interroge le gestionnaire et réécrit le cache. C'est le chemin lent
// (plusieurs secondes) : il n'est jamais appelé que détaché, en arrière-plan. Verrou
// déjà pris : il renonce aussitôt — un autre shell fait déjà le travail.
func Refresh(dir string) (Status, error) { return refresh(dir, SondePlateforme(), 0) }

// RefreshWait est Refresh pour le cas où le cache est *connu* faux : le shell vient de
// voir passer une commande qui change l'état. Renoncer serait alors laisser un
// compteur mensonger jusqu'à la péremption du TTL, y compris dans le cas le plus
// probable — un rafraîchissement paresseux lancé pendant l'upgrade, et qui poireaute sur
// le verrou tout du long. Ce chemin attend donc son tour.
func RefreshWait(dir string) (Status, error) {
	return refresh(dir, SondePlateforme(), AttenteVerrou)
}

// RefreshWith est Refresh sur Homebrew avec un exécuteur injectable (tests).
func RefreshWith(dir string, run Runner) (Status, error) {
	return refresh(dir, SondeBrew(run), 0)
}

// RefreshWaitWith est RefreshWith avec une attente du verrou (tests).
func RefreshWaitWith(dir string, run Runner, attente time.Duration) (Status, error) {
	return refresh(dir, SondeBrew(run), attente)
}

// refresh est le corps commun : `attente` est le temps qu'on accepte de passer à
// guetter le verrou (zéro = une seule tentative). La sonde échoue ? On renonce sans
// avoir touché au cache, qui garde sa dernière valeur connue.
func refresh(dir string, sonde Sonde, attente time.Duration) (Status, error) {
	libere, ok := lockWait(dir, attente)
	if !ok {
		return Status{}, ErrVerrouille
	}
	defer libere()

	s, err := sonde()
	if err != nil {
		return Status{}, err
	}
	s.At = time.Now()

	return s, ecrire(dir, s)
}

// SondePlateforme rend la sonde de la machine : winget et scoop sous Windows, pacman et
// yay là où pacman règne, Homebrew partout ailleurs.
//
// Le test porte sur la présence de pacman, pas sur GOOS : le greffon zsh tourne aussi sur
// les Linux qui n'ont que Homebrew (cf. README), et ceux-là doivent garder leur bloc.
func SondePlateforme() Sonde {
	switch {
	case runtime.GOOS == "windows":
		return SondeWindows
	case pacman.Present("pacman"):
		return SondePacman
	}
	return SondeBrew(brewReel)
}

// SondePacman interroge pacman, et yay s'il est là. La répartition primaire/secondaire
// garde le sens qu'elle a partout ailleurs — formulae/casks, winget/scoop : ici, **dépôts
// et AUR**. Le format du cache d'une ligne ne bouge donc pas, et ni le hook zsh ni les
// segments oh-my-posh et starship n'ont à changer.
//
// Comme SondeWindows, elle ne renonce que si la machine n'a rien à dire : un compteur AUR
// injoignable (pas de yay, ou pas de réseau) laisse le compteur des dépôts s'afficher seul.
func SondePacman() (Status, error) {
	// La version d'abord : si pacman est injoignable, on renonce tout de suite.
	v, err := pacman.Version()
	if err != nil {
		return Status{}, err
	}
	s := Status{Version: v}

	depots, errDepots := pacman.Outdated()
	if errDepots != nil {
		return Status{}, errDepots
	}
	s.Primary = depots

	if aur, err := pacman.OutdatedAUR(); err == nil {
		s.Secondary = aur
	}
	return s, nil
}

// SondeBrew interroge Homebrew : sa version, puis ses obsolètes répartis en formulae et
// casks.
func SondeBrew(run Runner) Sonde {
	return func() (Status, error) {
		// La version d'abord : si brew est injoignable, on renonce tout de suite.
		brut, err := run("--version")
		if err != nil {
			return Status{}, err
		}
		s := Status{Version: ParseVersion(string(brut))}

		brut, err = run("outdated", "--json=v2")
		if err != nil {
			return Status{}, err
		}
		if s.Primary, s.Secondary, err = ParseOutdated(brut); err != nil {
			return Status{}, err
		}
		return s, nil
	}
}

// SondeWindows interroge winget et scoop. Les deux sont facultatifs : sur une machine
// qui n'a que scoop, le compteur winget reste à zéro et la version manque — le prompt
// masque alors ce qu'il ne sait pas, plutôt que de disparaître entièrement. On ne
// renonce que si aucun des deux n'est là.
func SondeWindows() (Status, error) {
	var s Status

	// La version de winget est la seule question rapide de tout ce fichier (~100 ms) ;
	// son échec ne dit rien de plus que celui du compteur, qui suit.
	s.Version, _ = winget.Version()

	obsoletes, errWinget := winget.Outdated()
	if errWinget == nil {
		s.Primary = obsoletes
	}

	if scoop.Present() {
		if n, err := scoop.Outdated(); err == nil {
			s.Secondary = n
		}
	} else if errWinget != nil {
		// Le %w est dans le catalogue : c'est fmt.Errorf qui l'interprète (Tf passe par
		// Sprintf, qui ne sait pas envelopper), et la ponctuation avant le deux-points
		// diffère d'une langue à l'autre.
		return Status{}, fmt.Errorf(i18n.T("cli.no_winget_no_scoop"), errWinget)
	}

	return s, nil
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

// lockWait est lock qui réessaie jusqu'à `attente`. Une attente nulle donne exactement
// lock : une tentative, puis on rend la main.
func lockWait(dir string, attente time.Duration) (func(), bool) {
	fin := time.Now().Add(attente)
	for {
		if libere, ok := lock(dir); ok {
			return libere, true
		}
		if !time.Now().Before(fin) {
			return nil, false
		}
		time.Sleep(intervalleVerrou)
	}
}

// lock prend le verrou de rafraîchissement. Il renvoie la fonction de libération et un
// booléen : faux si un autre rafraîchissement est déjà en cours.
func lock(dir string) (func(), bool) { return pm.Lock(verrou(dir)) }

// brewReel exécute le vrai brew. Sa sortie d'erreur est ignorée : `brew outdated`
// bavarde volontiers (téléchargement du catalogue) sans que ce soit une panne.
func brewReel(args ...string) ([]byte, error) {
	return pm.Run(brew.Path(), args...)
}
