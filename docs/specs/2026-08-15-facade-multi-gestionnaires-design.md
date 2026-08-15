# Façade multi-gestionnaires — conception

15 août 2026 — état : validé, prêt pour le plan d'implémentation

## Objet

Donner à jigger une **syntaxe unique** au-dessus de Homebrew, winget et scoop :
`jg install fd` plutôt que `brew install fd` ou `scoop install fd`, avec une sortie
uniforme quel que soit le gestionnaire qui répond.

Ce document couvre la phase 1. Il ne remplace pas [Description.md](../Description.md),
qui porte la vision d'ensemble.

## Point de départ

jigger est en v0.7.0 : ~5 000 lignes de Go, les trois gestionnaires de la phase 1
implémentés, un popup de complétion branché dans zsh et PowerShell, un bloc oh-my-posh.
L'interface `pm.Manager` et le registre `internal/managers` fournissent déjà la
modularité par gestionnaire.

Ce que jigger fait aujourd'hui est **passif** : il observe la ligne qu'on tape et la
complète. Il n'émet jamais de commande. La façade ajoute une nature d'**action** à côté
de cette nature d'**observation**.

## Décisions structurantes

Quatre choix arrêtés, dont tout le reste découle.

1. **Exécuter et normaliser.** jigger lance le gestionnaire et uniformise sa sortie —
   plutôt que de se contenter de réécrire la ligne du shell ou de relayer la sortie brute
   sans la retoucher.
2. **Résolution par le nom, partout.** jigger cherche le paquet nommé dans tous les
   gestionnaires disponibles. Un seul le connaît, il gagne ; plusieurs, l'utilisateur
   tranche.
3. **Verbes propres exposés, capacités déclarées.** Les verbes qu'un seul gestionnaire
   sait rendre passent quand même par la façade, qui sait dire qui peut quoi.
4. **Nommage : canonique quand ça converge, natif sinon.**

## §1 — Architecture

### Frontières

```text
main.go                    aiguillage CLI : mot réservé ou verbe de façade ?
   │
   ├── internal/facade     NOUVEAU — le moteur
   │      résout le verbe, route par le nom, construit l'argv,
   │      exécute, agrège, formate
   │
   ├── internal/complete   existant — étendu pour connaître le vocabulaire de la façade
   │
   └── internal/pm         existant — le contrat
          pm.go       Manager : répondre à des questions      (inchangé)
          verbs.go    NOUVEAU — Bindings : savoir agir
          package.go  NOUVEAU — Package : la ligne normalisée
```

Chaque gestionnaire gagne deux fichiers, sans qu'aucun des siens ne bouge :
`internal/{brew,winget,scoop}/verbs.go` (la table) et `parse.go` (les parsers).

### Pourquoi `pm.Manager` n'est pas élargi

`pm.Manager` se définit ainsi : *« Une implémentation ne fait que répondre à des
questions : c'est `complete` qui décide quoi demander, et `ui` comment l'afficher. »*
C'est une frontière voulue — **le contrat actuel n'agit jamais**.

Y greffer `Install()` mélangerait deux natures et rendrait `Manager` inutilisable comme
contrat de complétion seule. Le contrat d'exécution est donc un **second contrat**,
`Bindings`, indépendant du premier. Un gestionnaire implémente les deux ; rien ne l'y
oblige structurellement.

Les deux vivent dans le paquet `pm` — qui se définit comme « le contrat que jigger attend
d'un gestionnaire de paquets » — mais dans des **fichiers séparés** : `pm.go` fait déjà
341 lignes, et les deux contrats ne se lisent pas ensemble.

### Aiguillage dans `main.go`

**Six mots réservés** : `pick`, `render`, `complete`, `prompt`, `warm`, `demo` (plus
`version` / `--version` / `-v`). Tout autre premier mot est un verbe de façade.

