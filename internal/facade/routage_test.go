package facade

import (
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
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
	cibles, amb, err := Router("install", []string{"fd"}, "", deuxGestionnaires(), catalogues())
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
	_, amb, err := Router("install", []string{"git"}, "", deuxGestionnaires(), catalogues())
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
	cibles, amb, err := Router("install", []string{"git"}, "scoop", deuxGestionnaires(), catalogues())
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
	cibles, amb, err := Router("install", []string{"fd", "Git.Git"}, "", deuxGestionnaires(), catalogues())
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
	cibles, amb, err := Router("uninstall", []string{"git"}, "", deuxGestionnaires(), catalogues())
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
	cibles, amb, err := Router("outdated", nil, "", deuxGestionnaires(), catalogues())
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
	_, _, err := Router("install", []string{"fdfind"}, "", deuxGestionnaires(), catalogues())
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
	_, _, err := Router("uninstall", []string{"gitt"}, "", deuxGestionnaires(), catalogues())
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

	_, _, err := Router("install", []string{"Git.Git"}, "", deuxGestionnaires(), cats)
	if err == nil {
		t.Fatal("attendu une erreur")
	}
	if !strings.Contains(err.Error(), "en cours de constitution") {
		t.Errorf("la note du catalogue doit primer : %s", err.Error())
	}
}

func TestRoutageForcePMInconnu(t *testing.T) {
	_, _, err := Router("install", []string{"fd"}, "apt", deuxGestionnaires(), catalogues())
	if err == nil {
		t.Fatal("attendu une erreur : « apt » n'est pas un gestionnaire disponible")
	}
}

// Un drapeau natif (« --cask ») ne doit pas être avalé comme s'il était un nom de paquet à
// résoudre : il doit ressortir intact dans les arguments de la cible, et en tête — c'est
// « brew install --cask firefox », pas « brew install firefox --cask ».
func TestRoutageDrapeauxNatifsAccompagnentLesNoms(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firefox"}, nil)}
	cibles, amb, err := Router("install", []string{"--cask", "firefox"}, "", []pm.Manager{brew.New()}, cats)
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

// PoolAucun (list, outdated…) ne résout aucun nom : les drapeaux doivent lui parvenir
// intacts, sans passer par le partitionnement drapeaux/noms.
func TestRoutagePoolAucunLaisseFilerLesDrapeaux(t *testing.T) {
	cibles, amb, err := Router("list", []string{"--versions"}, "", []pm.Manager{brew.New()}, nil)
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
	cibles, amb, err := Router("install", []string{"firealpaca"}, "", []pm.Manager{brew.New()}, cats)
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

	cibles, amb, err := Router("install", []string{"flux"}, "", []pm.Manager{scoop.New()}, cats)
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

// Les guillemets que winget pose autour d'un identifiant à espace protègent la commande
// shell — mais l'exécution façade passe par exec.Command, pas par un shell : ils
// doivent donc être retirés, et l'identifiant tenir en un seul élément d'argv.
func TestRoutageProtegeIdentifiantWingetAvecEspace(t *testing.T) {
	w := pm.NewCatalog()
	w.MarkInstalled("Canon IJ Scan Utility", "2.2.0.5", pm.BadgeOther)
	w.Sort()
	cats := map[string]*pm.Catalog{"winget": w}

	cibles, amb, err := Router("uninstall", []string{"Canon IJ Scan Utility"}, "", []pm.Manager{winget.New()}, cats)
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
