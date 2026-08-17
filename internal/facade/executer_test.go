package facade

import (
	"errors"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// appel enregistre ce que le moteur aurait lancé.
type appel struct {
	cmd  string
	args []string
}

// simuler remplace le lanceur réel le temps d'un test, et rend les appels enregistrés.
func simuler(t *testing.T, sortie []byte, codes map[string]int) *[]appel {
	t.Helper()
	var vus []appel
	precedent := lancer
	lancer = func(cmd string, args []string, relais bool) ([]byte, int, error) {
		vus = append(vus, appel{cmd, args})
		if c := codes[cmd]; c != 0 {
			return nil, c, errors.New("échec simulé")
		}
		return sortie, 0, nil
	}
	t.Cleanup(func() { lancer = precedent })
	return &vus
}

func TestExecuterRelaieEtPropageLeCode(t *testing.T) {
	vus := simuler(t, nil, map[string]int{"scoop": 3})

	_, code := Executer("install", []Cible{{Mgr: scoop.New(), Args: []string{"fd"}}}, Opts{})
	if code != 3 {
		t.Fatalf("code = %d, attendu 3 — le code du gestionnaire doit passer tel quel", code)
	}
	if len(*vus) != 1 || (*vus)[0].cmd != "scoop" {
		t.Fatalf("appels = %v", *vus)
	}
}

// winget ne prend qu'un id par appel : deux paquets font deux invocations.
func TestExecuterUnAppelParPaquetChezWinget(t *testing.T) {
	vus := simuler(t, nil, nil)

	_, code := Executer("install",
		[]Cible{{Mgr: winget.New(), Args: []string{"Git.Git", "7zip.7zip"}}}, Opts{})
	if code != 0 {
		t.Fatalf("code = %d, attendu 0", code)
	}
	if len(*vus) != 2 {
		t.Fatalf("%d appels, attendu 2 : %v", len(*vus), *vus)
	}
}

// Verbe mutant : on n'installe pas depuis scoop si winget vient d'échouer.
func TestEcritureSArreteALaPremiereErreur(t *testing.T) {
	vus := simuler(t, nil, map[string]int{"winget": 1})

	_, code := Executer("install", []Cible{
		{Mgr: winget.New(), Args: []string{"Git.Git"}},
		{Mgr: scoop.New(), Args: []string{"fd"}},
	}, Opts{})

	if code != 1 {
		t.Fatalf("code = %d, attendu 1", code)
	}
	for _, a := range *vus {
		if a.cmd == "scoop" {
			t.Fatal("scoop ne devait pas être lancé après l'échec de winget")
		}
	}
}

// Verbe en lecture : au mieux. Un gestionnaire en panne ne doit pas rendre `jg outdated`
// inutile.
func TestLectureContinueMalgreUnEchec(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 1})

	_, code := Executer("outdated", []Cible{
		{Mgr: winget.New(), Args: nil},
		{Mgr: scoop.New(), Args: nil},
	}, Opts{})

	if code != 0 {
		t.Fatalf("code = %d, attendu 0 — scoop a répondu", code)
	}
}

// "list", pas "outdated" : chez scoop, "outdated" est Direct (aucun sous-processus, cf.
// TestDirectNeLancePersonne) et OutdatedApps() n'a délibérément aucun chemin d'erreur —
// lire un disque vide est une réponse, pas un échec. Le seam `lancer` ne peut donc pas
// faire échouer scoop sur ce verbe. "list" est Native chez les deux gestionnaires : les
// deux passent par `lancer`, et le scénario « personne ne répond » devient possible.
func TestLectureEchoueSiPersonneNeRepond(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 1, "scoop": 1})

	_, code := Executer("list", []Cible{
		{Mgr: winget.New(), Args: nil},
		{Mgr: scoop.New(), Args: nil},
	}, Opts{})

	if code == 0 {
		t.Fatal("aucun gestionnaire n'a répondu : le code ne peut pas être 0")
	}
}

// --yes ajoute les accords winget. Sans lui, jigger n'accepte rien à la place de
// l'utilisateur : les invites passent, puisque la sortie est relayée.
func TestYesAjouteLesAccordsWinget(t *testing.T) {
	vus := simuler(t, nil, nil)

	Executer("install", []Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{Yes: true})

	if len(*vus) != 1 {
		t.Fatalf("appels = %v", *vus)
	}
	var accord, source bool
	for _, a := range (*vus)[0].args {
		switch a {
		case "--accept-package-agreements":
			accord = true
		case "--accept-source-agreements":
			source = true
		}
	}
	if !accord || !source {
		t.Fatalf("--yes doit ajouter les deux accords : %v", (*vus)[0].args)
	}
}

