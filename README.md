# jigger

**Assistance [Homebrew](https://brew.sh) dans le terminal** — complétion contextuelle
et sélecteur interactif, dans _ton_ vrai shell.

`jigger` est un petit binaire Go autonome (démarrage quasi instantané) branché dans zsh :
quand tu tapes une commande `brew`, une touche ouvre un **sélecteur** ([Bubble Tea] /
[Lip Gloss]) qui propose les bons candidats selon le contexte, puis les insère dans la
ligne — ou exécute la commande si elle est complète.

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
- **⇥ insérer / ↩ exécuter** : Tab insère le choix (tu continues d'éditer) ; Entrée
  exécute quand la commande est complète (contexte paquet).

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

Tape une commande brew et appuie sur **Tab** :

```
brew install fire⇥      → sélecteur des paquets « fire… » ; ↩ installe (⌘cask auto)
brew uninstall ⇥        → sélecteur des paquets installés
brew list --⇥           → sélecteur des options de list
```

La touche est configurable (défaut Tab) :

```sh
JIGGER_KEY='^ '   # Ctrl-Espace, avant le `source`
```

## Sous la capot (CLI)

Le plugin s'appuie sur deux sous-commandes ; utilisables seules :

```sh
jigger complete "install fire"   # candidats, un par ligne (complétion classique)
jigger pick "brew uninstall wg"  # sélecteur interactif ; imprime la nouvelle ligne
                                 # code retour : 0 = insérer, 10 = exécuter, 2 = annulé
jigger demo                      # aperçu statique coloré du sélecteur
```

Le catalogue (`brew formulae` / `brew casks`) est mis en cache 24 h sous
`~/.cache/jigger` ; la liste des installés est relue à chaque appel (rapide).

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
