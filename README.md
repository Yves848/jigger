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

## Sous la capot (CLI)

Le plugin s'appuie sur ces sous-commandes ; utilisables seules :

```sh
jigger render --line "brew install fire" --sel 0 --cols 80   # une frame du popup vivant
                                 # 1re ligne : count=… sel=… exec=… left=<ligne complétée>
jigger complete "install fire"   # candidats, un par ligne (complétion classique)
jigger pick "brew uninstall wg"  # sélecteur interactif ; imprime la nouvelle ligne
                                 # code retour : 0 = insérer, 10 = exécuter, 2 = annulé
jigger demo                      # aperçu statique coloré du sélecteur
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
