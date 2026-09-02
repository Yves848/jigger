# Conception — le module pacman / yay

2 septembre 2026

Premier gestionnaire **Linux** de jigger. Il répond à la grille de l'entrée
[A-16](../ameliorations.md) — l'étude qu'elle demandait est faite ici, et conclut à
« faisable, bon marché », d'où l'implémentation dans la foulée plutôt qu'une note.

Toutes les mesures de ce document ont été prises le 2 septembre 2026 sur la machine de
développement (Arch Linux / Omarchy, pacman 7.1.0, yay 13.0.1, 1 055 paquets installés).

## 1. La grille A-16, remplie

| Question | Réponse mesurée |
|---|---|
| **Le catalogue** | `pacman -Sl` : **166 ms**, 15 604 lignes, format `dépôt nom version [installed[: v]]`. Une seule commande donne le dépôt, le nom, la version disponible, l'état installé et — quand elles diffèrent — la version installée. C'est la sortie la plus riche des quatre gestionnaires de jigger. |
| **Les installés** | `/var/lib/pacman/local/<nom>-<pkgver>-<pkgrel>/` : un `readdir` de 1 057 entrées, **sous la milliseconde**. C'est exactement le Cellar de Homebrew et le `apps/` de scoop. `pacman -Qq` coûte 10 ms — négligeable en absolu, mais c'est un sous-processus, donc hors du chemin d'un rendu par principe. |
| **Les obsolètes** | `checkupdates` (pacman-contrib) : **510 ms**, et il **synchronise une base privée** — donc réseau. `pacman -Qu` est instantané et local, mais ne dit que ce que la dernière `-Sy` sait. Côté AUR, `yay -Qua` interroge le RPC de l'AUR. Tout cela vit en tâche de fond, comme `brew outdated`. |
| **Une sortie machine ?** | Pas de JSON, mais **pas besoin** : les sorties sont déjà des colonnes séparées par des espaces, sans largeur fixe, sans en-tête, sans ANSI quand la sortie n'est pas un terminal. C'est le cas facile — l'inverse de la leçon scoop. |
| **La table des verbes** | Voir §6. Elle tient en quinze lignes. |
| **Élévation, plateforme, portée** | pacman exige root pour `-S`, `-R`, `-U` ; yay **refuse** de tourner en root et appelle `sudo` lui-même. C'est la seule vraie difficulté du module, et elle est tranchée par l'[ADR-0007](../adr/0007-pacman-lit-yay-pilote.md). |
| **Le verdict** | **Faisable, bon marché** — comparable à brew, très loin de winget. |

## 2. Deux fournisseurs, un paquet

`internal/pacman` fournit `New("pacman")` et `New("yay")`, sur le patron exact de
`ssh.New("ssh") / ("scp") / ("sftp")` : `Manager.Cmd()` ne rend qu'un mot, et l'élargir
obligerait les autres gestionnaires à répondre à une question qu'ils ne se posent pas.

Ce qui les sépare tient en trois points, et rien d'autre :

| | `pacman` | `yay` |
|---|---|---|
| Catalogue | dépôts | dépôts **+ AUR** |
| Opérations proposées | celles de pacman | + `-Y`, `-P`, `-G`, `--aur`, `--repo` |
| Verbes pilotables | lecture seule, et seulement si yay est absent ([ADR-0007](../adr/0007-pacman-lit-yay-pilote.md)) | tous |

`paru` et les autres assistants AUR ne sont pas traités. Ils s'ajouteraient par un
troisième `New("paru")` le jour où le besoin se pose ; rien dans la conception ne s'y
oppose.

## 3. La grammaire : les opérations sont des drapeaux

C'est le seul point où pacman ne ressemble à aucun des trois gestionnaires déjà branchés.
`brew install`, `winget install`, `scoop install` : un verbe. pacman écrit `pacman -S`.

`complete.completeWith` route sur un seul test :

```go
isOption := strings.HasPrefix(word, "-")
```

Le mot qui commence par `-` va chercher `Options(sub)` ; les autres vont chercher
`Subcommands()` (premier mot) ou le catalogue (ensuite). Une opération de pacman tombe donc
**du mauvais côté** de ce test.