Aucun n'est un verbe de gestionnaire de paquets, donc zéro collision aujourd'hui. Cela
devient une **contrainte permanente** : *aucune sous-commande interne future ne peut
porter le nom d'un verbe canonique.* Si `jigger list` devait un jour désigner un usage
interne, c'est le mot interne qui change.

### Alias `jg`

Posé par les deux greffons — `alias jg=jigger` sous zsh, `Set-Alias jg jigger` sous
PowerShell — et ajouté à `motCommande()` pour que le popup le reconnaisse comme jigger.

Un alias shell ne suit pas dans un script. C'est accepté en phase 1 : la façade s'utilise
au clavier. Le jour où elle doit servir dans un script, `make install` posera un lien
symbolique.

## §2 — Le vocabulaire

### Règle de nommage

1. **Deux gestionnaires ou plus couvrent le concept** → un nom canonique unique, chaque
   table traduit.
2. **Un seul le couvre** → son nom natif, tel quel, sans habillage.
3. **Choix du nom canonique** : la majorité des noms natifs l'emporte ; à égalité, le plus
   largement connu. On n'invente un mot que si aucun natif ne décrit le concept — et la
   table dit alors pourquoi.

### Verbes composés

La clé de table est le **membre de phrase entier** : `"source add"`, pas `source` avec un
sous-argument. C'est ce qui garde les gabarits d'argv triviaux — sans quoi brew
(`tap` / `untap`, deux mots sans rapport) exigerait aussitôt l'échappatoire `Build`.

Bénéfice secondaire : `jg source ⇥` complète `add` et `rm` dans le popup sans une ligne de
code de plus, les clés de table *étant* le vocabulaire.

### Table de correspondance — phase 1

**Universels** — les trois savent faire :

| Verbe jigger | brew | winget | scoop |
|---|---|---|---|
| `install {pkgs}` | `install {pkgs}` ¹ | `install --id {pkg} --exact` | `install {pkgs}` |
| `uninstall {pkgs}` | `uninstall {pkgs}` | `uninstall --id {pkg} --exact` | `uninstall {pkgs}` |
| `upgrade [pkgs]` | `upgrade [pkgs]` | `upgrade --id {pkg}` / `--all` | `update {pkgs}` / `update *` |
| `list` | `list --versions` | `list` | `list` |
| `outdated` | `outdated --json=v2` | `list --upgrade-available` | *(`Direct`, cf. §4)* |
| `search {q}` | `search {q}` | `search {q}` | `search {q}` |
| `info {pkg}` | `info {pkg}` | `show --id {pkg}` | `info {pkg}` |

¹ le `--cask` de `brew.Manager.Insert` se rebranche ici : la logique existe, elle change
d'appelant.

**Convergents** — un concept, des noms différents, pas toujours les trois :

| Verbe jigger | brew | winget | scoop | Pourquoi ce nom |
|---|---|---|---|---|
| `source` | `tap` | `source list` | `bucket list` | seul nom compréhensible sans connaître l'outil |
| `source add {arg}` | `tap {arg}` | `source add {arg}` | `bucket add {arg}` | |
| `source rm {arg}` | `untap {arg}` | `source remove {arg}` | `bucket rm {arg}` | |
| `pin {pkg}` | `pin {pkg}` | `pin add --id {pkg}` | `hold {pkg}` | majorité (brew + winget) |
| `unpin {pkg}` | `unpin {pkg}` | `pin remove --id {pkg}` | `unhold {pkg}` | idem |
| `cleanup` | `cleanup` | — | `cleanup *` | 2/3 ; winget n'a pas le concept |
| `doctor` | `doctor` | — | `checkup` | égalité ; `doctor` est le plus connu |

`cleanup` et `doctor` **exercent le modèle de capacités dès la phase 1** : sous Windows
sans scoop, `jg doctor` doit dire proprement qu'aucun gestionnaire disponible ne sait
faire ça. Ce chemin d'erreur est couvert par les tests.

