package facade

import (
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// catalogues fabrique deux catalogues où « git » est connu des deux gestionnaires et
// « fd » de scoop seul.
func catalogues() map[string]*pm.Catalog {
	w := pm.NewCatalog()
	w.Add("Git.Git", pm.BadgeWinget)
	w.Add("git", pm.BadgeWinget)
	w.MarkInstalled("Git.Git", "2.55.0", pm.BadgeWinget)
	w.MarkInstalled("git", "2.55.0", pm.BadgeWinget)
	w.Sort()

	s := pm.NewCatalog()
	s.Add("git", pm.BadgeScoop)
	s.Add("fd", pm.BadgeScoop)
	s.MarkInstalled("fd", "10.2.0", pm.BadgeScoop)
	s.Sort()

	return map[string]*pm.Catalog{"winget": w, "scoop": s}
}

func deuxGestionnaires() []pm.Manager {
	return []pm.Manager{winget.New(), scoop.New()}
}

func TestRoutageUnSeulGestionnaireConnaitLeNom(t *testing.T) {
	cibles, amb, err := Router("install", []string{"fd"}, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 || cibles[0].Mgr.Cmd() != "scoop" {
		t.Fatalf("cibles = %v, attendu scoop seul", cibles)
	}
}

func TestRoutageAmbiguite(t *testing.T) {
	_, amb, err := Router("install", []string{"git"}, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb == nil {
		t.Fatal("« git » est connu des deux : une ambiguïté était attendue")
	}
	if amb.Nom != "git" || len(amb.Candidats) != 2 {
		t.Fatalf("ambiguïté = %+v", amb)
	}
}

// --pm tranche sans ouvrir le sélecteur.
func TestRoutageForcePM(t *testing.T) {
	cibles, amb, err := Router("install", []string{"git"}, "scoop", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatal("--pm doit lever l'ambiguïté")
	}
	if len(cibles) != 1 || cibles[0].Mgr.Cmd() != "scoop" {
		t.Fatalf("cibles = %v, attendu scoop", cibles)
	}
}

// Chaque nom est résolu indépendamment : une ligne peut viser deux gestionnaires.
func TestRoutageDeuxNomsDeuxGestionnaires(t *testing.T) {
	cibles, amb, err := Router("install", []string{"fd", "Git.Git"}, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 2 {
		t.Fatalf("%d cibles, attendu 2", len(cibles))
	}
	vu := map[string][]string{}
	for _, c := range cibles {
		vu[c.Mgr.Cmd()] = c.Args
	}
	if len(vu["scoop"]) != 1 || vu["scoop"][0] != "fd" {
		t.Errorf("scoop = %v, attendu [fd]", vu["scoop"])
	}
	if len(vu["winget"]) != 1 || vu["winget"][0] != "Git.Git" {
		t.Errorf("winget = %v, attendu [Git.Git]", vu["winget"])
	}
}

// PoolInstalles ne fouille que les installés : « git » est au catalogue de scoop mais
// n'y est pas installé, donc uninstall ne doit pas le proposer.
func TestRoutagePoolInstalles(t *testing.T) {
	cibles, amb, err := Router("uninstall", []string{"git"}, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v — seul winget a « git » installé", amb)
	}
	if len(cibles) != 1 || cibles[0].Mgr.Cmd() != "winget" {
		t.Fatalf("cibles = %v, attendu winget", cibles)
	}
}

// PoolAucun : pas de nom à résoudre, tous les gestionnaires capables agissent.
func TestRoutagePoolAucun(t *testing.T) {
	cibles, amb, err := Router("outdated", nil, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatal("un verbe sans nom ne peut pas être ambigu")
	}
	if len(cibles) != 2 {
		t.Fatalf("%d cibles, attendu 2 (tous les capables)", len(cibles))
	}
}

func TestRoutageNomInconnu(t *testing.T) {
	_, _, err := Router("install", []string{"fdfind"}, "", nil, deuxGestionnaires(), catalogues())
	if err == nil {
		t.Fatal("attendu une erreur pour un nom inconnu de tous")
	}
	msg := err.Error()
	if !strings.Contains(msg, "fdfind") {
		t.Errorf("le message doit nommer le paquet : %s", msg)
	}
	// Le voisin le plus proche aide plus qu'un « inconnu » sec.
	if !strings.Contains(msg, "fd") {
		t.Errorf("le message doit proposer « fd » : %s", msg)
	}
}

// Sous PoolInstalles, le voisin proposé doit lui-même être installé : uninstall ne peut
// viser qu'un paquet installé, donc suggérer un paquet simplement catalogué (« git » chez
// scoop, qui l'a au catalogue mais ne l'a pas installé) mènerait droit à un second
// « nom inconnu ». Seul winget a « git » installé : c'est lui, et lui seul, qui doit
// apparaître dans la suggestion.
func TestRoutageNomInconnuRespectePoolInstalles(t *testing.T) {
	_, _, err := Router("uninstall", []string{"gitt"}, "", nil, deuxGestionnaires(), catalogues())
	if err == nil {
		t.Fatal("attendu une erreur pour un nom inconnu de tous")
	}
	msg := err.Error()
	if !strings.Contains(msg, "gitt") {
		t.Errorf("le message doit nommer le paquet : %s", msg)
	}
	if !strings.Contains(msg, "git (winget)") {
		t.Errorf("le message doit proposer « git » installé chez winget : %s", msg)
	}
	if strings.Contains(msg, "(scoop)") {
		t.Errorf("scoop n'a « git » qu'au catalogue, pas installé : il ne doit pas être suggéré : %s", msg)
	}
}

// Un catalogue vide et en cours de constitution ne doit pas produire « paquet inconnu ».
func TestRoutageCatalogueEnConstruction(t *testing.T) {
	cats := catalogues()
	vide := pm.NewCatalog()
	vide.Note = "catalogue winget en cours de constitution"
	cats["winget"] = vide

	_, _, err := Router("install", []string{"Git.Git"}, "", nil, deuxGestionnaires(), cats)
	if err == nil {
		t.Fatal("attendu une erreur")
	}
	if !strings.Contains(err.Error(), "en cours de constitution") {
		t.Errorf("la note du catalogue doit primer : %s", err.Error())
	}
}

// La note d'un catalogue en construction (cat.Note) est injectée dans facade.note :
// « jigger : %s » ou « jigger: %s » selon la langue. Avant la relecture, le préfixe
// restait « jigger : » (espace avant le deux-points, ponctuation française) même en
// anglais — ce test aurait attrapé l'oubli, faute de quoi rien n'asserait ce préfixe.
func TestRoutageCatalogueEnConstructionEnAnglais(t *testing.T) {
	t.Cleanup(i18n.Recharger)
	t.Setenv("JIGGER_LANG", "en")
	i18n.Recharger()

	cats := catalogues()
	vide := pm.NewCatalog()
	vide.Note = "winget catalog under construction"
	cats["winget"] = vide

	_, _, err := nomInconnuVia(cats)
	if err == nil {
		t.Fatal("attendu une erreur")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "jigger: ") {
		t.Errorf("le préfixe anglais doit être « jigger: » (pas de ponctuation française) : %q", msg)
	}
	if strings.HasPrefix(msg, "jigger : ") {
		t.Errorf("le préfixe ne doit plus être « jigger : » en anglais : %q", msg)
	}
}

// nomInconnuVia route « install Git.Git » à travers les deux gestionnaires habituels,
// pour exercer nomInconnu avec le catalogue winget donné (vide, avec une Note).
func nomInconnuVia(cats map[string]*pm.Catalog) ([]Cible, *Ambiguite, error) {
	return Router("install", []string{"Git.Git"}, "", nil, deuxGestionnaires(), cats)
}

// search prend une requête, pas un nom de paquet à résoudre au catalogue : une requête
// qui ne correspond à aucun paquet connu doit tout de même atteindre les gestionnaires
// capables — c'est à eux de chercher, pas à Router de refuser une recherche au prétexte
// que la requête n'est pas déjà un nom exact du catalogue.
func TestRoutageSearchAtteintLesGestionnairesMemeSansCorrespondanceExacte(t *testing.T) {
	cibles, amb, err := Router("search", []string{"fire"}, "", nil, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatalf("« fire » ne correspond à aucun paquet exact des catalogues de test, mais search doit tout de même router : %v", err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 2 {
		t.Fatalf("%d cibles, attendu 2 (tous les gestionnaires capables) : %v", len(cibles), cibles)
	}
	for _, c := range cibles {
		if len(c.Args) != 1 || c.Args[0] != "fire" {
			t.Errorf("cible %s : args = %v, attendu [fire] (la requête, intacte)", c.Mgr.Cmd(), c.Args)
		}
	}
}

// « apt » n'est un gestionnaire connu de jigger en aucun cas : « indisponible pour ce
// verbe » suggérerait à tort qu'il existe mais ne sait pas rendre ce verbe précis, alors
// que jigger ne sait tout simplement pas ce qu'est apt.
func TestRoutageForcePMInconnuDeJigger(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()
	_, _, err := Router("install", []string{"fd"}, "apt", nil, deuxGestionnaires(), catalogues())
	if err == nil {
		t.Fatal("attendu une erreur : « apt » n'est pas un gestionnaire de jigger")
	}
	msg := err.Error()
	if !strings.Contains(msg, "inconnu") {
		t.Errorf("« apt » n'existe pas du tout chez jigger, le message doit le dire : %s", msg)
	}
	if strings.Contains(msg, "indisponible pour ce verbe") {
		t.Errorf("« apt » ne doit pas être dit « indisponible pour ce verbe » (il n'existe pas, point) : %s", msg)
	}
}

// « winget » est un gestionnaire que jigger connaît bien — mais ici il est absent des
// capables (par exemple : doctor, que winget ne sait pas rendre). Le message doit rester
// « indisponible pour ce verbe », pas « inconnu » : jigger sait très bien ce qu'est
// winget.
func TestRoutageForcePMConnuMaisIndisponiblePourCeVerbe(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()
	_, _, err := Router("doctor", nil, "winget", nil, []pm.Manager{scoop.New()}, catalogues())
	if err == nil {
		t.Fatal("attendu une erreur : winget n'est pas dans les capables pour doctor")
	}
	msg := err.Error()
	if !strings.Contains(msg, "indisponible pour ce verbe") {
		t.Errorf("« winget » est connu de jigger, seulement absent des capables pour ce verbe : %s", msg)
	}
	if strings.Contains(msg, "inconnu") {
		t.Errorf("« winget » ne doit pas être dit inconnu de jigger : %s", msg)
	}
}

// Un drapeau natif (« --cask ») ne doit pas être avalé comme s'il était un nom de paquet à
// résoudre : il doit ressortir intact dans les arguments de la cible, et en tête — c'est
// « brew install --cask firefox », pas « brew install firefox --cask ».
func TestRoutageDrapeauxNatifsAccompagnentLesNoms(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firefox"}, nil)}
	cibles, amb, err := Router("install", []string{"--cask", "firefox"}, "", nil, []pm.Manager{brew.New()}, cats)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 {
		t.Fatalf("%d cibles, attendu 1", len(cibles))
	}
	if len(cibles[0].Args) != 2 || cibles[0].Args[0] != "--cask" || cibles[0].Args[1] != "firefox" {
		t.Fatalf("args = %v, attendu [--cask firefox] (drapeau en tête)", cibles[0].Args)
	}
}

// jg install --cask (des drapeaux, aucun nom) ne doit pas produire zéro cible en silence
// — un verbe mutant qui « réussit » sans avoir rien fait. Router doit laisser les
// drapeaux filer au gestionnaire, qui refusera lui-même (vérifié : `brew install --cask`
// sort en erreur, « This command requires at least 1 formula or cask argument ») —
// exactement le chemin qu'emprunte déjà `jg install` sans le moindre argument, qu'accepte
// ce code depuis toujours (len(args) == 0). Les deux cas — aucun argument du tout, des
// arguments qui ne sont que des drapeaux — doivent s'accorder.
func TestRoutageDrapeauxSeulsSansNomNeDonnePasZeroCibleEnSilence(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firefox"}, nil)}
	cibles, amb, err := Router("install", []string{"--cask"}, "", nil, []pm.Manager{brew.New()}, cats)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 || cibles[0].Mgr.Cmd() != "brew" {
		t.Fatalf("cibles = %v, attendu brew seul (le --cask lui parvient, il refusera lui-même)", cibles)
	}
	if len(cibles[0].Args) != 1 || cibles[0].Args[0] != "--cask" {
		t.Fatalf("args = %v, attendu [--cask] intact", cibles[0].Args)
	}
}

// PoolAucun (list, outdated…) ne résout aucun nom : les drapeaux doivent lui parvenir
// intacts, sans passer par le partitionnement drapeaux/noms.
func TestRoutagePoolAucunLaisseFilerLesDrapeaux(t *testing.T) {
	cibles, amb, err := Router("list", []string{"--versions"}, "", nil, []pm.Manager{brew.New()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatal("un verbe sans nom ne peut pas être ambigu")
	}
	if len(cibles) != 1 || len(cibles[0].Args) != 1 || cibles[0].Args[0] != "--versions" {
		t.Fatalf("cibles = %v, attendu [--versions] intact", cibles)
	}
}

// C'est le cœur de la note ¹ de la spec : le --cask de brew.Manager.Insert doit se
// rebrancher sur l'exécution façade, pas seulement sur la complétion native. Sans ça,
// `jg install <cask pur>` lance `brew install <cask>`, que brew refuse.
func TestRoutageAppliqueLeCaskDeBrew(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firealpaca"}, nil)}
	cibles, amb, err := Router("install", []string{"firealpaca"}, "", nil, []pm.Manager{brew.New()}, cats)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 {
		t.Fatalf("%d cibles, attendu 1", len(cibles))
	}
	if len(cibles[0].Args) != 2 || cibles[0].Args[0] != "--cask" || cibles[0].Args[1] != "firealpaca" {
		t.Fatalf("args = %v, attendu [--cask firealpaca]", cibles[0].Args)
	}
}

// La qualification par bucket de scoop (« main/flux ») doit, elle aussi, atteindre argv —
// en un seul élément, puisque « main/flux » est un identifiant unique pour scoop, pas deux
// arguments.
func TestRoutageAppliqueLaQualificationDeScoop(t *testing.T) {
	s := pm.NewCatalog()
	s.Add("flux", pm.BadgeScoop)
	s.Qualified["flux"] = "main/flux"
	s.Sort()
	cats := map[string]*pm.Catalog{"scoop": s}

	cibles, amb, err := Router("install", []string{"flux"}, "", nil, []pm.Manager{scoop.New()}, cats)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 || len(cibles[0].Args) != 1 || cibles[0].Args[0] != "main/flux" {
		t.Fatalf("args = %v, attendu [main/flux]", cibles[0].Args)
	}
}

// Une désambiguïsation ne doit lier que le nom pour lequel elle a été faite : `jg install
// git fd`, avec « git » ambigu (winget et scoop) et « fd » scoop seul, doit laisser « fd »
// se résoudre tout seul une fois « git » tranché — pas forcer winget sur toute la ligne
// (cf. tâche « installation mixte »). C'est ce que `resolu` porte : une résolution par nom,
// distincte de --pm qui, lui, s'applique à toute la ligne.
func TestRoutageResolutionDUnNomNeForcePasLesAutres(t *testing.T) {
	resolu := map[string]string{"git": "winget"}
	cibles, amb, err := Router("install", []string{"git", "fd"}, "", resolu, deuxGestionnaires(), catalogues())
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v — « git » est déjà résolu par `resolu`", amb)
	}
	if len(cibles) != 2 {
		t.Fatalf("%d cibles, attendu 2 : %v", len(cibles), cibles)
	}
	vu := map[string][]string{}
	for _, c := range cibles {
		vu[c.Mgr.Cmd()] = c.Args
	}
	if len(vu["winget"]) != 1 || vu["winget"][0] != "git" {
		t.Errorf("winget = %v, attendu [git] (résolu par `resolu`)", vu["winget"])
	}
	if len(vu["scoop"]) != 1 || vu["scoop"][0] != "fd" {
		t.Errorf("scoop = %v, attendu [fd] (résolu tout seul, sans que le choix sur git ne le force)", vu["scoop"])
	}
}

// Les guillemets que winget pose autour d'un identifiant à espace protègent la commande
// shell — mais l'exécution façade passe par exec.Command, pas par un shell : ils
// doivent donc être retirés, et l'identifiant tenir en un seul élément d'argv.
func TestRoutageProtegeIdentifiantWingetAvecEspace(t *testing.T) {
	w := pm.NewCatalog()
	w.MarkInstalled("Canon IJ Scan Utility", "2.2.0.5", pm.BadgeOther)
	w.Sort()
	cats := map[string]*pm.Catalog{"winget": w}

	cibles, amb, err := Router("uninstall", []string{"Canon IJ Scan Utility"}, "", nil, []pm.Manager{winget.New()}, cats)
	if err != nil {
		t.Fatal(err)
	}
	if amb != nil {
		t.Fatalf("ambiguïté inattendue : %v", amb)
	}
	if len(cibles) != 1 || len(cibles[0].Args) != 1 || cibles[0].Args[0] != "Canon IJ Scan Utility" {
		t.Fatalf("args = %v, attendu [\"Canon IJ Scan Utility\"] en un seul élément, sans guillemets", cibles[0].Args)
	}
}
