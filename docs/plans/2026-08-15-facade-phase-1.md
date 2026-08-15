# Façade multi-gestionnaires — plan d'implémentation, phase 1

> **Pour les agents :** SOUS-SKILL REQUIS — utiliser `superpowers:subagent-driven-development`
> (recommandé) ou `superpowers:executing-plans` pour dérouler ce plan tâche par tâche.
> Les étapes sont en cases à cocher (`- [ ]`) pour le suivi.

**But :** donner à jigger une syntaxe unique au-dessus de Homebrew, winget et scoop —
`jg install fd` — avec routage par résolution du nom et sortie normalisée.

**Architecture :** un second contrat `pm.Bindings`, indépendant de `pm.Manager`, où chaque
gestionnaire **déclare une table** `verbe → liaison`. Un moteur générique (`internal/facade`)
résout le verbe, résout la cible, construit l'argv, exécute et normalise. Les capacités
sont les clés de la table : un verbe absent est un verbe non supporté.

**Stack :** Go ≥ 1.24, Bubble Tea / Lip Gloss (déjà en place). Aucune dépendance nouvelle.

**Spec :** [docs/specs/2026-08-15-facade-multi-gestionnaires-design.md](../specs/2026-08-15-facade-multi-gestionnaires-design.md)
**Décisions :** [ADR-0001](../adr/0001-go-confirme.md) · [ADR-0002](../adr/0002-facade-table-declarative.md)

## Contraintes globales

Ces règles s'appliquent à **toutes** les tâches, sans être répétées dans chacune.

- **Go ≥ 1.24.** Aucune dépendance nouvelle ; le `go.mod` ne doit pas changer.
- **`pm.Manager` n'est pas modifié.** Le contrat d'exécution est un second contrat. En
  particulier, `Manager.InstalledOnly() bool` **reste tel quel** et continue de servir la
  complétion native ; `pm.Pool` est un type neuf porté par `Binding`.
- **Commentaires et messages en français**, comme tout le dépôt. Les tests aussi :
  `t.Fatalf("argv = %v, attendu %v", got, want)`.
- **Rien de lent dans le chemin d'un rendu.** `Manager.Load()` ne lit que des caches et des
  répertoires — cette interdiction vaut pour tout code appelé depuis `render`.
- **Aucun test ne lance de gestionnaire de paquets.** Les tables se testent sur l'argv
  produit, les parsers sur des fichiers de `testdata/`.
- **`make test` doit passer** à la fin de chaque tâche. `make test-all` ajoute les suites
  shell de la plateforme et n'est exigé que pour les tâches 14 et 15.
- **Un commit par tâche**, message en français, style nominal (cf. `git log --oneline`).

## Découpage en deux parties

**Partie A (tâches 1–11)** livre la façade en ligne de commande : `jigger install fd`
fonctionne de bout en bout.
**Partie B (tâches 12–16)** livre le popup et les greffons : `jg install ⇥` complète.

Les deux sont dans la phase 1 et s'exécutent d'affilée. La partie B n'est pas un
« plus tard » — la spec §5 en fait le fil conducteur visuel du produit.

## Carte des fichiers

**Créés**

| Fichier | Responsabilité |
|---|---|
| `internal/pm/verbs.go` | `Verb`, `Pool`, `Parser`, `Binding`, `Bindings`, construction d'argv |
| `internal/pm/package.go` | `Package` — la ligne de sortie normalisée |
| `internal/pm/verbs_test.go` | argv, validation des liaisons |
| `internal/brew/verbs.go` · `parse.go` | table brew · parsers JSON |
| `internal/winget/verbs.go` · `parse.go` | table winget · parsers de tableaux |
| `internal/scoop/verbs.go` · `parse.go` | table scoop · adaptateurs `Direct` |
| `internal/facade/verbe.go` | résolution du verbe, capacités, messages |
| `internal/facade/routage.go` | résolution de la cible, ambiguïté |
| `internal/facade/executer.go` | exécution, codes de retour |
| `internal/facade/format.go` | tableau aligné, `--json` |

**Modifiés**

| Fichier | Changement |
|---|---|
| `internal/pm/pm.go` | `Item` gagne `PM string` |
| `internal/managers/managers.go` | accès aux tables, vocabulaire réuni |
| `internal/complete/complete.go` | vocabulaire de la façade, fusion multi-catalogues |
| `main.go` | aiguillage mots réservés / verbes, drapeaux |
| `shell/jigger.plugin.zsh` · `jigger.psm1` | alias `jg`, reconnaissance de la commande |
| `README.md` | section façade |

---

# Partie A — le moteur

## Tâche 1 : vérifier la table de correspondance et capturer les jeux d'essai

La spec §2 le dit : la table est écrite de mémoire. Rien ne doit être codé avant qu'elle
soit vérifiée. Cette tâche produit aussi les fixtures dont la tâche 9 aura besoin.

**Fichiers :**
- Modifier : `docs/specs/2026-08-15-facade-multi-gestionnaires-design.md` (section « Table de correspondance »)
- Créer : `internal/brew/testdata/outdated.json`, `internal/brew/testdata/list-versions.txt`
- Créer : `internal/scoop/testdata/status.txt`
- Créer : `internal/winget/testdata/source-list-fr.txt`

**Interfaces :**
- Consomme : rien
- Produit : une table §2 exacte, et des fixtures réelles pour la tâche 9

- [ ] **Étape 1 : vérifier chaque verbe sur macOS**

Pour chacun, confirmer que la sous-commande et les options existent :

```sh
brew help | head -40
brew outdated --help | head -20
brew pin --help ; brew list --help | head
```

Noter tout écart avec la table §2 de la spec.

- [ ] **Étape 2 : capturer les jeux d'essai brew**

```sh
brew outdated --json=v2 > internal/brew/testdata/outdated.json
brew list --versions   > internal/brew/testdata/list-versions.txt
```

Si `brew outdated` ne renvoie rien, forcer un cas non vide en désinstallant puis
réinstallant une version ancienne, ou fabriquer le JSON à la main **en respectant
exactement la forme réelle** (clés `formulae` / `casks`, `installed_versions` en tableau,
`current_version` en chaîne).

- [ ] **Étape 3 : vérifier chaque verbe sous Windows**

Les trois points les plus incertains de la table :

```powershell
winget pin --help          # « pin add » / « pin remove » existent-ils ?
winget source --help       # « source list » / « source add » / « source remove »
scoop help                 # « checkup », « hold », « unhold », « cleanup »
scoop update --help        # « scoop update * » met-il bien tout à jour ?
```

- [ ] **Étape 4 : capturer les jeux d'essai Windows**

```powershell
scoop status                 > internal/scoop/testdata/status.txt
winget source list           > internal/winget/testdata/source-list-fr.txt
```

Les jeux `list-fr.txt`, `search-fr.txt` et `upgrade-fr.txt` existent déjà dans
`internal/winget/testdata/` — les réutiliser, ne pas les écraser.

- [ ] **Étape 5 : corriger la table dans la spec**

Reporter chaque écart constaté dans les deux tableaux de la section « Table de
correspondance ». Si un verbe se révèle inexistant chez un gestionnaire, le retirer de sa
colonne et mettre `—` : le modèle de capacités s'en accommode par construction.

- [ ] **Étape 6 : commit**

```bash
git add docs/specs internal/brew/testdata internal/scoop/testdata internal/winget/testdata
git commit -m "Table de correspondance vérifiée contre les CLI réelles, avec jeux d'essai"
```

---

## Tâche 2 : les types du contrat d'exécution

**Fichiers :**
- Créer : `internal/pm/verbs.go`, `internal/pm/package.go`, `internal/pm/verbs_test.go`

**Interfaces :**
- Consomme : rien
- Produit : `pm.Verb`, `pm.Pool` (`PoolAucun`/`PoolCatalogue`/`PoolInstalles`),
  `pm.Parser`, `pm.Package`, `pm.Binding`, `pm.Bindings`,
  `Binding.Argv(args []string) [][]string`, `Binding.Valid() error`,
  `Binding.NomNatif() string`

**Note sur `Argv`.** Elle rend **plusieurs** lignes d'arguments, pas une : winget
n'installe qu'un paquet à la fois (`install --id X --exact`), là où brew en prend
plusieurs. Deux marqueurs encodent la différence :

- `{args}` — tous les arguments, dans une seule invocation
- `{arg}` — un seul argument ; le moteur invoque une fois par argument

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/pm/verbs_test.go` :

```go
package pm

import "testing"

func TestArgvDeveloppeTousLesArguments(t *testing.T) {
	b := Binding{Native: []string{"install", "{args}"}}
	got := b.Argv([]string{"fd", "git"})

	if len(got) != 1 {
		t.Fatalf("{args} doit produire une seule invocation, obtenu %v", got)
	}
	want := []string{"install", "fd", "git"}
	if len(got[0]) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got[0], want)
	}
	for i := range want {
		if got[0][i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got[0], want)
		}
	}
}

func TestArgvUnAppelParArgument(t *testing.T) {
	b := Binding{Native: []string{"install", "--id", "{arg}", "--exact"}}
	got := b.Argv([]string{"Git.Git", "7zip.7zip"})

	if len(got) != 2 {
		t.Fatalf("{arg} doit produire une invocation par argument, obtenu %v", got)
	}
	if got[0][2] != "Git.Git" || got[1][2] != "7zip.7zip" {
		t.Fatalf("argv = %v", got)
	}
	if got[0][3] != "--exact" {
		t.Fatalf("le suffixe du gabarit est perdu : %v", got[0])
	}
}

func TestArgvSansMarqueur(t *testing.T) {
	b := Binding{Native: []string{"outdated", "--json=v2"}}
	got := b.Argv(nil)

	if len(got) != 1 || len(got[0]) != 2 || got[0][0] != "outdated" {
		t.Fatalf("argv = %v, attendu une invocation [outdated --json=v2]", got)
	}
}

func TestArgvPrefereBuild(t *testing.T) {
	b := Binding{Build: func(args []string) []string {
		return append([]string{"untap"}, args...)
	}}
	got := b.Argv([]string{"extras"})

	if len(got) != 1 || got[0][0] != "untap" || got[0][1] != "extras" {
		t.Fatalf("argv = %v, attendu [[untap extras]]", got)
	}
}

func TestValidRefuseDeuxFaconsDAgir(t *testing.T) {
	cas := []struct {
		nom string
		b   Binding
		ok  bool
	}{
		{"native seul", Binding{Native: []string{"install"}}, true},
		{"build seul", Binding{Build: func([]string) []string { return nil }}, true},
		{"direct seul", Binding{Direct: func([]string) ([]Package, error) { return nil, nil }}, true},
		{"aucun", Binding{}, false},
		{"native + build", Binding{
			Native: []string{"install"},
			Build:  func([]string) []string { return nil },
		}, false},
		{"direct + parse", Binding{
			Direct: func([]string) ([]Package, error) { return nil, nil },
			Parse:  func([]byte) ([]Package, error) { return nil, nil },
		}, false},
	}
	for _, c := range cas {
		err := c.b.Valid()
		if (err == nil) != c.ok {
			t.Errorf("%s : Valid() = %v, attendu ok=%v", c.nom, err, c.ok)
		}
	}
}

