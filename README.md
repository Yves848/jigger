# jigger

**Assistance [Homebrew](https://brew.sh) dans le terminal** — complétion contextuelle
et sélecteur interactif, dans _ton_ vrai shell.

`jigger` est un petit binaire Go autonome (démarrage quasi instantané) branché dans zsh :
dès que tu tapes une commande `brew`, un **sélecteur** ([Bubble Tea] / [Lip Gloss])
s'affiche sous le prompt et suit ta frappe, proposant les bons candidats selon le
contexte. ⇥ insère le candidat courant dans la ligne ; tu n'as jamais à demander.

Compagnon en ligne de commande de l'app GUI **Cocktails**, mais **totalement indépendant** :
il ne requiert que `brew`.

## Ce qu'il fait

- **Complétion contextuelle**
  - premier mot → sous-commandes brew (`install`, `uninstall`, `search`…) ;
  - après `install`, `info`, `deps`… → **tous** les paquets (formulae + casks) ;
  - après `uninstall`, `reinstall`, `pin`… → seulement les **paquets installés** ;
  - après `-` → les **options** de la sous-commande (`list --versions`, `install --cask`…).
- **Badges** F / C (formula / cask) et **indicateur « installé »** dans le sélecteur.
- **`--cask` automatique** : choisir un cask « pur » derrière `install`/`reinstall` insère
  `--cask <nom>` (évite l'erreur brew « use --cask »).
- **Popup vivant** : le cadre apparaît dès « `brew ` » et se filtre au fil de la frappe,
  sans presser la moindre touche. `^N`/`^P` naviguent, `⇥` insère, `^G` ferme.
  Les flèches `↑`/`↓` ne sont **jamais** détournées : l'historique zsh reste l'historique
  zsh.
- **Bloc oh-my-posh** (optionnel) : version de brew et mises à jour en attente (formulae
  et casks séparément) dans le prompt, sans jamais le ralentir.

## Installation

```sh
# 1. le binaire (Go ≥ 1.24)
go install gitlab.yg-devworks.com/yves/jigger@latest   # → $GOBIN/jigger
#   ou :  git clone … && make install

# 2. le plugin zsh (dans ~/.zshrc)
source /chemin/vers/jigger/shell/jigger.plugin.zsh
```

Le binaire `jigger` doit être dans le `PATH`. Recharge ton shell (`exec zsh`).

## Usage

Tape simplement une commande brew — le popup vit tout seul :

```
brew ␣                  → les sous-commandes
brew install fire       → les paquets « fire… », mis à jour à chaque lettre
brew uninstall ␣        → les paquets installés
brew list --            → les options de list
```

| Touche | Effet |
|---|---|
| `⇥` | insère le candidat courant (`--cask` ajouté si besoin) |
| `^N` / `^P` | candidat suivant / précédent |
| `^G` | ferme le popup pour la ligne en cours (`⇥` le rouvre) |
| `↑` / `↓` | **inchangées** — historique zsh |

Après `brew install`, le mot est vide et le catalogue compte des milliers d'entrées :
le cadre invite alors à taper au moins une lettre plutôt que d'égrener la liste.

### Réglages

```sh
JIGGER_LIVE=0     # désactive le popup vivant : ⇥ ouvre le sélecteur plein écran
JIGGER_ROWS=12    # candidats affichés (défaut 8 ; réduit si le terminal est court)
JIGGER_KEY='^ '   # touche d'insertion (défaut Tab)
```

à poser **avant** le `source`. Le popup s'efface de lui-même si le terminal est trop
étroit, trop court, ou ne répond pas à l'interrogation de position du curseur.

## Bloc oh-my-posh

Un bloc Homebrew dans le prompt : la **version de brew**, et les **mises à jour en
attente**, formulae et casks comptés séparément.

```
 yves@MacBook  ~/git/jigger   main   6.0.17 ⇡7F ⇡2C ❯
                                     ▲       ▲    ▲
                                     │       │    └─ 2 casks obsolètes
                                     │       └────── 7 formulae obsolètes
                                     └────────────── version de brew
```

Chaque compteur disparaît quand il tombe à zéro — ` 6.0.17 ⇡2C` s'il ne reste que des
casks, ` 6.0.17` tout court quand tout est à jour.

`brew outdated` coûte de une à cinq secondes : il est donc **exclu du chemin du prompt**.
jigger le lance en tâche de fond et dépose le résultat dans un fichier d'une ligne, que le
hook `precmd` relit avec les seuls builtins de zsh — **0,03 ms par prompt, aucun fork**. La
valeur affichée est celle du dernier calcul ; passé `JIGGER_PROMPT_TTL`, un rafraîchissement
part détaché et le prompt suivant est à jour.

oh-my-posh n'ayant plus de segment `command` depuis la v26, tout passe par deux variables
d'environnement et un segment `text`.

**1. Activer le hook** — dans `~/.zshrc`, *avant* le `source` :

```sh
JIGGER_PROMPT=1
source /chemin/vers/jigger/shell/jigger.plugin.zsh
```

**2. Ajouter le segment** — les thèmes livrés par Homebrew sont écrasés à chaque mise à
jour : travaille sur une copie.

```sh
mkdir -p ~/.config/oh-my-posh
cp "$(brew --prefix oh-my-posh)/themes/catppuccin_mocha.omp.json" \
   ~/.config/oh-my-posh/mon-theme.omp.json
```

Colle le contenu de [`shell/oh-my-posh/brew.segment.json`](shell/oh-my-posh/brew.segment.json)
dans le tableau `segments` du bloc voulu, puis fais pointer `~/.zshrc` sur ta copie :

```sh
eval "$(oh-my-posh init zsh --config ~/.config/oh-my-posh/mon-theme.omp.json)"
```

Le segment n'affiche **rien** tant que le cache n'existe pas — le bloc apparaît au
deuxième prompt, une fois le premier rafraîchissement terminé.

### Réglages

```sh
JIGGER_PROMPT=1        # active le bloc (défaut 0)
JIGGER_PROMPT_TTL=1800 # âge du cache, en secondes, avant rafraîchissement (défaut 30 min)
JIGGER_CACHE_DIR=…     # emplacement du cache (défaut ~/Library/Caches/jigger)
```

### Variables exposées

| Variable | Contenu |
|---|---|
| `JIGGER_BREW_VERSION` | version de brew, sans suffixe de commits : `6.0.17` |
| `JIGGER_BREW_FORMULAE` | formulae obsolètes |
| `JIGGER_BREW_CASKS` | casks obsolètes |
| `JIGGER_BREW_OUTDATED` | total des deux |

Un compteur **n'est pas défini** quand il vaut zéro. Le template se réduit ainsi à un
`{{ if .Env.JIGGER_BREW_FORMULAE }}`, sans comparaison de chaînes — et le bloc s'efface
tout seul quand il n'y a rien à dire. Pour n'afficher qu'un chiffre plutôt que le détail
F/C, remplace les deux blocs du template par :

```
{{ if .Env.JIGGER_BREW_OUTDATED }} <#F9E2AF>⇡{{ .Env.JIGGER_BREW_OUTDATED }}</>{{ end }}
```

Rien n'interdit de se servir de ces variables ailleurs que dans oh-my-posh (starship, un
prompt maison…).

## Sous la capot (CLI)

Le plugin s'appuie sur ces sous-commandes ; utilisables seules :

```sh
jigger render --line "brew install fire" --sel 0 --cols 80   # une frame du popup vivant
                                 # 1re ligne : count=… sel=… exec=… left=<ligne complétée>
jigger complete "install fire"   # candidats, un par ligne (complétion classique)
jigger pick "brew uninstall wg"  # sélecteur interactif ; imprime la nouvelle ligne
                                 # code retour : 0 = insérer, 10 = exécuter, 2 = annulé
jigger demo                      # aperçu statique coloré du sélecteur
jigger prompt                    # état brew en cache : version⇥formulae⇥casks⇥epoch
jigger prompt --refresh          # interroge brew et réécrit le cache (lent, détaché)
jigger prompt --path             # chemin du fichier de cache
```

`render` est **sans état** : l'index sélectionné vit côté shell et lui revient par
`--sel`. C'est ce qui permet au widget de rester maître du clavier — zsh garde sa ligne,
jigger ne fait qu'imprimer un cadre.

Le catalogue (`brew formulae` / `brew casks`) est mis en cache 24 h sous
`~/Library/Caches/jigger`. Les paquets installés sont lus directement dans
`Cellar`/`Caskroom` (~1 ms) plutôt que par `brew list --versions` (~300 ms) : c'est ce
qui rend tenable un rendu à chaque frappe (**~8 ms** par appel, de bout en bout).

## Tests

```sh
make test-all     # tests Go + suite zsh
```

Le widget ne se teste que dans un vrai pseudo-terminal : `tests/zpty.zsh` lance un zsh
interactif sous `zpty`, tape une séquence de touches et vérifie ce qui est réellement
écrit à l'écran. `JIGGER_TEST_PLUGINS=1` y ajoute zsh-autosuggestions et
zsh-syntax-highlighting, pour prouver qu'ils cohabitent.

## Feuille de route

- Complétion **fish** et **bash**.
- Wrapper `brew()` : proposer d'**enchaîner** sur les commandes suggérées par brew
  (« To install …, run: … »).
- Aperçu (`brew desc`) dans le sélecteur.
- Distribution comme **commande externe brew** (`brew jigger`) via un tap.

## Licence

Apache-2.0.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
