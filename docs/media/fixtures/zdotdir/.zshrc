# Fixture de capture — zsh
#
# Ce fichier n'est PAS une configuration d'utilisateur : c'est le décor figé dans
# lequel toutes les captures de jigger sont tournées, pour que celle de macOS et
# celle d'Omarchy soient comparables au pixel près. Il est chargé par VHS via
# ZDOTDIR (cf. docs/captures.md), et ne touche jamais au ~/.zshrc de la machine.

# Un prompt d'un seul caractère, celui des cadres de la documentation. Pas de git,
# pas de chemin, pas de durée : tout ce qui varie d'une machine à l'autre est
# écarté, sinon la capture daterait et localiserait la machine qui l'a produite.
PROMPT='%F{#89b4fa}❯%f '
RPROMPT=''

# L'historique est la principale source de non-déterminisme d'un shell : ↑ y pioche
# ce que la machine a tapé avant. On le coupe, plutôt que de le vider.
HISTFILE=/dev/null
HISTSIZE=0
SAVEHIST=0

setopt no_beep
unsetopt correct correct_all

# La langue de la capture. Le tape la pose (Set Env JIGGER_LANG fr) ; sans elle,
# l'anglais, qui est la langue des documents de référence.
: "${JIGGER_LANG:=en}"
export JIGGER_LANG

# Le greffon, pris dans le dépôt et non dans l'installation de la machine : une
# capture doit montrer le code qu'on est en train de documenter.
source "${JIGGER_REPO:?JIGGER_REPO doit pointer sur la racine du dépôt}/shell/jigger.plugin.zsh"