func TestSansYesAucunAccord(t *testing.T) {
	vus := simuler(t, nil, nil)

	Executer("install", []Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{})

	for _, a := range (*vus)[0].args {
		if a == "--accept-package-agreements" {
			t.Fatal("jigger ne doit jamais accepter une licence sans --yes")
		}
	}
}

// Direct ne lance aucun sous-processus.
func TestDirectNeLancePersonne(t *testing.T) {
	vus := simuler(t, nil, nil)

	Executer("outdated", []Cible{{Mgr: scoop.New(), Args: nil}}, Opts{})

	for _, a := range *vus {
		if a.cmd == "scoop" {
			t.Fatal("outdated chez scoop passe par Direct : aucun appel attendu")
		}
	}
}

// jouetSansParse est un gestionnaire fictif dont la table déclare un verbe normalisé
// (Pool: Aucun, comme list/outdated/search/source) sans Parse ni Direct — exactement la
// forme que scoop avait avant la tâche 9 (list, search, source en Native sans parser).
// Elle sert à prouver que la garde de Executer protège N'IMPORTE QUELLE table dans cette
// forme, pas seulement scoop une fois corrigé.
type jouetSansParse struct{}

func (jouetSansParse) Cmd() string                                       { return "jouet" }
func (jouetSansParse) Subcommands() []string                             { return nil }
func (jouetSansParse) Options(string) []string                           { return nil }
func (jouetSansParse) InstalledOnly(string) bool                         { return false }
func (jouetSansParse) Available() bool                                   { return true }
func (jouetSansParse) Load() *pm.Catalog                                 { return pm.NewCatalog() }
func (jouetSansParse) Insert(*pm.Catalog, string, string, string) string { return "" }
func (jouetSansParse) Warm(pm.Scope) error                               { return nil }
func (jouetSansParse) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"list": {Native: []string{"list"}, Pool: pm.PoolAucun}, // pas de Parse
	}
}

// Un verbe normalisé (sortie capturée) sans Parse ni Direct ne doit jamais rendre un
// succès silencieux : la sortie captée serait perdue sans que rien ne le dise. C'est
// exactement le bug de scoop avant la tâche 9 — list, search et source y étaient Native
// sans parser, donc `jg list` sous Windows omettait purement et simplement scoop en
// sortant 0.
func TestVerbeNormaliseSansParseNAvalePasLaSortieEnSilence(t *testing.T) {
	simuler(t, []byte("des données que personne ne lira\n"), nil)

	rows, code := Executer("list", []Cible{{Mgr: jouetSansParse{}, Args: nil}}, Opts{})
	if code == 0 {
		t.Fatal("code = 0 : un verbe normalisé sans analyseur déclaré ne doit pas réussir silencieusement")
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu aucune ligne : rien n'a été analysé", rows)
	}
}

func TestVerbesNormalisesSontCaptures(t *testing.T) {
	if !Normalise("outdated") || !Normalise("list") || !Normalise("search") || !Normalise("source") {
		t.Error("les quatre verbes tabulaires doivent être normalisés")
	}
	for _, v := range []pm.Verb{"install", "info", "doctor", "source add"} {
		if Normalise(v) {
			t.Errorf("le verbe %q doit être relayé, pas normalisé", v)
		}
	}
}

// ── Élévation constatée (ADR-0004) ────────────────────────────────────────────────────
//
// La façade ne demande rien et n'élève rien : elle constate un code de sortie et le rend.
// Ces tests ne touchent donc ni au terminal, ni à Windows, ni à UAC.

// Le cas nominal : winget refuse faute de droits, et la façade rend de quoi rejouer.
func TestExecuterRemonteUnRejeuSurCodeAdmin(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 0x8A150019})

	res := ExecuterAvec("install", []Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{})

	if res.Rejeu == nil {
		t.Fatal("aucun rejeu : le code COMMAND_REQUIRES_ADMIN doit être reconnu")
	}
	if res.Rejeu.Droits != pm.DroitsRequis {
		t.Fatalf("droits = %v, attendu DroitsRequis", res.Rejeu.Droits)
	}
	if res.Rejeu.Cmd != "winget" {
		t.Fatalf("cmd = %q", res.Rejeu.Cmd)
	}
	if res.Code != 0x8A150019 {
		t.Fatalf("code = %d : le code du gestionnaire doit passer tel quel", res.Code)
	}
}

