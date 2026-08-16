package facade

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Opts porte les drapeaux de la ligne qui ne concernent pas le routage.
type Opts struct {
	JSON bool // sortie en JSON plutôt qu'en tableau
	Yes  bool // accepter les accords de licence (winget)
}

// verbesNormalises : ceux dont la sortie est tabulaire, donc capturée et refondue. Tout
// le reste est relayé — et c'est ce qui fait que les invites, les barres de progression et
// l'élévation UAC fonctionnent sans une ligne de code de TTY.
var verbesNormalises = map[pm.Verb]bool{
	"list": true, "outdated": true, "search": true, "source": true,
}

// Normalise dit si un verbe rend un tableau plutôt qu'une sortie relayée.
func Normalise(v pm.Verb) bool { return verbesNormalises[v] }

// lancer est le point d'injection des tests. relais dit si le processus hérite du
// terminal (verbe relayé) ou si sa sortie est capturée (verbe normalisé).
var lancer = lancerReel

func lancerReel(cmd string, args []string, relais bool) ([]byte, int, error) {
	c := exec.Command(cmd, args...)
	if relais {
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := c.Run()
		return nil, code(err), err
	}
	out, err := c.Output()
	return out, code(err), err
}

func code(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

// Executer déroule les cibles en séquence. L'ordre vient de Router, qui le tient de
// managers.All() : deux exécutions successives font la même chose dans le même ordre.
//
// Lecture et écriture ne traitent pas l'échec de la même façon, et c'est délibéré : la
// lecture est au mieux, l'écriture ne devine pas.
func Executer(v pm.Verb, cibles []Cible, o Opts) ([]pm.Package, int) {
	lecture := Normalise(v)
	var rows []pm.Package
	var reussites, echecs int
	dernierCode := 0

	for _, cible := range cibles {
		b, ok := cible.Mgr.(pm.Bindings)
		if !ok {
			continue
		}
		liaison, ok := b.Verbs()[v]
		if !ok {
			continue
		}

		// Direct : jigger sait déjà répondre, aucun sous-processus.
		if liaison.Direct != nil {
			out, err := liaison.Direct(cible.Args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "jigger (%s) : %v\n", cible.Mgr.Cmd(), err)
				echecs++
				dernierCode = 1
				continue
			}
			rows = append(rows, out...)
			reussites++
			continue
		}

		echoue := false
		for _, argv := range liaison.Argv(cible.Args) {
			argv = accords(cible.Mgr.Cmd(), v, argv, o)
			out, c, err := lancer(cible.Mgr.Cmd(), argv, !lecture)
			if err != nil {
				if !lecture {
					// Écriture : on n'enchaîne pas sur un gestionnaire suivant après
					// un échec.
					fmt.Fprint(os.Stderr, i18n.Tf("facade.failed", cible.Mgr.Cmd()))
					return rows, c
				}
				fmt.Fprintf(os.Stderr, "jigger (%s) : %v\n", cible.Mgr.Cmd(), err)
				echoue = true
				dernierCode = c
				break
			}
			if liaison.Parse != nil {
				parsed, perr := liaison.Parse(out)
				if perr != nil {
					fmt.Fprint(os.Stderr, i18n.Tf("facade.unreadable", cible.Mgr.Cmd(), perr))
					echoue = true
					dernierCode = 1
					break
				}
				for i := range parsed {
					parsed[i].PM = cible.Mgr.Cmd()
				}
				rows = append(rows, parsed...)
			} else if lecture {
				// Un verbe normalisé (sortie capturée, pas relayée) sans Parse ni Direct
				// jetterait `out` en silence : reussites++ , code 0, et le contenu
				// disparaîtrait sans qu'on l'ait dit. C'est le bug qu'avait scoop avant
				// que sa table gagne ses parsers (list, search, source) — cette garde
				// protège toute table future qui prendrait la même forme mal remplie.
				// Un verbe relayé (lecture == false) n'entre jamais ici : `out` y est
				// toujours nil, rien à perdre.
				fmt.Fprint(os.Stderr, i18n.Tf("facade.no_parser", cible.Mgr.Cmd(), v))
				echoue = true
				dernierCode = 1
				break
			}
		}
		if echoue {
			echecs++
		} else {
			reussites++
		}
	}

	if lecture {
		// Au mieux : 0 dès qu'un gestionnaire a répondu.
		if reussites > 0 {
			return rows, 0
		}
		if echecs > 0 {
			return rows, dernierCode
		}
	}
	return rows, 0
}

// accords ajoute les acceptations de licence de winget, et seulement sur --yes : jigger
// n'accepte jamais une licence à la place de l'utilisateur. Sans le drapeau, l'invite
// s'affiche — la sortie étant relayée, il peut y répondre.
func accords(cmd string, v pm.Verb, argv []string, o Opts) []string {
	if !o.Yes || cmd != "winget" {
		return argv
	}
	switch v {
	case "install", "uninstall", "upgrade":
		return append(argv, "--accept-package-agreements", "--accept-source-agreements")
	}
	return argv
}
