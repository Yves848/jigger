# Sélecteur de serveurs SSH — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal :** proposer les serveurs de `~/.ssh/config` dès qu'on tape `ssh`, `scp` ou `sftp`, dans la popup de jigger.

**Architecture :** un paquet `internal/ssh` qui implémente `pm.Manager` — et **jamais** `pm.Bindings`. Trois fournisseurs partageant une implémentation et un catalogue. Une règle générale ajoutée à `completeWith` : un fournisseur sans sous-commandes propose son catalogue dès le premier mot.

**Tech Stack :** Go, bibliothèque standard seule. Tests `go test ./...`.

**Spec :** `docs/specs/2026-08-30-selecteur-ssh-design.md`
**ADR :** `docs/adr/0005-completion-sans-facade.md`

## Global Constraints

- **Aucune dépendance nouvelle.** Bibliothèque standard seule — le parseur de configuration SSH s'écrit en une centaine de lignes.
- **Commentaires, messages de commit et documentation en français accentué**, comme tout le dépôt.
- Chemin de module : `gitlab.yg-devworks.com/yves/jigger`.
- **`pm.Manager` n'est pas modifié.** C'est la conséquence explicite de l'ADR-0005, et le troisième ADR d'affilée à le laisser intact.
- **`Load()` est sur le chemin du rendu** : il ne doit lire que des fichiers locaux, jamais lancer de sous-processus ni toucher au réseau.
- **Aucun test ne lit le `~/.ssh/config` réel de la machine.** Tous construisent leurs fichiers dans `t.TempDir()`.
- **Les tests existants de `internal/complete` ne doivent pas changer.** Ils sont la preuve que la règle générale ne touche pas brew, winget et scoop.

---

### Task 1 : Le parseur de `~/.ssh/config`

**Files :**
- Create : `internal/ssh/config.go`
- Test : `internal/ssh/config_test.go`

**Interfaces :**
- Consumes : rien.
- Produces :
  - `type Hote struct { Nom, HostName string }`
  - `func Lire(chemin string) []Hote` — les hôtes déclarés par le fichier et ses `Include`, triés, dédoublonnés

- [ ] **Step 1 : Écrire les tests qui échouent**

Créer `internal/ssh/config_test.go` :

```go
package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

// ecrire pose un fichier et rend son chemin.
func ecrire(t *testing.T, dir, nom, contenu string) string {
	t.Helper()
	p := filepath.Join(dir, nom)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func noms(hotes []Hote) []string {
	out := make([]string, 0, len(hotes))
	for _, h := range hotes {
		out = append(out, h.Nom)
	}
	return out
}

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLireNomsEtHostName(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host pve
    HostName 192.168.50.8
    User root

Host archlight
    HostName 192.168.50.207
`)
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"archlight", "pve"})
	if hotes[1].HostName != "192.168.50.8" {
		t.Errorf("HostName de pve = %q, attendu 192.168.50.8", hotes[1].HostName)
	}
}

func TestLireUnBlocAPlusieursMotifs(t *testing.T) {
	// « Host archlight aquarium 192.168.50.207 » declare trois facons valides de
	// designer la meme machine : les trois sont des candidats.
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host archlight aquarium 192.168.50.207\n    HostName 192.168.50.207\n")
	egal(t, noms(Lire(p)), []string{"192.168.50.207", "aquarium", "archlight"})
}