La résolution ne demande aucune modification de `complete` : le fournisseur déclare la même
liste des deux côtés.

- `Subcommands()` rend les opérations. `pacman ` (espace, mot vide) : `firstWord` et
  `!isOption`, donc la liste s'ouvre entière.
- `Options("")` rend **la même** liste. `pacman -` : `isOption` avec `sub` encore vide, donc
  les opérations à nouveau, filtrées sur `-`.
- `Options("-s")`, `Options("-r")`… rendent les drapeaux secondaires de l'opération
  (`--needed`, `--noconfirm`, `--nosave`…).

Une propriété du moteur rend cela sûr : `complete` **minuscule** la sous-commande
(`sub = strings.ToLower(before[0])`). Les tables sont donc indexées en minuscules —
`"-s"` pour `-S`, `"-rns"` pour `-Rns`. Aucune collision n'est possible dans le sens qui
compte : `-s` seul n'est pas une opération valide de pacman, il n'existe que comme
modificateur derrière une opération.

Ce que ça donne à l'usage :

```
pacman ⇥            → -S  -Syu  -Ss  -Si  -R  -Rns  -Q  -Qu  -U  -F …
pacman -⇥           → la même liste, filtrée
pacman -S fire⇥     → le catalogue des dépôts (◆), filtré sur « fire »
pacman -Rns fire⇥   → les seuls paquets installés
pacman -S --⇥       → --needed --noconfirm --asdeps --overwrite …
yay -S fire⇥        → dépôts (◆) + AUR (▣)
```

`InstalledOnly` est vrai pour toute la famille `-R` et toute la famille `-Q` : ce sont
précisément les opérations qui interrogent la base locale.

## 4. Le catalogue, en deux étages

**Dépôts.** `pacman -Sl` au réchauffement (**166 ms**), mis en cache brut sous le nom
`pacman-sync`, TTL déclaré `pacman_ttl` (défaut 24 h, comme brew). Une ligne se lit en
quatre champs :

```
core acl 2.4.0-1 [installed]
omarchy gpu-screen-recorder 5.12.3-2 [installed: 6.0.1-1]
extra zsh 5.9-8
```

**AUR** (yay seul). `yay -Slq aur` au réchauffement : **799 ms**, 118 582 noms, cache brut
`aur-names`, TTL déclaré `aur_ttl` (défaut 24 h).

> *Écarté :* lire `~/.cache/yay/completion.cache`, qui contient déjà `nom<TAB>source` pour
> 134 108 entrées et dispenserait du sous-processus. C'est un format privé, rafraîchi selon
> le calendrier de yay et non le nôtre ; jigger tiendrait un cache dont il ne contrôle ni la
> forme ni la péremption.

**Installés.** `readdir` de `/var/lib/pacman/local`. Chaque entrée est un répertoire nommé
`<nom>-<pkgver>-<pkgrel>`. Le découpage est **exact et non heuristique** : Arch interdit le
tiret dans `pkgver` comme dans `pkgrel`, donc les deux derniers tirets du nom de répertoire
sont toujours les bons séparateurs. Le fichier `ALPM_DB_VERSION`, seule entrée non
répertoire, est ignoré par le même test que brew applique à Cellar.

Rien de tout cela n'est mis en cache : c'est frais à chaque appel, pour moins d'une
milliseconde — la propriété que scoop résume par « rien à mettre en cache, donc rien qui
puisse mentir ».

### Le second étage, et pourquoi il existe

La première rédaction s'arrêtait là : `Charger` lisait les deux caches bruts, fusionnait,
dédupliquait et triait. **Mesuré à 45 ms**, à chaque frappe — parce que `Charger` est dans
le chemin de `jigger render`, que le greffon appelle en substitution de commande dans le
widget zle (`shell/jigger.plugin.zsh:222`). Ces millisecondes-là sont de la latence de
frappe, et le budget que le projet se donne est de « quelques ms au plus »
(`internal/complete/complete_test.go:332`).

La fusion est donc faite **une fois par réchauffement**, et déposée dans un fichier déjà
trié — `pacman-catalog` ou `yay-catalog` selon le fournisseur, puisque le catalogue de
pacman ne contient que les dépôts. Trois formes de ligne :