**Singuliers** — `brew services`, `brew leaves`, `winget export`, `scoop reset`… Le
mécanisme les accepte (une ligne de table chacun) ; **aucun n'entre dans la phase 1**.

**Décompte.** Douze verbes de premier niveau — ceux que `jg ⇥` propose — pour **quatorze
clés de table**, `source` en comptant trois (`source`, `source add`, `source rm`).

### État de vérification de la table

La colonne **brew** ci-dessus a été vérifiée le 15 août 2026 contre `brew 6.0.17-73-gc68efb6`
(macOS) : `brew help`, `brew commands` et `brew <verbe> --help` pour chacun des douze verbes
(`install`, `uninstall`, `upgrade`, `list`, `outdated`, `search`, `info`, `tap`, `untap`,
`pin`, `unpin`, `cleanup`, `doctor`). Tous les verbes et toutes les options citées existent
tels quels ; aucun écart avec ce qui était écrit de mémoire. Le détail des sorties observées
est dans le rapport de la tâche 1.

> **Colonnes winget et scoop : non vérifiées.** Cette machine est un Mac ; `winget` et
> `scoop` sont des binaires Windows absents de cet environnement. Les valeurs des colonnes
> **winget** et **scoop** ci-dessus restent donc **écrites de mémoire, non confirmées** —
> en particulier les trois points les plus incertains :
>
> - `winget pin add` / `winget pin remove` (existent-ils sous cette forme ?)
> - `winget source list` / `source add` / `source remove`
> - `scoop checkup`, `scoop hold` / `unhold`, `scoop cleanup`, et la portée réelle de
>   `scoop update *`
>
> **Vérification différée à une tâche 1b**, à exécuter depuis Windows :
>
> ```powershell
> winget pin --help
> winget source --help
> scoop help
> scoop update --help
> ```
>
> et à capturer :
>
> ```powershell
> scoop status                 > internal/scoop/testdata/status.txt
> winget source list           > internal/winget/testdata/source-list-fr.txt
> ```
>
> Aucun moteur ne doit être écrit contre ces deux colonnes tant que la tâche 1b n'a pas
> confirmé (ou corrigé) les valeurs ci-dessus.

## §3 — Résolution et routage

### Le pipeline

```text
jg install fd
   │
   ├─1─ résoudre le VERBE      qui, parmi les gestionnaires disponibles,
   │                            a « install » dans sa table ?
   │
   ├─2─ résoudre la CIBLE      selon Pool du verbe :
   │                            Aucun     → pas de cible, tous les candidats agissent
   │                            Catalogue → chercher « fd » dans chaque catalogue
   │                            Installés → chercher « fd » parmi les installés
   │
   ├─3─ trancher l'AMBIGUÏTÉ   0 → erreur ; 1 → il gagne ; ≥2 → le sélecteur
   │
   └─4─ GROUPER et exécuter    un appel par gestionnaire retenu, en séquence
```

`Pool` reprend la notion que porte déjà `Manager.InstalledOnly()`, en l'élargissant d'un
booléen à trois valeurs pour couvrir « ce verbe ne prend pas de paquet du tout ».

**`Manager.InstalledOnly()` n'est pas modifié pour autant** : il continue de servir le
chemin de complétion natif (`brew uninstall ⇥`). `Pool` est un type neuf, porté par
`Binding`, qui ne concerne que le chemin de la façade. Les deux coexistent, conformément à
[ADR-0002](../adr/0002-facade-table-declarative.md) qui laisse `pm.Manager` intact.

Seuls les gestionnaires rendus par `managers.Available()` participent.

### Résoudre ne coûte rien de lent

L'étape 2 appelle `Manager.Load()`, qui a déjà l'interdiction de toucher au gestionnaire —
il ne lit que des caches et des répertoires. La discipline posée pour le popup se paie
ici sans rien ajouter.

