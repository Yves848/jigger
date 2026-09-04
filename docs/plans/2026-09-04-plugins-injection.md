# Plan — Injection de plugins tiers

**Date :** 2026-09-04

---

## 1. Contexte et objectif

L'ADR-0001 a acté que l'extensibilité par des tiers **sans recompilation** demanderait un
protocole de sous-processus (des binaires `jigger-<pm>` dialoguant en JSON). Cet document
définit le design de cette extension.

**Objectif :** permettre à n'importe qui d'écrire un gestionnaire de paquets personnalisé
sans toucher au code source de jigger, en livrant un seul binaire et quelques fichiers
de configuration.

---

## 2. Contraintes héritées

1. **Règle d'or** : rien de lent dans le chemin du rendu (`jigger render` tourne à chaque
   frappe). Un plugin ne peut pas être un sous-processus appelé par `render`.
2. **Modularité actuelle** : les gestionnaires sont des paquets Go compilés statiquement.
3. **Multi-gestionnaires** : la façade résout une ligne entre plusieurs gestionnaires — le
   système de plugins doit s'intégrer dans ce modèle.
4. **Sécurité** : un plugin tiers ne doit pas pouvoir exécuter de commande arbitraire.

---

## 3. Architecture proposée

### 3.1 Principe : deux modes, deux chemins

```
┌──────────────────────────────────────────────┐
│              jigger (binaire principal)       │
│                                              │
│  ┌─ Mode compilé (natif) ─────────────────┐  │
│  │ internal/brew/, internal/winget/, …   │  │
│  │ Go statique, accès direct aux structs  │  │
│  └────────────────────────────────────────┘  │
│                                              │
│  ┌─ Mode plugin (dynamique) ──────────────┐  │
│  │ jigger-<nom>/                          │  │
│  │   ├── jigger-mespa      (binaire)       │  │
│  │   └── config.json     (descripteur)    │  │
│  └────────────────────────────────────────┘  │
│                                              │
│  ┌─ Orchestrateur de plugins ─────────────┐  │
│  │ internal/plugin/                       │  │
│  │   discovery.go                         │  │
│  │   protocol.go                          │  │
│  │   cache.go                             │  │
│  └────────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

**Mode natif** : inchangé. Les gestionnaires internes restent des paquets Go.
**Mode plugin** : un sous-processus externe appelé uniquement par `jigger warm` et
par les commandes d'exécution (`jg install …`). Il n'intervient **jamais** dans le
chemin de `render`.

### 3.2 Le descripteur de plugin (`config.json`)

Chaque plugin est un dossier contenant un binaire et un descripteur :

```json
{
  "name": "mespa",
  "version": "1.0.0",
  "cmd": "jigger-mespa",
  "platforms": ["darwin", "linux", "windows"],
  "verbs": {
    "install":  {"native": ["install", "{arg}"],     "pool": "catalogue"},
    "search":   {"native": ["search", "{arg}"],      "pool": "catalogue"},
    "upgrade":  {"native": ["upgrade", "--all"],     "pool": "aucun"},
    "list":     {"build": "BuildList",               "pool": "catalogue"}
  },
  "warmup": {
    "catalog":  {"cmd": "jigger-mespa", "args": ["catalog", "--json"]},
    "installed":{"cmd": "jigger-mespa", "args": ["list", "--installed", "--json"]}
  },
  "parse": {
    "package_fields": ["name", "version", "kind", "source"],
    "encoding": "utf-8"
  }
}
```

Ce descripteur remplace `verbs.go` d'un gestionnaire natif. Il dit à jigger :
- quels verbes le plugin supporte (`verbs`)
- comment construire l'argv pour chaque verbe (`native` ou `build`)
- comment réchauffer les caches (`warmup`)
- comment parser la sortie JSON (`parse`)

### 3.3 Le protocole de communication (JSON)

Le binaire du plugin communique avec jigger via **stdin/stdout** en JSON.
Pas d'IPC, pas de socket — juste un processus qui se lève, produit du JSON, et meurt.

#### Réchauffement (warm)

```bash
# jigger lance ça pendant `jigger warm`
jigger-mespa catalog --json     →  {"names":["foo","bar"],"badges":{"foo":"X"}}
jigger-mespa list --installed   →  [{"name":"foo","version":"1.2.3","kind":"X"}]
```

Sortie : un seul document JSON sur stdout, exit 0 en cas de succès.
L'erreur se signale par un code de sortie non nul + un message d'erreur sur stderr.

#### Exécution (façade)

Quand la façade doit exécuter une cible sur un plugin :

```bash
jigger-mespa run install foo bar --json   →  {"stdout":"...\n","code":0}
jigger-mespa run upgrade --all            →  {"stdout":"...\n","code":0}
```

Sortie : le stdout brut du gestionnaire (pour que `facade.Formater` s'applique).

#### Complétion (complete) — **optionnel, pas recommandé**

Les plugins *peuvent* exposer un endpoint de complétion, mais jigger doit ignorer
silencieusement un plugin dont le binaire est lent à lancer. Si le délai dépasse
50 ms, jigger retourne une liste vide et lance `warm` en tâche de fond.

```bash
jigger-mespa complete "ins" --json   →  ["install","inspect"]
```

### 3.4 Discovery (`internal/plugin/discovery.go`)

Pendant `managers.Available()` (appelé par `warm`), jigger parcourt :

```
$XDG_CACHE_HOME/jigger/plugins/     (plugins installés)
~/.config/jigger/plugins/           (plugins utilisateur, non versionnés)
/usr/local/lib/jigger-plugins/      (plugins système, lecture seule)
```

Chaque sous-dossier contenant un `config.json` valide est un plugin candidat.
`Available()` teste si le binaire existe et est exécutable ; sinon il passe.

### 3.5 Cache des plugins

Les plugins utilisent **le même système de cache** que les gestionnaires natifs :
`CacheDir()/mespa-catalog` et `CacheDir()/mespa-installed`.
Le fichier `.stamp` et le verrou `warm.lock` sont partagés — un plugin ne peut pas
bloquer le réchauffement d'un autre.

---

## 4. Intégration avec la façade

### 4.1 Nouveau contrat : `pm.PluginManager`

```go
// PluginManager est implémenté par l'orchestrateur de plugins, pas par le plugin lui-même.
// Il expose la même interface que pm.Manager pour la complétion et le popup.
type PluginManager struct {
    name     string
    config   PluginConfig
    cacheDir string
}