```
firefox-nightly            un paquet de l'AUR
zsh<TAB>extra              un paquet de dépôt
rustup<TAB>extra<TAB>+     un paquet que les deux portent (cf. §5)
```

`Relire` remplit alors `Names` et `Badges` **directement**, sans repasser par
`Catalog.Add` : la déduplication a déjà eu lieu, et refaire la vérification cent
trente-quatre mille fois coûterait précisément ce qu'on vient d'économiser. C'est le même
déplacement que celui qui a sorti `brew list` du chemin du rendu — le travail n'a pas
disparu, il a changé de côté.

Deux mesures secondaires vont avec :

- `pm.NewCatalogDe(n)` dimensionne les tables d'avance. Une table de hachage Go qui reçoit
  134 000 entrées sans indication de taille les réalloue une vingtaine de fois en chemin :
  **49 ms → 35 ms** sur la seule construction. Le champ est additif, et
  `NewCatalog()` reste `NewCatalogDe(0)` — rien ne change pour brew, winget et scoop.
- `pacman -Sl` est mémoïsé pour la durée du processus. `jigger warm` appelle `Warm` sur les
  **deux** fournisseurs, et `warm --all` met le TTL à zéro : sans cela, la commande tournait
  deux fois par réchauffement — et le greffon en lance un après chaque `pacman -Sy`.

Le résultat, mesuré sur la machine de développement :

| | avant | après |
|---|---|---|
| `render` sur `pacman -S fire` | 13 ms | **7 ms** |
| `render` sur `yay -S fire` | 70 ms | **28 ms** |
| `warm --all` (caches bruts frais) | 1 058 ms | **710 ms** |

Les 28 ms restants sont la lecture de 2,1 Mo et le remplissage d'une table de 134 000
entrées. Les faire tomber encore demanderait un badge par défaut sur `pm.Catalog` — un
changement du type partagé par les quatre gestionnaires, pour une dizaine de
millisecondes. Ce n'est pas fait, et c'est délibéré.

**Badges.** Deux classes, comme partout ailleurs dans jigger :

| Badge | Glyphe | Sens |
|---|---|---|
| `pm.BadgeRepo` | ◆ ambré | paquet d'un dépôt binaire |
| `pm.BadgeAUR` | ▣ violet | paquet de l'AUR, ou installé hors dépôts |

C'est la dichotomie formula/cask et catalogue/hors-catalogue : le cas ordinaire d'abord,
l'autre ensuite. `core`, `extra`, `multilib` et `omarchy` sont tous des dépôts binaires
signés, et les distinguer au glyphe n'apprendrait rien à qui tape ; le nom du dépôt sert à
la qualification (§5), pas au badge.

**Note d'attente.** Catalogue vide au premier usage : `popup.catalog_pacman`, sur le modèle
de `popup.catalog_brew`.

## 5. Insert : une seule correction, et elle est pour yay

`pacman -S <nom>` n'a jamais besoin d'être corrigé : quand plusieurs dépôts portent le même
nom, pacman tranche par ordre de priorité, sans erreur ni question.

`yay -S <nom>` en a besoin dans un cas : un nom présent **à la fois** dans un dépôt et dans
l'AUR — 121 des 134 000 sur cette machine. yay ouvre alors un menu interactif au milieu de ce que
jigger vient d'insérer. La correction est celle de scoop, au mot près — `cat.Qualified`
existe déjà pour cela, et c'est la troisième forme de ligne du fichier fusionné qui la
porte :

```
yay -S ⇥ … visual-studio-code-bin   →   yay -S aur/visual-studio-code-bin
```

La qualification n'est posée **que** sur les noms réellement partagés. Un nom AUR unique
s'insère nu, comme un nom de dépôt.

## 6. La table des verbes

pacman, quand yay est absent — lecture seule, [ADR-0007](../adr/0007-pacman-lit-yay-pilote.md) :

| Verbe | Natif | Pool |
|---|---|---|
| `list` | `pacman -Q` | aucun |
| `outdated` | `pacman -Qu` | aucun |
| `search` | `pacman -Ss {args}` | aucun |
| `info` | `pacman -Si {args}` | catalogue |

