package facade

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Cible est un gestionnaire et les arguments qui lui reviennent. Une ligne peut en
// produire plusieurs : `jg install fd Git.Git` route fd vers scoop et Git.Git vers winget.
type Cible struct {
	Mgr  pm.Manager
	Args []string
}

// Candidat est un gestionnaire qui connaît un nom ambigu.
type Candidat struct {
	Mgr      pm.Manager
	Badge    string
	Qualifie string // texte à insérer si le nom demande une qualification (« main/flux »)
}

// Ambiguite est un nom que plusieurs gestionnaires connaissent. Le moteur ne tranche
// jamais tout seul : un choix silencieux entre deux « git » qui ne sont pas le même
// logiciel est ce qui rend une façade impossible à croire.
type Ambiguite struct {
	Nom       string
	Candidats []Candidat
}

// Router résout chaque nom et rend les cibles. Il rend une Ambiguite — et aucune cible —
// dès qu'un nom est connu de plusieurs gestionnaires et que forcePM ne tranche pas.
func Router(v pm.Verb, args []string, forcePM string, mgrs []pm.Manager, cats map[string]*pm.Catalog) ([]Cible, *Ambiguite, error) {
	capables, err := filtrerParPM(mgrs, forcePM)
	if err != nil {
		return nil, nil, err
	}

	pool := pm.PoolAucun
	for _, m := range capables {
		if b, ok := m.(pm.Bindings); ok {
			if liaison, ok := b.Verbs()[v]; ok {
				pool = liaison.Pool
				break
			}
		}
	}

	// Pas de nom à résoudre : tous les gestionnaires capables agissent.
	if pool == pm.PoolAucun || len(args) == 0 {
		cibles := make([]Cible, 0, len(capables))
		for _, m := range capables {
			cibles = append(cibles, Cible{Mgr: m, Args: args})
		}
		return cibles, nil, nil
	}

	parPM := map[string][]string{}
	ordre := []pm.Manager{}
	for _, nom := range args {
		proprios := connaissent(nom, pool, capables, cats)
		switch len(proprios) {
		case 0:
			return nil, nil, nomInconnu(nom, pool, capables, cats)
		case 1:
			m := proprios[0]
			if _, vu := parPM[m.Cmd()]; !vu {
				ordre = append(ordre, m)
			}
			parPM[m.Cmd()] = append(parPM[m.Cmd()], nom)
		default:
			return nil, ambiguite(nom, proprios, cats), nil
		}
	}

	cibles := make([]Cible, 0, len(ordre))
	for _, m := range ordre {
		cibles = append(cibles, Cible{Mgr: m, Args: parPM[m.Cmd()]})
	}
	return cibles, nil, nil
}

// filtrerParPM applique --pm. Un nom de gestionnaire absent des capables est une erreur :
// mieux vaut le dire que de router ailleurs en silence.
func filtrerParPM(mgrs []pm.Manager, forcePM string) ([]pm.Manager, error) {
	if forcePM == "" {
		return mgrs, nil
	}
	for _, m := range mgrs {
		if m.Cmd() == forcePM {
			return []pm.Manager{m}, nil
		}
	}
	noms := make([]string, 0, len(mgrs))
	for _, m := range mgrs {
		noms = append(noms, m.Cmd())
	}
	return nil, fmt.Errorf("jigger : --pm %s — gestionnaire indisponible pour ce verbe. Disponibles : %s",
		forcePM, strings.Join(noms, ", "))
}

// connaissent rend les gestionnaires dont le vivier contient ce nom exactement.
func connaissent(nom string, pool pm.Pool, mgrs []pm.Manager, cats map[string]*pm.Catalog) []pm.Manager {
	var out []pm.Manager
	for _, m := range mgrs {
		cat := cats[m.Cmd()]
		if cat == nil {
			continue
		}
		if pool == pm.PoolInstalles {
			if cat.Installed[nom] {
				out = append(out, m)
			}
			continue
		}
		if _, connu := cat.Badges[nom]; connu {
			out = append(out, m)
		}
	}
	return out
}

func ambiguite(nom string, proprios []pm.Manager, cats map[string]*pm.Catalog) *Ambiguite {
	amb := &Ambiguite{Nom: nom}
	for _, m := range proprios {
		cat := cats[m.Cmd()]
		amb.Candidats = append(amb.Candidats, Candidat{
			Mgr:      m,
			Badge:    cat.Badge(nom),
			Qualifie: cat.Qualified[nom],
		})
	}
	return amb
}

// nomInconnu distingue trois situations que l'utilisateur ne doit pas confondre : un
// catalogue en cours de constitution, une faute de frappe (avec les voisins), et un
// paquet trop récent pour le cache (avec l'échappatoire --pm).
func nomInconnu(nom string, pool pm.Pool, mgrs []pm.Manager, cats map[string]*pm.Catalog) error {
	for _, m := range mgrs {
		if cat := cats[m.Cmd()]; cat != nil && len(cat.Names) == 0 && cat.Note != "" {
			return fmt.Errorf("jigger : %s", cat.Note)
		}
	}

	var noms []string
	for _, m := range mgrs {
		noms = append(noms, m.Cmd())
	}
	msg := fmt.Sprintf("jigger : « %s » — inconnu de %s", nom, strings.Join(noms, " et "))

	if proches := voisins(nom, pool, mgrs, cats); len(proches) > 0 {
		msg += "\n        Proche : " + strings.Join(proches, ", ")
	}
	msg += fmt.Sprintf("\n        Si le paquet est trop récent pour le catalogue : jg … --pm %s %s", noms[0], nom)
	return fmt.Errorf("%s", msg)
}

// voisins cherche les noms qui partagent un préfixe avec le nom demandé — de la longueur
// du nom moins deux caractères, ce qui rattrape une faute de frappe en fin de mot sans
// noyer le message. Il respecte le pool du verbe : sous PoolInstalles, un voisin qui
// n'est que catalogué et pas installé ne serait pas une cible valide, donc on ne le
// propose pas.
func voisins(nom string, pool pm.Pool, mgrs []pm.Manager, cats map[string]*pm.Catalog) []string {
	n := len(nom) - 2
	if n < 2 {
		return nil
	}
	prefixe := strings.ToLower(nom[:n])

	var out []string
	for _, m := range mgrs {
		cat := cats[m.Cmd()]
		if cat == nil {
			continue
		}
		vivier := cat.Names
		if pool == pm.PoolInstalles {
			vivier = cat.InstalledNames()
		}
		for _, candidat := range vivier {
			if !strings.HasPrefix(strings.ToLower(candidat), prefixe) {
				continue
			}
			if q := cat.Qualified[candidat]; q != "" {
				out = append(out, fmt.Sprintf("%s (%s)", candidat, q))
			} else {
				out = append(out, fmt.Sprintf("%s (%s)", candidat, m.Cmd()))
			}
			if len(out) >= 5 {
				sort.Strings(out)
				return out
			}
		}
	}
	sort.Strings(out)
	return out
}
