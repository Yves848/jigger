package facade

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
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
// dès qu'un nom est connu de plusieurs gestionnaires, que forcePM ne tranche pas, et que
// resolu ne le connaît pas déjà.
//
// resolu porte les désambiguïsations déjà obtenues, nom par nom (typiquement : l'appelant
// a ouvert le sélecteur sur une Ambiguite précédente et rappelle Router avec le choix de
// l'utilisateur). Elle ne lie QUE le nom pour lequel elle a été faite : contrairement à
// forcePM — qui restreint tous les gestionnaires capables pour toute la ligne, et sert
// --pm —, choisir un gestionnaire pour « git » ne doit pas empêcher « fd » de se résoudre
// tout seul sur la même ligne (`jg install git fd`, spec §3 : « chaque nom résolu
// indépendamment »). nil est la valeur normale d'un premier appel, sans rien à rejouer.
func Router(v pm.Verb, args []string, forcePM string, resolu map[string]string, mgrs []pm.Manager, cats map[string]*pm.Catalog) ([]Cible, *Ambiguite, error) {
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

	drapeaux, noms := partitionner(args)

	// Pas de nom à résoudre : tous les gestionnaires capables agissent, avec les
	// arguments (drapeaux compris) tels quels. C'est délibérément la même branche pour
	// « aucun argument du tout » (len(args) == 0) et « des arguments qui ne sont que des
	// drapeaux » (jg install --cask, len(noms) == 0) : dans les deux cas, jigger n'a rien
	// à résoudre lui-même, et laisse le gestionnaire recevoir la ligne telle quelle — qui
	// refusera lui-même s'il exige un nom (`brew install --cask` : « This command
	// requires at least 1 formula or cask argument »). Construire zéro cible ici rendrait
	// un verbe mutant silencieusement no-op (tâche « install --cask »).
	if pool == pm.PoolAucun || len(noms) == 0 {
		cibles := make([]Cible, 0, len(capables))
		for _, m := range capables {
			cibles = append(cibles, Cible{Mgr: m, Args: args})
		}
		return cibles, nil, nil
	}

	parPM := map[string][]string{}
	ordre := []pm.Manager{}
	for _, nom := range noms {
		var proprios []pm.Manager
		if choisi, ok := resolu[nom]; ok {
			// Ce nom précis a déjà été tranché (cf. la doc de resolu ci-dessus) : on ne
			// repasse pas par connaissent, qui rouvrirait la même Ambiguite.
			if m := trouverCmd(capables, choisi); m != nil {
				proprios = []pm.Manager{m}
			}
		}
		if proprios == nil {
			proprios = connaissent(nom, pool, capables, cats)
		}
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

	// Les drapeaux déjà tapés servent de préfixe à Insert : c'est ce qui permet à brew de
	// voir qu'un --cask est déjà là et de ne pas le doubler (cf. note ¹ de la spec — « la
	// ligne tranche déjà », le même garde-fou que la complétion native).
	prefixeDrapeaux := strings.Join(drapeaux, " ")

	cibles := make([]Cible, 0, len(ordre))
	for _, m := range ordre {
		// Les drapeaux en tête : `brew install --cask firefox`, pas
		// `brew install firefox --cask` — plus lisible, et certains gestionnaires y sont
		// pointilleux.
		argv := make([]string, 0, len(drapeaux)+len(parPM[m.Cmd()]))
		argv = append(argv, drapeaux...)

		cat := cats[m.Cmd()]
		for _, nom := range parPM[m.Cmd()] {
			// Insert porte les petites corrections qui évitent une commande fautive —
			// --cask de brew, qualification par bucket de scoop, guillemets de winget.
			// La correction accompagne son nom, elle n'est pas hoistée en tête : c'est
			// bien « brew install --cask firealpaca », jamais « brew install --cask
			// firealpaca --cask » ni « --cask » livré à part des autres noms.
			inséré := m.Insert(cat, string(v), prefixeDrapeaux, nom)
			argv = append(argv, splitInsert(inséré)...)
		}
		cibles = append(cibles, Cible{Mgr: m, Args: argv})
	}
	return cibles, nil, nil
}

// partitionner sépare les drapeaux natifs (« --cask », « -1 »…) des noms à résoudre :
// Router ne doit chercher au catalogue que les seconds. Aucun nom de paquet, chez aucun
// gestionnaire, ne commence par « - », donc ce partitionnement est sans ambiguïté.
func partitionner(args []string) (drapeaux, noms []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			drapeaux = append(drapeaux, a)
		} else {
			noms = append(noms, a)
		}
	}
	return drapeaux, noms
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

	// Deux causes bien distinctes derrière la même absence de mgrs : --pm nomme un mot
	// que jigger ne connaît sous aucune forme (faute de frappe, autre gestionnaire —
	// apt, dnf…), ou nomme un gestionnaire que jigger connaît très bien mais qui ne
	// figure pas parmi les capables de CE verbe (pas installé, ou son concept n'existe
	// pas chez lui — winget --pm pour doctor). Confondre les deux sous « indisponible
	// pour ce verbe » ferait croire à l'utilisateur qu'apt est un gestionnaire que
	// jigger pourrait faire agir sur d'autres verbes.
	if _, connu := managers.For(forcePM); !connu {
		return nil, fmt.Errorf("jigger : --pm %s — gestionnaire inconnu de jigger. Connus : %s",
			forcePM, strings.Join(nomsConnus(), ", "))
	}
	return nil, fmt.Errorf("jigger : --pm %s — gestionnaire indisponible pour ce verbe. Disponibles : %s",
		forcePM, strings.Join(noms, ", "))
}

// nomsConnus rend les mots de commande de tous les gestionnaires que jigger sait
// reconnaître, qu'ils soient installés ou non — c'est le référentiel dont --pm doit
// piocher un nom, pas la liste (potentiellement restreinte) des capables de ce verbe.
func nomsConnus() []string {
	tous := managers.All()
	noms := make([]string, 0, len(tous))
	for _, m := range tous {
		noms = append(noms, m.Cmd())
	}
	return noms
}

// trouverCmd rend, parmi les gestionnaires capables, celui dont Cmd() correspond — ou nil
// si aucun (une résolution obtenue avant que --pm ne réduise ensuite les capables, par
// exemple).
func trouverCmd(mgrs []pm.Manager, cmd string) pm.Manager {
	for _, m := range mgrs {
		if m.Cmd() == cmd {
			return m
		}
	}
	return nil
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
