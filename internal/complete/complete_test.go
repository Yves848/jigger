package complete

import (
	"fmt"
	"os"
	"path/filepath"
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
	if got := res.InsertItem(Item{Name: "firefox"}); got != "--cask firefox" {
		t.Fatalf("insertion de firefox = %q, attendu « --cask firefox »", got)
	}
	if got := brewComplete("install git", cat).InsertItem(Item{Name: "git"}); got != "git" {
		t.Fatalf("insertion de git = %q, attendu « git »", got)
	}
	// Ligne qui tranche déjà : on n'ajoute rien.
	if got := brewComplete("install --cask fire", cat).InsertItem(Item{Name: "firefox"}); got != "firefox" {
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
	if got := res.InsertItem(Item{Name: "flux"}); got != "main/flux" {
		t.Fatalf("insertion = %q, attendu « main/flux »", got)
	}
	// Un nom sans ambiguïté s'insère tel quel.
	if got := res.InsertItem(Item{Name: "winrar"}); got != "winrar" {
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
	if got := res.InsertItem(Item{Name: "Canon IJ Scan Utility"}); got != `"Canon IJ Scan Utility"` {
		t.Fatalf("insertion = %q, attendue entre guillemets", got)
	}
}

// dispoDeTest fournit une liste de gestionnaires explicite. `Complete` interroge la
// machine — sans gestionnaire installé, elle ne propose aucun verbe, ce qui est le bon
// comportement mais rend le test dépendant de l'endroit où il tourne : il passait sur un
// Mac avec Homebrew et échouait dans le conteneur de la CI.
func dispoDeTest() ([]pm.Manager, map[string]*pm.Catalog) {
	dispo := []pm.Manager{brew.New(), scoop.New(), winget.New()}
	cats := map[string]*pm.Catalog{}
	for _, m := range dispo {
		cats[m.Cmd()] = testCatalog()
	}
	return dispo, cats
}

// « jg ⇥ » complète le vocabulaire de la façade, pas les sous-commandes d'un
// gestionnaire : les clés des tables SONT le vocabulaire.
func TestFacade_CompleteLesVerbes(t *testing.T) {
	dispo, cats := dispoDeTest()
	res := CompleteFacade("jg ", dispo, cats)
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
	dispo, cats := dispoDeTest()
	res := CompleteFacade("jg source ", dispo, cats)
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

// « jigger » en toutes lettres marche comme « jg ». Le routage se teste sur estFacade,
// qui est la décision elle-même ; le contenu proposé, lui, dépendrait des gestionnaires
// présents sur la machine.
func TestFacade_NomComplet(t *testing.T) {
	for _, mot := range []string{"jigger", "jg"} {
		if !estFacade(mot) {
			t.Errorf("%q doit déclencher la façade", mot)
		}
	}
	dispo, cats := dispoDeTest()
	if len(CompleteFacade("jigger ", dispo, cats).Items) == 0 {
		t.Fatal("« jigger » doit proposer les verbes comme « jg »")
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

// C'est le popup de la note ¹ : « jg install fire » doit produire la même correction que
// « brew install fire » — --cask, pas le nom brut. Un résultat façade mélange les
// gestionnaires d'un Item à l'autre (cf. TestFacade_ItemPorteSonPM), donc la correction se
// résout par Item, via son champ PM, et non plus une fois pour tout le Result.
func TestFacade_InsertItemAppliqueLaCorrectionDuBonGestionnaire(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firealpaca"}, nil)}

	res := CompleteFacade("jg install fire", []pm.Manager{brew.New()}, cats)
	if len(res.Items) != 1 {
		t.Fatalf("items = %v, attendu 1", names(res.Items))
	}
	if got := res.InsertItem(res.Items[0]); got != "--cask firealpaca" {
		t.Fatalf("insertion = %q, attendu « --cask firealpaca »", got)
	}
}

// Un Item sans PM — construit à la main, hors de ce que CompleteFacade a produit — n'a
// par construction aucun moyen de savoir quel gestionnaire l'a proposé sur un résultat
// façade : InsertItem rend le nom brut, tel quel, plutôt que de deviner un gestionnaire
// par défaut qui n'existe pas.
func TestFacade_InsertItemSansPMNeCorrigePas(t *testing.T) {
	cats := map[string]*pm.Catalog{"brew": brew.NewCatalog(nil, []string{"firealpaca"}, nil)}

	res := CompleteFacade("jg install fire", []pm.Manager{brew.New()}, cats)
	if got := res.InsertItem(Item{Name: "firealpaca"}); got != "firealpaca" {
		t.Fatalf("insertion = %q, attendu « firealpaca » (aucun contexte PM)", got)
	}
}

// Un résultat façade avec plusieurs gestionnaires : chaque Item se corrige avec le sien,
// pas avec celui d'un autre.
func TestFacade_InsertItemMelangeDeuxGestionnaires(t *testing.T) {
	cats := map[string]*pm.Catalog{
		"brew": brew.NewCatalog(nil, []string{"firealpaca"}, nil),
	}
	s := pm.NewCatalog()
	s.Add("flux", pm.BadgeScoop)
	s.Qualified["flux"] = "main/flux"
	s.Sort()
	cats["scoop"] = s

	res := CompleteFacade("jg install f", []pm.Manager{brew.New(), scoop.New()}, cats)
	got := map[string]string{}
	for _, it := range res.Items {
		got[it.Name] = res.InsertItem(it)
	}
	if got["firealpaca"] != "--cask firealpaca" {
		t.Fatalf("firealpaca = %q, attendu « --cask firealpaca »", got["firealpaca"])
	}
	if got["flux"] != "main/flux" {
		t.Fatalf("flux = %q, attendu « main/flux »", got["flux"])
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

// BenchmarkCompleteFacade garde un œil sur le chemin le plus coûteux du popup : « jg
// install g » doit filtrer TROIS catalogues à chaque frappe, là où le chemin natif n'en
// filtre qu'un. Sous Windows, celui de winget compte à lui seul 14 401 noms.
//
// Budget indicatif : du même ordre que BenchmarkComplete. S'il s'en écarte d'un facteur,
// c'est que le filtrage est reparti dans le mauvais sens — réunir puis balayer, au lieu
// de filtrer chez chaque gestionnaire puis réunir (cf. spec §5).
func BenchmarkCompleteFacade(b *testing.B) {
	gros := func(prefixe string, n int) *pm.Catalog {
		cat := pm.NewCatalog()
		for i := range n {
			cat.Add(fmt.Sprintf("%s-%05d", prefixe, i), pm.BadgeWinget)
		}
		cat.Sort()
		return cat
	}
	cats := map[string]*pm.Catalog{
		"winget": gros("gadget", 14401),
		"scoop":  gros("gizmo", 3000),
		"brew":   gros("gubbins", 8000),
	}
	dispo := []pm.Manager{brew.New(), winget.New(), scoop.New()}

	b.ResetTimer()
	for b.Loop() {
		CompleteFacade("jg install g", dispo, cats)
	}
}

// fauxManagerSansSub est un fournisseur de complétion sans verbes, comme ssh.
type fauxManagerSansSub struct{ cmd string }

func (f fauxManagerSansSub) Cmd() string                               { return f.cmd }
func (fauxManagerSansSub) Subcommands() []string                       { return nil }
func (fauxManagerSansSub) Options(string) []string                     { return nil }
func (fauxManagerSansSub) InstalledOnly(string) bool                   { return false }
func (fauxManagerSansSub) Available() bool                             { return true }
func (fauxManagerSansSub) Load() *pm.Catalog                           { return nil }
func (fauxManagerSansSub) Warm(pm.Scope) error                         { return nil }
func (fauxManagerSansSub) Insert(_ *pm.Catalog, _, _, n string) string { return n }

func catalogueHotes() *pm.Catalog {
	c := pm.NewCatalog()
	c.Add("archlight", "")
	c.Add("pve", "")
	c.Versions["pve"] = "192.168.50.8"
	c.Sort()
	return c
}

func TestSansSousCommandeLeCatalogueVientDesLePremierMot(t *testing.T) {
	// C'est la regle de l'ADR-0005. « ssh arch » n'a pas de verbe : arch est deja
	// l'operande.
	res := CompleteWith("ssh arch", fauxManagerSansSub{"ssh"}, catalogueHotes())
	if len(res.Items) != 1 || res.Items[0].Name != "archlight" {
		t.Fatalf("Items = %v, attendu [archlight]", res.Items)
	}
}

func TestSansSousCommandeLaCommandeSeuleProposeTout(t *testing.T) {
	res := CompleteWith("ssh ", fauxManagerSansSub{"ssh"}, catalogueHotes())
	if len(res.Items) != 2 {
		t.Fatalf("Items = %v, attendu les deux hotes", res.Items)
	}
}

func TestSansSousCommandeLAdresseSuitDansVersion(t *testing.T) {
	res := CompleteWith("ssh pve", fauxManagerSansSub{"ssh"}, catalogueHotes())
	if len(res.Items) != 1 || res.Items[0].Version != "192.168.50.8" {
		t.Fatalf("Items = %+v, attendu pve avec son adresse", res.Items)
	}
}

func TestSansSousCommandeNEstJamaisExecutable(t *testing.T) {
	// Executable commande si ⏎ EXECUTE dans le selecteur plein ecran. Un fournisseur
	// sans pm.Bindings n'a rien a executer : le picker doit inserer, pas lancer.
	res := CompleteWith("ssh arch", fauxManagerSansSub{"ssh"}, catalogueHotes())
	if res.Executable {
		t.Error("Executable = true pour un fournisseur sans verbes")
	}
}

// fauxIndisponible est le meme fournisseur sans verbes, mais sur une machine qui n'a pas
// sa configuration : c'est ce qu'est ssh sans ~/.ssh/config.
type fauxIndisponible struct{ fauxManagerSansSub }

func (fauxIndisponible) Available() bool { return false }

// fauxAVerbesIndisponible a des sous-commandes et n'est pas installe : c'est brew sur une
// machine sans Homebrew, cas que pm.Manager.Available() documente comme devant rester
// complete (« la completion repond pour tous »).
type fauxAVerbesIndisponible struct{ fauxManagerSansSub }

func (fauxAVerbesIndisponible) Subcommands() []string { return []string{"install", "search"} }
func (fauxAVerbesIndisponible) Available() bool       { return false }

func TestFournisseurSansVerbesIndisponibleSeTait(t *testing.T) {
	// Sans cette regle, chaque frappe d'une ligne ssh sur une machine sans
	// ~/.ssh/config dessinait un cadre « aucun candidat » sous le prompt.
	res := CompleteWith("ssh serv", fauxIndisponible{fauxManagerSansSub{"ssh"}}, pm.NewCatalog())
	if len(res.Items) != 0 {
		t.Fatalf("Items = %v, attendu aucun", res.Items)
	}
	if !res.Silencieux {
		t.Error("Silencieux = false : le popup dessinerait une boite vide a chaque frappe")
	}
}

func TestFournisseurSansVerbesDisponibleNeSeTaitPas(t *testing.T) {
	// Une liste vide n'est pas un silence quand le fournisseur est la : « ssh zzz » sur
	// une vraie configuration doit dire « aucun candidat », pas disparaitre.
	res := CompleteWith("ssh zzz", fauxManagerSansSub{"ssh"}, catalogueHotes())
	if len(res.Items) != 0 {
		t.Fatalf("Items = %v, attendu aucun", res.Items)
	}
	if res.Silencieux {
		t.Error("Silencieux = true alors que le fournisseur est disponible")
	}
}

func TestGestionnaireAVerbesIndisponibleNeSeTaitPas(t *testing.T) {
	// Le cas qui compte est celui ou la liste est VIDE : c'est la seule facon d'atteindre
	// la condition, donc la seule ou le garde-fou garde quelque chose. « brew zzz » ne
	// correspond a aucune sous-commande, et brew est indisponible — si la regle oubliait
	// d'exiger sansVerbes, le popup disparaitrait ici. Sur une machine sans Homebrew, un
	// « winget zzz » sous Windows doit continuer d'afficher son cadre « aucune
	// correspondance » plutot que de s'evanouir sous les doigts.
	res := CompleteWith("brew zzz", fauxAVerbesIndisponible{fauxManagerSansSub{"brew"}}, pm.NewCatalog())
	if len(res.Items) != 0 {
		t.Fatalf("Items = %v, attendu aucune correspondance", res.Items)
	}
	if res.Silencieux {
		t.Error("Silencieux = true pour un gestionnaire a verbes : le cadre disparaitrait")
	}
}

func TestSshSeTaitSansConfigurationSSH(t *testing.T) {
	// Le vrai chemin, de bout en bout : Complete → managers.Detect → ssh.Manager. C'est
	// le test qui tombe si ssh.Available() est mutee en « return true », mutation a
	// laquelle toute la suite survivait jusqu'ici.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir() sous Windows

	if res := Complete("ssh serv"); !res.Silencieux {
		t.Errorf("Silencieux = false sans ~/.ssh/config (Items = %v)", res.Items)
	}

	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".ssh", "config")
	if err := os.WriteFile(cfg, []byte("Host serveur\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := Complete("ssh serv")
	if res.Silencieux {
		t.Error("Silencieux = true alors que ~/.ssh/config existe")
	}
	if len(res.Items) != 1 || res.Items[0].Name != "serveur" {
		t.Fatalf("Items = %v, attendu [serveur]", res.Items)
	}
}
