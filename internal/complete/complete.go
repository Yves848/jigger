// Package complete analyse une ligne de commande et propose des candidats contextuels :
// sous-commandes, options (--…) ou noms de paquets selon la position, et — pour
// uninstall, upgrade… — seulement les paquets installés.
//
// Rien ici ne connaît Homebrew, winget ou scoop : c'est le premier mot de la ligne qui
// désigne le gestionnaire (cf. internal/managers), lequel fournit ses sous-commandes,
// ses options et son catalogue.
package complete

import (
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Item est un candidat de complétion. C'est le type du gestionnaire : le popup affiche
// tel quel ce que celui-ci a produit.
type Item = pm.Item

// Result décrit le contexte de complétion et les candidats.
type Result struct {
	Prefix     string // texte avant le mot courant (pour reconstruire la ligne)
	Word       string // mot en cours de complétion
	Cmd        string // gestionnaire : « brew », « winget », « scoop »
	Sub        string // sous-commande courante (« install », « uninstall »…)
	Items      []Item
	Executable bool   // contexte paquet → accepter complète une commande exécutable
	Note       string // message à afficher à la place de « aucun candidat »

	mgr pm.Manager
	cat *pm.Catalog

	// mgrsParPM et catsParPM ne sont peuplés que par CompleteFacade. Un résultat natif a
	// un seul gestionnaire (mgr/cat ci-dessus) ; un résultat façade en mélange
	// plusieurs — Item.PM dit lequel a proposé chaque candidat (cf.
	// TestFacade_ItemPorteSonPM) — donc la correction ne peut se résoudre qu'Item par
	// Item, pas une fois pour tout le Result.
	mgrsParPM map[string]pm.Manager
	catsParPM map[string]*pm.Catalog
}

// InsertItem rend le texte à insérer pour un candidat précis : le nom, éventuellement
// corrigé par le gestionnaire (`--cask` de brew, qualification par bucket de scoop). Sur
// un résultat natif, un seul gestionnaire, donc aucune ambiguïté sur lequel corrige. Sur
// un résultat façade, elle résout la correction du gestionnaire qui a proposé CET Item —
// via it.PM — plutôt que celle d'un gestionnaire par défaut qui n'existe pas. Sans PM
// connu (candidat construit à la main, hors contexte façade), elle rend le nom brut.
func (r Result) InsertItem(it Item) string {
	mgr, cat := r.mgr, r.cat
	if mgr == nil {
		mgr, cat = r.mgrsParPM[it.PM], r.catsParPM[it.PM]
	}
	if mgr == nil || cat == nil {
		return it.Name
	}
	return mgr.Insert(cat, r.Sub, r.Prefix, it.Name)
}

// Title résume le contexte, en tête du popup : « winget install ».
func (r Result) Title() string {
	if r.Sub == "" {
		return r.Cmd
	}
	return r.Cmd + " " + r.Sub
}

// estFacade dit si le premier mot d'une ligne désigne jigger lui-même — « jigger » ou son
// alias « jg » — plutôt qu'un gestionnaire.
func estFacade(mot string) bool {
	m := motCommande(mot)
	return m == "jigger" || m == "jg"
}

// Complete calcule le contexte et les candidats pour la ligne donnée, en interrogeant
// le gestionnaire qu'elle nomme.
func Complete(line string) Result { return CompleteAvec(line, false) }

// CompleteAvec est Complete, en choisissant comment le mot filtre les **noms de paquets** :
// par préfixe (le comportement historique) ou par expression rationnelle. Les verbes, les
// sous-commandes et les options gardent leur préfixe dans les deux cas.
func CompleteAvec(line string, regex bool) Result {
	premier, _, _ := strings.Cut(strings.TrimSpace(line), " ")
	if estFacade(premier) {
		dispo := managers.Available()
		cats := map[string]*pm.Catalog{}
		for _, m := range dispo {
			cats[m.Cmd()] = m.Load()
		}
		return completeFacade(line, dispo, cats, regex)
	}
	m := managers.Detect(line)
	return completeWith(line, m, m.Load(), regex)
}

// CompleteFacade complète la syntaxe unique : « jg ⇥ » propose les verbes, « jg source ⇥ »
// les sous-verbes, « jg install g » les paquets de tous les gestionnaires disponibles.
//
// Les catalogues sont filtrés CHEZ CHAQUE GESTIONNAIRE avant d'être réunis. L'ordre
// compte : concaténer trois catalogues — 14 401 noms rien que pour winget — puis balayer
// coûterait le budget de la frappe (cf. spec §5).
func CompleteFacade(line string, dispo []pm.Manager, cats map[string]*pm.Catalog) Result {
	return completeFacade(line, dispo, cats, false)
}

func completeFacade(line string, dispo []pm.Manager, cats map[string]*pm.Catalog, regex bool) Result {
	var prefix, word string
	if i := strings.LastIndex(line, " "); i < 0 {
		word = line
	} else {
		prefix, word = line[:i+1], line[i+1:]
	}

	champs := strings.Fields(strings.TrimSpace(prefix))
	if len(champs) > 0 && estFacade(champs[0]) {
		champs = champs[1:]
	}

	res := Result{
		Prefix: prefix, Word: word, Cmd: "jigger",
		mgrsParPM: indexParPM(dispo), catsParPM: cats,
	}
	lw := strings.ToLower(word)
	filtre := NouveauFiltre(word, regex)
	tables := managers.Tables(dispo)

	// Premier mot : les verbes.
	if len(champs) == 0 {
		for _, v := range managers.Vocabulaire(dispo) {
			if strings.HasPrefix(v, lw) {
				res.Items = append(res.Items, Item{Name: v})
			}
		}
		return res
	}

	res.Sub = strings.ToLower(champs[0])

	// Deuxième mot d'un verbe composé : « source ⇥ » → add, rm.
	if len(champs) == 1 {
		var sous []string
		for v := range tables {
			tete, queue, compose := strings.Cut(string(v), " ")
			if compose && tete == res.Sub && strings.HasPrefix(queue, lw) {
				sous = append(sous, queue)
			}
		}
		if len(sous) > 0 {
			sort.Strings(sous)
			for _, s := range sous {
				res.Items = append(res.Items, Item{Name: s})
			}
			return res
		}
	}

	// Sinon : des paquets. Le Pool du verbe dit lequel des deux viviers fouiller.
	verbe := pm.Verb(res.Sub)
	if len(champs) >= 2 {
		if _, ok := tables[pm.Verb(res.Sub+" "+strings.ToLower(champs[1]))]; ok {
			verbe = pm.Verb(res.Sub + " " + strings.ToLower(champs[1]))
			res.Sub = string(verbe)
		}
	}

	res.Executable = true
	for _, m := range dispo {
		b, ok := m.(pm.Bindings)
		if !ok {
			continue
		}
		liaison, ok := b.Verbs()[verbe]
		if !ok || liaison.Pool == pm.PoolAucun {
			continue
		}
		cat := cats[m.Cmd()]
		if cat == nil {
			continue
		}
		vivier := cat.Names
		if liaison.Pool == pm.PoolInstalles {
			vivier = cat.InstalledNames()
		}
		// Filtrer ici, chez le gestionnaire : c'est ce qui tient le budget.
		for _, n := range vivier {
			if !filtre.Correspond(n) {
				continue
			}
			res.Items = append(res.Items, Item{
				Name:      n,
				Badge:     cat.Badge(n),
				Installed: cat.Installed[n],
				Version:   cat.Version(n),
				PM:        m.Cmd(),
			})
		}
	}
	sort.Slice(res.Items, func(i, j int) bool {
		return pm.LessFold(res.Items[i].Name, res.Items[j].Name)
	})
	return res
}

// indexParPM range les gestionnaires disponibles par Cmd(), pour qu'InsertItem retrouve
// celui qui a proposé un Item façade à partir de son seul champ PM.
func indexParPM(dispo []pm.Manager) map[string]pm.Manager {
	idx := make(map[string]pm.Manager, len(dispo))
	for _, m := range dispo {
		idx[m.Cmd()] = m
	}
	return idx
}

// CompleteWith est Complete sur un gestionnaire et un catalogue donnés (tests, et tout
// appelant qui a déjà chargé le catalogue).
func CompleteWith(line string, m pm.Manager, cat *pm.Catalog) Result {
	return completeWith(line, m, cat, false)
}

func completeWith(line string, m pm.Manager, cat *pm.Catalog, regex bool) Result {
	var prefix, word string
	if i := strings.LastIndex(line, " "); i < 0 {
		word = line
	} else {
		prefix, word = line[:i+1], line[i+1:]
	}

	before := strings.Fields(strings.TrimSpace(prefix))
	if len(before) > 0 && strings.EqualFold(motCommande(before[0]), m.Cmd()) {
		before = before[1:]
	}
	sub := ""
	if len(before) > 0 {
		sub = strings.ToLower(before[0])
	}

	firstWord := len(before) == 0

	// « winget » tout court : le mot en cours *est* le nom de la commande. Le chercher
	// parmi les sous-commandes ne donnerait rien — aucune ne s'appelle « winget… » — et
	// le popup annoncerait « aucun candidat » là où l'utilisateur attend justement de
	// voir ce qu'il peut taper. On considère donc la commande comme acquise, et on
	// propose la suite.
	if firstWord && prefix == "" && strings.EqualFold(motCommande(word), m.Cmd()) {
		prefix, word = word+" ", ""
	}

	// Un fournisseur qui ne déclare aucune sous-commande n'a pas de verbe entre la
	// commande et son opérande : `ssh archlight` met l'hôte là où `brew install` met un
	// verbe. Le catalogue vient donc dès le premier mot (ADR-0005).
	subs := m.Subcommands()
	sansVerbes := len(subs) == 0

	isOption := strings.HasPrefix(word, "-")
	isPackage := !isOption && (!firstWord || sansVerbes)

	res := Result{
		Prefix: prefix, Word: word, Cmd: m.Cmd(), Sub: sub,
		// Un fournisseur sans verbes n'a pas de pm.Bindings : rien à exécuter. Le
		// sélecteur plein écran doit insérer, pas lancer.
		Executable: isPackage && !sansVerbes, mgr: m, cat: cat,
	}
	lw := strings.ToLower(word)
	filtre := NouveauFiltre(word, regex)

	switch {
	case isOption:
		for _, o := range m.Options(sub) {
			if strings.HasPrefix(strings.ToLower(o), lw) {
				res.Items = append(res.Items, Item{Name: o})
			}
		}
	case firstWord && !sansVerbes:
		for _, s := range subs {
			if strings.HasPrefix(s, lw) {
				res.Items = append(res.Items, Item{Name: s})
			}
		}
	default: // paquet
		pool := cat.Names
		if m.InstalledOnly(sub) {
			pool = cat.InstalledNames()
		}
		for _, n := range pool {
			if filtre.Correspond(n) {
				res.Items = append(res.Items, Item{
					Name:      n,
					Badge:     cat.Badge(n),
					Installed: cat.Installed[n],
					Version:   cat.Version(n),
				})
			}
		}
		if len(res.Items) == 0 {
			res.Note = cat.Note
		}
	}
	return res
}

// motCommande ramène un mot à un nom de commande : sans chemin ni extension. Une ligne
// peut très bien commencer par `C:\Users\…\scoop\shims\scoop.exe`.
func motCommande(s string) string {
	s = strings.Trim(s, `"'`)
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	return strings.TrimSuffix(strings.ToLower(s), ".exe")
}