func TestNomNatif(t *testing.T) {
	if got := (Binding{Native: []string{"checkup"}}).NomNatif(); got != "checkup" {
		t.Errorf("NomNatif = %q, attendu « checkup »", got)
	}
	if got := (Binding{Build: func([]string) []string { return nil }}).NomNatif(); got != "" {
		t.Errorf("NomNatif d'une liaison calculée = %q, attendu vide", got)
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/pm/ -run 'Argv|Valid|NomNatif' -v`
Attendu : ÉCHEC à la compilation — `undefined: Binding`

- [ ] **Étape 3 : écrire `internal/pm/package.go`**

```go
package pm

// Package est une ligne de sortie normalisée : ce que jigger sait dire d'un paquet, quel
// que soit le gestionnaire qui l'a produit. Un seul type sert les quatre verbes
// normalisés — list, outdated, search et source.
type Package struct {
	Name      string // identifiant natif : « fd », « Git.Git »
	Version   string // version installée ; vide si non installé
	Available string // version disponible ; vide si à jour ou inconnue
	Kind      string // badge Badge* — le popup l'affiche déjà
	Source    string // provenance fine : « main », « extras », « homebrew/core »
	PM        string // « brew », « winget », « scoop »
}
```

- [ ] **Étape 4 : écrire `internal/pm/verbs.go`**

```go
// Ce fichier porte le second contrat de pm : savoir **agir**. Le premier — pm.go — ne
// sait que répondre à des questions, et n'est pas modifié (cf. ADR-0002).
//
// Un gestionnaire déclare une table `verbe → liaison`. Les capacités en découlent sans
// drapeau à tenir : un verbe absent de la table est un verbe que ce gestionnaire ne sait
// pas rendre.
package pm

import (
	"errors"
	"strings"
)

// Verb est un membre de phrase du vocabulaire de jigger : « install », « source add ».
// La clé porte la phrase entière, ce qui garde les gabarits d'argv triviaux — sans quoi
// `brew tap` / `untap`, deux mots sans rapport, exigeraient aussitôt Build.
type Verb string

// Pool dit où chercher les candidats d'un verbe. Il reprend la notion de
// Manager.InstalledOnly en l'élargissant au cas « ce verbe ne prend pas de paquet ».
// Manager.InstalledOnly reste en place pour la complétion native : les deux coexistent.
type Pool int

const (
	PoolAucun     Pool = iota // le verbe ne prend pas de nom de paquet
	PoolCatalogue             // tous les paquets connus
	PoolInstalles             // les seuls paquets installés
)

// Parser refond la sortie d'un gestionnaire en lignes normalisées. Fonction pure : elle
// ne lance rien, ce qui la rend testable sur fichier.
type Parser func(out []byte) ([]Package, error)

// Marqueurs de gabarit. La distinction n'est pas cosmétique : winget n'installe qu'un
// paquet par appel, brew en prend plusieurs.
const (
	MarqueurTous = "{args}" // tous les arguments, une seule invocation
	MarqueurUn   = "{arg}"  // un argument par invocation
)

// Binding lie un verbe de jigger à ce qu'il faut faire chez un gestionnaire. Une liaison
// agit d'**une seule** des trois façons ci-dessous.
type Binding struct {
	Native []string                          // gabarit d'argv — le cas ordinaire
	Build  func(args []string) []string      // argv calculé — le cas rétif
	Direct func(args []string) ([]Package, error) // sans sous-processus du tout

	Pool  Pool   // où chercher les candidats
	Parse Parser // nil → la sortie est relayée telle quelle
}

// Bindings est le contrat d'exécution. Un gestionnaire peut implémenter Manager sans
// l'implémenter : on saura le compléter sans savoir le piloter.
type Bindings interface {
	Verbs() map[Verb]Binding
}

// Argv construit les lignes d'arguments passées au gestionnaire. Elle en rend
// **plusieurs** quand le gabarit porte MarqueurUn : un appel par argument.
func (b Binding) Argv(args []string) [][]string {
	if b.Build != nil {
		return [][]string{b.Build(args)}
	}
	if !contient(b.Native, MarqueurUn) {
		return [][]string{developper(b.Native, args)}
	}
	if len(args) == 0 {
		return [][]string{retirer(b.Native, MarqueurUn)}
	}
	lignes := make([][]string, 0, len(args))
	for _, a := range args {
		lignes = append(lignes, developper(b.Native, []string{a}))
	}
	return lignes
}

// developper remplace les marqueurs par les arguments. MarqueurTous s'étale en autant
// d'éléments qu'il y a d'arguments ; MarqueurUn prend le premier.
func developper(gabarit, args []string) []string {
	out := make([]string, 0, len(gabarit)+len(args))
	for _, mot := range gabarit {
		switch mot {
		case MarqueurTous:
			out = append(out, args...)
		case MarqueurUn:
			if len(args) > 0 {
				out = append(out, args[0])
			}
		default:
			out = append(out, mot)
		}
	}
	return out
}

func retirer(gabarit []string, marqueur string) []string {
	out := make([]string, 0, len(gabarit))
	for _, mot := range gabarit {
		if mot != marqueur {
			out = append(out, mot)
		}
	}
	return out
}

func contient(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Valid vérifie qu'une liaison est bien formée. Elle est appelée par les tests de table
// de chaque gestionnaire : une table mal formée doit se voir à la compilation des tests,
// pas au premier usage.
func (b Binding) Valid() error {
	n := 0
	for _, pose := range []bool{b.Native != nil, b.Build != nil, b.Direct != nil} {
		if pose {
			n++
		}
	}
	switch {
	case n == 0:
		return errors.New("liaison sans Native, Build ni Direct")
	case n > 1:
		return errors.New("liaison avec plusieurs façons d'agir")
	case b.Direct != nil && b.Parse != nil:
		return errors.New("Direct rend déjà des Package : Parse est de trop")
	}
	return nil
}

// NomNatif rend le nom que le gestionnaire donne au verbe — « checkup » là où jigger dit
// « doctor ». Vide pour une liaison calculée, dont le nom dépend des arguments. Sert aux
// messages de capacité : « scoop le sait (checkup), mais n'est pas installé ».
func (b Binding) NomNatif() string {
	if len(b.Native) == 0 {
		return ""
	}
	if strings.HasPrefix(b.Native[0], "{") {
		return ""
	}
	return b.Native[0]
}
```

- [ ] **Étape 5 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/pm/ -v`
Attendu : SUCCÈS sur tous les tests, anciens compris

- [ ] **Étape 6 : commit**

```bash
git add internal/pm/verbs.go internal/pm/package.go internal/pm/verbs_test.go
git commit -m "Contrat d'exécution : Verb, Pool, Binding et la construction d'argv"
```

---

## Tâche 3 : la table brew

**Fichiers :**
- Créer : `internal/brew/verbs.go`, `internal/brew/verbs_test.go`

**Interfaces :**
- Consomme : `pm.Verb`, `pm.Binding`, `pm.Pool*`, `pm.Bindings`, `Binding.Argv`, `Binding.Valid`
- Produit : `brew.Manager` implémente `pm.Bindings`

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/brew/verbs_test.go` :

```go
package brew

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// argv1 rend l'unique ligne d'arguments d'un verbe, et échoue s'il y en a plusieurs.
func argv1(t *testing.T, v pm.Verb, args []string) []string {
	t.Helper()
	b, ok := New().Verbs()[v]
	if !ok {
		t.Fatalf("verbe %q absent de la table brew", v)
	}
	lignes := b.Argv(args)
	if len(lignes) != 1 {
		t.Fatalf("verbe %q : %d invocations, attendu 1", v, len(lignes))
	}
	return lignes[0]
}

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got, want)
		}
	}
}

func TestTableBrewEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

func TestArgvBrew(t *testing.T) {
	egal(t, argv1(t, "install", []string{"fd", "git"}), []string{"install", "fd", "git"})
	egal(t, argv1(t, "uninstall", []string{"fd"}), []string{"uninstall", "fd"})
	egal(t, argv1(t, "outdated", nil), []string{"outdated", "--json=v2"})
	egal(t, argv1(t, "list", nil), []string{"list", "--versions"})
	egal(t, argv1(t, "info", []string{"fd"}), []string{"info", "fd"})
	egal(t, argv1(t, "pin", []string{"fd"}), []string{"pin", "fd"})
	egal(t, argv1(t, "doctor", nil), []string{"doctor"})
}

// tap et untap sont deux mots sans rapport : c'est le cas qui justifie Build.
func TestArgvBrewSource(t *testing.T) {
	egal(t, argv1(t, "source", nil), []string{"tap"})
	egal(t, argv1(t, "source add", []string{"homebrew/cask-fonts"}),
		[]string{"tap", "homebrew/cask-fonts"})
	egal(t, argv1(t, "source rm", []string{"homebrew/cask-fonts"}),
		[]string{"untap", "homebrew/cask-fonts"})
}

func TestPoolBrew(t *testing.T) {
	cas := map[pm.Verb]pm.Pool{
		"install":   pm.PoolCatalogue,
		"search":    pm.PoolCatalogue,
		"uninstall": pm.PoolInstalles,
		"upgrade":   pm.PoolInstalles,
		"pin":       pm.PoolInstalles,
		"outdated":  pm.PoolAucun,
		"doctor":    pm.PoolAucun,
	}
	table := New().Verbs()
	for v, want := range cas {
		if got := table[v].Pool; got != want {
			t.Errorf("verbe %q : Pool = %v, attendu %v", v, got, want)
		}
	}
}