// L'argv rendu est celui qui a RÉELLEMENT tourné, accords compris. Rejouer autre chose que
// ce qui a échoué serait un piège : l'utilisateur accepterait une commande, une autre
// partirait.
func TestLeRejeuPorteLArgvReellementLance(t *testing.T) {
	vus := simuler(t, nil, map[string]int{"winget": 0x8A150019})

	res := ExecuterAvec("install",
		[]Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{Yes: true})

	if res.Rejeu == nil {
		t.Fatal("aucun rejeu")
	}
	if len(*vus) != 1 {
		t.Fatalf("appels = %v", *vus)
	}
	lance := (*vus)[0].args
	if len(res.Rejeu.Argv) != len(lance) {
		t.Fatalf("argv du rejeu = %v, lancé = %v", res.Rejeu.Argv, lance)
	}
	for i := range lance {
		if res.Rejeu.Argv[i] != lance[i] {
			t.Fatalf("argv du rejeu = %v, lancé = %v", res.Rejeu.Argv, lance)
		}
	}
	var accord bool
	for _, a := range res.Rejeu.Argv {
		if a == "--accept-package-agreements" {
			accord = true
		}
	}
	if !accord {
		t.Fatalf("les accords de --yes manquent au rejeu : %v", res.Rejeu.Argv)
	}
}

// Le contre-cas remonte aussi — mais marqué comme tel, pour que l'appelant dise l'inverse
// au lieu de proposer d'élever.
func TestExecuterRemonteLInterdictionDElevation(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 0x8A150056})

	res := ExecuterAvec("install", []Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{})

	if res.Rejeu == nil {
		t.Fatal("aucun rejeu : INSTALLER_PROHIBITS_ELEVATION doit être reconnu")
	}
	if res.Rejeu.Droits != pm.DroitsInterdits {
		t.Fatalf("droits = %v, attendu DroitsInterdits", res.Rejeu.Droits)
	}
}

// Un échec ordinaire ne propose rien. C'est la garde qui empêche « code non nul → propose
// d'élever », lequel serait faux la plupart du temps et nuisible deux fois sur quatre.
func TestPasDeRejeuSurUnEchecOrdinaire(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 0x8A150014}) // NO_APPLICATIONS_FOUND

	res := ExecuterAvec("install", []Cible{{Mgr: winget.New(), Args: []string{"Zzz.Zzz"}}}, Opts{})
	if res.Rejeu != nil {
		t.Fatalf("rejeu = %+v : un échec qui ne parle pas de droits ne propose rien", res.Rejeu)
	}
}

// Un gestionnaire qui n'implémente pas le contrat ne fait rien proposer, quel que soit le
// code — même si celui-ci vaut par coïncidence celui de winget.
func TestPasDeRejeuSansContrat(t *testing.T) {
	simuler(t, nil, map[string]int{"scoop": 0x8A150019})

	res := ExecuterAvec("install", []Cible{{Mgr: scoop.New(), Args: []string{"fd"}}}, Opts{})
	if res.Rejeu != nil {
		t.Fatalf("rejeu = %+v : scoop ne déclare pas savoir lire ses codes", res.Rejeu)
	}
}

// Une lecture qui échoue enchaîne sur le gestionnaire suivant : il n'y a rien à rejouer,
// et proposer une élévation au milieu d'un `jg list` n'aurait aucun sens.
func TestPasDeRejeuSurUneLecture(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 0x8A150019})

	res := ExecuterAvec("list", []Cible{{Mgr: winget.New(), Args: nil}}, Opts{})
	if res.Rejeu != nil {
		t.Fatalf("rejeu = %+v : une lecture ne se rejoue pas", res.Rejeu)
	}
}

// Executer garde sa signature courte, et son comportement : c'est ce que dix sites
// d'appel attendent.
func TestExecuterCourtRendToujoursLeCouple(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 0x8A150019})

	rows, code := Executer("install", []Cible{{Mgr: winget.New(), Args: []string{"Git.Git"}}}, Opts{})
	if code != 0x8A150019 {
		t.Fatalf("code = %d", code)
	}
	if rows != nil {
		t.Fatalf("rows = %v, attendu nil", rows)
	}
}
