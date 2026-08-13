# jigger.plugin.zsh — assistance Homebrew dans zsh.
#
# Deux modes, complémentaires :
#
#   • popup vivant — dès que la ligne courante est une commande brew, le sélecteur
#     s'affiche sous le prompt et suit la frappe, sans rien demander. ^N/^P naviguent,
#     ⇥ insère le candidat courant, ^G ferme le popup pour la ligne en cours.
#   • Tab classique — quand le popup vivant est éteint (JIGGER_LIVE=0), ⇥ ouvre le
#     sélecteur interactif plein écran (`jigger pick`, avec son filtre en sous-chaîne).
#
# L'affichage est écrit directement sur /dev/tty, sous la ligne de commande. `zle -M`,
# qui aurait laissé zsh gérer l'affichage, n'est pas utilisable : il échappe les
# caractères de contrôle du message, donc les séquences de couleur s'afficheraient en
# clair (« ^[[38;2;… »). On dessine donc à la main, et pour ne jamais faire défiler
# l'écran sous les pieds de zsh — ce qui décalerait tout son affichage — on demande
# d'abord au terminal où est le curseur (DSR) et on n'affiche que ce qui tient dessous.
#
# Installation :
#   source /chemin/vers/jigger.plugin.zsh   # dans ~/.zshrc
# (le binaire `jigger` doit être dans le PATH.)

# Touche du sélecteur classique / d'insertion (par défaut Tab). Surcharge possible avant
# le source :  JIGGER_KEY='^ '   # Ctrl-Espace
: "${JIGGER_KEY:=^I}"
# Popup vivant. JIGGER_LIVE=0 revient au comportement « Tab seulement » — utile pour
# isoler un conflit avec un autre plugin de ligne d'édition.
: "${JIGGER_LIVE:=1}"
# Candidats affichés dans le popup vivant.
: "${JIGGER_ROWS:=8}"
# En dessous de cette largeur de terminal, le cadre n'a plus de sens : on n'affiche rien.
: "${JIGGER_MIN_COLUMNS:=30}"

autoload -Uz add-zle-hook-widget

typeset -g _jigger_sel=0        # candidat courant
typeset -g _jigger_selline=''   # ligne pour laquelle _jigger_sel a un sens
typeset -g _jigger_dismissed=0  # ^G : popup fermé jusqu'à la fin de la ligne
typeset -g _jigger_count=0      # candidats du dernier rendu
typeset -g _jigger_left=''      # LBUFFER après insertion du candidat courant
typeset -g _jigger_frame=''     # cadre déjà rendu (évite un fork par redisplay)
typeset -g _jigger_key=''       # signature du dernier rendu
typeset -g _jigger_shown=0      # un cadre est-il actuellement à l'écran ?
typeset -g _jigger_row_value='' # ligne écran du curseur (réponse au DSR)
typeset -gi _jigger_dsr_misses=0 # DSR restés sans réponse d'affilée

_jigger_is_brew() { [[ $LBUFFER == 'brew' || $LBUFFER == 'brew '* ]] }

# Le profil couleur ne peut pas être deviné par jigger : sa sortie est capturée. C'est
# donc le shell, qui connaît $COLORTERM et $TERM, qui tranche.
_jigger_color() {
  case $COLORTERM in
    (*truecolor*|*24bit*) print -r -- truecolor; return ;;
  esac
  case $TERM in
    (*256color*) print -r -- 256 ;;
    (dumb)       print -r -- never ;;
    (*)          print -r -- 16 ;;
  esac
}

