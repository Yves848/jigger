# Premiers pas avec jigger

*Ce document existe aussi en [anglais](../getting-started.md).*

De l'installation à la première complétion, en une dizaine de minutes. Ce document se lit
d'un bout à l'autre ; le [README](../../README.fr.md) reprend chaque point en détail, et explique
_pourquoi_ les choses sont faites ainsi.

`jigger` branche un sélecteur de paquets dans ton shell : dès que tu tapes une commande de
gestionnaire de paquets, un cadre s'affiche sous le prompt et suit ta frappe.

```
❯ brew install fire
╭──────────────────────────────────────────────────────╮
│❯ brew install                            jigger 0.8.0│
│  ▣  firealpaca                                       │
│  ▣  firebase-admin                                   │
│  ◆  firebase-cli                                     │
│  ▣  firebird-emu                                     │
│  ▣  firecamp                                         │
│                                                      │
│   ⇥  insérer   ↓  parcourir   ^G  fermer             │
╰──────────────────────────────────────────────────────╯
```

Et, par-dessus les trois gestionnaires, **une seule syntaxe** : `jg install fd` s'adresse à
celui qui connaît `fd`, sans que tu aies à savoir lequel (§ 6).

| Plateforme | Shell | Gestionnaires |
|---|---|---|
| macOS, Linux | zsh | [Homebrew](https://brew.sh) |
| Windows | PowerShell 7 | [winget](https://learn.microsoft.com/windows/package-manager/), [scoop](https://scoop.sh) |

## 1. Prérequis

- **Le gestionnaire lui-même** — et rien d'autre. jigger ne dépend d'aucun service, ne
  parle à aucun réseau, et se contente de ce que `brew`, `winget` ou `scoop` a déjà sur le
  disque.
- **zsh** (livré avec macOS) ou **PowerShell 7** avec PSReadLine (livré avec Windows).
- **Go ≥ 1.24**, uniquement pour compiler — le paquet Homebrew s'en charge tout seul.

## 2. Installer le binaire

### macOS et Linux — par Homebrew (recommandé)

Le tap est hébergé sur le GitLab du projet, d'où l'URL explicite :

```sh
brew tap yves/cocktails https://gitlab.yg-devworks.com/yves/homebrew-cocktails.git
brew install jigger
```

La formule compile le binaire chez toi (`go` est tiré comme dépendance de compilation),
installe le greffon zsh sous `share/`, et pose au passage `brew-jigger` — ce qui rend
`brew jigger …` utilisable comme n'importe quelle commande brew.

Mise à jour, ensuite : `brew upgrade jigger`.

### Toutes plateformes — par Go

```sh
go install gitlab.yg-devworks.com/yves/jigger@latest
```

Le binaire atterrit dans `$GOBIN` (à défaut `~/go/bin`, ou `%USERPROFILE%\go\bin` sous
Windows). Vérifie que ce répertoire est dans ton `PATH`.

### Depuis les sources

```sh
git clone https://gitlab.yg-devworks.com/yves/jigger.git
cd jigger
make install            # → ~/.local/bin/jigger  (PREFIX=… pour changer)
```

C'est la voie à prendre sous **Windows** : il n'existe pas encore de paquet winget ni
scoop pour jigger (c'est à la feuille de route). Le greffon PowerShell est alors celui du
dépôt cloné.

> **Un seul binaire dans le `PATH`.** Si tu as installé par plusieurs voies, `which -a
> jigger` (ou `Get-Command jigger -All`) le dira. Un binaire ancien devant un greffon
> récent est la panne la plus pénible à diagnostiquer — d'où la vérification du § 4.

## 3. Brancher le greffon dans le shell

### zsh

```sh
# dans ~/.zshrc
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

Depuis les sources, remplace le chemin par `/chemin/vers/jigger/shell/jigger.plugin.zsh`.
Puis recharge : `exec zsh`.

L'ordre des `source` dans `~/.zshrc` n'a aucune importance — le greffon se place lui-même
là où il faut dans les hooks de zsh.

### PowerShell

```powershell
# dans $PROFILE   (notepad $PROFILE pour l'ouvrir)
Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
```

Puis recharge : `. $PROFILE`, ou ouvre un nouvel onglet.

Une seule contrainte d'ordre, celle-ci réelle : si tu utilises oh-my-posh ou starship,
importe jigger **après** lui (cf. § 8).

## 4. Vérifier que ça marche

```sh
jigger --version        # → jigger 0.8.0, ou plus récent
```

Ouvre un shell neuf et tape `brew ins` (ou `winget ins`) **sans valider**. Le cadre doit
apparaître sous le prompt et se filtrer à chaque lettre.

Rien ne s'affiche ? Le greffon le dit quand il refuse de se charger : un message, au
démarrage du shell, signale que le binaire est introuvable dans le `PATH` — ou qu'il est
trop ancien pour ce greffon. Les deux vont par paire : un binaire en retard ne comprend pas
les options que le greffon lui passe, et le popup ne s'afficherait jamais, sans un mot. Si
aucun message n'apparaît, va au § 9.

**À la toute première utilisation**, le cadre peut annoncer « catalogue en préparation… » :
jigger ne fait jamais attendre une frappe après le gestionnaire de paquets, il constitue
donc son catalogue en tâche de fond. Quelques secondes plus tard, il est là — et il le
reste (cache de 24 h, renouvelé tout seul).

## 5. Utiliser

Tape simplement une commande. Le popup vit tout seul :

```
brew install fire         les paquets « fire… », mis à jour à chaque lettre
brew uninstall ␣          seulement les paquets installés
brew list --              les options de list
winget install Git.       idem, côté Windows
scoop uninstall 7z
```

| Touche | Effet |
|---|---|
| `⇥` | insère le candidat courant |
| `↓` | entre dans la liste, puis descend d'un candidat |
| `↑` | remonte ; au premier candidat, rend le clavier au shell |
| `^N` / `^P` | les mêmes, pour qui les préfère aux flèches |
| `^G` | ferme le popup pour la ligne en cours (`⇥` le rouvre) |

Deux choses à savoir, qui font l'essentiel du confort :

- **Les flèches restent ton historique** tant que le popup n'a pas le clavier — popup
  ouvert ou non. Le cadre le montre : ligne courante soulignée et pied `↑↓ naviguer` quand
  il a le focus, au repos et `↓ parcourir` quand il ne l'a pas.
- **jigger corrige ce qu'il insère** quand la commande serait fautive sans cela : `--cask`
  ajouté devant un cask Homebrew, nom qualifié `main/flux` pour un paquet scoop présent
  dans plusieurs buckets, guillemets autour d'un identifiant winget à espaces.

Les badges devant les noms distinguent les deux natures de paquets : ◆ pour le cas
ordinaire (formula, catalogue winget, bucket `main`), ▣ pour l'autre (cask, application
hors catalogue, bucket tiers).

## 6. Une seule syntaxe : `jg`

Tout ce qui précède parle la langue de chaque gestionnaire. `jg` en parle une seule pour
les trois :

```sh
jg install fd            # brew, winget ou scoop — celui qui connaît « fd »
jg outdated              # ce qui est à mettre à jour, partout
jg search ripgrep
jg info fd
```

`jg` est un alias de `jigger`, posé par le greffon zsh ; les deux s'écrivent
indifféremment. **La façade s'ajoute, elle ne remplace rien** : `brew install fd` continue
de marcher exactement comme avant, popup compris.

### Les douze verbes

`jg ⇥` te les rappelle, et le popup les propose comme il propose les paquets :

```
❯ jg
╭──────────────────────────────────────────────────────╮
│❯ jigger                                  jigger 0.8.0│
│  •  cleanup                                          │
│  •  doctor                                           │
│  •  info                                             │
│  •  install                                          │
│                                                      │
│   ⇥  insérer   ↓  parcourir   ^G  fermer             │
╰──────────────────────────────────────────────────────╯
```

`install`, `uninstall`, `upgrade`, `list`, `outdated`, `search`, `info` — les sept que les
trois gestionnaires savent tous faire. Puis `source` (le `tap` de brew, le `bucket` de
scoop), `pin`, `unpin`, `cleanup` et `doctor`, qui n'existent pas partout. Demander à
winget un verbe qu'il n'a pas — `cleanup`, `doctor` — échoue proprement, en disant qui
saurait le faire.

`source` prend trois formes : `jg source` liste, `jg source add <dépôt>` ajoute,
`jg source rm <dépôt>` retire.

### Comment le gestionnaire est choisi

jigger cherche le nom dans le catalogue de chacun des gestionnaires présents :

- **un seul le connaît** → il gagne, sans rien demander ;
- **plusieurs le connaissent** → le sélecteur s'ouvre et tu tranches ;
- **aucun ne le connaît** → erreur, avec les voisins les plus proches.

Il n'y a **jamais de choix automatique** entre deux gestionnaires, et aucun réglage n'en
introduit : deux paquets qui portent le même nom ne sont pas forcément le même logiciel.

`--pm <gestionnaire>` est l'échappatoire — pour trancher hors terminal (script, CI, pipe),
atteindre un paquet trop récent pour le catalogue en cache, ou viser un verbe sans nom :

```sh
jg install git --pm scoop
jg doctor --pm brew
```

### Des tableaux, et `--json`

Les quatre verbes qui rendent une liste — `list`, `outdated`, `search`, `source` — sortent
un tableau aligné, et le même contenu en JSON avec `--json` :

```
$ jg list
PAQUET                    ACTUEL
alembic                   1.8.12
aom                       3.14.1
assimp                    6.0.5
```

`jg outdated` y ajoute une colonne `DISPO`, la version qui t'attend — et répond
« rien à signaler » quand tout est à jour.

Une colonne `PM` s'ajoute quand **plusieurs** gestionnaires ont répondu — inutile de la
montrer quand elle serait partout la même.

Tout le reste (`install`, `info`…) **relaie la sortie du gestionnaire telle quelle** :
invites, barres de progression et élévation UAC fonctionnent comme si tu avais tapé la
commande native, précisément parce que jigger ne s'interpose pas. Sous winget, `--yes`
accepte les accords de licence ; il n'est jamais implicite.

### Ce qui n'est pas encore là

- **PowerShell n'a pas l'alias `jg`.** Seul le greffon zsh arme la façade ; sous Windows,
  `jigger install …` reste utilisable en tapant le nom complet, mais le popup ne le suit
  pas encore.
- **Les traductions winget et scoop n'ont pas été vérifiées contre les vraies CLI** — le
  développement s'est fait sur Mac. Seule la colonne brew a tourné pour de vrai. La table
  complète, avec cet avertissement, est dans le
  [README](../../README.fr.md#une-seule-syntaxe).

## 7. Configurer

Les réglages sont des **variables d'environnement**, à poser **avant** le `source` ou
l'`Import-Module` — c'est au chargement que le greffon lit ses touches et pose ses hooks.

```sh
# ~/.zshrc, avant le source
JIGGER_ROWS=12
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

```powershell
# $PROFILE, avant l'Import-Module
$env:JIGGER_ROWS = '12'
Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
```

| Variable | Défaut | Rôle |
|---|---|---|
| `JIGGER_LIVE` | `1` | popup vivant. `0` = ⇥ ouvre le sélecteur plein écran, et rien ne s'affiche sans le demander |
| `JIGGER_ROWS` | `8` | candidats affichés — à réduire sur un terminal court |
| `JIGGER_KEY` | `^I` (Tab) | touche d'insertion. `'^ '` pour Ctrl-Espace ; sous PowerShell, un nom PSReadLine (`Ctrl+Spacebar`) |
| `JIGGER_MIN_COLUMNS` | `30` | en dessous de cette largeur, le cadre n'a plus de sens : rien ne s'affiche |
| `JIGGER_CACHE_DIR` | `~/Library/Caches/jigger`, `%LOCALAPPDATA%\jigger` | emplacement du cache |
| `JIGGER_LANG` | la langue de ta locale | messages : `en` ou `fr`. Lu avant `LC_ALL`, `LC_MESSAGES` et `LANG` — c'est lui qui rend le français à un shell qui tourne en anglais. Ce que jigger ne sait pas traduire retombe sur l'anglais |

Deux réglages n'existent que sous PowerShell, faute d'équivalent utile côté zsh :

| Variable | Défaut | Rôle |
|---|---|---|
| `JIGGER_COMMANDS` | `winget,scoop` | commandes qui déclenchent le popup |
| `JIGGER_KEYS_EXTRA` | `éèêàçùâîôûëïüö°²µ§£€` | touches relayées en plus des ASCII imprimables |

`JIGGER_KEYS_EXTRA` mérite un mot : PSReadLine n'offre aucun crochet appelé à chaque
frappe, jigger réenregistre donc une à une les touches qui modifient la ligne. Sur un
clavier AZERTY, la rangée des chiffres non pressée donne « éèçàù » — d'où cette valeur par
défaut, et le réglage pour les dispositions qu'elle ne couvre pas.

## 8. Le bloc de prompt (optionnel)

jigger sait aussi afficher dans ton prompt la **version du gestionnaire** et les **mises à
jour en attente** :

```
 yves@MacBook  ~/git/jigger   main  🍺 6.0.17  🧪 7  📦 2 ❯      ← macOS
 PS D:\jigger  🪟 1.29.280  📦 48  🥄 1 ❯                        ← Windows
```

Rien de lent n'est dans le chemin du prompt : le comptage tourne détaché et dépose son
résultat dans un fichier d'une ligne, que le hook relit avec les seules primitives du
shell. Chaque compteur disparaît quand il tombe à zéro.

**Activer le hook** — avant le chargement du greffon :

```sh
JIGGER_PROMPT=1                                    # ~/.zshrc
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

```powershell
$env:JIGGER_PROMPT = '1'                           # $PROFILE, APRÈS oh-my-posh/starship
Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
```

**Ajouter le segment** — un fichier prêt à coller par prompt et par plateforme.

*oh-my-posh* : travaille sur une copie, les thèmes livrés sont écrasés à chaque mise à
jour :

```sh
mkdir -p ~/.config/oh-my-posh
cp "$(brew --prefix oh-my-posh)/themes/catppuccin_mocha.omp.json" \
   ~/.config/oh-my-posh/mon-theme.omp.json
```

Colle le contenu de [`shell/oh-my-posh/brew.segment.json`](../../shell/oh-my-posh/brew.segment.json)
— ou de [`windows.segment.json`](../../shell/oh-my-posh/windows.segment.json) — dans le
tableau `segments` du bloc voulu, puis fais pointer ton profil sur ta copie :

```sh
eval "$(oh-my-posh init zsh --config ~/.config/oh-my-posh/mon-theme.omp.json)"
```

*starship* : rien à copier au préalable, il n'y a qu'un fichier de configuration —
ajoute-lui [`shell/starship/brew.toml`](../../shell/starship/brew.toml), ou
[`windows.toml`](../../shell/starship/windows.toml) :

```sh
cat /chemin/vers/jigger/shell/starship/brew.toml >> ~/.config/starship.toml
```

Ce sont des modules `env_var`, que le format par défaut de starship affiche déjà : il n'y
a rien d'autre à faire.

Le bloc n'apparaît qu'au **deuxième prompt** : rien ne s'affiche tant que le premier
comptage n'est pas terminé. Les réglages associés (`JIGGER_PROMPT_TTL`,
`JIGGER_PROMPT_SYNC`) et les variables exposées sont décrits dans le
[README](../../README.fr.md#bloc-de-prompt) ; elles servent aussi bien à un prompt maison.

## 9. Quand ça ne marche pas

| Symptôme | Cause probable |
|---|---|
| « binaire introuvable dans le PATH » au démarrage du shell | le répertoire d'installation n'est pas dans le `PATH` — ou le shell n'a pas été rechargé |
| « le binaire … est en X, or ce greffon en demande Y » | deux installations concurrentes. `which -a jigger` ; `brew upgrade jigger` ou `make install` |
| aucun cadre, aucun message | terminal trop étroit (`JIGGER_MIN_COLUMNS`), ou terminal qui ne répond pas à l'interrogation de position du curseur — jigger s'abstient alors plutôt que de dessiner à l'aveugle |
| cadre absent sous PowerShell en **mode Vi** | le popup vivant y est désactivé exprès : relayer les caractères imprimables casserait le mode commande. ⇥ reste disponible |
| affichage qui se bat avec la prédiction PSReadLine | jigger range `PredictionViewStyle = ListView` le temps du cadre et le rend ensuite ; s'il reste en `InlineView`, un shell neuf remet tout d'aplomb |
| « catalogue en préparation… » qui dure | lance `jigger warm --all` à la main pour voir ce que dit le gestionnaire |
| le compteur du prompt est faux | il ne voit que ce qui passe par ce shell ; une mise à jour lancée ailleurs est rattrapée à l'expiration du TTL (30 min par défaut) |
| `jg` : « verbe inconnu » | ce n'est pas un des douze — `jg ⇥` les liste. La commande native, elle, s'écrit toujours en entier : `brew tap`, pas `jg tap` |
| `jg` : « inconnu de brew » sur un paquet qui existe | le catalogue en cache est plus vieux que le paquet. `jg … --pm brew <nom>` passe outre, `jigger warm --all` remet le cache à jour |
| `jg` : « gestionnaire indisponible pour ce verbe » | le `--pm` demandé n'est pas installé, ou ne sait pas faire ce verbe ; le message dit lesquels le savent |

Pour isoler un conflit avec un autre greffon de ligne d'édition, `JIGGER_LIVE=0` éteint
tout ce qui touche à la frappe : seul ⇥ reste, et ouvre le sélecteur plein écran.

**Désinstaller** : retire la ligne de `~/.zshrc` (ou de `$PROFILE`), puis
`brew uninstall jigger` — ou supprime le binaire. Le cache se jette avec
`rm -rf "$(dirname "$(jigger prompt --path)")"`.

## 10. Aller plus loin

Le greffon n'est qu'un client : les sous-commandes s'utilisent seules, et c'est le meilleur
moyen de comprendre ce qui se passe.

```sh
jigger complete "brew install fire" # les candidats, un par ligne
jigger complete "jg "               # … et les verbes de la façade
jigger render --line "brew ins" --cols 80   # une frame du popup, métadonnées comprises
jigger pick "brew uninstall 7z"     # le sélecteur plein écran
jigger demo                         # aperçu statique coloré
jigger prompt                       # l'état en cache, tel que le lit le hook
jigger warm --all                   # reconstitue les catalogues (lent)
```

Les verbes de la façade s'appellent de la même façon sans le greffon — `jg` n'étant qu'un
alias, `jigger outdated --json` marche partout, y compris dans un script ou une CI où
aucun shell interactif n'est chargé.

- [README](../../README.fr.md) — ce que fait jigger, et pourquoi chaque choix a été fait ainsi.
- [CHANGELOG](../../CHANGELOG.md) — ce qui a changé d'une version à l'autre.
- `docs/` — les décisions d'architecture (ADR), les conceptions en cours et le journal du
  projet.