func (m *PluginManager) Cmd() string                          { … }
func (m *PluginManager) Subcommands() []string                  { … }
func (m *PluginManager) Options(sub string) []string            { … }
func (m *PluginManager) InstalledOnly(sub string) bool          { … }
func (m *PluginManager) Available() bool                        { … }
func (m *PluginManager) Load() *Catalog                         { … }  // lit le cache
func (m *PluginManager) Insert(cat *Catalog, sub, prefix, name string) string { … }
func (m *PluginManager) Warm(scope Scope) error                 { … }  // lance le sous-processus
```

### 4.2 Intégration dans `managers.All()`

```go
func All() []pm.Manager {
    var out []pm.Manager
    // Gestionnaires natifs (inchangés)
    out = append(out, brew.New(), winget.New(), scoop.New(),
        pacman.New("pacman"), pacman.New("yay"), ssh.New("ssh"), …)

    // Plugins tiers (découverts dynamiquement)
    for _, p := range plugin.Discover() {
        out = append(out, p)
    }
    return out
}
```

### 4.3 Exécution via la façade

`facade.ExecuterAvec` doit distinguer les cibles natives des cibles plugins :

```go
func ExecuterAverb(verbe Verb, cibles []Cible, opts Opts) Result {
    for _, cible := range cibles {
        if pm, ok := cible.Mgr.(*PluginManager); ok {
            // Lance le sous-processus plugin
            res := lancePlugin(pm, verbe, cible.Args)
            rows = append(rows, res.Rows…)
        } else {
            // Chemin natif (inchangé)
            res := execNatif(cible.Mgr, verbe, cible.Args)
            rows = append(rows, res.Rows…)
        }
    }
    return Result{Rows: rows, Code: …}
}
```

---

## 5. Sécurité

| Menace | Protection |
|---|---|
| Exécution arbitraire de commandes | Seul le binaire `config.json` pointe est autorisé ; pas de `$PATH` |
| Injection via noms de paquets | Le plugin produit du JSON structuré, pas du shell |
| Plugin malveillant qui vole des données | Le plugin ne voit **que** les noms qu'on lui passe ; il n'a pas accès au terminal ni aux variables d'environnement sensibles (sauf `$PATH` pour trouver son propre binaire) |
| Déni de service via binaire bloquant | Timeout 30 s par sous-processus ; le plugin est lancé détaché |

---

## 6. Installation d'un plugin par l'utilisateur

```bash
# 1. Placer le binaire dans $PATH
cp /tmp/jigger-mespa /usr/local/bin/

# 2. Placer le descripteur
mkdir -p ~/.config/jigger/plugins/mespa
cat > ~/.config/jigger/plugins/mespa/config.json <<'EOF'
{ … }
EOF