// winget n'a ni cleanup ni doctor : la capacité se lit dans l'absence de clé. brew, lui,
// doit les avoir.
func TestBrewSaitFaireCleanupEtDoctor(t *testing.T) {
	table := New().Verbs()
	for _, v := range []pm.Verb{"cleanup", "doctor"} {
		if _, ok := table[v]; !ok {
			t.Errorf("verbe %q absent de la table brew", v)
		}
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/brew/ -run Brew -v`
Attendu : ÉCHEC à la compilation — `New().Verbs undefined`

- [ ] **Étape 3 : écrire `internal/brew/verbs.go`**

Reprendre les valeurs exactes de la table §2 **telle que corrigée par la tâche 1**.

```go
package brew

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que brew sait faire. La table est le modèle de capacités : un verbe
// absent est un verbe que brew ne sait pas rendre.
//
// Les parsers (Parse) sont branchés en tâche 9 ; ils restent nil ici, ce qui signifie
// « sortie relayée telle quelle ».
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		// Universels
		"install":   {Native: []string{"install", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"upgrade":   {Native: []string{"upgrade", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"list":      {Native: []string{"list", "--versions"}, Pool: pm.PoolAucun},
		"outdated":  {Native: []string{"outdated", "--json=v2"}, Pool: pm.PoolAucun},
		"search":    {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"info":      {Native: []string{"info", pm.MarqueurTous}, Pool: pm.PoolCatalogue},

		// Convergents
		"source":     {Native: []string{"tap"}, Pool: pm.PoolAucun},
		"source add": {Native: []string{"tap", pm.MarqueurTous}, Pool: pm.PoolAucun},
		// untap n'est pas « tap » avec une option : d'où Build plutôt qu'un gabarit.
		"source rm": {
			Build: func(args []string) []string {
				return append([]string{"untap"}, args...)
			},
			Pool: pm.PoolAucun,
		},
		"pin":     {Native: []string{"pin", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"unpin":   {Native: []string{"unpin", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"cleanup": {Native: []string{"cleanup"}, Pool: pm.PoolAucun},
		"doctor":  {Native: []string{"doctor"}, Pool: pm.PoolAucun},
	}
}
```

- [ ] **Étape 4 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/brew/ -v`
Attendu : SUCCÈS

- [ ] **Étape 5 : commit**

```bash
git add internal/brew/verbs.go internal/brew/verbs_test.go
git commit -m "Table des verbes de brew"
```

---

## Tâche 4 : la table winget

**Fichiers :**
- Créer : `internal/winget/verbs.go`, `internal/winget/verbs_test.go`

**Interfaces :**
- Consomme : mêmes types que la tâche 3
- Produit : `winget.Manager` implémente `pm.Bindings`

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/winget/verbs_test.go` :

```go
package winget

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got, want)
		}
	}
}

func TestTableWingetEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

// winget n'installe qu'un paquet par appel : deux noms font deux invocations.
func TestArgvWingetUnAppelParPaquet(t *testing.T) {
	b := New().Verbs()["install"]
	lignes := b.Argv([]string{"Git.Git", "7zip.7zip"})

	if len(lignes) != 2 {
		t.Fatalf("%d invocations, attendu 2 — winget ne prend qu'un id", len(lignes))
	}
	egal(t, lignes[0], []string{"install", "--id", "Git.Git", "--exact"})
	egal(t, lignes[1], []string{"install", "--id", "7zip.7zip", "--exact"})
}

func TestArgvWinget(t *testing.T) {
	table := New().Verbs()
	egal(t, table["list"].Argv(nil)[0], []string{"list"})
	egal(t, table["outdated"].Argv(nil)[0], []string{"list", "--upgrade-available"})
	egal(t, table["search"].Argv([]string{"git"})[0], []string{"search", "git"})
	egal(t, table["source"].Argv(nil)[0], []string{"source", "list"})
	egal(t, table["source add"].Argv([]string{"x"})[0], []string{"source", "add", "x"})
	egal(t, table["pin"].Argv([]string{"Git.Git"})[0],
		[]string{"pin", "add", "--id", "Git.Git"})
}

// winget n'a ni cleanup ni doctor. Leur absence EST la capacité déclarée — c'est ce qui
// fait dire « aucun gestionnaire disponible ne sait faire ça » en tâche 6.
func TestWingetNeSaitNiCleanupNiDoctor(t *testing.T) {
	table := New().Verbs()
	for _, v := range []pm.Verb{"cleanup", "doctor"} {
		if _, ok := table[v]; ok {
			t.Errorf("verbe %q présent dans la table winget, attendu absent", v)
		}
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/winget/ -run Winget -v`
Attendu : ÉCHEC à la compilation — `New().Verbs undefined`

- [ ] **Étape 3 : écrire `internal/winget/verbs.go`**

Reprendre les valeurs exactes de la table §2 **telle que corrigée par la tâche 1** — en
particulier la forme de `winget pin`, qui est le point le plus incertain.

```go
package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que winget sait faire. Ni cleanup ni doctor n'y figurent : winget n'a
// pas ces concepts, et leur absence est précisément ce que le modèle de capacités lit.
//
// MarqueurUn partout où un identifiant est attendu : winget ne prend qu'un `--id` par
// appel, là où brew et scoop acceptent une liste.
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"install":   {Native: []string{"install", "--id", pm.MarqueurUn, "--exact"}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", "--id", pm.MarqueurUn, "--exact"}, Pool: pm.PoolInstalles},
		"upgrade":   {Native: []string{"upgrade", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
		"list":      {Native: []string{"list"}, Pool: pm.PoolAucun},
		"outdated":  {Native: []string{"list", "--upgrade-available"}, Pool: pm.PoolAucun},
		"search":    {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"info":      {Native: []string{"show", "--id", pm.MarqueurUn}, Pool: pm.PoolCatalogue},

		"source":     {Native: []string{"source", "list"}, Pool: pm.PoolAucun},
		"source add": {Native: []string{"source", "add", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"source rm":  {Native: []string{"source", "remove", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"pin":        {Native: []string{"pin", "add", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
		"unpin":      {Native: []string{"pin", "remove", "--id", pm.MarqueurUn}, Pool: pm.PoolInstalles},
	}
}
```

- [ ] **Étape 4 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/winget/ -v`
Attendu : SUCCÈS

- [ ] **Étape 5 : commit**

```bash
git add internal/winget/verbs.go internal/winget/verbs_test.go
git commit -m "Table des verbes de winget"
```

---

## Tâche 5 : la table scoop, avec `Direct`

`outdated` chez scoop ne lance rien : `internal/scoop/outdated.go` compare déjà les
manifestes sur le disque. C'est le cas qui justifie `Direct`.

**Fichiers :**
- Créer : `internal/scoop/verbs.go`, `internal/scoop/verbs_test.go`
- Lire (sans modifier pour l'instant) : `internal/scoop/outdated.go`

**Interfaces :**
- Consomme : mêmes types que la tâche 3
- Produit : `scoop.Manager` implémente `pm.Bindings` ; `outdated` passe par `Direct`

- [ ] **Étape 1 : lire l'existant**

Lire `internal/scoop/outdated.go` en entier et relever la signature de la fonction qui
rend les applications à mettre à jour. Le nom exact conditionne l'adaptateur de l'étape 3 :
`Direct` doit l'envelopper, **pas** la réécrire.

- [ ] **Étape 2 : écrire le test qui échoue**

`internal/scoop/verbs_test.go` :

```go
package scoop

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv = %v, attendu %v", got, want)
		}
	}
}

func TestTableScoopEstBienFormee(t *testing.T) {
	for v, b := range New().Verbs() {
		if err := b.Valid(); err != nil {
			t.Errorf("verbe %q : %v", v, err)
		}
	}
}

func TestArgvScoop(t *testing.T) {
	table := New().Verbs()
	egal(t, table["install"].Argv([]string{"fd", "7zip"})[0],
		[]string{"install", "fd", "7zip"})
	egal(t, table["list"].Argv(nil)[0], []string{"list"})
	egal(t, table["source"].Argv(nil)[0], []string{"bucket", "list"})
	egal(t, table["source add"].Argv([]string{"extras"})[0],
		[]string{"bucket", "add", "extras"})
	egal(t, table["source rm"].Argv([]string{"extras"})[0],
		[]string{"bucket", "rm", "extras"})
	egal(t, table["pin"].Argv([]string{"fd"})[0], []string{"hold", "fd"})
	egal(t, table["unpin"].Argv([]string{"fd"})[0], []string{"unhold", "fd"})
	egal(t, table["doctor"].Argv(nil)[0], []string{"checkup"})
}

// `scoop update` sans argument met à jour scoop lui-même et les buckets, pas les
// applications : le verbe upgrade sans nom doit donc viser « * ».
func TestUpgradeScoopSansNomViseTout(t *testing.T) {
	got := New().Verbs()["upgrade"].Argv(nil)
	if len(got) != 1 {
		t.Fatalf("%d invocations, attendu 1", len(got))
	}
	egal(t, got[0], []string{"update", "*"})
}

// outdated ne lance pas scoop : la réponse se lit sur le disque. C'est ce que Direct
// exprime, et c'est ce qui rend `jg outdated` instantané côté scoop.
func TestOutdatedScoopEstDirect(t *testing.T) {
	b := New().Verbs()["outdated"]
	if b.Direct == nil {
		t.Fatal("outdated doit passer par Direct, pas par un sous-processus")
	}
	if b.Native != nil || b.Parse != nil {
		t.Errorf("outdated : Direct exclut Native et Parse (%v / %v)", b.Native, b.Parse)
	}
}
```

- [ ] **Étape 3 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/scoop/ -run Scoop -v`
Attendu : ÉCHEC à la compilation — `New().Verbs undefined`

- [ ] **Étape 4 : écrire `internal/scoop/verbs.go`**

Remplacer `applicationsAMettreAJour` par le nom réel relevé à l'étape 1.

```go
package scoop

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Verbs déclare ce que scoop sait faire.
//
// outdated est le seul verbe en Direct de tout jigger : scoop range ses applications dans
// une arborescence qui ressemble au Cellar de Homebrew, et la comparaison des manifestes
// se fait sur le disque (cf. outdated.go). Passer par un sous-processus pour redemander ce
// que jigger sait déjà — en démarrant PowerShell, qui plus est — serait absurde.
func (Manager) Verbs() map[pm.Verb]pm.Binding {
	return map[pm.Verb]pm.Binding{
		"install":   {Native: []string{"install", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"uninstall": {Native: []string{"uninstall", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		// `scoop update` seul met à jour scoop et les buckets : viser « * » pour les
		// applications quand aucun nom n'est donné.
		"upgrade": {
			Build: func(args []string) []string {
				if len(args) == 0 {
					return []string{"update", "*"}
				}
				return append([]string{"update"}, args...)
			},
			Pool: pm.PoolInstalles,
		},
		"list":     {Native: []string{"list"}, Pool: pm.PoolAucun},
		"outdated": {Direct: outdatedDirect, Pool: pm.PoolAucun},
		"search":   {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue},
		"info":     {Native: []string{"info", pm.MarqueurTous}, Pool: pm.PoolCatalogue},

		"source":     {Native: []string{"bucket", "list"}, Pool: pm.PoolAucun},
		"source add": {Native: []string{"bucket", "add", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"source rm":  {Native: []string{"bucket", "rm", pm.MarqueurTous}, Pool: pm.PoolAucun},
		"pin":        {Native: []string{"hold", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"unpin":      {Native: []string{"unhold", pm.MarqueurTous}, Pool: pm.PoolInstalles},
		"cleanup":    {Native: []string{"cleanup", "*"}, Pool: pm.PoolAucun},
		"doctor":     {Native: []string{"checkup"}, Pool: pm.PoolAucun},
	}
}

// outdatedDirect enveloppe la comparaison de manifestes déjà écrite dans outdated.go.
// Il ne la réécrit pas : il la traduit en pm.Package.
func outdatedDirect([]string) ([]pm.Package, error) {
	apps, err := applicationsAMettreAJour() // ← nom réel relevé à l'étape 1
	if err != nil {
		return nil, err
	}
	out := make([]pm.Package, 0, len(apps))
	for _, a := range apps {
		out = append(out, pm.Package{
			Name:      a.Nom,
			Version:   a.Installee,
			Available: a.Disponible,
			Kind:      pm.BadgeScoop,
			Source:    a.Bucket,
			PM:        "scoop",
		})
	}
	return out, nil
}
```

- [ ] **Étape 5 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/scoop/ -v`
Attendu : SUCCÈS

- [ ] **Étape 6 : commit**

```bash
git add internal/scoop/verbs.go internal/scoop/verbs_test.go
git commit -m "Table des verbes de scoop, outdated branché en Direct"
```

---

## Tâche 6 : résolution du verbe et messages de capacité

**Fichiers :**
- Créer : `internal/facade/verbe.go`, `internal/facade/verbe_test.go`
- Modifier : `internal/managers/managers.go`

**Interfaces :**
- Consomme : `pm.Bindings`, `managers.All()`, `managers.Available()`
- Produit :
  - `managers.Tables(mgrs []pm.Manager) map[pm.Verb][]pm.Manager`
  - `managers.Vocabulaire() []string` — les verbes de premier niveau, triés
  - `facade.ResoudreVerbe(ligne []string) (pm.Verb, []string, []pm.Manager, error)`
  - `facade.ErrVerbeInconnu` — erreur dont le message porte les capacités

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/facade/verbe_test.go` :

```go
package facade

import (
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/brew"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/scoop"
	"gitlab.yg-devworks.com/yves/jigger/internal/winget"
)

// Le verbe composé se reconnaît avant le verbe simple : « source add extras » est
// « source add », pas « source » avec un argument.
func TestResoudreVerbeComposeDAbord(t *testing.T) {
	dispo := []pm.Manager{brew.New()}

	v, args, _, err := resoudreVerbe([]string{"source", "add", "extras"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if v != "source add" {
		t.Fatalf("verbe = %q, attendu « source add »", v)
	}
	if len(args) != 1 || args[0] != "extras" {
		t.Fatalf("args = %v, attendu [extras]", args)
	}

	v, args, _, err = resoudreVerbe([]string{"source"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if v != "source" || len(args) != 0 {
		t.Fatalf("verbe = %q, args = %v, attendu « source » sans argument", v, args)
	}
}

func TestResoudreVerbeRendLesGestionnairesCapables(t *testing.T) {
	dispo := []pm.Manager{winget.New(), scoop.New()}

	_, _, capables, err := resoudreVerbe([]string{"install", "fd"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if len(capables) != 2 {
		t.Fatalf("%d gestionnaires capables, attendu 2", len(capables))
	}

	// doctor : scoop sait (checkup), winget non.
	_, _, capables, err = resoudreVerbe([]string{"doctor"}, dispo, dispo)
	if err != nil {
		t.Fatal(err)
	}
	if len(capables) != 1 || capables[0].Cmd() != "scoop" {
		t.Fatalf("capables = %v, attendu [scoop]", capables)
	}
}

// Le message doit nommer qui saurait faire, et sous quel nom : c'est tout l'intérêt du
// modèle de capacités.
func TestVerbeConnuAilleursMaisIndisponible(t *testing.T) {
	dispo := []pm.Manager{winget.New()}
	tous := []pm.Manager{brew.New(), winget.New(), scoop.New()}

	_, _, _, err := resoudreVerbe([]string{"doctor"}, dispo, tous)
	if err == nil {
		t.Fatal("attendu une erreur : winget seul ne sait pas doctor")
	}
	msg := err.Error()
	for _, attendu := range []string{"doctor", "scoop", "checkup"} {
		if !strings.Contains(msg, attendu) {
			t.Errorf("le message ne contient pas %q : %s", attendu, msg)
		}
	}
}

func TestVerbeInconnuDeTous(t *testing.T) {
	tous := []pm.Manager{brew.New(), winget.New(), scoop.New()}

	_, _, _, err := resoudreVerbe([]string{"teleporter"}, tous, tous)
	if err == nil {
		t.Fatal("attendu une erreur pour un verbe qui n'existe nulle part")
	}
	if !strings.Contains(err.Error(), "teleporter") {
		t.Errorf("le message doit nommer le verbe : %s", err.Error())
	}
}

func TestLigneVide(t *testing.T) {
	tous := []pm.Manager{brew.New()}
	if _, _, _, err := resoudreVerbe(nil, tous, tous); err == nil {
		t.Fatal("attendu une erreur sur une ligne vide")
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/facade/ -v`
Attendu : ÉCHEC — le paquet n'existe pas

- [ ] **Étape 3 : ajouter les accès dans `internal/managers/managers.go`**

```go
// Tables rend, pour chaque verbe, les gestionnaires qui savent le rendre. C'est le modèle
// de capacités vu depuis le verbe : la clé existe si au moins un gestionnaire la déclare.
func Tables(mgrs []pm.Manager) map[pm.Verb][]pm.Manager {
	out := map[pm.Verb][]pm.Manager{}
	for _, m := range mgrs {
		b, ok := m.(pm.Bindings)
		if !ok {
			continue // on sait le compléter sans savoir le piloter
		}
		for v := range b.Verbs() {
			out[v] = append(out[v], m)
		}
	}
	return out
}

// Vocabulaire rend les verbes de premier niveau proposés par les gestionnaires donnés,
// triés et dédupliqués. « source add » y figure comme « source » : le popup complète le
// premier mot, puis le second.
func Vocabulaire(mgrs []pm.Manager) []string {
	vu := map[string]bool{}
	for v := range Tables(mgrs) {
		premier, _, _ := strings.Cut(string(v), " ")
		vu[premier] = true
	}
	out := make([]string, 0, len(vu))
	for v := range vu {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
```

Ajouter `"sort"` aux imports.

- [ ] **Étape 4 : écrire `internal/facade/verbe.go`**

```go
// Package facade est le moteur de la syntaxe unique de jigger : `jg install fd` plutôt
// que `brew install fd` ou `scoop install fd`.
//
// Il ne connaît aucun gestionnaire en particulier. Tout ce qu'il sait, il le lit dans les
// tables que ceux-ci déclarent (cf. pm.Bindings) : quels verbes existent, comment ils se
// traduisent, où chercher leurs candidats.
package facade

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// ResoudreVerbe reconnaît le verbe en tête de ligne et rend les gestionnaires installés
// qui savent le rendre.
func ResoudreVerbe(ligne []string) (pm.Verb, []string, []pm.Manager, error) {
	return resoudreVerbe(ligne, managers.Available(), managers.All())
}

// resoudreVerbe est ResoudreVerbe sur des listes données — c'est la forme testable, qui
// permet de simuler une machine où tel gestionnaire manque.
func resoudreVerbe(ligne []string, dispo, tous []pm.Manager) (pm.Verb, []string, []pm.Manager, error) {
	if len(ligne) == 0 {
		return "", nil, nil, fmt.Errorf("jigger : aucun verbe. Essaie « jg install <paquet> » ou « jg outdated »")
	}

	tablesDispo := managers.Tables(dispo)

	// Le verbe composé d'abord : « source add extras » est « source add », pas « source »
	// avec un argument.
	if len(ligne) >= 2 {
		compose := pm.Verb(ligne[0] + " " + ligne[1])
		if capables, ok := tablesDispo[compose]; ok {
			return compose, ligne[2:], trier(capables), nil
		}
	}

	simple := pm.Verb(ligne[0])
	if capables, ok := tablesDispo[simple]; ok {
		return simple, ligne[1:], trier(capables), nil
	}

	return "", nil, nil, verbeIndisponible(ligne, dispo, tous)
}

// verbeIndisponible construit le message qui distingue « personne ne sait faire ça » de
// « quelqu'un saurait, mais il n'est pas installé ». C'est le modèle de capacités qui
// parle : sans cette distinction, l'utilisateur ne sait pas s'il s'est trompé de mot ou
// s'il lui manque un outil.
func verbeIndisponible(ligne []string, dispo, tous []pm.Manager) error {
	mot := ligne[0]
	if len(ligne) >= 2 {
		if _, ok := managers.Tables(tous)[pm.Verb(mot+" "+ligne[1])]; ok {
			mot = mot + " " + ligne[1]
		}
	}

	var ailleurs []string
	for _, m := range tous {
		if estDispo(m, dispo) {
			continue
		}
		b, ok := m.(pm.Bindings)
		if !ok {
			continue
		}
		liaison, ok := b.Verbs()[pm.Verb(mot)]
		if !ok {
			continue
		}
		if natif := liaison.NomNatif(); natif != "" && natif != mot {
			ailleurs = append(ailleurs, fmt.Sprintf("%s le sait (%s)", m.Cmd(), natif))
		} else {
			ailleurs = append(ailleurs, fmt.Sprintf("%s le sait", m.Cmd()))
		}
	}

	if len(ailleurs) == 0 {
		return fmt.Errorf("jigger : « %s » — verbe inconnu. « jg ⇥ » liste ce que jigger sait faire", mot)
	}
	sort.Strings(ailleurs)
	return fmt.Errorf("jigger : « %s » — aucun gestionnaire disponible ne sait faire ça.\n        %s, mais n'est pas installé",
		mot, strings.Join(ailleurs, " ; "))
}

func estDispo(m pm.Manager, dispo []pm.Manager) bool {
	for _, d := range dispo {
		if d.Cmd() == m.Cmd() {
			return true
		}
	}
	return false
}

// trier range les gestionnaires dans l'ordre de managers.All(), pour que l'exécution
// séquentielle soit reproductible d'un appel à l'autre.
func trier(mgrs []pm.Manager) []pm.Manager {
	rang := map[string]int{}
	for i, m := range managers.All() {
		rang[m.Cmd()] = i
	}
	out := append([]pm.Manager(nil), mgrs...)
	sort.Slice(out, func(i, j int) bool { return rang[out[i].Cmd()] < rang[out[j].Cmd()] })
	return out
}
```

- [ ] **Étape 5 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/facade/ ./internal/managers/ -v`
Attendu : SUCCÈS

- [ ] **Étape 6 : commit**

```bash
git add internal/facade/verbe.go internal/facade/verbe_test.go internal/managers/managers.go
git commit -m "Résolution du verbe et messages de capacité"
```

---

## Tâche 7 : routage par résolution du nom

**Fichiers :**
- Créer : `internal/facade/routage.go`, `internal/facade/routage_test.go`

**Interfaces :**
- Consomme : `pm.Pool*`, `pm.Catalog`, `Manager.Load()`, `facade.resoudreVerbe`
- Produit :
  - `facade.Cible{Mgr pm.Manager; Args []string}`
  - `facade.Ambiguite{Nom string; Candidats []Candidat}` avec `Candidat{Mgr, Badge, Qualifie}`
  - `facade.Router(v pm.Verb, args []string, forcePM string, mgrs []pm.Manager, cats map[string]*pm.Catalog) ([]Cible, *Ambiguite, error)`

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/facade/routage_test.go` :

```go
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
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/facade/ -run Routage -v`
Attendu : ÉCHEC à la compilation — `undefined: Router`

- [ ] **Étape 3 : écrire `internal/facade/routage.go`**

```go
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
// noyer le message.
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
		pool := cat.Names
		for _, candidat := range pool {
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
```

- [ ] **Étape 4 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/facade/ -v`
Attendu : SUCCÈS

- [ ] **Étape 5 : commit**

```bash
git add internal/facade/routage.go internal/facade/routage_test.go
git commit -m "Routage par résolution du nom, avec --pm et détection d'ambiguïté"
```

---

## Tâche 8 : exécution et codes de retour

**Fichiers :**
- Créer : `internal/facade/executer.go`, `internal/facade/executer_test.go`

**Interfaces :**
- Consomme : `facade.Cible`, `pm.Binding.Argv`, `pm.Bindings`
- Produit :
  - `facade.Opts{JSON bool; Yes bool}`
  - `facade.Executer(v pm.Verb, cibles []Cible, o Opts) (rows []pm.Package, code int)`
  - `facade.lancer` — point d'injection remplacé dans les tests

**Rappel de la spec §4.** Les verbes relayés héritent de `os.Std*` : les invites de winget,
les barres de progression et l'élévation UAC fonctionnent comme si l'utilisateur avait
tapé la commande. Il n'y a donc **aucun code de TTY à écrire**.

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/facade/executer_test.go` :

```go
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

func TestLectureEchoueSiPersonneNeRepond(t *testing.T) {
	simuler(t, nil, map[string]int{"winget": 1, "scoop": 1})

	_, code := Executer("outdated", []Cible{
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

func TestVerbesNormalisesSontCaptures(t *testing.T) {
	if !normalise("outdated") || !normalise("list") || !normalise("search") || !normalise("source") {
		t.Error("les quatre verbes tabulaires doivent être normalisés")
	}
	for _, v := range []pm.Verb{"install", "info", "doctor", "source add"} {
		if normalise(v) {
			t.Errorf("le verbe %q doit être relayé, pas normalisé", v)
		}
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/facade/ -run 'Executer|Ecriture|Lecture|Yes|Direct|Verbes' -v`
Attendu : ÉCHEC à la compilation — `undefined: Executer`

- [ ] **Étape 3 : écrire `internal/facade/executer.go`**

```go
package facade

import (
	"fmt"
	"os"
	"os/exec"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Opts porte les drapeaux de la ligne qui ne concernent pas le routage.
type Opts struct {
	JSON bool // sortie en JSON plutôt qu'en tableau
	Yes  bool // accepter les accords de licence (winget)
}

// verbesNormalises : ceux dont la sortie est tabulaire, donc capturée et refondue. Tout
// le reste est relayé — et c'est ce qui fait que les invites, les barres de progression et
// l'élévation UAC fonctionnent sans une ligne de code de TTY.
var verbesNormalises = map[pm.Verb]bool{
	"list": true, "outdated": true, "search": true, "source": true,
}

func normalise(v pm.Verb) bool { return verbesNormalises[v] }

// lancer est le point d'injection des tests. relais dit si le processus hérite du
// terminal (verbe relayé) ou si sa sortie est capturée (verbe normalisé).
var lancer = lancerReel

func lancerReel(cmd string, args []string, relais bool) ([]byte, int, error) {
	c := exec.Command(cmd, args...)
	if relais {
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := c.Run()
		return nil, code(err), err
	}
	out, err := c.Output()
	return out, code(err), err
}

func code(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errorsAs(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

// Executer déroule les cibles en séquence. L'ordre vient de Router, qui le tient de
// managers.All() : deux exécutions successives font la même chose dans le même ordre.
//
// Lecture et écriture ne traitent pas l'échec de la même façon, et c'est délibéré : la
// lecture est au mieux, l'écriture ne devine pas.
func Executer(v pm.Verb, cibles []Cible, o Opts) ([]pm.Package, int) {
	lecture := normalise(v)
	var rows []pm.Package
	var reussites, echecs int
	dernierCode := 0

	for _, cible := range cibles {
		b, ok := cible.Mgr.(pm.Bindings)
		if !ok {
			continue
		}
		liaison, ok := b.Verbs()[v]
		if !ok {
			continue
		}

		// Direct : jigger sait déjà répondre, aucun sous-processus.
		if liaison.Direct != nil {
			out, err := liaison.Direct(cible.Args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "jigger (%s) : %v\n", cible.Mgr.Cmd(), err)
				echecs++
				dernierCode = 1
				continue
			}
			rows = append(rows, out...)
			reussites++
			continue
		}

		echoue := false
		for _, argv := range liaison.Argv(cible.Args) {
			argv = accords(cible.Mgr.Cmd(), v, argv, o)
			out, c, err := lancer(cible.Mgr.Cmd(), argv, !lecture)
			if err != nil {
				if !lecture {
					// Écriture : on n'enchaîne pas sur un gestionnaire suivant après
					// un échec.
					fmt.Fprintf(os.Stderr, "jigger (%s) : échec\n", cible.Mgr.Cmd())
					return rows, c
				}
				fmt.Fprintf(os.Stderr, "jigger (%s) : %v\n", cible.Mgr.Cmd(), err)
				echoue = true
				dernierCode = c
				break
			}
			if liaison.Parse != nil {
				parsed, perr := liaison.Parse(out)
				if perr != nil {
					fmt.Fprintf(os.Stderr, "jigger (%s) : sortie illisible — %v\n", cible.Mgr.Cmd(), perr)
					echoue = true
					dernierCode = 1
					break
				}
				for i := range parsed {
					parsed[i].PM = cible.Mgr.Cmd()
				}
				rows = append(rows, parsed...)
			}
		}
		if echoue {
			echecs++
		} else {
			reussites++
		}
	}

	if lecture {
		// Au mieux : 0 dès qu'un gestionnaire a répondu.
		if reussites > 0 {
			return rows, 0
		}
		if echecs > 0 {
			return rows, dernierCode
		}
	}
	return rows, 0
}

// accords ajoute les acceptations de licence de winget, et seulement sur --yes : jigger
// n'accepte jamais une licence à la place de l'utilisateur. Sans le drapeau, l'invite
// s'affiche — la sortie étant relayée, il peut y répondre.
func accords(cmd string, v pm.Verb, argv []string, o Opts) []string {
	if !o.Yes || cmd != "winget" {
		return argv
	}
	switch v {
	case "install", "uninstall", "upgrade":
		return append(argv, "--accept-package-agreements", "--accept-source-agreements")
	}
	return argv
}
```

Ajouter en tête du fichier, avec les imports, l'alias qui évite d'importer `errors` pour
un seul appel :

```go
import "errors"

var errorsAs = errors.As
```

- [ ] **Étape 4 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/facade/ -v`
Attendu : SUCCÈS

- [ ] **Étape 5 : commit**

```bash
git add internal/facade/executer.go internal/facade/executer_test.go
git commit -m "Exécution des cibles, codes de retour et accords winget sur --yes"
```

---

## Tâche 9 : les parsers des verbes normalisés

**Fichiers :**
- Créer : `internal/brew/parse.go`, `internal/brew/parse_test.go`
- Créer : `internal/winget/parse.go`, `internal/winget/parse_test.go`
- Modifier : `internal/brew/verbs.go`, `internal/winget/verbs.go` (brancher `Parse`)
- Lire : `internal/winget/table.go` (le découpage à largeur fixe existe déjà)

**Interfaces :**
- Consomme : `pm.Package`, `pm.Parser`, les fixtures de la tâche 1
- Produit : `Parse` branché sur `list`, `outdated`, `search`, `source` chez brew et winget

- [ ] **Étape 1 : lire l'existant**

Lire `internal/winget/table.go` en entier. Le découpage aux frontières de colonnes, avec
en-têtes traduits, y est déjà résolu — les parsers doivent s'appuyer dessus, **pas**
refaire un découpage à eux.

- [ ] **Étape 2 : écrire les tests qui échouent**

`internal/brew/parse_test.go` :

```go
package brew

import (
	"os"
	"testing"
)

func TestParseOutdated(t *testing.T) {
	data, err := os.ReadFile("testdata/outdated.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOutdated(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne : le jeu d'essai de la tâche 1 est-il non vide ?")
	}
	for _, r := range rows {
		if r.Name == "" {
			t.Errorf("ligne sans nom : %+v", r)
		}
		if r.Version == "" || r.Available == "" {
			t.Errorf("outdated doit porter les deux versions : %+v", r)
		}
		if r.Kind == "" {
			t.Errorf("ligne sans badge : %+v", r)
		}
	}
}

func TestParseList(t *testing.T) {
	data, err := os.ReadFile("testdata/list-versions.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne")
	}
	for _, r := range rows {
		if r.Name == "" || r.Version == "" {
			t.Errorf("« nom version » attendu sur chaque ligne : %+v", r)
		}
	}
}

// Une sortie vide est un résultat valide — rien n'est périmé —, pas une erreur.
func TestParseOutdadedVide(t *testing.T) {
	rows, err := parseOutdated([]byte(`{"formulae":[],"casks":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu vide", rows)
	}
}
```

`internal/winget/parse_test.go` :

```go
package winget

import (
	"os"
	"testing"
)

// Les en-têtes de winget sont traduits : le parser doit tenir sur un jeu français.
func TestParseUpgradeFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/upgrade-fr.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOutdated(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne extraite du tableau français")
	}
	for _, r := range rows {
		if r.Name == "" {
			t.Errorf("ligne sans identifiant : %+v", r)
		}
		if r.Version == "" || r.Available == "" {
			t.Errorf("outdated doit porter les deux versions : %+v", r)
		}
	}
}

func TestParseListFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/list-fr.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne")
	}
}

func TestParseSearchFrancais(t *testing.T) {
	data, err := os.ReadFile("testdata/search-fr.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseSearch(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne")
	}
}

// Une barre de progression ou un en-tête de copyright ne doivent pas devenir des paquets.
func TestParseIgnoreLeBruit(t *testing.T) {
	bruit := []byte("   \\\n   |\nAucun paquet installé ne correspond aux critères.\n")
	rows, err := parseList(bruit)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, attendu vide", rows)
	}
}
```

- [ ] **Étape 3 : lancer les tests, vérifier qu'ils échouent**

Lancer : `go test ./internal/brew/ ./internal/winget/ -run Parse -v`
Attendu : ÉCHEC à la compilation — `undefined: parseOutdated`

- [ ] **Étape 4 : écrire `internal/brew/parse.go`**

Ajuster les balises JSON à la forme réelle capturée en tâche 1.

```go
package brew

import (
	"encoding/json"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// brew est le seul des trois à offrir une vraie sortie machine : --json=v2 dispense de
// tout découpage de colonnes.
type sortieOutdated struct {
	Formulae []entreeOutdated `json:"formulae"`
	Casks    []entreeOutdated `json:"casks"`
}

type entreeOutdated struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

func parseOutdated(out []byte) ([]pm.Package, error) {
	var s sortieOutdated
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(s.Formulae)+len(s.Casks))
	for _, groupe := range []struct {
		entrees []entreeOutdated
		badge   string
	}{
		{s.Formulae, pm.BadgeFormula},
		{s.Casks, pm.BadgeCask},
	} {
		for _, e := range groupe.entrees {
			installee := ""
			if len(e.InstalledVersions) > 0 {
				installee = e.InstalledVersions[len(e.InstalledVersions)-1]
			}
			rows = append(rows, pm.Package{
				Name:      e.Name,
				Version:   installee,
				Available: e.CurrentVersion,
				Kind:      groupe.badge,
				PM:        "brew",
			})
		}
	}
	return rows, nil
}

// parseList lit « nom version [version…] », une ligne par formula ou cask.
func parseList(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		champs := strings.Fields(ligne)
		if len(champs) < 2 {
			continue
		}
		rows = append(rows, pm.Package{
			Name:    champs[0],
			Version: champs[len(champs)-1], // la plus récente des versions gardées
			Kind:    pm.BadgeFormula,
			PM:      "brew",
		})
	}
	return rows, nil
}

// parseSearch : brew search rend un nom par ligne, sans version.
func parseSearch(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		nom := strings.TrimSpace(ligne)
		// Les en-têtes de section (« ==> Formulae ») ne sont pas des paquets.
		if nom == "" || strings.HasPrefix(nom, "==>") {
			continue
		}
		rows = append(rows, pm.Package{Name: nom, Kind: pm.BadgeFormula, PM: "brew"})
	}
	return rows, nil
}

// parseSource : brew tap rend un tap par ligne.
func parseSource(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		rows = append(rows, pm.Package{Name: strings.TrimSpace(ligne), PM: "brew"})
	}
	return rows, nil
}
```

- [ ] **Étape 5 : écrire `internal/winget/parse.go`**

S'appuyer sur les fonctions de découpage de `table.go` relevées à l'étape 1 ; remplacer
`decouperTableau` par leur nom réel.

```go
package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// winget est à l'opposé de brew : aucune sortie machine, que des tableaux à largeur fixe
// aux en-têtes traduits. Le découpage aux frontières de colonnes vit déjà dans table.go —
// ces parsers ne font que nommer les colonnes qui les intéressent.

func parseOutdated(out []byte) ([]pm.Package, error) {
	lignes, err := decouperTableau(out) // ← nom réel relevé à l'étape 1
	if err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		if l.ID == "" {
			continue
		}
		rows = append(rows, pm.Package{
			Name:      l.ID,
			Version:   l.Version,
			Available: l.Disponible,
			Kind:      pm.BadgeWinget,
			PM:        "winget",
		})
	}
	return rows, nil
}

func parseList(out []byte) ([]pm.Package, error) {
	lignes, err := decouperTableau(out)
	if err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		if l.ID == "" {
			continue
		}
		badge := pm.BadgeWinget
		if l.Source == "" {
			badge = pm.BadgeOther // installé hors catalogue (ARP/MSIX)
		}
		rows = append(rows, pm.Package{
			Name:    l.ID,
			Version: l.Version,
			Kind:    badge,
			Source:  l.Source,
			PM:      "winget",
		})
	}
	return rows, nil
}

func parseSearch(out []byte) ([]pm.Package, error) {
	lignes, err := decouperTableau(out)
	if err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		if l.ID == "" {
			continue
		}
		rows = append(rows, pm.Package{
			Name:   l.ID,
			Kind:   pm.BadgeWinget,
			Source: l.Source,
			PM:     "winget",
		})
	}
	return rows, nil
}

func parseSource(out []byte) ([]pm.Package, error) {
	lignes, err := decouperTableau(out)
	if err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		if l.Nom == "" {
			continue
		}
		rows = append(rows, pm.Package{Name: l.Nom, Source: l.URL, PM: "winget"})
	}
	return rows, nil
}
```

- [ ] **Étape 6 : brancher `Parse` dans les deux tables**

Dans `internal/brew/verbs.go`, ajouter le champ aux quatre verbes normalisés :

```go
"list":     {Native: []string{"list", "--versions"}, Pool: pm.PoolAucun, Parse: parseList},
"outdated": {Native: []string{"outdated", "--json=v2"}, Pool: pm.PoolAucun, Parse: parseOutdated},
"search":   {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue, Parse: parseSearch},
"source":   {Native: []string{"tap"}, Pool: pm.PoolAucun, Parse: parseSource},
```

Dans `internal/winget/verbs.go`, de même :

```go
"list":     {Native: []string{"list"}, Pool: pm.PoolAucun, Parse: parseList},
"outdated": {Native: []string{"list", "--upgrade-available"}, Pool: pm.PoolAucun, Parse: parseOutdated},
"search":   {Native: []string{"search", pm.MarqueurTous}, Pool: pm.PoolCatalogue, Parse: parseSearch},
"source":   {Native: []string{"source", "list"}, Pool: pm.PoolAucun, Parse: parseSource},
```

- [ ] **Étape 7 : lancer les tests, vérifier qu'ils passent**

Lancer : `go test ./... -v`
Attendu : SUCCÈS partout, y compris les tests de table des tâches 3 et 4

- [ ] **Étape 8 : commit**

```bash
git add internal/brew internal/winget
git commit -m "Parsers des verbes normalisés, branchés dans les tables"
```

---

## Tâche 10 : formatage de la sortie

**Fichiers :**
- Créer : `internal/facade/format.go`, `internal/facade/format_test.go`

**Interfaces :**
- Consomme : `pm.Package`
- Produit : `facade.Formater(v pm.Verb, rows []pm.Package, json bool) string`

- [ ] **Étape 1 : écrire le test qui échoue**

`internal/facade/format_test.go` :

```go
package facade

import (
	"encoding/json"
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func lignes() []pm.Package {
	return []pm.Package{
		{Name: "fd", Version: "10.1.0", Available: "10.2.0", PM: "scoop"},
		{Name: "Git.Git", Version: "2.54.0", Available: "2.55.0", PM: "winget"},
	}
}

// La colonne PM n'apparaît que si plus d'un gestionnaire a contribué : sur macOS, où seul
// brew répond, elle serait du bruit.
func TestColonnePMSeulementSiPlusieurs(t *testing.T) {
	avec := Formater("outdated", lignes(), false)
	if !strings.Contains(avec, "PM") {
		t.Errorf("deux gestionnaires : la colonne PM est attendue\n%s", avec)
	}
	if !strings.Contains(avec, "scoop") || !strings.Contains(avec, "winget") {
		t.Errorf("les deux gestionnaires doivent apparaître\n%s", avec)
	}

	seul := Formater("outdated", lignes()[:1], false)
	if strings.Contains(seul, "PM") {
		t.Errorf("un seul gestionnaire : pas de colonne PM\n%s", seul)
	}
}

func TestColonnesAlignees(t *testing.T) {
	out := Formater("outdated", lignes(), false)
	var largeurs []int
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		largeurs = append(largeurs, strings.Index(l, strings.Fields(l)[1]))
	}
	for i := 1; i < len(largeurs); i++ {
		if largeurs[i] != largeurs[0] {
			t.Fatalf("colonnes désalignées :\n%s", out)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	out := Formater("outdated", lignes(), true)

	var relu []pm.Package
	if err := json.Unmarshal([]byte(out), &relu); err != nil {
		t.Fatalf("sortie JSON illisible : %v\n%s", err, out)
	}
	if len(relu) != 2 || relu[0].Name != "fd" {
		t.Fatalf("relu = %+v", relu)
	}
}

// `list` n'a pas de version disponible : la colonne DISPO n'a rien à faire là.
func TestListNAPasDeColonneDispo(t *testing.T) {
	rows := []pm.Package{{Name: "fd", Version: "10.2.0", PM: "scoop"}}
	out := Formater("list", rows, false)
	if strings.Contains(out, "DISPO") {
		t.Errorf("colonne DISPO inattendue pour list :\n%s", out)
	}
}

func TestAucuneLigne(t *testing.T) {
	if out := Formater("outdated", nil, false); strings.TrimSpace(out) == "" {
		t.Error("une liste vide doit dire quelque chose, pas rien")
	}
	if out := Formater("outdated", nil, true); strings.TrimSpace(out) != "[]" {
		t.Errorf("JSON vide = %q, attendu « [] »", out)
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/facade/ -run 'Colonne|Format|List|Aucune' -v`
Attendu : ÉCHEC à la compilation — `undefined: Formater`

- [ ] **Étape 3 : écrire `internal/facade/format.go`**

```go
package facade

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Formater rend la sortie d'un verbe normalisé. Les colonnes s'adaptent aux données :
// PM n'apparaît que si plusieurs gestionnaires ont contribué, DISPO que si au moins une
// ligne porte une version disponible. Une colonne toujours vide n'apprend rien.
func Formater(v pm.Verb, rows []pm.Package, enJSON bool) string {
	if enJSON {
		if rows == nil {
			rows = []pm.Package{}
		}
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return "[]"
		}
		return string(data)
	}

	if len(rows) == 0 {
		return "rien à signaler\n"
	}

	avecPM := plusieursPM(rows)
	avecDispo := false
	avecSource := false
	for _, r := range rows {
		if r.Available != "" {
			avecDispo = true
		}
		if r.Source != "" {
			avecSource = true
		}
	}

	entete := []string{"PAQUET", "ACTUEL"}
	if avecDispo {
		entete = append(entete, "DISPO")
	}
	if avecSource {
		entete = append(entete, "SOURCE")
	}
	if avecPM {
		entete = append(entete, "PM")
	}

	table := [][]string{entete}
	for _, r := range rows {
		ligne := []string{r.Name, r.Version}
		if avecDispo {
			ligne = append(ligne, r.Available)
		}
		if avecSource {
			ligne = append(ligne, r.Source)
		}
		if avecPM {
			ligne = append(ligne, r.PM)
		}
		table = append(table, ligne)
	}
	return aligner(table)
}

func plusieursPM(rows []pm.Package) bool {
	vu := ""
	for _, r := range rows {
		if r.PM == "" {
			continue
		}
		if vu == "" {
			vu = r.PM
		} else if vu != r.PM {
			return true
		}
	}
	return false
}

// aligner pose les colonnes à largeur fixe, deux espaces de gouttière. Le même principe
// que les tableaux de winget — à ceci près qu'ici, c'est nous qui les écrivons.
func aligner(table [][]string) string {
	if len(table) == 0 {
		return ""
	}
	largeurs := make([]int, len(table[0]))
	for _, ligne := range table {
		for i, cell := range ligne {
			if n := len([]rune(cell)); n > largeurs[i] {
				largeurs[i] = n
			}
		}
	}

	var b strings.Builder
	for _, ligne := range table {
		for i, cell := range ligne {
			if i == len(ligne)-1 {
				b.WriteString(cell)
			} else {
				fmt.Fprintf(&b, "%-*s  ", largeurs[i], cell)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
```

- [ ] **Étape 4 : lancer le test, vérifier qu'il passe**

Lancer : `go test ./internal/facade/ -v`
Attendu : SUCCÈS

- [ ] **Étape 5 : commit**

```bash
git add internal/facade/format.go internal/facade/format_test.go
git commit -m "Formatage de la sortie : tableau à colonnes adaptatives et --json"
```

---

## Tâche 11 : aiguillage `main.go` et drapeaux

Fin de la partie A : `jigger install fd` fonctionne de bout en bout.

**Fichiers :**
- Modifier : `main.go`
- Créer : `main_test.go`

**Interfaces :**
- Consomme : `facade.ResoudreVerbe`, `facade.Router`, `facade.Executer`, `facade.Formater`
- Produit : `motsReserves`, `runFacade(args []string) int`

- [ ] **Étape 1 : écrire le test qui échoue**

`main_test.go` :

```go
package main

import "testing"

// Les six mots réservés ne doivent jamais devenir des verbes de façade. Contrainte
// permanente : aucune sous-commande interne future ne peut porter le nom d'un verbe
// canonique (cf. spec §1).
func TestMotsReserves(t *testing.T) {
	attendus := []string{"pick", "render", "complete", "prompt", "warm", "demo"}
	for _, m := range attendus {
		if !motsReserves[m] {
			t.Errorf("« %s » doit être réservé", m)
		}
	}
	// Un verbe de la façade ne doit surtout pas y figurer.
	for _, v := range []string{"install", "list", "outdated", "search", "info"} {
		if motsReserves[v] {
			t.Errorf("« %s » est un verbe de façade, il ne peut pas être réservé", v)
		}
	}
}

func TestSeparerDrapeaux(t *testing.T) {
	verbe, args, o, err := separerDrapeaux(
		[]string{"install", "--pm", "scoop", "fd", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if verbe != "install" {
		t.Fatalf("verbe = %q", verbe)
	}
	if len(args) != 1 || args[0] != "fd" {
		t.Fatalf("args = %v, attendu [fd]", args)
	}
	if o.PM != "scoop" {
		t.Fatalf("PM = %q, attendu scoop", o.PM)
	}
	if !o.Yes {
		t.Fatal("--yes non pris en compte")
	}
}

func TestSeparerDrapeauxJSON(t *testing.T) {
	_, _, o, err := separerDrapeaux([]string{"outdated", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.JSON {
		t.Fatal("--json non pris en compte")
	}
}

// Un drapeau destiné au gestionnaire ne doit pas être avalé par jigger.
func TestDrapeauxInconnusPassentAuGestionnaire(t *testing.T) {
	_, args, _, err := separerDrapeaux([]string{"install", "--cask", "firefox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "--cask" || args[1] != "firefox" {
		t.Fatalf("args = %v, attendu [--cask firefox]", args)
	}
}

func TestPMSansValeur(t *testing.T) {
	if _, _, _, err := separerDrapeaux([]string{"install", "--pm"}); err == nil {
		t.Fatal("attendu une erreur : --pm sans valeur")
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test . -run 'Mots|Separer|Drapeaux|PMSans' -v`
Attendu : ÉCHEC à la compilation — `undefined: motsReserves`

- [ ] **Étape 3 : modifier `main.go`**

Remplacer le `switch os.Args[1]` par un aiguillage à deux temps :

```go
// motsReserves sont les sous-commandes internes de jigger. Tout autre premier mot est un
// verbe de façade.
//
// Contrainte permanente : aucune sous-commande interne future ne peut porter le nom d'un
// verbe canonique. Si « jigger list » devait un jour désigner un usage interne, c'est le
// mot interne qui change — pas le verbe.
var motsReserves = map[string]bool{
	"pick": true, "render": true, "complete": true,
	"prompt": true, "warm": true, "demo": true,
}

func main() {
	ui.Version = version

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "--version", "-v", "version":
		fmt.Println("jigger", version)
		return
	case "--help", "-h", "help":
		usage()
		return
	}

	if !motsReserves[os.Args[1]] {
		os.Exit(runFacade(os.Args[1:]))
	}

	switch os.Args[1] {
	case "pick":
		os.Exit(runPick(arg(2)))
	case "render":
		os.Exit(runRender(os.Args[2:]))
	case "complete":
		runComplete(arg(2))
	case "prompt":
		os.Exit(runPrompt(os.Args[2:]))
	case "warm":
		os.Exit(runWarm(os.Args[2:]))
	case "demo":
		runDemo()
	}
}
```

Ajouter à la suite :

```go
// optsCLI rassemble les drapeaux que jigger interprète lui-même. Tous les autres mots en
// « -- » sont passés au gestionnaire : `jg install --cask firefox` doit marcher.
type optsCLI struct {
	PM   string
	JSON bool
	Yes  bool
}

func separerDrapeaux(argv []string) (verbe string, args []string, o optsCLI, err error) {
	if len(argv) == 0 {
		return "", nil, o, fmt.Errorf("aucun verbe")
	}
	verbe = argv[0]
	for i := 1; i < len(argv); i++ {
		switch argv[i] {
		case "--pm":
			if i+1 >= len(argv) {
				return "", nil, o, fmt.Errorf("jigger : --pm attend un nom de gestionnaire")
			}
			i++
			o.PM = argv[i]
		case "--json":
			o.JSON = true
		case "--yes":
			o.Yes = true
		default:
			args = append(args, argv[i])
		}
	}
	return verbe, args, o, nil
}

// runFacade déroule le pipeline de la spec §3 : résoudre le verbe, résoudre la cible,
// exécuter, formater.
func runFacade(argv []string) int {
	premier, reste, o, err := separerDrapeaux(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	ligne := append([]string{premier}, reste...)
	verbe, args, capables, err := facade.ResoudreVerbe(ligne)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	cats := map[string]*pm.Catalog{}
	for _, m := range capables {
		cats[m.Cmd()] = m.Load()
	}

	cibles, amb, err := facade.Router(verbe, args, o.PM, capables, cats)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if amb != nil {
		choisi, ok := trancher(amb) // branché en tâche 13
		if !ok {
			return 2
		}
		cibles, amb, err = facade.Router(verbe, args, choisi, capables, cats)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}

	rows, code := facade.Executer(verbe, cibles, facade.Opts{JSON: o.JSON, Yes: o.Yes})
	if len(rows) > 0 || facadeNormalise(verbe) {
		fmt.Print(facade.Formater(verbe, rows, o.JSON))
	}
	return code
}
```

Exposer le prédicat depuis `facade` (`internal/facade/executer.go`) :

```go
// Normalise dit si un verbe rend un tableau plutôt qu'une sortie relayée.
func Normalise(v pm.Verb) bool { return normalise(v) }
```

et appeler `facade.Normalise(verbe)` dans `runFacade`.

- [ ] **Étape 4 : poser un `trancher` provisoire**

La tâche 13 le remplacera par le sélecteur. Pour l'instant, il échoue proprement :

```go
// trancher demande à l'utilisateur quel gestionnaire retenir. Le sélecteur arrive en
// tâche 13 ; d'ici là, on échoue en listant les candidats — ce qui est de toute façon le
// comportement attendu hors terminal.
func trancher(amb *facade.Ambiguite) (string, bool) {
	fmt.Fprintf(os.Stderr, "jigger : « %s » — connu de plusieurs gestionnaires :\n", amb.Nom)
	for _, c := range amb.Candidats {
		fmt.Fprintf(os.Stderr, "        %s\n", c.Mgr.Cmd())
	}
	fmt.Fprintf(os.Stderr, "        Choisis avec --pm <gestionnaire>\n")
	return "", false
}
```

- [ ] **Étape 5 : mettre à jour `usage()`**

```go
func usage() {
	fmt.Fprintln(os.Stderr, "usage: jigger <verbe> [--pm <gestionnaire>] [--json] [--yes] [arguments…]")
	fmt.Fprintln(os.Stderr, "       jigger pick|complete \"<ligne>\" | jigger render --line \"<ligne>\"")
	fmt.Fprintln(os.Stderr, "       jigger prompt [--refresh [--wait]|--path] | jigger warm [--all|--installed]")
}
```

- [ ] **Étape 6 : lancer les tests, vérifier qu'ils passent**

Lancer : `go test ./... -v && make build`
Attendu : SUCCÈS et binaire construit

- [ ] **Étape 7 : essai manuel de bout en bout**

```sh
./jigger outdated
./jigger list --json | head -20
./jigger doctor
./jigger teleporter          # doit dire « verbe inconnu »
```

- [ ] **Étape 8 : commit**

```bash
git add main.go main_test.go internal/facade/executer.go
git commit -m "Aiguillage de la façade dans main, avec --pm, --json et --yes"
```

---

# Partie B — le popup et les greffons

## Tâche 12 : `Item.PM` et le vocabulaire de la façade dans le popup

**Fichiers :**
- Modifier : `internal/pm/pm.go` (`Item`), `internal/complete/complete.go`
- Modifier : `internal/complete/complete_test.go`

**Interfaces :**
- Consomme : `managers.Vocabulaire`, `managers.Tables`
- Produit : `pm.Item.PM` ; `complete.Complete` reconnaît `jigger` et `jg`

- [ ] **Étape 1 : écrire le test qui échoue**

Ajouter à `internal/complete/complete_test.go` :

```go
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
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/complete/ -run Facade -v`
Attendu : ÉCHEC — `undefined: CompleteFacade`, et `Item.PM` inconnu

- [ ] **Étape 3 : ajouter `PM` à `pm.Item`**

Dans `internal/pm/pm.go` :

```go
// Item est un candidat de complétion tel que l'affiche le popup.
type Item struct {
	Name      string
	Badge     string // classe du paquet ; "" pour une sous-commande ou une option
	Installed bool
	Version   string // version installée (vide si non installé/inconnue)
	// PM nomme le gestionnaire d'où vient le candidat, et n'est rempli que par la
	// façade : sur `brew install ⇥`, il n'y a rien à désambiguïser. Le badge ne
	// suffirait pas — BadgeOther est partagé par winget et scoop.
	PM string
}
```

- [ ] **Étape 4 : étendre `internal/complete/complete.go`**

Ajouter la reconnaissance de jigger avant le routage vers un gestionnaire :

```go
// estFacade dit si le premier mot d'une ligne désigne jigger lui-même — « jigger » ou son
// alias « jg » — plutôt qu'un gestionnaire.
func estFacade(mot string) bool {
	m := motCommande(mot)
	return m == "jigger" || m == "jg"
}

// Complete calcule le contexte et les candidats pour la ligne donnée.
func Complete(line string) Result {
	premier, _, _ := strings.Cut(strings.TrimSpace(line), " ")
	if estFacade(premier) {
		dispo := managers.Available()
		cats := map[string]*pm.Catalog{}
		for _, m := range dispo {
			cats[m.Cmd()] = m.Load()
		}
		return CompleteFacade(line, dispo, cats)
	}
	m := managers.Detect(line)
	return CompleteWith(line, m, m.Load())
}

// CompleteFacade complète la syntaxe unique : « jg ⇥ » propose les verbes, « jg source ⇥ »
// les sous-verbes, « jg install g » les paquets de tous les gestionnaires disponibles.
//
// Les catalogues sont filtrés CHEZ CHAQUE GESTIONNAIRE avant d'être réunis. L'ordre
// compte : concaténer trois catalogues — 14 401 noms rien que pour winget — puis balayer
// coûterait le budget de la frappe (cf. spec §5).
func CompleteFacade(line string, dispo []pm.Manager, cats map[string]*pm.Catalog) Result {
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

	res := Result{Prefix: prefix, Word: word, Cmd: "jigger"}
	lw := strings.ToLower(word)
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
			if !strings.HasPrefix(strings.ToLower(n), lw) {
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
```

Ajouter `"sort"` aux imports.

- [ ] **Étape 5 : lancer les tests, vérifier qu'ils passent**

Lancer : `go test ./internal/complete/ ./internal/pm/ -v`
Attendu : SUCCÈS, y compris tous les tests natifs existants

- [ ] **Étape 6 : commit**

```bash
git add internal/pm/pm.go internal/complete
git commit -m "Le popup complète le vocabulaire de la façade et nomme le gestionnaire"
```

---

## Tâche 13 : colonne PM dans le cadre et sélecteur de désambiguïsation

**Fichiers :**
- Modifier : `internal/ui/frame.go`, `internal/ui/frame_test.go`
- Modifier : `main.go` (remplacer le `trancher` provisoire)

**Interfaces :**
- Consomme : `pm.Item.PM`, `facade.Ambiguite`
- Produit : le cadre affiche la colonne PM quand au moins un item en porte un ;
  `trancher` ouvre le sélecteur

- [ ] **Étape 1 : écrire le test qui échoue**

Ajouter à `internal/ui/frame_test.go` :

```go
// La colonne PM apparaît selon les données, pas selon un drapeau — même règle que les
// tableaux de sortie.
func TestFrameColonnePMSelonLesDonnees(t *testing.T) {
	avec := Frame{
		Title: "jigger install",
		Items: []pm.Item{
			{Name: "Git.Git", Badge: pm.BadgeWinget, PM: "winget"},
			{Name: "git", Badge: pm.BadgeScoop, PM: "scoop"},
		},
		Rows: 8,
	}.Render()
	if !strings.Contains(avec, "winget") || !strings.Contains(avec, "scoop") {
		t.Errorf("les deux gestionnaires doivent apparaître dans le cadre :\n%s", avec)
	}

	sans := Frame{
		Title: "brew install",
		Items: []pm.Item{{Name: "git", Badge: pm.BadgeFormula}},
		Rows:  8,
	}.Render()
	if strings.Contains(sans, "brew") && !strings.Contains(sans, "brew install") {
		t.Errorf("aucun PM sur les items : rien ne doit s'ajouter :\n%s", sans)
	}
}
```

- [ ] **Étape 2 : lancer le test, vérifier qu'il échoue**

Lancer : `go test ./internal/ui/ -run ColonnePM -v`
Attendu : ÉCHEC — le cadre n'affiche pas le PM

- [ ] **Étape 3 : afficher le PM dans `internal/ui/frame.go`**

Lire d'abord la fonction qui rend une ligne d'item, puis y ajouter le PM en suffixe, aligné
à droite comme la version. La règle : **présent si au moins un item du cadre porte un
`PM`**, absent sinon — de sorte que `brew install ⇥` est inchangé au pixel près.

Réutiliser les couleurs de badge déjà définies dans le fichier : la spec §5 exige que le
popup, les tableaux de sortie et le bloc oh-my-posh partagent une seule identité visuelle.

- [ ] **Étape 4 : remplacer `trancher` dans `main.go`**

```go
// trancher ouvre le sélecteur sur les candidats d'un nom ambigu. Ce n'est pas un nouvel
// écran : c'est le popup, avec un autre titre et d'autres touches de pied.
//
// Hors terminal — pipe, script, CI — il n'y a personne pour choisir : on échoue en
// listant les candidats et en rappelant --pm. Jamais de choix automatique.
func trancher(amb *facade.Ambiguite) (string, bool) {
	items := make([]complete.Item, 0, len(amb.Candidats))
	for _, c := range amb.Candidats {
		items = append(items, complete.Item{
			Name:  c.Mgr.Cmd(),
			Badge: c.Badge,
			PM:    c.Mgr.Cmd(),
		})
	}

	tty, err := openTTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "jigger : « %s » — connu de plusieurs gestionnaires :\n", amb.Nom)
		for _, c := range amb.Candidats {
			fmt.Fprintf(os.Stderr, "        %s\n", c.Mgr.Cmd())
		}
		fmt.Fprintln(os.Stderr, "        Choisis avec --pm <gestionnaire>")
		return "", false
	}
	defer tty.Close()

	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)
	titre := fmt.Sprintf("%s : %d gestionnaires", amb.Nom, len(amb.Candidats))
	model := ui.New(titre, complete.Result{Executable: true, Items: items})

	fmt.Fprint(tty.Out, "\r\n")
	prog := tea.NewProgram(model, tea.WithInput(tty.In), tea.WithOutput(tty.Out))
	final, err := prog.Run()
	fmt.Fprint(tty.Out, "\x1b[1A\r")
	if err != nil {
		return "", false
	}

	m := final.(ui.Model)
	if m.Chosen == nil {
		return "", false // annulé
	}
	return m.Chosen.Name, true
}
```

- [ ] **Étape 5 : lancer les tests, vérifier qu'ils passent**

Lancer : `go test ./... && make build`
Attendu : SUCCÈS

- [ ] **Étape 6 : essai manuel**

Sur une machine où deux gestionnaires connaissent le même nom :

```sh
./jigger install git      # le sélecteur doit s'ouvrir
./jigger install git --pm scoop   # il ne doit PAS s'ouvrir
./jigger install git | cat        # hors TTY : message + --pm, code 2
```

- [ ] **Étape 7 : commit**

```bash
git add internal/ui main.go
git commit -m "Colonne PM dans le cadre et sélecteur de désambiguïsation"
```

---

## Tâche 14 : les greffons — alias `jg` et reconnaissance

**Fichiers :**
- Modifier : `shell/jigger.plugin.zsh`, `shell/jigger.psm1`
- Modifier : `tests/zpty.zsh`, `tests/smoke.ps1`

**Interfaces :**
- Consomme : `jigger render --line "jg …"` (tâche 12)
- Produit : `jg` défini dans les deux shells ; le popup se déclenche dessus

- [ ] **Étape 1 : lire les deux greffons**

Relever comment la liste des commandes qui déclenchent le popup est constituée —
`JIGGER_COMMANDS` sous PowerShell, son équivalent sous zsh — et où l'alias doit être posé
pour que le widget le voie.

- [ ] **Étape 2 : écrire le test qui échoue (zsh)**

Ajouter un cas à `tests/zpty.zsh`, sur le modèle des cas existants : taper `jg inst`,
vérifier que le cadre apparaît et contient `install`.

- [ ] **Étape 3 : poser l'alias sous zsh**

Dans `shell/jigger.plugin.zsh` :

```sh
# jg : l'alias court de la façade. Ajouté à la liste des commandes qui arment le popup —
# sans quoi le widget ne se déclencherait que sur « jigger » en toutes lettres.
alias jg=jigger
```

Ajouter `jigger` et `jg` à la liste des commandes surveillées.

- [ ] **Étape 4 : poser l'alias sous PowerShell**

Dans `shell/jigger.psm1` :

```powershell
# jg : l'alias court de la façade.
Set-Alias -Name jg -Value jigger -Scope Global

# Les deux noms arment le popup, au même titre que winget et scoop.
if (-not $env:JIGGER_COMMANDS) { $env:JIGGER_COMMANDS = 'winget,scoop,jigger,jg' }
```

Vérifier que la valeur par défaut de `JIGGER_COMMANDS` documentée dans le README suit.

- [ ] **Étape 5 : ajouter le cas dans `tests/smoke.ps1`**

Vérifier que `jg` figure bien dans les commandes surveillées et que
`jigger render --line "jg inst"` produit un cadre non vide.

- [ ] **Étape 6 : lancer les suites de la plateforme**

Lancer : `make test-all`
Attendu : SUCCÈS — tests Go **et** suite shell de la plateforme

- [ ] **Étape 7 : commit**

```bash
git add shell tests
git commit -m "Alias jg dans les deux greffons, et popup armé dessus"
```

---

## Tâche 15 : le banc d'essai de latence

La spec §5 identifie ce point comme le seul du design qui menace une propriété acquise.
La parade — filtrer chez chaque gestionnaire avant de réunir — est déjà écrite en tâche 12 ;
cette tâche la **mesure**, pour qu'une régression se voie au lieu de se deviner.

**Fichiers :**
- Modifier : `internal/complete/complete_test.go`

**Interfaces :**
- Consomme : `complete.CompleteFacade`
- Produit : `BenchmarkCompleteFacade`

- [ ] **Étape 1 : écrire le banc d'essai**

Ajouter à `internal/complete/complete_test.go`, sur le modèle de `BenchmarkComplete` :

```go
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
```

- [ ] **Étape 2 : mesurer les deux chemins**

Lancer : `go test ./internal/complete/ -bench 'Complete' -benchmem -run '^$'`

Relever les deux lignes. Le chemin façade parcourt environ trois fois plus de noms que le
chemin natif ; un rapport nettement supérieur à trois signale une fusion faite dans le
mauvais ordre.

- [ ] **Étape 3 : corriger si le rapport dérape**

Si `BenchmarkCompleteFacade` dépasse largement trois fois `BenchmarkComplete`, relire
`CompleteFacade` : la boucle de filtrage doit rester **à l'intérieur** de la boucle sur les
gestionnaires, et le tri final ne porter que sur les survivants.

- [ ] **Étape 4 : consigner le résultat**

Ajouter les deux mesures dans la section « Tests » de la spec, avec la machine et la date :
un budget sans chiffre de référence ne se défend pas.

- [ ] **Étape 5 : commit**

```bash
git add internal/complete/complete_test.go docs/specs
git commit -m "Banc d'essai du chemin façade du popup, avec mesure de référence"
```

---

## Tâche 16 : le README

**Fichiers :**
- Modifier : `README.md`

- [ ] **Étape 1 : ajouter la section « Une seule syntaxe »**

À placer **avant** « Sous le capot (CLI) », après « Usage ». Elle doit couvrir :

- ce que fait `jg <verbe>`, avec les 12 verbes en tableau et leur traduction par
  gestionnaire — reprendre la table §2 de la spec **telle que corrigée en tâche 1** ;
- la règle de routage : le nom est cherché partout ; un seul gestionnaire le connaît, il
  gagne ; plusieurs, le sélecteur tranche ; jamais de choix automatique ;
- `--pm`, `--json`, `--yes`, avec un mot sur le fait que `--yes` accepte des licences et
  n'est jamais implicite ;
- que la colonne PM des tableaux n'apparaît que si plusieurs gestionnaires ont répondu ;
- que les commandes natives continuent de marcher, popup compris : la façade s'ajoute,
  elle ne remplace rien.

- [ ] **Étape 2 : mettre à jour la feuille de route**

Retirer ce qui est livré. Ajouter, au titre des non-buts assumés de la phase 1 : les verbes
singuliers, le volet d'aperçu, les gestionnaires tiers par sous-processus.

- [ ] **Étape 3 : mettre à jour « Sous le capot (CLI) »**

Ajouter les verbes de façade à la liste des sous-commandes, et mentionner la contrainte des
mots réservés.

- [ ] **Étape 4 : vérifier les exemples**

Chaque commande citée dans la nouvelle section doit être lancée pour de vrai sur la
plateforme courante, et sa sortie recopiée fidèlement — pas reconstituée de mémoire.

- [ ] **Étape 5 : commit**

```bash
git add README.md
git commit -m "README : la syntaxe unique de la façade"
```

---

## Auto-relecture du plan

**Couverture de la spec.** Chaque section a sa tâche : §1 architecture → 2, 11 ;
§2 vocabulaire → 1, 3, 4, 5 ; §3 routage → 6, 7 ; §4 normalisation → 8, 9, 10 ;
§5 popup → 12, 13, 14, 15. Portée phase 1 : les huit livrables sont couverts (types → 2 ;
tables → 1, 3, 4, 5 ; moteur → 6, 7, 8 ; parsers → 9 ; `main.go` → 11 ; popup → 12, 13 ;
greffons → 14 ; documentation → 16).

**Deux dépendances externes assumées, et pourquoi elles ne sont pas des trous :**

- La tâche 5 dépend de la signature réelle de `internal/scoop/outdated.go`, et la tâche 9
  de celle de `internal/winget/table.go`. Les deux commencent par une étape de lecture
  explicite, et le nom à substituer est signalé en commentaire dans le code proposé.
  Écrire une signature inventée ici serait pire qu'une étape de lecture.
- La tâche 9 dépend des jeux d'essai capturés en tâche 1. C'est voulu : la tâche 1 existe
  précisément pour que les parsers soient écrits contre des sorties réelles.

**Cohérence des types.** `pm.Verb`, `pm.Pool`/`PoolAucun`/`PoolCatalogue`/`PoolInstalles`,
`pm.Binding`, `pm.Parser`, `pm.Package`, `pm.Bindings`, `pm.MarqueurTous`/`MarqueurUn`,
`Binding.Argv`/`Valid`/`NomNatif`, `facade.Cible`/`Candidat`/`Ambiguite`/`Opts`,
`facade.ResoudreVerbe`/`Router`/`Executer`/`Formater`/`Normalise`,
`managers.Tables`/`Vocabulaire`, `complete.CompleteFacade`, `pm.Item.PM` — définis une
fois, employés sous le même nom partout.

**Un ajout en cours de route.** `facade.Normalise` (exporté) n'apparaissait pas dans la
carte des fichiers ; il est créé en tâche 8 et exporté en tâche 11, où `runFacade` en a
besoin pour décider s'il faut imprimer un tableau.