yay — tout :

| Verbe | Natif | Pool |
|---|---|---|
| `install` | `yay -S {args}` | catalogue |
| `uninstall` | `yay -Rns {args}` | installés |
| `upgrade` | `yay -Syu {args}` | installés |
| `list` | `yay -Q` | aucun |
| `outdated` | `yay -Qu` | aucun |
| `search` | `yay -Ss {args}` | aucun |
| `info` | `yay -Si {args}` | catalogue |
| `cleanup` | `yay -Sc` | aucun |

Absents chez les deux, et c'est le modèle de capacités qui parle : **`source`** — un dépôt
Arch s'ajoute en éditant `/etc/pacman.conf`, pas par une commande ; **`pin` / `unpin`** —
`IgnorePkg` est une ligne de configuration, pas un verbe ; **`doctor`** — pacman n'a pas
d'équivalent de `brew doctor`.

Les parsers des trois verbes normalisés :

- `parseList` — `nom version-release`, une ligne par paquet.
- `parseOutdated` — `nom ancienne -> nouvelle`. La flèche est le séparateur.
- `parseSearch` — deux lignes par résultat : `dépôt/nom version [groupes]` puis la
  description indentée. Seule la première est retenue, et le `dépôt/` en fait un candidat
  qualifié gratuitement.

## 7. Le bloc de prompt

`prompt.SondePlateforme` choisit aujourd'hui entre Windows et « Homebrew là où il règne ».
Elle gagne un troisième cas : **Linux avec pacman** passe par `SondePacman`, et Linux sans
pacman garde brew (le greffon zsh y tourne déjà, cf. README).

| Champ | Source |
|---|---|
| `Version` | `pacman --version` → `Pacman v7.1.0 - libalpm v16.0.1` → `7.1.0` |
| `Primary` | obsolètes des dépôts — `checkupdates` s'il existe, sinon `pacman -Qu` |
| `Secondary` | obsolètes de l'AUR — `yay -Qua`, zéro si yay est absent |

La répartition primaire/secondaire garde exactement le sens qu'elle a partout : formulae /
casks, winget / scoop, **dépôts / AUR**. Le format du cache d'une ligne ne bouge pas, donc
ni le hook zsh, ni les segments oh-my-posh et starship n'ont à changer.

`checkupdates` est préféré quand il est là parce qu'il synchronise une base privée : il ne
touche pas à `/var/lib/pacman/sync`, donc il ne peut pas laisser le système dans l'état
« `-Sy` sans `-u` » qui casse une installation Arch. C'est aussi pourquoi jigger ne lance
**jamais** `pacman -Sy` lui-même.

## 8. Le greffon zsh

Deux changements, tous deux dans `shell/jigger.plugin.zsh` :

- `JIGGER_COMMANDS` passe de `brew ssh scp sftp` à `brew pacman yay ssh scp sftp`. Le
  défaut vaut pour toutes les machines : un `pacman` tapé sur macOS reste complété au pire
  sur un catalogue vide, exactement comme un `brew` tapé sous Windows.
- La détection des commandes mutantes (`_jigger_brew_mutants`) apprend les opérations qui
  changent l'état — `-S`, `-Syu`, `-R`, `-U` et leurs variantes — sous `pacman` comme sous
  `yay`, pour que le compteur du prompt ne mente pas une demi-heure après une mise à jour.

## 9. Ce que le module ne fait pas

- **Il n'élève rien.** L'[ADR-0004](../adr/0004-elevation-constatee.md) tient : jigger
  constate, il n'intercepte pas. `pm.Elevateur` n'est pas implémenté — pacman rend 1 pour
  tous ses échecs, y compris « you cannot perform this operation unless you are root », et
  un code qui ne distingue rien ne permet de constater rien.
- **Il ne synchronise jamais.** Aucun chemin de jigger ne lance `pacman -Sy`.
- **Il ne construit rien.** `yay -G`, `makepkg`, les PKGBUILD : hors du contrat
  `pm.Manager`, qui ne sait que répondre à des questions et relayer des commandes.
