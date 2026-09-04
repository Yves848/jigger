package facade

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/plugin"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Opts porte les drapeaux de la ligne qui ne concernent pas le routage.
type Opts struct {
	JSON bool // sortie en JSON plutôt qu'en tableau
	Yes  bool // accepter les accords de licence (winget)
}

// Normalise dit si un verbe rend un tableau plutôt qu'une sortie relayée. La table vit
// désormais dans pm : les plugins doivent en dériver leurs liaisons sans pouvoir importer
// la façade (cf. pm.Normalise).
func Normalise(v pm.Verb) bool { return pm.Normalise(v) }

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

// Rejeu décrit un échec de privilèges qu'un appelant peut proposer de rejouer. L'argv est
// celui qui a **réellement** tourné, `accords()` compris : rejouer autre chose que ce qui
// a échoué serait un piège.
type Rejeu struct {
	Cmd    string
	Argv   []string
	Droits pm.Droits // DroitsRequis ou DroitsInterdits — jamais DroitsRien
}

// Resultat est ce qu'une exécution produit, élévation comprise.
type Resultat struct {
	Rows  []pm.Package
	Code  int
	Rejeu *Rejeu // non nil quand un code de sortie a parlé de privilèges
}

// Executer déroule les cibles en séquence, et rend le couple court dont presque tout le
// monde se contente. La forme longue est ExecuterAvec — même partage que
// complete.Complete / CompleteAvec.
func Executer(v pm.Verb, cibles []Cible, o Opts) ([]pm.Package, int) {
	r := ExecuterAvec(v, cibles, o)
	return r.Rows, r.Code
}

// ExecuterAvec déroule les cibles en séquence. L'ordre vient de Router, qui le tient de
// managers.All() : deux exécutions successives font la même chose dans le même ordre.
//
// Lecture et écriture ne traitent pas l'échec de la même façon, et c'est délibéré : la
// lecture est au mieux, l'écriture ne devine pas.
//
// Rien n'est intercepté et rien n'est élevé ici : la commande tourne relayée, exactement
// comme avant, et son code de sortie est lu **après coup** (ADR-0004). La façade constate
// et rend ; c'est l'appelant qui a le terminal, donc c'est lui qui propose.
func ExecuterAvec(v pm.Verb, cibles []Cible, o Opts) Resultat {
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
				fmt.Fprint(os.Stderr, i18n.Tf("facade.manager_error", cible.Mgr.Cmd(), err))
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

			// Un plugin s'exécute exactement comme un gestionnaire natif : même relais de
			// terminal, même lecture du code de sortie, même rejeu sur défaut de droits. Seul
			// le programme lancé change — le mot de la ligne (« git ») n'est pas le binaire
			// (« jigger-git »), et les confondre lancerait le vrai git.
			binaire := cible.Mgr.Cmd()
			if b, ok := plugin.Binaire(cible.Mgr); ok {
				binaire = b
			}

			out, c, err := lancer(binaire, argv, !lecture)
			if err != nil {
				if !lecture {
					// Écriture : on n'enchaîne pas sur un gestionnaire suivant après
					// un échec.
					fmt.Fprint(os.Stderr, i18n.Tf("facade.failed", cible.Mgr.Cmd()))
					res := Resultat{Rows: rows, Code: c}
					// Seule l'écriture porte un rejeu : une lecture qui échoue passe au
					// gestionnaire suivant, il n'y a rien à relancer.
					if d := pm.DroitsDe(cible.Mgr, c); d != pm.DroitsRien {
						res.Rejeu = &Rejeu{Cmd: binaire, Argv: argv, Droits: d}
					}
					return res
				}
				fmt.Fprint(os.Stderr, i18n.Tf("facade.manager_error", cible.Mgr.Cmd(), err))
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
			return Resultat{Rows: rows}
		}
		if echecs > 0 {
			return Resultat{Rows: rows, Code: dernierCode}
		}
	}
	return Resultat{Rows: rows}
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
