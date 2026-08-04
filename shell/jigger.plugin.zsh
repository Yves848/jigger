# jigger.plugin.zsh — assistance Homebrew dans zsh.
#
# Lie Tab (^I) à un sélecteur interactif (jigger) quand la ligne courante est une
# commande brew ; sinon la complétion zsh normale s'applique. Le sélecteur insère
# le paquet/l'option choisi ; sur ↩ dans une commande complète, il l'exécute.
#
# Installation :
#   source /chemin/vers/jigger.plugin.zsh   # dans ~/.zshrc
# (le binaire `jigger` doit être dans le PATH.)

# Touche déclenchant le sélecteur (par défaut Tab). Surcharge possible avant le source :
#   JIGGER_KEY='^ '   # Ctrl-Espace
: "${JIGGER_KEY:=^I}"

_jigger_widget() {
  # N'assiste que les commandes brew ; sinon complétion zsh standard.
  if [[ "$BUFFER" != "brew" && "$BUFFER" != brew\ * ]]; then
    zle expand-or-complete
    return
  fi

  local out ret
  out="$(command jigger pick "$BUFFER")"
  ret=$?

  # 2 = annulé, ou sortie vide : on ne touche à rien.
  if (( ret == 2 )) || [[ -z "$out" ]]; then
    zle reset-prompt
    return
  fi

  BUFFER="$out"
  CURSOR=${#BUFFER}
  zle reset-prompt

  # 10 = la commande est complète → on l'exécute directement (↩ dans le sélecteur).
  if (( ret == 10 )); then
    zle accept-line
  fi
}

zle -N _jigger_widget
bindkey "$JIGGER_KEY" _jigger_widget