### Les messages d'erreur

Pour une façade, les cas d'échec *sont* le produit.

**Verbe inconnu de tous** — le modèle de capacités parle :

```text
$ jg doctor                       # Windows, winget seul
jigger: « doctor » — aucun gestionnaire disponible ne sait faire ça.
        scoop le sait (checkup), mais n'est pas installé.
```

**Nom inconnu de tous** — avec les préfixes voisins, tirés des catalogues déjà chargés :

```text
$ jg install fdfind
jigger: « fdfind » — inconnu de winget et scoop.
        Proche : fd (scoop/main), fd-find (scoop/extras)
        Si le paquet est trop récent pour le catalogue : jg install --pm winget fdfind
```

**Catalogue en cours de constitution** — surtout pas « paquet inconnu ». `Catalog.Note`
existe pour ça et se réutilise tel quel :

```text
$ jg install Git.Git
jigger: catalogue winget en cours de constitution — réessaie dans un instant.
```

**Ambiguïté** — le sélecteur existant, avec ses badges :

```text
$ jg install git
┌─ git : 2 gestionnaires ──────────┐
│ ◆ Git.Git            winget      │
│ ▣ git                scoop/main  │
└─ ↵ choisir   ^G annuler ─────────┘
```

Hors terminal (pipe, script, CI), pas de sélecteur : on échoue en listant les candidats et
en rappelant `--pm`.

### `--pm`, l'échappatoire unique

`jg install --pm winget Git.Git` force le gestionnaire. Un seul mécanisme pour trois
besoins : lever une ambiguïté sans TTY, atteindre un paquet trop récent pour le catalogue
en cache, et cibler un verbe sans nom (`jg cleanup --pm scoop`).

La forme courte `jg install scoop:fd` est écartée : elle ne servirait que le cas
« plusieurs paquets de plusieurs gestionnaires sur une ligne », qui se résout déjà tout
seul — chaque nom étant résolu indépendamment, `jg install fd Git.Git` route `fd` vers
scoop et `Git.Git` vers winget sans qu'on ait rien à dire.

### Pas de départage automatique

Aucun `JIGGER_PM_ORDER` ne choisira à la place de l'utilisateur quand deux gestionnaires
ont le paquet. Un choix silencieux entre deux `git` qui ne sont pas le même logiciel est
précisément ce qui rend une façade impossible à croire. L'ambiguïté est tranchée par
l'utilisateur, ou par `--pm`.

### Exécution séquentielle

Quand une ligne touche deux gestionnaires, ils passent l'un après l'autre dans l'ordre de
`managers.All()`, jamais en parallèle : les sorties s'entremêleraient, et une installation
qui échoue doit pouvoir arrêter la suite.

## §4 — Normalisation et exécution

### La ligne de partage

Tous les verbes n'ont pas de quoi être normalisés. `brew doctor` produit de la prose,
`winget install` une barre de progression.

**Règle : on normalise ce qui est tabulaire, on relaie ce qui ne l'est pas.**

| | Verbes | Traitement |
|---|---|---|
| **Normalisés** | `list`, `outdated`, `search`, `source` | sortie capturée, parsée, refondue |
| **Relayés** | `install`, `uninstall`, `upgrade`, `info`, `pin`, `unpin`, `cleanup`, `doctor`, `source add/rm` | stdio hérité, sortie du gestionnaire intacte |

`info` est rangé en relayé délibérément : normaliser `brew info` demanderait d'inventer un
schéma commun pour des métadonnées qui n'ont presque rien en commun, et la sortie native
est déjà bonne.

Cette ligne de partage ramène le coût de parsing de la phase 1 à **quatre verbes** au lieu
de douze.

### Ce que le partage résout gratuitement

**Les verbes qui ont besoin d'un terminal sont exactement ceux qu'on relaie.**