func TestLireEcarteLesMotifs(t *testing.T) {
	// `Host *` n'est pas un serveur : c'est un bloc de defauts.
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host *
    AddKeysToAgent yes

Host web-?
    User deploy

Host !prod
    User dev

Host reel
    HostName 10.0.0.1
`)
	egal(t, noms(Lire(p)), []string{"reel"})
}

func TestLireSuitUnInclude(t *testing.T) {
	d := t.TempDir()
	ecrire(t, d, "config.d/homelab.conf", "Host archlight\n    HostName 192.168.50.207\n")
	p := ecrire(t, d, "config", "Include config.d/homelab.conf\n\nHost pve\n    HostName 192.168.50.8\n")
	egal(t, noms(Lire(p)), []string{"archlight", "pve"})
}

func TestLireResoutUnIncludeGlob(t *testing.T) {
	// OpenSSH accepte les jokers dans Include ; une configuration en fragments s'en sert.
	d := t.TempDir()
	ecrire(t, d, "config.d/a.conf", "Host aaa\n")
	ecrire(t, d, "config.d/b.conf", "Host bbb\n")
	p := ecrire(t, d, "config", "Include config.d/*.conf\n")
	egal(t, noms(Lire(p)), []string{"aaa", "bbb"})
}

func TestLireNeBouclePasSurUnIncludeCirculaire(t *testing.T) {
	// Une configuration fautive ne doit pas figer le popup pendant la frappe.
	d := t.TempDir()
	ecrire(t, d, "b.conf", "Include a.conf\nHost dansB\n")
	p := ecrire(t, d, "a.conf", "Include b.conf\nHost dansA\n")
	egal(t, noms(Lire(p)), []string{"dansA", "dansB"})
}

func TestLireIgnoreLaCasseDesMotsCles(t *testing.T) {
	// OpenSSH est insensible a la casse sur ses mots-cles.
	d := t.TempDir()
	p := ecrire(t, d, "config", "HOST pve\n    hostname 192.168.50.8\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"pve"})
	if hotes[0].HostName != "192.168.50.8" {
		t.Errorf("HostName = %q", hotes[0].HostName)
	}
}

func TestLireIgnoreCommentairesEtLignesVides(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "# un commentaire\n\n   # indente\nHost pve\n")
	egal(t, noms(Lire(p)), []string{"pve"})
}

func TestLireDedoublonne(t *testing.T) {
	// Le meme nom declare deux fois (fragment + fichier principal) ne doit sortir
	// qu'une fois : le popup afficherait sinon deux lignes identiques.
	d := t.TempDir()
	ecrire(t, d, "f.conf", "Host pve\n    HostName 10.0.0.1\n")
	p := ecrire(t, d, "config", "Include f.conf\nHost pve\n    HostName 192.168.50.8\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"pve"})
	// La premiere valeur rencontree gagne, comme le fait OpenSSH lui-meme.
	if hotes[0].HostName != "10.0.0.1" {
		t.Errorf("HostName = %q, attendu celui du fragment inclus en premier", hotes[0].HostName)
	}
}

func TestLireUnFichierAbsentRendVide(t *testing.T) {
	// Une machine sans configuration SSH n'est pas une erreur.
	if got := Lire(filepath.Join(t.TempDir(), "inexistant")); len(got) != 0 {
		t.Errorf("got %v, attendu vide", got)
	}
}

func TestLireHostSansHostName(t *testing.T) {
	// Un bloc peut n'avoir que son nom : c'est valide, HostName vaut alors le nom.
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host solo\n    User root\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"solo"})
	if hotes[0].HostName != "" {
		t.Errorf("HostName = %q, attendu vide", hotes[0].HostName)
	}
}
```

- [ ] **Step 2 : Lancer les tests, vérifier qu'ils échouent**

Run : `go test ./internal/ssh/`
Expected : FAIL — le paquet n'existe pas (`no Go files`).

- [ ] **Step 3 : Écrire le parseur**

Créer `internal/ssh/config.go` :

```go
// Package ssh lit ~/.ssh/config pour en tirer les serveurs connus, et les propose
// comme catalogue de complétion.
//
// Ce fournisseur n'exécute rien : il implémente pm.Manager et jamais pm.Bindings.
// C'est la décision de l'ADR-0005 — le contrat de complétion n'est pas réservé aux
// gestionnaires de paquets.
package ssh

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Hote est un nom déclaré par un bloc Host, avec l'adresse que HostName lui donne.
type Hote struct {
	Nom      string
	HostName string
}

// Lire rend les hôtes déclarés par le fichier et par tout ce qu'il inclut, triés et
// dédoublonnés. Un fichier absent ou illisible rend une liste vide : une machine sans
// configuration SSH n'est pas une erreur.
func Lire(chemin string) []Hote {
	vus := map[string]string{} // nom -> HostName, première valeur gagnante
	lireDans(chemin, vus, map[string]bool{})

	noms := make([]string, 0, len(vus))
	for n := range vus {
		noms = append(noms, n)
	}
	sort.Slice(noms, func(i, j int) bool {
		return strings.ToLower(noms[i]) < strings.ToLower(noms[j])
	})

	out := make([]Hote, 0, len(noms))
	for _, n := range noms {
		out = append(out, Hote{Nom: n, HostName: vus[n]})
	}
	return out
}

// lireDans analyse un fichier et suit ses Include. `visites` porte les chemins déjà
// ouverts : une configuration fautive qui s'inclut elle-même ne doit pas figer le popup
// pendant la frappe.
func lireDans(chemin string, vus map[string]string, visites map[string]bool) {
	abs, err := filepath.Abs(chemin)
	if err != nil || visites[abs] {
		return
	}
	visites[abs] = true

	f, err := os.Open(abs)
	if err != nil {
		return
	}
	defer f.Close()

	base := filepath.Dir(abs)
	var courants []string // motifs du bloc Host en cours

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ligne := strings.TrimSpace(sc.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		// OpenSSH accepte « Host x », « Host=x » et est insensible à la casse.
		ligne = strings.ReplaceAll(ligne, "=", " ")
		champs := strings.Fields(ligne)
		if len(champs) < 2 {
			continue
		}
		switch strings.ToLower(champs[0]) {
		case "host":
			courants = nil
			for _, motif := range champs[1:] {
				if estMotif(motif) {
					continue
				}
				courants = append(courants, motif)
				if _, déjà := vus[motif]; !déjà {
					vus[motif] = ""
				}
			}
		case "hostname":
			for _, n := range courants {
				if vus[n] == "" {
					vus[n] = champs[1]
				}
			}
		case "include":
			for _, motif := range champs[1:] {
				for _, p := range résoudreInclude(motif, base) {
					lireDans(p, vus, visites)
				}
			}
		}
	}
}

// estMotif dit si un mot est un gabarit plutôt qu'un serveur. `Host *` est un bloc de
// défauts, pas une machine — le proposer dans le popup n'aurait aucun sens.
func estMotif(s string) bool {
	return strings.ContainsAny(s, "*?!")
}

// résoudreInclude rend les fichiers désignés. OpenSSH résout un chemin relatif depuis
// ~/.ssh/ pour la configuration utilisateur ; on prend le répertoire du fichier qui
// inclut, ce qui revient au même dans le cas ordinaire et reste juste pour les tests.
func résoudreInclude(motif, base string) []string {
	if strings.HasPrefix(motif, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			motif = filepath.Join(home, motif[2:])
		}
	}
	if !filepath.IsAbs(motif) {
		motif = filepath.Join(base, motif)
	}
	if trouvés, err := filepath.Glob(motif); err == nil && len(trouvés) > 0 {
		return trouvés
	}
	return []string{motif}
}
```

- [ ] **Step 4 : Lancer les tests, vérifier qu'ils passent**

Run : `go test ./internal/ssh/ -v`
Expected : PASS, 11 tests.

- [ ] **Step 5 : Commit**

```bash
git add internal/ssh/config.go internal/ssh/config_test.go
git commit -m "Sélecteur SSH : le parseur de ~/.ssh/config

Suit les Include, jokers compris, et se garde d'un cycle : une configuration
fautive ne doit pas figer le popup pendant la frappe.

Écarte les motifs — « Host * » est un bloc de défauts, pas une machine — et
dédoublonne en gardant la première valeur rencontrée, comme OpenSSH lui-même.

Un fichier absent rend une liste vide : une machine sans configuration SSH
n'est pas une erreur."
```

---

### Task 2 : Le fournisseur `pm.Manager`

**Files :**
- Create : `internal/ssh/manager.go`
- Test : `internal/ssh/manager_test.go`

**Interfaces :**
- Consumes : `Lire`, `Hote` (tâche 1) ; `pm.Catalog`, `pm.NewCatalog` du paquet `pm`.
- Produces :
  - `func New(cmd string) Manager` — `cmd` vaut `"ssh"`, `"scp"` ou `"sftp"`
  - `Manager` implémentant `pm.Manager` en entier

- [ ] **Step 1 : Écrire les tests qui échouent**

Créer `internal/ssh/manager_test.go` :

```go
package ssh

import (
	"path/filepath"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func TestManagerImplementePmManager(t *testing.T) {
	// La preuve tient dans l'affectation : si l'interface n'est pas satisfaite, ça ne
	// compile pas.
	var _ pm.Manager = New("ssh")
}

func TestPasDeSousCommandeNiOption(t *testing.T) {
	// C'est ce vide qui declenche la regle generale de completeWith (ADR-0005).
	m := New("ssh")
	if got := m.Subcommands(); len(got) != 0 {
		t.Errorf("Subcommands() = %v, attendu vide", got)
	}
	if got := m.Options(""); len(got) != 0 {
		t.Errorf("Options() = %v, attendu vide", got)
	}
	if m.InstalledOnly("") {
		t.Error("InstalledOnly() = true, attendu false")
	}
	if err := m.Warm(pm.ScopeAll); err != nil {
		t.Errorf("Warm() = %v, attendu nil", err)
	}
}

func TestCmdRendLeMotDemande(t *testing.T) {
	for _, c := range []string{"ssh", "scp", "sftp"} {
		if got := New(c).Cmd(); got != c {
			t.Errorf("Cmd() = %q, attendu %q", got, c)
		}
	}
}

func TestCatalogueDepuisUnFichier(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host pve\n    HostName 192.168.50.8\n\nHost solo\n")
	cat := catalogueDe(p)

	egal(t, cat.Names, []string{"pve", "solo"})
	// L'adresse voyage dans Versions : c'est le seul champ rendu en texte libre a
	// droite de la ligne. Le nom du champ ment, la spec dit pourquoi.
	if got := cat.Version("pve"); got != "192.168.50.8" {
		t.Errorf("Version(pve) = %q, attendu 192.168.50.8", got)
	}
	// Un hote sans HostName n'affiche rien a droite plutot que de repeter son nom.
	if got := cat.Version("solo"); got != "" {
		t.Errorf("Version(solo) = %q, attendu vide", got)
	}
	// Aucun badge : le glyphe par defaut est la puce, qui convient — un hote
	// n'appartient a aucune des deux classes de paquets.
	if got := cat.Badge("pve"); got != "" {
		t.Errorf("Badge(pve) = %q, attendu vide", got)
	}
}

func TestInsertColleUnDeuxPointsPourScp(t *testing.T) {
	// « scp fichier archlight /tmp » copierait vers un FICHIER LOCAL nomme archlight,
	// en ecrasant peut-etre quelque chose. Le deux-points fait partie du candidat.
	cat := pm.NewCatalog()
	if got := New("scp").Insert(cat, "", "", "archlight"); got != "archlight:" {
		t.Errorf("scp Insert = %q, attendu archlight:", got)
	}
	for _, c := range []string{"ssh", "sftp"} {
		if got := New(c).Insert(cat, "", "", "archlight"); got != "archlight" {
			t.Errorf("%s Insert = %q, attendu archlight", c, got)
		}
	}
}

func TestInsertNeDoublePasLeDeuxPoints(t *testing.T) {
	// L'utilisateur a deja tape « archlight: » et complete le chemin : ne pas en
	// remettre un.
	cat := pm.NewCatalog()
	if got := New("scp").Insert(cat, "", "", "archlight:"); got != "archlight:" {
		t.Errorf("Insert = %q, attendu archlight:", got)
	}
}

func TestAvailableSuitLExistenceDuFichier(t *testing.T) {
	d := t.TempDir()
	absent := filepath.Join(d, "rien")
	if disponible(absent) {
		t.Error("disponible() = true sur un fichier absent")
	}
	présent := ecrire(t, d, "config", "Host x\n")
	if !disponible(présent) {
		t.Error("disponible() = false sur un fichier présent")
	}
}
```

- [ ] **Step 2 : Lancer les tests, vérifier qu'ils échouent**

Run : `go test ./internal/ssh/`
Expected : FAIL — `undefined: New`, `undefined: catalogueDe`, `undefined: disponible`.

- [ ] **Step 3 : Écrire le fournisseur**

Créer `internal/ssh/manager.go` :

```go
package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Manager implémente pm.Manager pour ssh, scp et sftp.
//
// Il n'implémente PAS pm.Bindings, et c'est le sujet : ces trois commandes n'ont pas de
// verbes à exécuter, jigger se contente de compléter la ligne que l'utilisateur lancera
// lui-même. Voir l'ADR-0005.
type Manager struct{ cmd string }

// New rend le fournisseur pour l'une des trois commandes.
func New(cmd string) Manager { return Manager{cmd: cmd} }

func (m Manager) Cmd() string { return m.cmd }

// Aucune sous-commande : c'est ce vide qui fait proposer le catalogue dès le premier
// mot, par la règle générale de completeWith (ADR-0005). `ssh archlight` n'a pas de
// verbe entre la commande et l'hôte.
func (Manager) Subcommands() []string { return nil }

// Aucune option proposée. La spec l'écarte explicitement : -p, -i, -L et les autres se
// tapent rarement à la main, et les proposer allongerait la liste sans la servir.
func (Manager) Options(string) []string { return nil }

// Sans objet : un hôte n'est ni installé ni absent.
func (Manager) InstalledOnly(string) bool { return false }

// Available dit si la machine a une configuration SSH. Sur une machine qui n'en a pas,
// le fournisseur se tait plutôt que de proposer une liste vide.
func (Manager) Available() bool { return disponible(cheminConfig()) }

func (Manager) Load() *pm.Catalog { return catalogue() }

// Insert colle le deux-points qu'attend scp. `scp fichier archlight /tmp` copierait vers
// un fichier LOCAL nommé archlight, en écrasant peut-être quelque chose — l'erreur est
// silencieuse, d'où la correction ici.
func (m Manager) Insert(_ *pm.Catalog, _, _, name string) string {
	if m.cmd == "scp" && !strings.HasSuffix(name, ":") {
		return name + ":"
	}
	return name
}

// Warm ne fait rien. Lire quelques fragments de configuration coûte une milliseconde :
// il n'y a ni sortie machine à analyser, ni service distant à interroger, ni cache de
// 24 h à tenir. C'est le seul fournisseur de jigger dans ce cas.
func (Manager) Warm(pm.Scope) error { return nil }

func cheminConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

func disponible(chemin string) bool {
	if chemin == "" {
		return false
	}
	st, err := os.Stat(chemin)
	return err == nil && !st.IsDir()
}

// Le catalogue est mémorisé et réévalué quand le fichier change. Load() est sur le
// chemin du rendu : le relire à chaque frappe serait gaspiller, et le figer pour la
// session ferait mentir le popup après un `reseau-outil rendre`.
var (
	mémo      *pm.Catalog
	mémoQuand time.Time
	mémoMu    sync.Mutex
)

func catalogue() *pm.Catalog {
	chemin := cheminConfig()
	mémoMu.Lock()
	defer mémoMu.Unlock()

	st, err := os.Stat(chemin)
	if err != nil {
		return pm.NewCatalog()
	}
	if mémo != nil && st.ModTime().Equal(mémoQuand) {
		return mémo
	}
	mémo, mémoQuand = catalogueDe(chemin), st.ModTime()
	return mémo
}

// catalogueDe construit le catalogue d'un fichier donné. Séparé de catalogue() pour être
// testable sans toucher au ~/.ssh/config réel de la machine.
func catalogueDe(chemin string) *pm.Catalog {
	cat := pm.NewCatalog()
	for _, h := range Lire(chemin) {
		// Badge vide : glyphe() rend alors la puce « • », qui dit « n'appartient à
		// aucune des deux classes de paquets ». C'est exactement le cas d'un hôte.
		cat.Add(h.Nom, "")
		if h.HostName != "" && h.HostName != h.Nom {
			// L'adresse voyage dans Versions parce que c'est le SEUL champ rendu en
			// texte libre à droite de la ligne (internal/ui/frame.go). Le nom du champ
			// ment ; le renommer toucherait les trois gestionnaires, l'UI et les tests
			// de rendu pour un gain de vocabulaire. La spec du 30 août l'assume.
			cat.Versions[h.Nom] = h.HostName
		}
	}
	cat.Sort()
	return cat
}
```

- [ ] **Step 4 : Lancer les tests, vérifier qu'ils passent**

Run : `go test ./internal/ssh/ -v`
Expected : PASS, 18 tests au total pour le paquet.

- [ ] **Step 5 : Vérifier que rien d'autre n'est cassé**

Run : `go test ./...`
Expected : PASS.

- [ ] **Step 6 : Commit**

```bash
git add internal/ssh/manager.go internal/ssh/manager_test.go
git commit -m "Sélecteur SSH : le fournisseur, sans table de verbes

Implémente pm.Manager et jamais pm.Bindings : ces trois commandes n'ont pas
de verbes, jigger complète la ligne que l'utilisateur lancera lui-même.

Subcommands() rend nil, et c'est ce vide qui fera proposer le catalogue dès
le premier mot. Insert() colle le deux-points qu'attend scp — sans lui, la
commande copierait vers un fichier local du même nom, en écrasant peut-être
quelque chose.

L'adresse voyage dans Versions, seul champ rendu en texte libre à droite ;
le commentaire dit pourquoi ce nom ment."
```

---

### Task 3 : La règle générale dans `completeWith`

**Files :**
- Modify : `internal/complete/complete.go`
- Test : `internal/complete/complete_test.go` (ajouts seulement)

**Interfaces :**
- Consumes : `pm.Manager` (tâche 2).
- Produces : le comportement décrit par l'ADR-0005 — un fournisseur sans sous-commandes propose son catalogue dès le premier mot.

- [ ] **Step 1 : Écrire les tests qui échouent**

Ajouter à `internal/complete/complete_test.go` :

```go
// fauxManagerSansSub est un fournisseur de complétion sans verbes, comme ssh.
type fauxManagerSansSub struct{ cmd string }

func (f fauxManagerSansSub) Cmd() string                  { return f.cmd }
func (fauxManagerSansSub) Subcommands() []string          { return nil }
func (fauxManagerSansSub) Options(string) []string        { return nil }
func (fauxManagerSansSub) InstalledOnly(string) bool      { return false }
func (fauxManagerSansSub) Available() bool                { return true }
func (fauxManagerSansSub) Load() *pm.Catalog              { return nil }
func (fauxManagerSansSub) Warm(pm.Scope) error            { return nil }
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
```

- [ ] **Step 2 : Lancer les tests, vérifier qu'ils échouent**

Run : `go test ./internal/complete/ -run SansSousCommande -v`
Expected : FAIL — la branche `firstWord` propose `Subcommands()`, donc `Items` est vide.

- [ ] **Step 3 : Poser la règle**

Dans `internal/complete/complete.go`, fonction `completeWith`, remplacer :

```go
	isOption := strings.HasPrefix(word, "-")
	isPackage := !isOption && !firstWord
```

par :

```go
	// Un fournisseur qui ne déclare aucune sous-commande n'a pas de verbe entre la
	// commande et son opérande : `ssh archlight` met l'hôte là où `brew install` met un
	// verbe. Le catalogue vient donc dès le premier mot (ADR-0005).
	subs := m.Subcommands()
	sansVerbes := len(subs) == 0

	isOption := strings.HasPrefix(word, "-")
	isPackage := !isOption && (!firstWord || sansVerbes)
```

puis, dans le `switch`, remplacer :

```go
	case firstWord:
		for _, s := range m.Subcommands() {
```

par :

```go
	case firstWord && !sansVerbes:
		for _, s := range subs {
```

Et `Executable: isPackage` devient :

```go
		// Un fournisseur sans verbes n'a pas de pm.Bindings : rien à exécuter. Le
		// sélecteur plein écran doit insérer, pas lancer.
		Executable: isPackage && !sansVerbes,
```

- [ ] **Step 4 : Lancer les tests, vérifier qu'ils passent**

Run : `go test ./internal/complete/ -v`
Expected : PASS — les 4 nouveaux **et les 18 existants**, aucun modifié.

- [ ] **Step 5 : Prouver que brew, winget et scoop ne changent pas**

Vérifier que les trois déclarent bien des sous-commandes, ce qui les met hors du chemin de la nouvelle règle :

```bash
grep -c '"' internal/brew/manager.go internal/winget/winget.go internal/scoop/scoop.go >/dev/null
go test ./internal/complete/ ./internal/brew/ ./internal/winget/ ./internal/scoop/
```

Expected : PASS. Consigner dans le rapport le nombre de sous-commandes de chacun — 24, 17 et 27 au 30 août.

- [ ] **Step 6 : Commit**

```bash
git add internal/complete/complete.go internal/complete/complete_test.go
git commit -m "Complétion : un fournisseur sans verbes propose son catalogue tout de suite

completeWith traitait le mot suivant la commande comme une sous-commande, et
ne passait au catalogue qu'ensuite — la grammaire de « brew install firefox ».
« ssh archlight » n'a pas de verbe : l'hôte est en deuxième position.

La règle posée n'est pas un cas particulier : « pas de sous-commande » a
toujours voulu dire « l'opérande commence tout de suite ». brew, winget et
scoop en déclarent 24, 17 et 27 — aucun ne change de comportement, et leurs
tests inchangés le prouvent.

Un tel fournisseur n'est jamais Executable : sans pm.Bindings, il n'y a rien
à lancer depuis le sélecteur plein écran."
```

---

### Task 4 : Enregistrement et greffons

**Files :**
- Modify : `internal/managers/managers.go`
- Modify : `shell/jigger.plugin.zsh`
- Modify : `shell/jigger.psm1`
- Modify : `README.md`, `README.fr.md` (le défaut documenté)
- Test : `internal/managers/managers_test.go` (créer s'il n'existe pas)

**Interfaces :**
- Consumes : `ssh.New` (tâche 2).
- Produces : les trois fournisseurs visibles de `managers.All()`, et les trois mots reconnus par les greffons.

- [ ] **Step 1 : Écrire le test qui échoue**

Créer ou compléter `internal/managers/managers_test.go` :

```go
package managers

import "testing"

func TestAllContientLesTroisCommandesSSH(t *testing.T) {
	vus := map[string]bool{}
	for _, m := range All() {
		vus[m.Cmd()] = true
	}
	for _, c := range []string{"ssh", "scp", "sftp"} {
		if !vus[c] {
			t.Errorf("%q absent de All()", c)
		}
	}
	// Les trois gestionnaires de paquets restent la.
	for _, c := range []string{"brew", "winget", "scoop"} {
		if !vus[c] {
			t.Errorf("%q disparu de All()", c)
		}
	}
}

func TestLesFournisseursSSHNeDeclarentAucunVerbe(t *testing.T) {
	for _, m := range All() {
		switch m.Cmd() {
		case "ssh", "scp", "sftp":
			if len(m.Subcommands()) != 0 {
				t.Errorf("%q déclare des sous-commandes", m.Cmd())
			}
		}
	}
}
```

- [ ] **Step 2 : Lancer le test, vérifier qu'il échoue**

Run : `go test ./internal/managers/ -v`
Expected : FAIL — `ssh`, `scp` et `sftp` absents.

- [ ] **Step 3 : Les enregistrer**

Dans `internal/managers/managers.go`, ajouter l'import et étendre `All()` :

```go
	"gitlab.yg-devworks.com/yves/jigger/internal/ssh"
```

```go
func All() []pm.Manager {
	return []pm.Manager{
		brew.New(), winget.New(), scoop.New(),
		// Trois fournisseurs plutôt qu'un à trois noms : Manager.Cmd() ne rend qu'un
		// mot, et l'élargir obligerait brew, winget et scoop à répondre à une question
		// qu'ils ne se posent pas. Ils partagent implémentation et catalogue.
		ssh.New("ssh"), ssh.New("scp"), ssh.New("sftp"),
	}
}
```

- [ ] **Step 4 : Lancer les tests**

Run : `go test ./...`
Expected : PASS.

- [ ] **Step 5 : Le greffon zsh**

Dans `shell/jigger.plugin.zsh`, ligne 169 :

```zsh
typeset -ga _jigger_commands=( brew jigger jg )
```

devient :

```zsh
typeset -ga _jigger_commands=( brew jigger jg ssh scp sftp )
```

Rien d'autre ne change côté zsh : la popup, les flèches, le rattrapage des frappes et la fermeture sur ⏎ ne connaissent pas la nature de ce qu'ils affichent.

- [ ] **Step 6 : Le greffon PowerShell — et une décision qui n'est pas la même**

Côté PowerShell, la liste n'est **pas** codée en dur : c'est un réglage utilisateur documenté.

```powershell
$script:Commands = @((Get-JiggerSetting 'JIGGER_COMMANDS' 'winget,scoop') -split '[,\s]+' | ...)
```

Le projet a déjà tranché un cas voisin, et sa leçon est écrite dans `docs/ameliorations.md` :
étendre le défaut à `winget,scoop,jigger,jg` **n'aurait rien changé** pour qui a recopié
`winget,scoop` dans son profil — ce que la documentation a montré pendant trois versions.
La façade arme donc le popup **toujours**, hors du réglage.

**Ce cas-ci se décide autrement, et il faut dire pourquoi.** `jigger` et `jg` sont les
commandes *de jigger* : les armer toujours va de soi, et les éteindre serait un défaut.
`ssh` est une commande **tierce**, que l'utilisateur peut légitimement ne pas vouloir voir
interceptée — c'est justement à quoi sert `JIGGER_COMMANDS`. L'armer de force lui retirerait
un choix qu'il avait.

Donc : **ajouter `ssh`, `scp` et `sftp` au défaut**, pas à la liste toujours-armée.

```powershell
Get-JiggerSetting 'JIGGER_COMMANDS' 'winget,scoop,ssh,scp,sftp'
```

**Conséquence assumée, à consigner dans le CHANGELOG** : qui a épinglé `winget,scoop` dans
son profil ne verra pas le sélecteur SSH tant qu'il n'aura pas allongé son réglage. Ce
n'est pas le défaut silencieux que la leçon dénonçait — c'est le comportement voulu d'une
liste blanche que l'utilisateur a explicitement posée.

Mettre à jour les deux README (`README.md` et `README.fr.md`), qui montrent le défaut
`'winget,scoop'` à la ligne 162-163.

`ssh` existe sous Windows depuis OpenSSH intégré, et `~/.ssh/config` s'y lit pareil : le
fournisseur fonctionne des deux côtés sans code spécifique.

**Note sur l'asymétrie, qui préexiste :** côté zsh, `_jigger_commands` est codé en dur et
n'offre aucun réglage — un utilisateur macOS n'a déjà pas le choix pour `brew`. Ce plan ne
la corrige pas ; il la constate. Si elle doit être réglée, c'est une entrée à part.

- [ ] **Step 7 : Commit**

```bash
git add internal/managers/ shell/jigger.plugin.zsh shell/jigger.psm1
git commit -m "Sélecteur SSH : enregistré, et reconnu par les deux greffons

Trois fournisseurs plutôt qu'un à trois noms : Cmd() ne rend qu'un mot, et
élargir l'interface obligerait brew, winget et scoop à répondre à une
question qu'ils ne se posent pas.

Côté zsh, un mot dans la liste codée en dur. Côté PowerShell, trois mots dans
le DÉFAUT de JIGGER_COMMANDS et non dans la liste toujours-armée : la leçon
d'A-20 valait pour la façade, commande de jigger qu'il aurait été fautif
d'éteindre. ssh est une commande tierce, et JIGGER_COMMANDS existe justement
pour que l'utilisateur choisisse ce qui l'intercepte."
```

---

### Task 5 : Vérification de bout en bout

**Files :**
- Modify : `docs/ameliorations.md` (A-25 descend en « Fait »)
- Create : `docs/historique/2026-08-30.md`

**Interfaces :**
- Consumes : tout ce qui précède.
- Produces : rien que le code consomme.

- [ ] **Step 1 : Le rendu, sans shell**

`jigger render` dessine le cadre seul — c'est ce que le CONTRIBUTING demande pour tout rapport sur la popup :

```bash
go build -o /tmp/jigger-ssh . && /tmp/jigger-ssh render --line "ssh arch" --cols 80
```

Expected : un cadre contenant les hôtes dont le nom commence par `arch`, chacun avec son adresse à droite. Consigner la sortie dans le rapport.

- [ ] **Step 2 : La complétion, sur les trois commandes**

```bash
for c in ssh scp sftp; do echo "--- $c"; /tmp/jigger-ssh complete --line "$c " ; done
```

Expected : les mêmes hôtes pour les trois. Vérifier que `scp` insère bien `hôte:` — visible dans la sortie de `complete` si elle rend l'insertion, sinon par le test unitaire de la tâche 2.

- [ ] **Step 3 : Vérifier qu'aucun gestionnaire n'a changé**

```bash
/tmp/jigger-ssh complete --line "brew ins"
/tmp/jigger-ssh complete --line "brew install fire"
```

Expected : les sous-commandes pour la première, les formules pour la seconde — exactement comme avant.

- [ ] **Step 4 : Dans un vrai shell**

Recharger le greffon et taper `ssh arch` sans valider :

```bash
exec zsh
# puis, à la main : taper « ssh arch » et constater la popup
```

Expected : la popup apparaît sous le prompt et suit la frappe. ⇥ insère l'hôte. ⎋ ferme sans rien insérer.

**C'est la seule étape qui ne peut pas être automatisée**, et la seule qui prouve que le greffon fonctionne. Consigner ce qui a été constaté.

- [ ] **Step 5 : La suite complète**

Run : `make test`
Expected : PASS, aucun test existant modifié.

- [ ] **Step 6 : A-25 descend en « Fait »**

Déplacer l'entrée `A-25` de la section « À faire » vers « Fait » dans `docs/ameliorations.md`, en y ajoutant la date et le commit, comme le fait le fichier pour les entrées réalisées.

- [ ] **Step 7 : Le journal**

Créer `docs/historique/2026-08-30.md` sur le modèle des journaux existants : ce qui a été demandé, ce qui a été fait, ce qui a été décidé, et ce qui reste ouvert.

- [ ] **Step 8 : Commit et merge request**

```bash
git add docs/
git commit -m "Sélecteur SSH : A-25 réalisée, et le journal du 30 août"
git push -u origin feat/selecteur-ssh
```

Puis ouvrir la merge request sur GitLab, comme le demande le CONTRIBUTING.

---

## Self-review

**Couverture de la spec :**

| Section de la spec | Tâche |
|---|---|
| § 1 — ce que ce fournisseur n'est pas (ni Bindings, ni verbes) | 2 |
| § 2 — la position de l'opérande | 3 |
| § 3 — trois commandes, trois fournisseurs | 2 (Insert), 4 (enregistrement) |
| § 3 — le deux-points de `scp` | 2 |
| § 4 — le catalogue, `Include`, motifs écartés | 1 |
| § 4 — pourquoi `Versions` porte une adresse | 2 |
| § 4 — pas de réchauffement, `Available()` | 2 |
| § 5 — les greffons | 4 |
| § 6 — les tests | chaque tâche |
| § 7 — ce que la spec ne fait pas | respecté : ni options, ni rsync, ni known_hosts, ni sonde réseau |

**Cohérence des noms** — vérifiée d'un bout à l'autre : `Hote{Nom, HostName}`, `Lire(chemin)`, `New(cmd)`, `catalogueDe(chemin)`, `disponible(chemin)`, `catalogue()`, `estMotif`, `résoudreInclude`.

**Points où le plan peut buter :**

- **`pm.Catalog.Versions` : vérifié.** Le champ est exporté et initialisé par `NewCatalog()` ; l'affectation directe fonctionne. Et il ne faut **pas** passer par `MarkInstalled`, qui poserait `Installed[nom] = true` — donc une pastille ● sur chaque hôte, alors qu'un serveur n'est ni installé ni absent.
- **La liste PowerShell : localisée et traitée** (`shell/jigger.psm1:61`). Elle a fait changer le plan — voir la tâche 4, étape 6.
- **`complete --line` rend-il l'insertion ?** Non vérifié. Si la sous-commande n'expose pas ce que `Insert()` produirait, l'étape 2 de la tâche 5 se limite aux tests unitaires — ce qui suffit, mais autant le savoir avant.
- **L'étape 4 de la tâche 5 n'est pas automatisable.** Constater la popup dans un vrai zsh demande une frappe humaine ; c'est pourtant la seule preuve que le greffon marche. Un exécuteur en sous-agent doit rendre la main plutôt que de la simuler.