# _jigger_fetch appelle le binaire et met à jour l'état. Renvoie non-zéro s'il n'y a rien
# à afficher.
_jigger_fetch() {
  local rows=${1:-$JIGGER_ROWS} out
  out=$(command jigger render --line "$LBUFFER" --sel "$_jigger_sel" --cols "$COLUMNS" \
        --rows "$rows" --color "$(_jigger_color)" 2>/dev/null) || return 1

  local -a lines
  lines=("${(@f)out}")
  (( ${#lines} > 1 )) || return 1

  # Métadonnées : des champs clé=valeur séparés par des tabulations, `left=` en dernier
  # car c'est le seul qui peut contenir des espaces.
  local meta=$lines[1]
  _jigger_left=${meta#*$'\t'left=}
  local -A m
  local kv
  for kv in ${(ps:\t:)${meta%$'\t'left=*}}; do m[${kv%%=*}]=${kv#*=}; done
  _jigger_count=${m[count]:-0}
  _jigger_sel=${m[sel]:-0}   # jigger a ramené l'index dans les bornes

  _jigger_frame=${(F)lines[2,-1]}
  return 0
}

# _jigger_row demande au terminal la ligne écran du curseur (DSR 6n) et la renvoie.
# Vide si le terminal ne répond pas — on s'abstient alors d'afficher plutôt que de
# dessiner à l'aveugle. Les octets lus qui ne font pas partie de la réponse sont des
# frappes de l'utilisateur arrivées entre-temps : on les rend à zle plutôt que de les
# perdre.
# Le résultat passe par une variable, pas par une substitution de commande : `zle -U`
# doit s'exécuter dans le shell du widget pour rendre les frappes rattrapées — dans un
# sous-shell, elles seraient purement perdues.
_jigger_row() {
  setopt local_options extendedglob
  _jigger_row_value=''
  local reply='' c
  print -n $'\e[6n' >/dev/tty
  while read -r -t 0.15 -k 1 c 2>/dev/null; do
    reply+=$c
    [[ $c == 'R' ]] && break
  done
  if [[ $reply == (#b)(*)$'\e['([0-9]##)';'[0-9]##'R' ]]; then
    [[ -n $match[1] ]] && zle -U -- "$match[1]"   # frappes arrivées entre-temps
    _jigger_row_value=$match[2]
    _jigger_dsr_misses=0
    return 0
  fi
  # Un terminal qui ne répond pas coûterait le délai d'attente à chaque frappe : au bout
  # de deux essais, on éteint le popup vivant pour la session (⇥ reste disponible).
  (( _jigger_dsr_misses += 1 ))
  if (( _jigger_dsr_misses >= 2 )); then
    JIGGER_LIVE=0
    _jigger_erase
  fi
  return 1
}

# _jigger_draw écrit le cadre sous la ligne de commande puis remet le curseur là où il
# l'a pris (DECSC/DECRC). Le contenu déjà là, sous le cadre, est effacé (ED) : un cadre
# précédent plus haut ne laisse donc pas de résidu.
_jigger_draw() {
  local frame=$1
  { print -n $'\e7\r\n'
    print -rn -- ${frame//$'\n'/$'\e[K\r\n'}
    print -n $'\e[K\e[J\e8'
  } >/dev/tty
  _jigger_shown=1
}

# _jigger_erase efface tout ce qui est sous la ligne de commande.
_jigger_erase() {
  (( _jigger_shown )) || return
  print -n $'\e7\r\n\e[J\e8' >/dev/tty
  _jigger_shown=0
}

# Popup effectivement à l'écran ?
_jigger_active() {
  (( JIGGER_LIVE )) || return 1
  (( _jigger_dismissed )) && return 1
  _jigger_is_brew || return 1
  (( _jigger_count > 0 ))
}

# Hook de redisplay : c'est lui qui fait vivre le popup. Il est appelé à chaque redessin
# de la ligne, donc à chaque frappe ; on ne relance le binaire que si la ligne, la
# sélection ou la géométrie ont bougé — le reste du temps, on réaffiche le cadre déjà
# rendu.
_jigger_redisplay() {
  (( JIGGER_LIVE )) || return
  if (( _jigger_dismissed )) || ! _jigger_is_brew || (( COLUMNS < JIGGER_MIN_COLUMNS )); then
    _jigger_erase
    return
  fi

  # Où est le curseur ? De sa position dépend le nombre de lignes qu'on peut occuper
  # sans faire défiler l'écran (ce qui décalerait tout l'affichage de zsh).
  _jigger_row || return
  # 5 lignes de décor : 2 bordures, l'en-tête, la respiration, le pied.
  local -i fits=$(( LINES - _jigger_row_value - 5 ))
  if (( fits < 1 )); then
    _jigger_erase
    return
  fi
  local -i rows=$(( fits < JIGGER_ROWS ? fits : JIGGER_ROWS ))

  # Changer la ligne change la liste : le candidat courant redevient le premier.
  if [[ $LBUFFER != $_jigger_selline ]]; then
    _jigger_sel=0
    _jigger_selline=$LBUFFER
  fi

  local key="$LBUFFER|$_jigger_sel|$COLUMNS|$rows"
  if [[ $key != $_jigger_key ]]; then
    _jigger_key=$key
    _jigger_fetch $rows || { _jigger_frame=''; _jigger_count=0; _jigger_erase; return }
  fi
  [[ -n $_jigger_frame ]] && _jigger_draw "$_jigger_frame"
}

# ^N / ^P : naviguent si le popup est là, sinon rendent la main au widget d'origine
# (historique). Les flèches ↑↓ ne sont jamais touchées.
_jigger_next() {
  if _jigger_active; then
    (( _jigger_sel++ ))
    _jigger_selline=$LBUFFER
  else
    zle .down-line-or-history
  fi
}
_jigger_prev() {
  if _jigger_active; then
    (( _jigger_sel > 0 )) && (( _jigger_sel-- ))
    _jigger_selline=$LBUFFER
  else
    zle .up-line-or-history
  fi
}

# ^G : ferme le popup jusqu'à la fin de la ligne. Sans popup, c'est l'abandon habituel.
_jigger_dismiss() {
  if _jigger_active; then
    _jigger_dismissed=1
    _jigger_frame=''
    _jigger_erase
  else
    zle .send-break
  fi
}

# ⇥ : insère le candidat courant si le popup est là ; sinon sélecteur classique sur une
# ligne brew (ou popup refermé), et complétion zsh normale partout ailleurs.
_jigger_widget() {
  if _jigger_active; then
    LBUFFER=$_jigger_left
    _jigger_key=''          # la ligne a changé : le cadre est à refaire
    _jigger_selline=$LBUFFER
    _jigger_sel=0
    return
  fi

  if ! _jigger_is_brew; then
    zle expand-or-complete
    return
  fi

  # Popup fermé par ^G : ⇥ le rouvre plutôt que d'ouvrir le sélecteur plein écran.
  if (( JIGGER_LIVE )) && (( _jigger_dismissed )); then
    _jigger_dismissed=0
    _jigger_key=''
    return
  fi

  local out ret
  out="$(command jigger pick "$LBUFFER")"
  ret=$?

  # 2 = annulé, ou sortie vide : on ne touche à rien.
  if (( ret == 2 )) || [[ -z "$out" ]]; then
    zle reset-prompt
    return
  fi

  LBUFFER="$out"
  zle reset-prompt

  # 10 = la commande est complète → on l'exécute directement (↩ dans le sélecteur).
  (( ret == 10 )) && zle accept-line
}

# Remise à zéro à chaque nouvelle ligne : ^G ne vaut que pour la ligne où il a été frappé.
# ⏎ : la commande va s'exécuter et écrire sous le prompt — on efface le cadre avant,
# sinon sa sortie se mélangerait au cadre.
_jigger_line_finish() { _jigger_erase }

_jigger_line_init() {
  _jigger_shown=0   # nouvelle ligne : l'écran a défilé, il n'y a plus rien à effacer
  _jigger_sel=0
  _jigger_selline=''
  _jigger_dismissed=0
  _jigger_count=0
  _jigger_frame=''
  _jigger_key=''
}

zle -N _jigger_widget
zle -N _jigger_next
zle -N _jigger_prev
zle -N _jigger_dismiss

bindkey "$JIGGER_KEY" _jigger_widget
if (( JIGGER_LIVE )); then
  bindkey '^N' _jigger_next
  bindkey '^P' _jigger_prev
  bindkey '^G' _jigger_dismiss
  add-zle-hook-widget line-pre-redraw _jigger_redisplay
  add-zle-hook-widget line-init _jigger_line_init
  add-zle-hook-widget line-finish _jigger_line_finish
fi

# ── Bloc Homebrew pour oh-my-posh ─────────────────────────────────────────────────────
#
# Exporte à chaque prompt JIGGER_BREW_VERSION et JIGGER_BREW_OUTDATED, qu'un segment
# `text` d'oh-my-posh affiche (cf. shell/oh-my-posh/brew.segment.json). Désactivé par
# défaut : ça ne sert qu'à ceux qui utilisent oh-my-posh.
#
#   JIGGER_PROMPT=1
#   source /chemin/vers/jigger.plugin.zsh
#
# Règle qui gouverne tout ce qui suit : **aucun fork dans le chemin normal**. `brew
# outdated` coûte plusieurs secondes ; il tourne en tâche de fond (`jigger prompt
# --refresh`) et dépose son résultat dans un fichier d'une ligne, que ce hook relit avec
# les seuls builtins de zsh. Un prompt ne doit rien attendre.
: "${JIGGER_PROMPT:=0}"
# Âge du cache (secondes) au-delà duquel on relance un rafraîchissement détaché.
: "${JIGGER_PROMPT_TTL:=1800}"

if (( JIGGER_PROMPT )); then
  zmodload -i zsh/datetime          # $EPOCHSECONDS, pour ne pas forker `date`
  autoload -Uz add-zsh-hook

  # Même règle que os.UserCacheDir() côté Go, qui sur macOS ignore XDG_CACHE_HOME.
  # `jigger prompt --path` donne le chemin exact — mais l'interroger coûterait un fork.
  if [[ -z $JIGGER_CACHE_DIR ]]; then
    if [[ $OSTYPE == darwin* ]]; then
      typeset -g JIGGER_CACHE_DIR="$HOME/Library/Caches/jigger"
    else
      typeset -g JIGGER_CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/jigger"
    fi
  fi
  typeset -g _jigger_status_file="$JIGGER_CACHE_DIR/status"
  typeset -gi _jigger_last_refresh=0

  _jigger_prompt_precmd() {
    local line
    local -a champs
    local -i age=$JIGGER_PROMPT_TTL   # cache absent → réputé périmé

    if [[ -r $_jigger_status_file ]] && read -r line < $_jigger_status_file; then
      # Découpage sur tabulation par expansion (pas de `cut`). Le drapeau `p` fait
      # interpréter \t dans le séparateur : pas de tabulation littérale dans ce
      # fichier, qu'un éditeur pourrait convertir en espaces.
      champs=( "${(@ps.\t.)line}" )
      # `<->` = suite de chiffres : une ligne bricolée à la main ne doit pas se
      # retrouver dans une expansion arithmétique.
      if (( $#champs == 4 )) && [[ "$champs[2]$champs[3]$champs[4]" == <-> ]]; then
        export JIGGER_BREW_VERSION="$champs[1]"
        # Le compteur n'est exporté que s'il y a quelque chose à signaler : le template
        # oh-my-posh se réduit alors à « {{ if .Env.JIGGER_BREW_OUTDATED }} ».
        if (( champs[2] + champs[3] > 0 )); then
          export JIGGER_BREW_OUTDATED=$(( champs[2] + champs[3] ))
        else
          unset JIGGER_BREW_OUTDATED
        fi
        age=$(( EPOCHSECONDS - champs[4] ))
      fi
    fi

    # Rafraîchissement paresseux : on affiche toujours la valeur connue, et on prépare
    # la suivante. Le garde-fou évite qu'une rafale de prompts (ou dix onglets) ne
    # relance brew en boucle ; le verrou côté Go couvre le cas inter-processus.
    if (( age >= JIGGER_PROMPT_TTL && EPOCHSECONDS - _jigger_last_refresh > 60 )); then
      _jigger_last_refresh=$EPOCHSECONDS
      { jigger prompt --refresh >/dev/null 2>&1 &! } 2>/dev/null
    fi
  }

  add-zsh-hook precmd _jigger_prompt_precmd
fi
