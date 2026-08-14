# jigger

**Assistance aux gestionnaires de paquets dans le terminal** — complétion contextuelle
et sélecteur interactif, dans _ton_ vrai shell.

`jigger` est un petit binaire Go autonome (démarrage quasi instantané) branché dans le
shell : dès que tu tapes une commande de gestionnaire de paquets, un **sélecteur**
([Bubble Tea] / [Lip Gloss]) s'affiche sous le prompt et suit ta frappe, proposant les
bons candidats selon le contexte. ⇥ insère le candidat courant dans la ligne ; tu n'as
jamais à demander.

| Plateforme | Shell | Gestionnaires |
|---|---|---|
| macOS, Linux | zsh (`shell/jigger.plugin.zsh`) | [Homebrew](https://brew.sh) |
| Windows | PowerShell (`shell/jigger.psm1`) | [winget](https://learn.microsoft.com/windows/package-manager/), [scoop](https://scoop.sh) |

C'est le **premier mot de la ligne** qui décide : `brew`, `winget` ou `scoop`. Chacun
apporte ses sous-commandes, ses options et son catalogue ; tout le reste — le popup, les
touches, le bloc de prompt — est commun.

Compagnon en ligne de commande de l'app GUI **Cocktails**, mais **totalement indépendant** :
il ne requiert que le gestionnaire lui-même.

## Ce qu'il fait

- **Complétion contextuelle**
  - premier mot → sous-commandes (`install`, `uninstall`, `search`…) ;
  - après `install`, `show`, `info`… → **tous** les paquets connus ;
  - après `uninstall`, `upgrade`, `pin`… → seulement les paquets **installés** ;
  - après `-` → les **options** de la sous-commande (`winget install --exact`,
    `brew list --versions`…).
- **Badges** et **indicateur « installé »** dans le sélecteur : ◆ pour le cas ordinaire
  (formula, paquet du catalogue winget, bucket `main`), ▣ pour l'autre (cask, application
  détectée hors catalogue, bucket tiers).
- **Corrections automatiques** — celles qui évitent une commande fautive :
  - brew : choisir un cask « pur » derrière `install`/`reinstall` insère `--cask <nom>` ;
  - scoop : un nom présent dans plusieurs buckets s'insère qualifié, `main/flux` ;
  - winget : un identifiant contenant des espaces s'insère entre guillemets.
- **Popup vivant** : le cadre apparaît dès « `winget ` » et se filtre au fil de la frappe,
  sans presser la moindre touche. `↓` fait entrer dans la liste, `⇥` insère, `^G` ferme.
- **Focus explicite** : le popup ne prend les flèches qu'une fois qu'on y est entré. `↓`
  l'y fait entrer, `↑` en ressort dès le premier candidat — et tant qu'il n'a pas le
  clavier, `↑`/`↓` restent l'historique du shell. La ligne courante le montre : soulignée
  quand le popup a le clavier, au repos quand il ne l'a pas.
- **Bloc oh-my-posh** (optionnel) : version du gestionnaire et mises à jour en attente
  dans le prompt, comptées séparément — sans jamais le ralentir.

## Installation

```sh
# le binaire (Go ≥ 1.24)
go install gitlab.yg-devworks.com/yves/jigger@latest   # → $GOBIN/jigger
#   ou :  git clone … && make install
```

Le binaire `jigger` doit être dans le `PATH`.

### zsh (Homebrew)

```sh
# dans ~/.zshrc
source /chemin/vers/jigger/shell/jigger.plugin.zsh
```

Recharge ton shell (`exec zsh`).

### PowerShell (winget, scoop)

```powershell
# dans $PROFILE  (notepad $PROFILE pour l'ouvrir)
Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
```

Recharge ton shell (`. $PROFILE`, ou un nouvel onglet). PowerShell 7 est recommandé ;
PSReadLine est requis (il est livré avec Windows).

Deux réserves, à connaître :

- le **mode Vi** de PSReadLine désactive le popup vivant (⇥ reste disponible) : relayer
  les caractères imprimables casserait la navigation en mode commande ;
- `PredictionViewStyle = ListView` dessine au même endroit que le popup. jigger range la
  vue le temps du cadre — la prédiction repasse en `InlineView`, à la suite du curseur —
  et la lui rend dès qu'il s'efface. Sans quoi les deux se disputeraient les mêmes lignes
  à chaque frappe.

## Usage

Tape simplement une commande — le popup vit tout seul :

```
winget ␣                 → les sous-commandes
winget install Git.      → les paquets « Git.… », mis à jour à chaque lettre
scoop uninstall ␣        → les applications installées
winget list --           → les options de list
brew install fire        → idem, côté macOS
```

| Touche | Effet |
|---|---|
| `⇥` | insère le candidat courant (corrigé si besoin) |
| `↓` | entre dans la liste, puis descend d'un candidat |
| `↑` | remonte ; au premier candidat, rend le clavier au shell |
| `^N` / `^P` | les mêmes, pour qui les préfère aux flèches |
| `^G` | ferme le popup pour la ligne en cours (`⇥` le rouvre) |

Tant que le popup n'a pas le clavier, `↑` et `↓` sont **l'historique du shell**, popup
ouvert ou non : ouvrir une liste de candidats ne coûte pas l'accès à la commande
précédente. Ce qu'elles feront se lit dans le cadre — pied `↓ parcourir` et ligne
courante au repos tant qu'il n'a pas le focus, `↑↓ naviguer` et ligne soulignée dès
qu'il l'a. Et jigger rend toujours la touche à ce qu'elle faisait avant lui : si un autre
greffon tient déjà tes flèches (recherche par préfixe dans l'historique, par exemple),
c'est lui qui reprend la main.

Après `winget install`, le mot est vide et le catalogue compte des milliers d'entrées :
le cadre invite alors à taper au moins une lettre plutôt que d'égrener la liste.

### Réglages

Identiques dans les deux shells — à poser **avant** le `source` / l'`Import-Module` :

```sh
JIGGER_LIVE=0     # désactive le popup vivant : ⇥ ouvre le sélecteur plein écran
JIGGER_ROWS=12    # candidats affichés (défaut 8 ; réduit si le terminal est court)
JIGGER_KEY='^ '   # touche d'insertion (défaut Tab)
```

```powershell
$env:JIGGER_LIVE = '0'
$env:JIGGER_ROWS = '12'
$env:JIGGER_KEY  = 'Ctrl+Spacebar'   # noms de touches PSReadLine
$env:JIGGER_COMMANDS = 'winget,scoop'      # commandes qui déclenchent le popup
$env:JIGGER_KEYS_EXTRA = 'éèçàù'           # touches à relayer en plus des ASCII
```

Le popup s'efface de lui-même si le terminal est trop étroit — et, sous zsh, s'il ne
répond pas à l'interrogation de position du curseur.

Quand le prompt occupe la dernière ligne de l'écran — le cas ordinaire d'un terminal en
usage —, **jigger pousse l'écran** pour dégager la place du cadre, comme le fait
`fzf --height`. C'est vrai des deux shells : sans cela, le popup ne s'afficherait pour
ainsi dire jamais.

Chacun des deux greffons **vérifie la version du binaire** au chargement. Greffon et
binaire vont par paire : un binaire plus ancien ne connaît pas les options que le greffon
lui passe, il sort en erreur, et le popup ne s'affiche jamais — sans un mot. Il le dit
désormais.

`JIGGER_KEYS_EXTRA` mérite un mot : PSReadLine n'offre aucun crochet appelé à chaque
frappe. jigger réenregistre donc, une à une, les touches qui modifient la ligne — les
ASCII imprimables, plus celles de cette liste. Sur un clavier AZERTY, la rangée des
chiffres non pressée donne « éèçàù » : d'où le réglage, et sa valeur par défaut.

## Bloc oh-my-posh

Un bloc dans le prompt : la **version du gestionnaire**, et les **mises à jour en
attente**, comptées séparément.

```
 yves@MacBook  ~/git/jigger   main  🍺 6.0.17  🧪 7  📦 2 ❯      ← macOS
 PS D:\jigger  🪟 1.29.280  📦 48  🥄 1 ❯                        ← Windows
```

Sur macOS : une **bière** pour brew, une **éprouvette** pour les formulae, un **colis**
pour les casks. Sous Windows : une **fenêtre** pour la version de winget, un **colis**
pour les paquets winget à mettre à niveau, une **cuillère** pour les applications scoop.

Chaque compteur disparaît quand il tombe à zéro — ` 1.29.280  🥄 1` s'il ne reste que
scoop, ` 1.29.280` tout court quand tout est à jour. Un compteur ne s'affichant **jamais**
à zéro, sa seule présence signifie « à mettre à jour » : ni flèche ni lettre à ajouter.

Ce sont des **émojis** : aucune police particulière n'est requise, et ils s'affichent
partout. Si tu préfères des glyphes **Nerd Font** monochromes — qui prennent la couleur du
segment, là où un émoji impose la sienne —, chaque fichier de segment indique les
codes correspondants.

Dans les deux cas, écris-les en **échappements JSON** plutôt qu'en clair : c'est la seule
forme qui traverse sans dommage les éditeurs, les copier-coller et les outils qui
normalisent l'Unicode. Les thèmes livrés par oh-my-posh font de même.

Compter les mises à jour coûte de une à cinq secondes — `brew outdated` comme
`winget list --upgrade-available` : c'est donc **exclu du chemin du prompt**. jigger le
lance en tâche de fond et dépose le résultat dans un fichier d'une ligne, que le hook
relit avec les seules primitives du shell — **aucun processus lancé, aucune attente**. La
valeur affichée est celle du dernier calcul ; passé `JIGGER_PROMPT_TTL`, un
rafraîchissement part détaché et le prompt suivant est à jour.

Ce TTL seul laisserait le compteur mentir une demi-heure après un `winget upgrade`. jigger
repère donc les commandes qui changent l'état — `install`, `upgrade`, `uninstall`,
`update`, `bucket`… — au moment où elles partent, et rafraîchit **dans la foulée** : le
prompt qui suit l'upgrade est déjà juste. C'est la seule fois où le prompt attend quelque
chose (moins d'une seconde en général, après une commande qui a duré bien plus) ;
`JIGGER_PROMPT_SYNC=0` rend ce rafraîchissement détaché, au prix d'un compteur juste au
prompt d'*après*.

Seul ce que le shell voit passer est détecté : une mise à jour lancée ailleurs — autre
onglet, application graphique — reste rattrapée par le TTL.

oh-my-posh n'ayant plus de segment `command` depuis la v26, tout passe par des variables
d'environnement et un segment `text`.

**1. Activer le hook** — *avant* le chargement du greffon :

```sh
JIGGER_PROMPT=1                                    # ~/.zshrc
source /chemin/vers/jigger/shell/jigger.plugin.zsh
```

```powershell
$env:JIGGER_PROMPT = '1'                           # $PROFILE
Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
```

Sous PowerShell, `prompt` est le seul « precmd » disponible : jigger **enveloppe** celui
qui est en place. Importe donc jigger **après** oh-my-posh, sans quoi le bloc aurait
toujours un coup de retard. (Sous zsh, l'ordre des `source` n'a pas d'importance : le
hook se place de lui-même en tête de `precmd_functions`.)

**2. Ajouter le segment** — les thèmes livrés avec oh-my-posh sont écrasés à chaque mise
à jour : travaille sur une copie.

```sh
mkdir -p ~/.config/oh-my-posh
cp "$(brew --prefix oh-my-posh)/themes/catppuccin_mocha.omp.json" \
   ~/.config/oh-my-posh/mon-theme.omp.json
```

Colle le contenu de [`shell/oh-my-posh/brew.segment.json`](shell/oh-my-posh/brew.segment.json)
— ou de [`windows.segment.json`](shell/oh-my-posh/windows.segment.json) — dans le tableau
`segments` du bloc voulu, puis fais pointer ton profil sur ta copie :

```sh
eval "$(oh-my-posh init zsh --config ~/.config/oh-my-posh/mon-theme.omp.json)"
```

Le segment n'affiche **rien** tant que le cache n'existe pas — le bloc apparaît au
deuxième prompt, une fois le premier rafraîchissement terminé.

### Réglages

```sh
JIGGER_PROMPT=1        # active le bloc (défaut 0)
JIGGER_PROMPT_TTL=1800 # âge du cache, en secondes, avant rafraîchissement (défaut 30 min)
JIGGER_PROMPT_SYNC=1   # après une commande mutante, rafraîchir avant d'afficher le
                       # prompt (défaut) ; 0 = détaché, juste au prompt suivant
JIGGER_CACHE_DIR=…     # emplacement du cache (défaut ~/Library/Caches/jigger sur macOS,
                       # %LOCALAPPDATA%\jigger sous Windows)
```

### Variables exposées

| Variable | Contenu |
|---|---|
| `JIGGER_BREW_VERSION` | version de brew, sans suffixe de commits : `6.0.17` |
| `JIGGER_BREW_FORMULAE` | formulae obsolètes |
| `JIGGER_BREW_CASKS` | casks obsolètes |
| `JIGGER_BREW_OUTDATED` | total des deux |
| `JIGGER_WINGET_VERSION` | version de winget : `1.29.280` |
| `JIGGER_WINGET_OUTDATED` | paquets winget à mettre à niveau |
| `JIGGER_SCOOP_OUTDATED` | applications scoop à mettre à niveau |
| `JIGGER_OUTDATED` | total des deux |

Un compteur **n'est pas défini** quand il vaut zéro. Le template se réduit ainsi à un
`{{ if .Env.JIGGER_WINGET_OUTDATED }}`, sans comparaison de chaînes — et le bloc s'efface
tout seul quand il n'y a rien à dire. Pour n'afficher qu'un chiffre plutôt que le détail
par gestionnaire, remplace les deux blocs du template par :

```
{{ if .Env.JIGGER_OUTDATED }} <#F9E2AF>\u21e1{{ .Env.JIGGER_OUTDATED }}</>{{ end }}
```

Rien n'interdit de se servir de ces variables ailleurs que dans oh-my-posh (starship, un
prompt maison…). Sous PowerShell, `Update-JiggerPrompt` est exportée exprès : appelle-la
depuis ta propre fonction `prompt`.

## Sous le capot (CLI)

Le greffon s'appuie sur ces sous-commandes ; utilisables seules :

```sh
jigger render --line "winget install Git." --sel 0 --cols 80   # une frame du popup vivant
                                 # 1re ligne : count=… sel=… exec=… left=<ligne complétée>
                                 # --focus=true : le popup a le clavier (cf. § Usage)
jigger complete "install fire"   # candidats, un par ligne (complétion classique)
jigger pick "scoop uninstall 7z" # sélecteur interactif ; imprime la nouvelle ligne
                                 # code retour : 0 = insérer, 10 = exécuter, 2 = annulé
jigger demo                      # aperçu statique coloré du sélecteur
jigger prompt                    # état en cache : version⇥compteur1⇥compteur2⇥epoch
jigger prompt --refresh          # interroge le gestionnaire et réécrit le cache (lent)
jigger prompt --path             # chemin du fichier de cache
jigger warm                      # reconstitue les catalogues périmés (lent, détaché)
jigger warm --installed          # les seules listes de paquets installés
jigger warm --all                # tout, périmé ou non
```

`render` est **sans état** : l'index sélectionné vit côté shell et lui revient par
`--sel`. C'est ce qui permet au greffon de rester maître du clavier — le shell garde sa
ligne, jigger ne fait qu'imprimer un cadre.

**Rien de lent n'est jamais dans le chemin d'un rendu** (~8 ms de travail, ~30 ms de bout
en bout sous Windows, où démarrer un processus coûte le plus cher). Chaque gestionnaire y
parvient à sa façon :

| | catalogue | installés | obsolètes |
|---|---|---|---|
| **brew** | `brew formulae` / `casks`, en cache 24 h | lus dans `Cellar`/`Caskroom` (~1 ms) | `brew outdated --json=v2`, en tâche de fond |
| **scoop** | lu dans `buckets/*/bucket/*.json` | lus dans `apps/<nom>/<version>` | comparaison des manifestes, sur le disque |
| **winget** | `winget search .`, en cache 24 h | `winget list`, en cache 10 min | `winget list --upgrade-available`, en tâche de fond |

scoop n'a besoin d'**aucun cache** : tout ce que jigger lui demande est déjà sur le
disque, dans une arborescence qui ressemble à s'y méprendre au `Cellar` de Homebrew. Même
le compte des mises à jour en attente s'y lit — c'est ce que fait `scoop status`, mais
sans démarrer PowerShell ni toucher au réseau.

winget est à l'opposé : aucune sortie machine (que des tableaux de largeur fixe, aux
en-têtes **traduits** — d'où un découpage aux frontières de colonnes, et des jeux d'essai
en français), et près de deux secondes par appel. Ses deux listes sont donc tenues en
cache et reconstituées par `jigger warm`, que `render` lance détaché dès qu'il les trouve
périmées. Le catalogue s'obtient en cherchant `.` : le point qui sépare l'éditeur du
paquet dans tous les identifiants de la source officielle (`Git.Git`,
`Microsoft.PowerShell`) — soit, ici, 14 401 noms.

## Tests

```sh
make test-all     # tests Go + suite du shell de la plateforme
```

Le widget zsh ne se teste que dans un vrai pseudo-terminal : `tests/zpty.zsh` lance un zsh
interactif sous `zpty`, tape une séquence de touches et vérifie ce qui est réellement
écrit à l'écran. `JIGGER_TEST_PLUGINS=1` y ajoute zsh-autosuggestions et
zsh-syntax-highlighting, pour prouver qu'ils cohabitent.

Sous Windows, `tests/conpty` joue le même rôle : il lance un pwsh dans un **pseudo-terminal**
(ConPTY), tape une séquence de touches et **rend l'écran** tel qu'on le verrait, cadre
compris. `tests/pty.ps1` (`make test-pty`) en tire ses assertions, dans la situation qui
compte — prompt sur la dernière ligne de l'écran, comme dans un terminal en usage.

```sh
go run ./tests/conpty -rc essai.ps1 -keys 'winget ins\t' -screen   # l'écran final
```

`tests/smoke.ps1` couvre ce qui ne demande pas de console : les touches effectivement
reprises, l'analyse de la sortie de `render`, les séquences d'affichage et d'effacement,
la détection des commandes mutantes, et l'export des variables du prompt depuis un cache
fabriqué.

## Feuille de route

- Complétion **fish** et **bash**.
- Wrapper de commande : proposer d'**enchaîner** sur les commandes suggérées par le
  gestionnaire (« To install …, run: … »).
- Aperçu (`brew desc`, `winget show`) dans le sélecteur.
- Distribution comme **commande externe brew** (`brew jigger`) via un tap, et paquet
  **scoop** / **winget**.

## Licence

Apache-2.0.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
