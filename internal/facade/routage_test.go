package facade

import (
	"strings"
	"testing"

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
