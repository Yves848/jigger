package config

import (
	"fmt"
	"os"
	"strings"
)

// Shell dit pour quel langage l'export est écrit.
type Shell int

const (
	Zsh Shell = iota
	PowerShell
)

// Export rend les lignes que le greffon évalue au chargement — une par réglage qu'il lit.
//
// Ce que cette fonction émet est un CONTRAT qui traverse deux langages de shell. Le projet
// s'est déjà fait prendre par une apostrophe non échappée, qui tronquait des messages dans
// les deux langues sur un chemin qu'aucun test n'exerçait. Les valeurs sont donc citées
// selon les règles de chaque shell, et la vérification se fait par **exécution** dans un
// vrai zsh et un vrai pwsh, jamais par relecture (spec §4).
//
// N'émet que les réglages que le greffon lit : lui dicter `pager`, qu'il ignore, serait du
// bruit — et une variable de plus dans son environnement.
func Export(f *Fichier, sh Shell) string {
	var b strings.Builder
	for _, r := range Declares {
		if r.Portee == Binaire {
			continue
		}
		valeur, prov := Resoudre(os.Getenv(r.Env()), f.Valeur(r.Cle), r.Defaut)
		// Ce qui vient de l'environnement y est déjà : le réémettre ne servirait qu'à
		// écraser une valeur par elle-même, et masquerait un `JIGGER_ROWS=3 zsh`.
		if prov == DeLEnvironnement {
			continue
		}
		// Un défaut n'a pas besoin d'être dicté : le greffon a le même. On n'émet que ce
		// que le fichier a réellement fixé.
		if prov == DuDefaut {
			continue
		}
		switch sh {
		case PowerShell:
			fmt.Fprintf(&b, "$env:%s = %s\n", r.Env(), citerPowerShell(valeur))
		default:
			fmt.Fprintf(&b, "export %s=%s\n", r.Env(), citerPosix(valeur))
		}
	}
	return b.String()
}

// citerPosix cite pour zsh et tout shell POSIX : apostrophes simples, et la seule séquence
// qui compte — une apostrophe interne se ferme, s'échappe et se rouvre. Rien d'autre n'a de
// sens à l'intérieur d'apostrophes simples, pas même $ ni `.
func citerPosix(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// citerPowerShell cite pour PowerShell : apostrophes simples également, où seule
// l'apostrophe se double. À l'intérieur, $ et ` sont inertes — contrairement aux guillemets
// doubles, qui interpoleraient.
func citerPowerShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