- **Relayés** → `cmd.Stdin/Stdout/Stderr = os.Std*`. Le gestionnaire hérite du terminal.
  Ses invites, ses barres de progression, son élévation UAC fonctionnent comme si
  l'utilisateur avait tapé la commande. **Aucun code de TTY, aucune gestion d'élévation.**
- **Normalisés** → sortie capturée. Ils sont tous en lecture seule : rien à demander,
  personne à élever.

Deux conséquences :

- **Les accords de licence winget.** `winget install` pose ses questions, l'utilisateur y
  répond. jigger n'accepte rien à sa place. Le drapeau `--yes`, **s'il est donné
  explicitement**, ajoute `--accept-package-agreements --accept-source-agreements`.
  Jamais par défaut.
- **Aucun champ `Elevate`** n'est nécessaire dans `Binding`.

### Le modèle de données

```go
// Package est une ligne de sortie normalisée — ce que jigger sait dire d'un paquet,
// quel que soit le gestionnaire qui l'a produit.
type Package struct {
    Name      string // identifiant natif : « fd », « Git.Git »
    Version   string // version installée ; vide si non installé
    Available string // version disponible ; vide si à jour ou inconnue
    Kind      string // badge pm.Badge* — le popup l'affiche déjà
    Source    string // provenance fine : « main », « extras », « homebrew/core »
    PM        string // « brew », « winget », « scoop »
}
```

Un seul type pour les quatre verbes normalisés. `Kind` réutilise les badges existants :
la sortie de `jg list` et le popup parlent le même langage visuel.

### `Binding`

Trois façons **mutuellement exclusives** de satisfaire un verbe :

```go
type Binding struct {
    Native []string                              // gabarit d'argv — le cas ordinaire
    Build  func(args []string) []string          // argv calculé — le cas rétif
    Direct func(args []string) ([]Package, error) // sans sous-processus du tout

    Pool  Pool   // Catalogue | Installés | Aucun
    Parse Parser // nil → relais brut
}

// Parser refond la sortie d'un gestionnaire en lignes normalisées. Fonction pure :
// elle ne lance rien, ce qui la rend testable sur fichier.
type Parser func(out []byte) ([]Package, error)
```

`Direct` existe parce que **scoop sait déjà répondre sans lancer scoop** :
`internal/scoop/outdated.go` compare les manifestes sur le disque, ce que fait
`scoop status` mais sans démarrer PowerShell ni toucher au réseau. Passer par un
sous-processus pour redemander ce que jigger sait déjà serait absurde. `jg outdated` sera
donc instantané côté scoop, et lent côté winget — comme aujourd'hui.

`Build` est une porte de sortie, pas la règle : un gabarit d'argv ne dira jamais tout
(`brew tap` / `untap`).

### Les parsers

| | Ce qu'on parse | Difficulté |
|---|---|---|
| **brew** | `--json=v2` sur `outdated` ; texte simple sur `list` | faible — c'est du vrai JSON |
| **scoop** | rien, ou presque : `Direct` lit le disque | nulle — le code existe |
| **winget** | tableaux à largeur fixe, en-têtes traduits | **c'est là qu'est le travail** |

winget réutilise `internal/winget/table.go` et les jeux d'essai de
`internal/winget/testdata/`, déjà en français. Chaque parser est une fonction pure
`[]byte → []Package`, testée sur fichier, sans rien lancer.

### Formats de sortie

Tableau aligné par défaut ; `--json` pour un tableau de `Package`. **La colonne `PM`
n'apparaît que si plus d'un gestionnaire a contribué** — sur macOS, où seul brew répond,
elle serait du bruit.

```text
$ jg outdated              # Windows
PAQUET             ACTUEL     DISPO      PM
fd                 10.1.0     10.2.0     scoop
Git.Git            2.54.0     2.55.0     winget

$ jg outdated              # macOS — pas de colonne PM
PAQUET             ACTUEL     DISPO
fd                 10.1.0     10.2.0
```

