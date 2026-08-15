package complete

import (
	"fmt"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

func testCatalog() *pm.Catalog {
	return brew.NewCatalog(
		[]string{"git", "wget", "node", "ripgrep", "firefly"},        // formulae
		[]string{"firefox", "visual-studio-code", "firefly-desktop"}, // casks
		[]string{"git", "ripgrep"},                                   // installés
	)
}

// complete est Complete sur un catalogue donné, pour le gestionnaire donné.
func complete(line string, m pm.Manager, cat *pm.Catalog) Result {
	return CompleteWith(line, m, cat)
}

func brewComplete(line string, cat *pm.Catalog) Result {
	return complete(line, brew.New(), cat)
}

func names(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}

func TestFirstWord_CompletesSubcommands(t *testing.T) {
	res := brewComplete("inst", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "install" {
		t.Fatalf("attendu [install], obtenu %v", got)
	}
	if res.Executable {
		t.Fatal("une sous-commande ne doit pas être « exécutable »")
	}
}

func TestOptionCompletion(t *testing.T) {
	res := brewComplete("list --vers", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "--versions" {
		t.Fatalf("attendu [--versions], obtenu %v", got)
	}
}

func TestInstall_CompletesAllPackages(t *testing.T) {
	res := brewComplete("install fire", testCatalog())
	got := names(res.Items)
	// firefly (formula) + firefox, firefly-desktop (casks)
	want := map[string]bool{"firefly": true, "firefox": true, "firefly-desktop": true}
	if len(got) != len(want) {
		t.Fatalf("attendu %d candidats, obtenu %v", len(want), got)
	}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("candidat inattendu %q dans %v", n, got)
		}
	}
	if !res.Executable {
		t.Fatal("un contexte paquet doit être « exécutable »")
	}
}

func TestUninstall_CompletesInstalledOnly(t *testing.T) {
	res := brewComplete("uninstall ", testCatalog())
	got := names(res.Items)
	if len(got) != 2 {
		t.Fatalf("attendu 2 installés, obtenu %v", got)
	}
	for _, n := range got {
		if n != "git" && n != "ripgrep" {
			t.Fatalf("non installé %q proposé pour uninstall: %v", n, got)
		}
	}
}

func TestBadgeAndCaskDetection(t *testing.T) {
	cat := testCatalog()
	res := brewComplete("install firefox", cat)
	if len(res.Items) != 1 || res.Items[0].Badge != pm.BadgeCask {
		t.Fatalf("firefox devrait porter le badge C, obtenu %v", res.Items)
	}
	// Un cask pur derrière install : brew exige --cask, jigger l'ajoute.
	if got := res.Insert("firefox"); got != "--cask firefox" {
		t.Fatalf("insertion de firefox = %q, attendu « --cask firefox »", got)
	}
	if got := brewComplete("install git", cat).Insert("git"); got != "git" {
		t.Fatalf("insertion de git = %q, attendu « git »", got)
	}
	// Ligne qui tranche déjà : on n'ajoute rien.
	if got := brewComplete("install --cask fire", cat).Insert("firefox"); got != "firefox" {
		t.Fatalf("insertion derrière --cask = %q, attendu « firefox »", got)
	}
}

func TestStripsLeadingCommand(t *testing.T) {
	res := brewComplete("brew uninstall gi", testCatalog())
	if got := names(res.Items); len(got) != 1 || got[0] != "git" {
		t.Fatalf("attendu [git], obtenu %v", got)
	}
}

// Le mot de commande peut arriver avec son chemin et son extension — c'est le cas d'une
// ligne complétée par PowerShell.
func TestStripsCommandPath(t *testing.T) {
	cat := scoopCatalog()
	res := complete(`C:\Users\y\scoop\shims\scoop.exe install 7z`, scoop.New(), cat)
	if got := names(res.Items); len(got) != 1 || got[0] != "7zip" {
		t.Fatalf("attendu [7zip], obtenu %v", got)
	}
	if res.Title() != "scoop install" {
		t.Fatalf("titre = %q", res.Title())
	}
}

// scoopCatalog imite ce que LoadFrom construit : deux buckets, dont un nom présent dans
// les deux.
func scoopCatalog() *pm.Catalog {
	cat := pm.NewCatalog()
	cat.Add("7zip", pm.BadgeScoop)
	cat.Add("flux", pm.BadgeScoop)
	cat.Add("winrar", pm.BadgeOther)
	cat.Qualified["flux"] = "main/flux"
	cat.MarkInstalled("7zip", "26.00", pm.BadgeScoop)
	cat.Sort()
	return cat
}

func TestScoop_QualifieUnNomAmbigu(t *testing.T) {
	res := complete("scoop install fl", scoop.New(), scoopCatalog())
	if got := res.Insert("flux"); got != "main/flux" {
		t.Fatalf("insertion = %q, attendu « main/flux »", got)
	}
	// Un nom sans ambiguïté s'insère tel quel.
	if got := res.Insert("winrar"); got != "winrar" {
		t.Fatalf("insertion = %q, attendu « winrar »", got)
	}
}

func TestWinget_SousCommandesEtOptions(t *testing.T) {
	cat := pm.NewCatalog()
	cat.Add("Git.Git", pm.BadgeWinget)
	cat.MarkInstalled("Canon IJ Scan Utility", "2.2.0.5", pm.BadgeOther)
	cat.Sort()

	if got := names(complete("winget unin", winget.New(), cat).Items); len(got) != 1 || got[0] != "uninstall" {
		t.Fatalf("attendu [uninstall], obtenu %v", got)
	}
	if got := names(complete("winget install --ex", winget.New(), cat).Items); len(got) != 1 || got[0] != "--exact" {
		t.Fatalf("attendu [--exact], obtenu %v", got)
	}
	// `uninstall` ne propose que des paquets installés — et protège celui dont
	// l'identifiant contient des espaces.
	res := complete("winget uninstall ", winget.New(), cat)
	if got := names(res.Items); len(got) != 1 || got[0] != "Canon IJ Scan Utility" {
		t.Fatalf("attendu le seul installé, obtenu %v", got)
	}
	if got := res.Insert("Canon IJ Scan Utility"); got != `"Canon IJ Scan Utility"` {
		t.Fatalf("insertion = %q, attendue entre guillemets", got)
	}
}

// « jg ⇥ » complète le vocabulaire de la façade, pas les sous-commandes d'un
// gestionnaire : les clés des tables SONT le vocabulaire.
func TestFacade_CompleteLesVerbes(t *testing.T) {
	res := Complete("jg ")
	got := names(res.Items)
	if len(got) == 0 {
		t.Fatal("aucun verbe proposé")
	}
	attendus := map[string]bool{"install": false, "outdated": false, "search": false}
	for _, n := range got {
		if _, veut := attendus[n]; veut {
			attendus[n] = true
		}
	}
	for v, vu := range attendus {
		if !vu {
			t.Errorf("verbe %q absent de %v", v, got)
		}
	}
}

func TestFacade_CompleteLeSousVerbe(t *testing.T) {
	res := Complete("jg source ")
	got := names(res.Items)
	var add, rm bool
	for _, n := range got {
		switch n {
		case "add":
			add = true
		case "rm":
			rm = true
		}
	}
	if !add || !rm {
		t.Fatalf("« source » doit proposer add et rm, obtenu %v", got)
	}
}

// Le titre du cadre dit jigger, pas le nom d'un gestionnaire.
func TestFacade_Titre(t *testing.T) {
	if got := Complete("jg install ").Title(); got != "jigger install" {
		t.Fatalf("titre = %q, attendu « jigger install »", got)
	}
}

// « jigger » en toutes lettres marche comme « jg ».
func TestFacade_NomComplet(t *testing.T) {
	if len(Complete("jigger ").Items) == 0 {
		t.Fatal("« jigger » doit déclencher la façade comme « jg »")
	}
}

// Un candidat de la façade porte son gestionnaire : le badge ne suffit pas, BadgeOther
// étant partagé par winget et scoop.
func TestFacade_ItemPorteSonPM(t *testing.T) {
	cats := map[string]*pm.Catalog{}
	w := pm.NewCatalog()
	w.Add("Git.Git", pm.BadgeWinget)
	w.Sort()
	cats["winget"] = w

	res := CompleteFacade("jg install Git", []pm.Manager{winget.New()}, cats)
	if len(res.Items) != 1 {
		t.Fatalf("items = %v, attendu 1", names(res.Items))
	}
	if res.Items[0].PM != "winget" {
		t.Fatalf("PM = %q, attendu « winget »", res.Items[0].PM)
	}
}

// Le chemin natif ne change pas : « brew install ⇥ » ne porte aucun PM.
func TestNatif_PasDePM(t *testing.T) {
	res := brewComplete("install fire", testCatalog())
	for _, it := range res.Items {
		if it.PM != "" {
			t.Errorf("le chemin natif ne doit pas remplir PM : %+v", it)
		}
	}
}

// BenchmarkComplete garde un œil sur le filtrage : le popup vivant appelle Complete à
// chaque frappe, sur un catalogue de plusieurs milliers de noms. Budget indicatif :
// quelques ms au plus (le reste du budget part dans le démarrage du binaire).
func BenchmarkComplete(b *testing.B) {
	formulae := make([]string, 0, 8000)
	casks := make([]string, 0, 2000)
	for i := range 8000 {
		formulae = append(formulae, fmt.Sprintf("formula-%04d", i))
	}
	for i := range 2000 {
		casks = append(casks, fmt.Sprintf("cask-%04d", i))
	}
	cat := brew.NewCatalog(formulae, casks, formulae[:200])

	b.ResetTimer()
	for b.Loop() {
		brewComplete("brew install form", cat)
	}
}
