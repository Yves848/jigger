package pacman

import (
	"os"
	"sort"
	"strings"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Les caches du module, et leur raison d'être.
//
// Deux étages plutôt qu'un, et c'est **mesuré**. Les listes brutes — `pacman -Sl` et
// `yay -Slq aur` — comptent ensemble 134 000 lignes ; les fusionner, les dédupliquer et
// les trier coûtait 45 ms, à chaque frappe, parce que Charger est dans le chemin du rendu
// (`jigger render` tourne dans le widget zle, en substitution de commande : ces
// millisecondes-là sont de la latence de frappe). Le budget que le projet se donne est de
// « quelques ms au plus » (cf. BenchmarkComplete).
//
// La fusion est donc faite **une fois par jour, au réchauffement**, et déposée dans un
// fichier déjà trié que Charger n'a plus qu'à relire. C'est le même déplacement que celui
// qui a sorti `brew list` du chemin du rendu : le travail n'a pas disparu, il a changé de
// côté.
const (
	cacheSync = "pacman-sync" // brut : sortie de `pacman -Sl`
	cacheAUR  = "aur-names"   // brut : sortie de `yay -Slq aur`
)

// fichierCatalogue rend le cache fusionné d'un fournisseur. Deux fichiers distincts : le
// catalogue de pacman ne contient QUE les dépôts, celui de yay y ajoute l'AUR.
func fichierCatalogue(cmd string) string { return cmd + "-catalog" }

// DossierLocal est la base des paquets installés. Chaque entrée est un répertoire nommé
// « <nom>-<pkgver>-<pkgrel> ».
const DossierLocal = "/var/lib/pacman/local"

// Charger construit le catalogue d'un des deux fournisseurs, sans jamais lancer pacman :
// un fichier de cache et un readdir. Un cache périmé est utilisé tel quel et déclenche un
// réchauffement détaché, qui le refera pour la frappe suivante (même règle que brew).
func Charger(cmd string) *pm.Catalog {
	lignes, frais := pm.Cached(fichierCatalogue(cmd), ttlCatalogue(cmd))
	if !frais {
		pm.TriggerWarm()
	}

	cat := Relire(lignes, InstallesDe(DossierLocal))
	if len(lignes) == 0 {
		// Première utilisation : le réchauffement vient d'être lancé, mais il dure une
		// seconde ou deux. Mieux vaut le dire que d'annoncer « aucun candidat ».
		cat.Note = i18n.T("popup.catalog_pacman")
	}
	return cat
}

// Fusionner réunit les dépôts et l'AUR en un catalogue trié, dédupliqué, prêt à relire.
// C'est le travail lent du module, et il est fait au réchauffement — jamais à la frappe.
//
// Le format de sortie, une ligne par paquet :
//
//	firefox-nightly            un paquet de l'AUR
//	zsh\textra                 un paquet de dépôt
//	rustup\textra\t+           un paquet que les deux portent
//
// La troisième forme est celle qui compte : c'est elle qui permettra à Insert de
// qualifier le nom (« extra/rustup ») et d'éviter que `yay -S rustup` n'ouvre son menu
// « dépôt ou AUR ? » au milieu de ce que jigger vient d'insérer.
//
// L'ordre est celui de pm.Catalog.Sort — sans égard à la casse —, pour que Relire n'ait
// plus rien à trier.
func Fusionner(sync, aur []string) []string {
	depots := make(map[string]string, len(sync))
	dansAUR := make(map[string]bool, len(aur))
	ordre := make([]string, 0, len(sync)+len(aur))

	// Les dépôts d'abord : un nom que portent plusieurs dépôts garde le premier, comme
	// pacman lui-même le résout par ordre de priorité.
	for _, ligne := range sync {
		depot, nom, _ := CoupeSync(ligne)
		if nom == "" {
			continue
		}
		if _, vu := depots[nom]; vu {
			continue
		}
		depots[nom] = depot
		ordre = append(ordre, nom)
	}

	for _, nom := range aur {
		if nom == "" || dansAUR[nom] {
			continue
		}
		dansAUR[nom] = true
		if _, dansDepot := depots[nom]; dansDepot {
			continue // déjà au catalogue, du côté du dépôt : on note seulement le partage
		}
		ordre = append(ordre, nom)
	}

	sort.Slice(ordre, func(i, j int) bool { return pm.LessFold(ordre[i], ordre[j]) })

	lignes := make([]string, 0, len(ordre))
	for _, nom := range ordre {
		depot, estDepot := depots[nom]
		switch {
		case !estDepot:
			lignes = append(lignes, nom)
		case dansAUR[nom]:
			lignes = append(lignes, nom+"\t"+depot+"\t+")
		default:
			lignes = append(lignes, nom+"\t"+depot)
		}
	}
	return lignes
}

// Relire reconstruit le catalogue depuis les lignes fusionnées et la liste des paquets
// installés. C'est tout ce qui reste dans le chemin d'une frappe.
//
// Elle remplit Names et Badges **directement**, sans passer par Catalog.Add : la
// déduplication a déjà eu lieu dans Fusionner, et refaire la vérification cent trente mille
// fois coûterait précisément ce qu'on vient d'économiser. Le contrat est donc : les lignes
// données sont uniques et triées — ce que Fusionner garantit, et ce que
// TestFusionnerPuisRelire vérifie.
func Relire(lignes, installes []string) *pm.Catalog {
	cat := pm.NewCatalogDe(len(lignes))

	for _, ligne := range lignes {
		nom, reste, avecDepot := strings.Cut(ligne, "\t")
		if nom == "" {
			continue
		}
		badge := pm.BadgeAUR
		if avecDepot {
			badge = pm.BadgeRepo
			depot, partage, _ := strings.Cut(reste, "\t")
			if partage != "" {
				cat.Qualified[nom] = depot + "/" + nom
			}
		}
		cat.Names = append(cat.Names, nom)
		cat.Badges[nom] = badge
	}

	// Les installés, eux, sont lus sur le disque à chaque appel : toujours frais. Un nom
	// déjà au catalogue est simplement marqué — pas de Add, donc pas de doublon dans
	// Names. Un nom inconnu (paquet bâti localement par `pacman -U`, ou retiré de l'AUR
	// depuis le dernier réchauffement) entre par la porte normale, et rouvre alors le
	// tri : c'est le seul cas où Relire trie, et il porte sur une poignée de noms.
	ajouts := false
	for _, ligne := range installes {
		nom, version, _ := strings.Cut(ligne, " ")
		version = strings.TrimSpace(version)
		if _, connu := cat.Badges[nom]; connu {
			cat.Installed[nom] = true
			if version != "" {
				cat.Versions[nom] = version
			}
			continue
		}
		cat.MarkInstalled(nom, version, pm.BadgeAUR)
		ajouts = true
	}
	if ajouts {
		cat.Sort()
	}
	return cat
}

// NewCatalog construit un catalogue à partir des trois listes brutes du module :
//
//   - `sync` : les lignes de `pacman -Sl`, « dépôt nom version [installed[: v]] ».
//   - `aur`  : des noms nus (`yay -Slq aur`), vide pour pacman.
//   - `installes` : des lignes « nom version », telles que les rend InstallesDe.
//
// C'est le chemin complet — fusion puis relecture —, tel que la production le déroule en
// deux temps. Il n'est plus emprunté d'un coup qu'ici et dans les tests, et c'est
// délibéré : un test qui passe par NewCatalog éprouve du même geste le format de fichier
// intermédiaire.
func NewCatalog(sync, aur, installes []string) *pm.Catalog {
	return Relire(Fusionner(sync, aur), installes)
}

// CoupeSync lit une ligne de `pacman -Sl` : « core acl 2.4.0-1 [installed] ». Une ligne
// qui n'a pas au moins trois champs n'est pas un paquet et ne rend rien.
func CoupeSync(ligne string) (depot, nom, version string) {
	champs := strings.Fields(ligne)
	if len(champs) < 3 {
		return "", "", ""
	}
	return champs[0], champs[1], champs[2]
}

// InstallesDe lit les paquets installés directement dans la base locale d'alpm, sous la
// forme « nom version » — le même format que celui de brew.
//
// Un readdir de mille entrées coûte moins d'une milliseconde, là où `pacman -Qq` coûte
// 10 ms : peu en absolu, mais c'est un sous-processus, donc exclu du chemin d'un rendu par
// principe (cf. la règle d'ouverture de pm).
//
// Un répertoire absent ou illisible n'est pas une erreur — machine sans pacman, chemin
// inattendu : il ne contribue simplement rien.
func InstallesDe(dir string) []string {
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	lignes := make([]string, 0, len(entrees))
	for _, e := range entrees {
		// ALPM_DB_VERSION est la seule entrée non répertoire de la base ; un nom qui
		// commence par un point n'en est pas une non plus.
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if nom, version := CoupeNomVersion(e.Name()); nom != "" {
			lignes = append(lignes, nom+" "+version)
		}
	}
	sort.Strings(lignes)
	return lignes
}

// CoupeNomVersion découpe « <nom>-<pkgver>-<pkgrel> ».
//
// Le découpage est **exact**, pas heuristique : Arch interdit le tiret dans pkgver comme
// dans pkgrel, donc les deux derniers tirets du nom de répertoire sont toujours les bons
// séparateurs — y compris pour « gpu-screen-recorder-6.0.1-1 », dont le nom en porte deux.
func CoupeNomVersion(entree string) (nom, version string) {
	rel := strings.LastIndexByte(entree, '-')
	if rel <= 0 {
		return "", ""
	}
	ver := strings.LastIndexByte(entree[:rel], '-')
	if ver <= 0 {
		return "", ""
	}
	return entree[:ver], entree[ver+1:]
}

// ttlCatalogue rend la durée de validité du fichier fusionné. Pour yay, c'est la plus
// courte des deux : le catalogue vaut ce que vaut la plus périmée de ses sources.
func ttlCatalogue(cmd string) time.Duration {
	depots, aur := durees()
	if cmd == "yay" && aur < depots {
		return aur
	}
	return depots
}

// cachedLines rend les lignes de `<bin> <args>`, mises en cache <ttl>. Un cache frais
// dispense du sous-processus ; un échec se rabat sur le cache périmé s'il existe.
func cachedLines(nomCache string, ttl time.Duration, bin string, args ...string) []string {
	if lignes, frais := pm.Cached(nomCache, ttl); frais {
		return lignes
	}
	out, err := pm.Run(Path(bin), args...)
	if err != nil {
		lignes, _ := pm.Cached(nomCache, ttl)
		return lignes
	}
	lignes := pm.SplitLines(out)
	_ = pm.Store(nomCache, lignes)
	return lignes
}