### Codes de retour et échec partiel

- **Un seul gestionnaire concerné** → son code de retour, propagé tel quel.
- **Verbe en lecture, plusieurs gestionnaires** → au mieux : on imprime ce qu'on a obtenu,
  on avertit sur stderr pour celui qui a échoué, et on sort en 0 si au moins un a répondu.
  `jg outdated` doit rester utile quand winget a un hoquet.
- **Verbe mutant, plusieurs gestionnaires** → **arrêt à la première erreur**, code de
  retour de celui qui a échoué. On n'installe pas depuis scoop si l'installation winget
  vient d'échouer.

L'asymétrie est délibérée : la lecture est au mieux, l'écriture ne devine pas.

## §5 — Le popup

Le popup est le **fil conducteur visuel** du produit. Il entre dans la phase 1, il n'en est
pas une suite.

### Ce que la façade lui demande

**Compléter le vocabulaire jigger.** Les clés des tables *sont* le vocabulaire — `jg ⇥`
propose les 12 verbes, `jg source ⇥` propose `add` et `rm`, sans une ligne de données en
plus.

**Dire de quel gestionnaire vient chaque candidat.** Le badge ne suffit pas : `BadgeOther`
est partagé par winget et scoop, donc deux candidats peuvent porter le même glyphe en
venant d'ailleurs. **`pm.Item` gagne un champ `PM`**, vide en contexte
mono-gestionnaire — donc sans rien changer à l'affichage actuel de `brew install ⇥`.

**Trancher les ambiguïtés.** Le sélecteur de §3 n'est pas un nouvel écran : c'est
`internal/ui/picker.go` avec un autre titre et d'autres touches de pied.

**Une seule identité visuelle.** Badges, glyphes et couleurs sont définis une fois et
partagés par le popup, les tableaux de sortie et le bloc oh-my-posh. Les tableaux de §4
consomment le rendu de badge de `ui`, ils ne le réinventent pas.

### Le risque de latence, et sa parade

Aujourd'hui `render` charge **un** catalogue par frappe. `jg install g` devra charger
**tous les catalogues disponibles** et les fusionner — sous Windows, les 14 401 noms de
winget plus ceux de scoop, à chaque lettre, contre un budget de ~8 ms.

C'est le seul endroit du design qui menace une propriété déjà acquise. La parade est
structurelle, pas une optimisation d'après-coup :

**Filtrer puis fusionner, jamais l'inverse.** Chaque catalogue est filtré par le préfixe
chez lui, et seuls les survivants sont fusionnés et triés. `complete.CompleteWith`
parcourt déjà `cat.Names` en filtrant — il suffit de le faire par gestionnaire avant de
réunir, plutôt que de concaténer trois catalogues puis de balayer.

Un banc d'essai Go sur ce chemin fait partie de la phase 1, pour que la régression se voie
au lieu de se deviner.

### Rester ouvert sans sur-concevoir

- **Le volet d'aperçu** (`brew desc`, `winget show` dans le cadre) est à la feuille de
  route. Il n'entre pas en phase 1, mais `Frame` ne doit pas figer sa largeur ni supposer
  une seule colonne de contenu.
- **La colonne PM apparaît selon les données**, pas selon un drapeau — même règle que les
  tableaux de §4. Une colonne ajoutée plus tard suivra le même chemin.

**Aucun framework de colonnes** n'est proposé aujourd'hui : deux cas connus ne justifient
pas une abstraction. Le champ `PM` et la règle d'apparition suffisent ; au troisième cas,
on aura trois exemples pour concevoir juste.

## Portée de la phase 1