# 3. Réchauffer les caches
jigger warm --all
```

Pas de `jigger install plugin` — c'est trop d'hypothèses sur la distribution.
L'utilisateur met le binaire où il veut, tant que `config.json` pointe le bon chemin.

---

## 7. Ordre des phases de développement

| Phase | Livrable | Dépendances |
|---|---|---|
| P0 | `internal/plugin/discovery.go` + `PluginManager` (squelette) | — |
| P1 | Protocole JSON : `catalog`, `list --installed`, `run <verbe> <args>` | P0 |
| P2 | Intégration dans `managers.All()` et `facade.ExecuterAverb` | P1 |
| P3 | Documentation + exemple de plugin (ex: `jigger-npm`) | P2 |
| P4 | Complétion en temps réel (optionnel, risqué pour la latence) | P3 |

---

## 8. Schéma global

```mermaid
flowchart TD
    subgraph Utilisateur
        U[utilisateur]
    end

    subgraph jigger[binaire principal — Go statique]
        subgraph Natifs[Gestionnaires natifs]
            B[brew/]
            W[winget/]
            S[scoop/]
            P[pacman/yay/]
            SSH[ssh/]
        end

        subgraph Plugins[Orchestrateur de plugins — internal/plugin/]
            D[discovery.go]
            PM[PluginManager]
            Prot[protocol.go — JSON stdin/stdout]
            Cache[cache.go — même système que pm]
        end

        F[facade/ — routage + exécution]
        UI[ui/ — popup]
        Comp[complete/ — complétion]
    end

    subgraph PluginsExternaux[plugins tiers — binaires externes]
        JP1[jigger-mespa\n+ config.json]
        JP2[jigger-autre\n+ config.json]
    end

    U -->|jg install fd Git.Git| F
    F -->|résolu: natif| B
    F -->|résolu: plugin| PM
    PM -->|lance sous-processus| Prot
    Prot -->|stdin/stdout JSON| JP1
    Prot -->|stdin/stdout JSON| JP2
    PM -->|lit cache| Cache
    Comp -->|Load() depuis cache| Cache
```

---

## 9. Décisions à prendre

| Question | Options | Recommandation |
|---|---|---|
| Le plugin parle-t-il en JSON brut ou en lignes (comme les natifs) ? | JSON structuré / lignes brutes | **JSON** — permet de typer `kind`, `source`, etc. sans parsing heuristique |
| La complétion passe-t-elle par le plugin ou est-elle gérée côté jigger ? | Plugin complet / jigger avec cache uniquement | **Cache uniquement** — la règle d'or §2 interdit un sous-processus dans le chemin de `render` |
| Où placer les plugins ? | `$PATH` + descripteur manuel / `jigger install plugin` / registre central | **Descripteur manuel** — zéro hypothèse sur la distribution |
| Le plugin peut-il modifier la ligne avant exécution (`Insert`) ? | Oui (callback) / Non | **Non** — un plugin tiers ne doit pas avoir ce pouvoir ; les corrections sont des bugs de façade |

---

## 10. Exemple concret : `jigger-npm`

Un développeur frontend veut compléter `npm install` via jigger :

```json
// ~/.config/jigger/plugins/npm/config.json
{
  "name": "npm",
  "version": "1.0.0",
  "cmd": "jigger-npm",
  "platforms": ["darwin", "linux", "windows"],
  "verbs": {
    "install":  {"native": ["install", "{arg}"],     "pool": "catalogue"},
    "uninstall":{"native": ["uninstall", "{arg}"],   "pool": "installes"},
    "list":     {"native": ["list", "--json"],       "pool": "catalogue"}
  },
  "warmup": {
    "catalog":  {"cmd": "jigger-npm", "args": ["catalog", "--json"]},
    "installed":{"cmd": "jigger-npm", "args": ["list", "--installed", "--json"]}
  }
}
```

Binaire `jigger-npm` produit :

```bash
$ jigger-npm catalog --json
{"names":["express","react","vue","typescript"],
 "badges":{"express":"N","react":"N"}}

$ jigger-npm list --installed --json
[{"name":"express","version":"4.21.0","kind":"N"},
 {"name":"react","version":"19.0.0","kind":"N"}]

$ jigger-npm run install express --json
{"stdout":"added 1 package in 2s\n","code":0}
```

---

## 11. Impact sur le code existant

| Fichier | Changement |
|---|---|
| `internal/managers/managers.go` | Appel à `plugin.Discover()` après les natifs |
| `internal/pm/pm.go` | Nouvelle interface `PluginManager` (ou paquet séparé) |
| `internal/facade/executer.go` | Distinction natif/plugin dans `ExecuterAverb` |
| `main.go` | Aucun — la CLI ne change pas |
| `shell/jigger.plugin.zsh` | Aucun — le widget appelle `jigger render` qui ne sait pas s'il consulte un plugin |
| `shell/jigger.psm1` | Idem |

Zéro changement requis dans les greffons shell : tout vit à l'intérieur du binaire.

---

## 12. Risques et atténuations

| Risque | Gravité | Atténuation |
|---|---|---|
| Un plugin lent bloque `warm` | Moyenne | Timeout 30 s, verrou partagé, pas de blocage en chaîne |
| Descriptions de plugins malformées | Faible | Validation stricte du JSON au discovery ; skip silencieux |
| Conflit de nom entre plugin et gestionnaire natif | Faible | Les natifs ont priorité dans `managers.All()` ; warning en stderr |
| Binaire plugin non trouvé après update | Moyenne | `Available()` retourne false si le binaire a disparu |