| # | Livrable |
|---|---|
| 1 | Types : `Verb`, `Binding`, `Pool`, `Package`, interface `Bindings` |
| 2 | Les trois tables — 14 clés, 12 verbes — **vérifiées contre les CLI réellement installées** |
| 3 | `internal/facade` : résolution, routage, exécution, agrégation, codes de retour |
| 4 | Parsers des 4 verbes normalisés (`list`, `outdated`, `search`, `source`) |
| 5 | Aiguillage `main.go`, drapeaux `--pm`, `--json`, `--yes` |
| 6 | Popup : `Item.PM`, colonne PM, vocabulaire façade, sélecteur de désambiguïsation |
| 7 | Greffons : alias `jg`, `jigger`/`jg` reconnus par `motCommande` et `JIGGER_COMMANDS` |
| 8 | Documentation : ADR, cette spec, section README |

## Non-buts

Ce que la phase 1 ne fait pas, et pourquoi :

- **Les verbes singuliers** (`brew services`, `winget export`, `scoop reset`) — une ligne
  de table chacun le jour où l'un manque.
- **Le départage automatique** (`JIGGER_PM_ORDER`) — un choix silencieux entre deux `git`
  différents ruine la confiance.
- **Le volet d'aperçu** — connu, planifié, pas maintenant.
- **`info` normalisé** — relayé brut, cf. §4.
- **Les PM tiers par sous-processus** — l'extensibilité sans recompilation est une vraie
  décision, elle mérite son ADR.
- **De nouveaux gestionnaires** (apt, dnf, pacman) — la phase 1 prouve le mécanisme sur
  trois, pas sur cinq.
- **Un lien symbolique `jg` pour les scripts** — l'alias shell suffit tant que la façade
  s'utilise au clavier.

## Tests

- **Tables** — test pur, sans sous-processus : `install` sur scoop produit bien
  `["scoop","install","fd"]`. C'est ce qui transforme la table de correspondance en
  promesse vérifiable.
- **Capacités** — `jg doctor` sans scoop disponible produit le message attendu de §3.
- **Résolution** — catalogues fabriqués, comme `complete.CompleteWith` le permet déjà :
  les cas 0, 1 et ≥2 gestionnaires.
- **Parsers** — fonctions pures sur fichier, `testdata/` par gestionnaire, locales
  comprises.
- **Popup** — `internal/ui/frame_test.go` et `picker_test.go` étendus ; et surtout les
  suites PTY, qui vérifient `jg install ⇥` dans un vrai terminal : `tests/zpty.zsh` côté
  zsh, `tests/conpty` côté PowerShell.
- **Latence** — banc d'essai sur le chemin `jg install g`, trois catalogues chargés.

  Mesures du 15 août 2026 (MacBook Pro M4 Max, macOS 14.7) :

  ```text
  BenchmarkComplete-16        2431    526887 ns/op   2558115 B/op     19 allocs/op
  BenchmarkCompleteFacade-16   338   3564154 ns/op   9674496 B/op    172 allocs/op
  ```

  **Rapport 6.77×**, terme non linéaire O(n log n) : le chemin natif marche dans `cat.Names` (déjà trié) et accumule les survivants en ordre — coût O(n), pas de tri. Le chemin façade fusionne les survivants de trois catalogues, puis appelle `sort.Slice` sur le résultat fusionné — coût O(n log n). Une fusion k-voies (k=3) réduirait ce terme à O(n log k).

  **Budget** : 3.56 ms contre ~8 ms, soit 45 % du budget. À l'intérieur du seuil d'acceptabilité, pas d'optimisation maintenant. Levier connu pour une régression future : passer de `sort.Slice` à une fusion k-voies ordonnée.

  **Fixture adversariale** : le préfixe `g` correspond à *tous* les 25 401 noms. L'usage réel ne ressemble pas à ça — `jg install fire` sur cette machine ne trouve que 18 candidats. C'est un cas pire, non une mesure typique.

## Décisions liées

- [ADR-0001 — Go confirmé](../adr/0001-go-confirme.md)
- [ADR-0002 — Façade à table déclarative](../adr/0002-facade-table-declarative.md)
