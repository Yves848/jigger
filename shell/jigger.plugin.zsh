# jigger.plugin.zsh — assistance Homebrew dans zsh.
#
# Deux modes, complémentaires :
#
#   • popup vivant — dès que la ligne courante est une commande brew, le sélecteur
#     s'affiche sous le prompt et suit la frappe, sans rien demander. ↓ (ou ^N) fait
#     entrer dans la liste, ↑ (ou ^P) en ressort et rend l'historique, ⇥ insère le
#     candidat courant, ^G ferme le popup pour la ligne en cours.
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

# La langue des messages du greffon. Même ordre que le binaire : JIGGER_LANG, puis les
# variables POSIX, puis l'anglais.
#
# L'export n'est pas décoratif : les réglages de jigger se posent sans lui (JIGGER_ROWS=12
# avant le source) parce que le greffon les lit lui-même, mais celui-ci doit atteindre le
# *binaire*, sans quoi le popup parlerait une autre langue que ces messages.
#
# Gardes ${VAR-} plutôt que $VAR nu : sous `setopt nounset` (que certains utilisateurs
# posent), référencer une variable non définie est une erreur. Seule la référence la plus
# interne (celle de $LANG) a besoin de la garde — les $VAR:-… qui l'entourent gèrent déjà
# le cas « non définie » sans y toucher ; mais y toucher aussi ne coûte rien et documente
# l'intention.
if [[ -n ${JIGGER_LANG-} ]]; then
  export JIGGER_LANG
fi
typeset -g _jigger_lang=${${JIGGER_LANG:-${LC_ALL:-${LC_MESSAGES:-${LANG-}}}}%%[_.]*}
[[ $_jigger_lang == fr ]] || _jigger_lang=en

# ── Vérifications d'installation ──────────────────────────────────────────────────────

if ! command -v jigger >/dev/null 2>&1; then
  if [[ $_jigger_lang == fr ]]; then
    print -u2 "jigger : binaire introuvable dans le PATH. Greffon inactif."
  else
    print -u2 "jigger: binary not found in PATH. Plugin inactive."
  fi
  return 0
fi

# Le greffon et le binaire vont par paire : le greffon passe à `jigger render` des options
# qu'un binaire plus ancien ne connaît pas (`--focus`, entre autres). Celui-ci sortirait en
# erreur, et le popup ne s'afficherait jamais — sans un mot, ce qui est la pire façon de
# tomber en panne. Un appel au source (quelques millisecondes) suffit à le dire.
#
# À relever **avec** la version du binaire, à chaque fois que le greffon se met à demander
# quelque chose de neuf : 0.8.0 pour l'alias `jg` et les verbes de la façade, qu'un binaire
# 0.7.0 ne connaît pas.
typeset -g JIGGER_VERSION_REQUISE=0.8.0
autoload -Uz is-at-least
typeset -g _jigger_v=${${(z)"$(command jigger --version 2>/dev/null)"}[2]}
if [[ -n $_jigger_v ]] && ! is-at-least $JIGGER_VERSION_REQUISE $_jigger_v; then
  if [[ $_jigger_lang == fr ]]; then
    print -u2 "jigger : le binaire $(command -v jigger) est en $_jigger_v, or ce greffon en demande $JIGGER_VERSION_REQUISE. Recompile-le (« make install ») ou remplace celui du PATH. Greffon inactif."
  else
    print -u2 "jigger: the binary at $(command -v jigger) is $_jigger_v, but this plugin requires $JIGGER_VERSION_REQUISE. Rebuild it (« make install ») or replace the one in PATH. Plugin inactive."
  fi
  unset _jigger_v
  return 0
fi
unset _jigger_v

# jg : l'alias court de la façade. Ajouté à la liste des commandes qui arment le popup —
# sans quoi le widget ne se déclencherait que sur « jigger » en toutes lettres.
alias jg=jigger

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
typeset -gi _jigger_focused=0   # le popup a-t-il le clavier ? (cf. _jigger_step)
typeset -g _jigger_up_widget='' _jigger_down_widget='' # ce que les flèches faisaient

# Commandes qui arment le popup vivant. « jigger » et son alias « jg » (posé plus haut)
# s'ajoutent à « brew » : l'expansion d'alias n'a pas encore eu lieu tant que la ligne
# n'est pas exécutée, $LBUFFER contient donc « jg » tel quel — c'est lui qu'il faut
# reconnaître, pas « jigger ».
typeset -ga _jigger_commands=( brew jigger jg )

_jigger_is_watched() {
  local c
  for c in $_jigger_commands; do
    [[ $LBUFFER == $c || $LBUFFER == "$c "* ]] && return 0
  done
  return 1
}

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
  local rows=${1:-$JIGGER_ROWS} out focus=false
  (( _jigger_focused )) && focus=true
  out=$(command jigger render --line "$LBUFFER" --sel "$_jigger_sel" --cols "$COLUMNS" \
        --rows "$rows" --color "$(_jigger_color)" --focus=$focus 2>/dev/null) || return 1

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

# _jigger_room fait la place sous la ligne de commande en poussant l'écran de <n> lignes,
# et rend 0 si la place est faite.
#
# Sans cela, le popup ne s'afficherait pour ainsi dire jamais : dans un terminal en usage,
# le prompt occupe la dernière ligne de l'écran, et il n'y a rien en dessous. On fait donc
# ce que fait n'importe quel sélecteur en incrustation (fzf --height, entre autres) : on
# défile.
#
# Le curseur, lui, doit rester sur la ligne de commande — laquelle a monté de <n>. D'où le
# `\e[nA` final : `\e8` (DECRC) ramène à la position *absolue* mémorisée, que le
# défilement a laissée <n> lignes trop bas. Rien de tout cela ne trouble zle, qui se
# déplace en relatif à partir de là où il a laissé le curseur.
_jigger_room() {
  local -i n=$1
  (( n > 0 )) || return 0
  # On ne pousse jamais la ligne de commande hors de l'écran : le prompt, qui peut tenir
  # sur plusieurs lignes, doit rester lisible — et c'est là que zle redessine.
  (( n > _jigger_row_value - 1 )) && n=$(( _jigger_row_value - 1 ))
  (( n > 0 )) || return 1
  { print -n $'\e7\e['$LINES';1H'
    repeat $n print -n $'\n'
    print -n $'\e8\e['$n'A'
  } >/dev/tty
  # La ligne de commande a monté, et le cadre déjà dessiné avec elle : il est toujours
  # juste en dessous, donc toujours à effacer le moment venu (_jigger_shown ne bouge pas).
  _jigger_row_value=$(( _jigger_row_value - n ))
  return 0
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
  _jigger_is_watched || return 1
  (( _jigger_count > 0 ))
}

# Hook de redisplay : c'est lui qui fait vivre le popup. Il est appelé à chaque redessin
# de la ligne, donc à chaque frappe ; on ne relance le binaire que si la ligne, la
# sélection ou la géométrie ont bougé — le reste du temps, on réaffiche le cadre déjà
# rendu.
_jigger_redisplay() {
  (( JIGGER_LIVE )) || return
  if (( _jigger_dismissed )) || ! _jigger_is_watched || (( COLUMNS < JIGGER_MIN_COLUMNS )); then
    _jigger_erase
    return
  fi

  # Où est le curseur ? De sa position dépend le nombre de lignes qu'on peut occuper
  # sans faire défiler l'écran (ce qui décalerait tout l'affichage de zsh).
  _jigger_row || return
  # 5 lignes de décor : 2 bordures, l'en-tête, la respiration, le pied.
  local -i fits=$(( LINES - _jigger_row_value - 5 ))
  # Pas la place : on la fait, en poussant l'écran. Si le défilement échoue, on se
  # contente de ce qu'il reste — et de rien du tout s'il ne reste rien.
  if (( fits < JIGGER_ROWS )); then
    _jigger_room $(( JIGGER_ROWS - fits ))
    fits=$(( LINES - _jigger_row_value - 5 ))
  fi
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

  local key="$LBUFFER|$_jigger_sel|$_jigger_focused|$COLUMNS|$rows"
  if [[ $key != $_jigger_key ]]; then
    _jigger_key=$key
    _jigger_fetch $rows || { _jigger_frame=''; _jigger_count=0; _jigger_erase; return }
  fi
  [[ -n $_jigger_frame ]] && _jigger_draw "$_jigger_frame"
}

# _jigger_step décide de ce que fait une flèche, et rend 0 si le popup l'a prise. C'est
# là que vit le focus, et la règle tient en deux phrases :
#
#   • ↓ fait entrer dans la liste (et y descend). Le popup a dès lors le clavier.
#   • ↑ y remonte, et rend la main dès qu'on est déjà sur le premier candidat.
#
# Hors focus, les deux flèches restent l'historique — c'est le point : on ne perd pas
# l'historique parce qu'un popup est ouvert. Le cadre montre où en est le focus (ligne
# courante soulignée ou au repos), sans quoi la règle serait invisible.
_jigger_step() {
  _jigger_active || return 1
  if (( $1 > 0 )); then
    _jigger_focused=1
    (( _jigger_sel++ ))
  else
    (( _jigger_focused )) || return 1   # popup ouvert, mais pas au clavier
    if (( _jigger_sel > 0 )); then
      (( _jigger_sel-- ))
    else
      _jigger_focused=0
    fi
  fi
  _jigger_selline=$LBUFFER
  return 0
}

# ↑↓ : au popup s'il a (ou prend) le clavier, sinon au widget qui les tenait avant nous —
# celui de l'historique, ou celui qu'un autre plugin y a mis.
_jigger_down() { _jigger_step 1  || zle ${_jigger_down_widget:-.down-line-or-history} }
_jigger_up()   { _jigger_step -1 || zle ${_jigger_up_widget:-.up-line-or-history} }

# ^N / ^P : les mêmes, pour qui les préfère aux flèches.
_jigger_next() { _jigger_step 1  || zle .down-line-or-history }
_jigger_prev() { _jigger_step -1 || zle .up-line-or-history }

# _jigger_bound_widget rend le widget déjà lié à une séquence, pour le lui rendre quand
# le popup n'a pas le focus. Échoue si la touche est libre — ou si elle est déjà à nous,
# ce qui arriverait en re-sourçant le plugin : on s'appellerait en boucle.
_jigger_bound_widget() {
  local -a champs
  champs=( ${(z)"$(bindkey -- "$1" 2>/dev/null)"} )
  (( ${#champs} >= 2 )) || return 1
  local w=${(Q)champs[2]}
  [[ -z $w || $w == undefined-key || $w == _jigger_* ]] && return 1
  print -r -- $w
}

# ^G : ferme le popup jusqu'à la fin de la ligne. Sans popup, c'est l'abandon habituel.
_jigger_dismiss() {
  if _jigger_active; then
    _jigger_dismissed=1
    _jigger_focused=0
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

  if ! _jigger_is_watched; then
    zle expand-or-complete
    return
  fi

  if (( JIGGER_LIVE )); then
    # Popup fermé par ^G : ⇥ le rouvre.
    if (( _jigger_dismissed )); then
      _jigger_dismissed=0
      _jigger_key=''
      return
    fi
    # Popup allumé mais sans rien à proposer — aucun candidat, ou pas la place de
    # l'afficher. ⇥ rend alors la main à la complétion du shell. Surtout pas au sélecteur
    # plein écran : il dessinerait par-dessus le prompt et attendrait une touche, ce qui
    # n'a rien à voir avec ce qu'on vient de demander.
    zle expand-or-complete
    return
  fi

  # JIGGER_LIVE=0 : le sélecteur plein écran, celui qu'on a explicitement choisi. Il
  # possède le terminal le temps du choix.
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
  _jigger_focused=0
  _jigger_dismissed=0
  _jigger_count=0
  _jigger_frame=''
  _jigger_key=''
}

zle -N _jigger_widget
zle -N _jigger_next
zle -N _jigger_prev
zle -N _jigger_up
zle -N _jigger_down
zle -N _jigger_dismiss

bindkey "$JIGGER_KEY" _jigger_widget
if (( JIGGER_LIVE )); then
  bindkey '^N' _jigger_next
  bindkey '^P' _jigger_prev
  bindkey '^G' _jigger_dismiss

  # Les flèches : on relève d'abord ce qu'elles faisaient, puis on les prend. Les deux
  # formes des séquences (mode normal et mode application) sont couvertes, plus celle
  # que déclare le terminfo — c'est ce que fait zsh lui-même.
  typeset -ga _jigger_up_keys=( '^[[A' '^[OA' )
  typeset -ga _jigger_down_keys=( '^[[B' '^[OB' )
  if zmodload -i zsh/terminfo 2>/dev/null; then
    [[ -n ${terminfo[kcuu1]-} ]] && _jigger_up_keys+=( ${terminfo[kcuu1]} )
    [[ -n ${terminfo[kcud1]-} ]] && _jigger_down_keys+=( ${terminfo[kcud1]} )
  fi
  for _jigger_k in $_jigger_up_keys;   do _jigger_up_widget=$(_jigger_bound_widget $_jigger_k)   && break; done
  for _jigger_k in $_jigger_down_keys; do _jigger_down_widget=$(_jigger_bound_widget $_jigger_k) && break; done
  for _jigger_k in $_jigger_up_keys;   do bindkey $_jigger_k _jigger_up;   done
  for _jigger_k in $_jigger_down_keys; do bindkey $_jigger_k _jigger_down; done
  unset _jigger_k
  add-zle-hook-widget line-pre-redraw _jigger_redisplay
  add-zle-hook-widget line-init _jigger_line_init
  add-zle-hook-widget line-finish _jigger_line_finish
fi

# ── Après une commande brew qui change l'état ─────────────────────────────────────────
#
# Deux choses vieillissent quand une commande brew passe, et une seule des deux regarde
# le bloc de prompt :
#
#   • le **catalogue** — `brew tap`, `untap` et `update` changent la liste des noms
#     connus. Sans rafraîchissement, une formule fraîchement tapée manquerait à la
#     complétion jusqu'à la péremption du cache (24 h) ;
#   • le **compteur du prompt**, faux dès qu'on installe ou qu'on met à niveau.
#
# La détection est donc ici, hors du bloc oh-my-posh : le catalogue doit être à jour même
# sans JIGGER_PROMPT (c'est le pendant du `warm --installed` du module PowerShell).

: "${JIGGER_PROMPT:=0}"
# Âge du cache (secondes) au-delà duquel on relance un rafraîchissement détaché.
: "${JIGGER_PROMPT_TTL:=1800}"
# Le TTL seul laisserait le compteur mentir une demi-heure après un `brew upgrade`. On
# repère donc les commandes brew qui changent l'état, et on rafraîchit dans la foulée —
# en avant-plan, pour que le prompt qui suit immédiatement l'upgrade soit déjà juste.
# C'est le seul endroit où le plugin fait attendre le prompt (quelques dixièmes de
# seconde, après une commande qui a duré bien plus). JIGGER_PROMPT_SYNC=0 rend ce
# rafraîchissement détaché : plus rien n'attend, mais le compteur ne se corrige qu'au
# prompt suivant.
: "${JIGGER_PROMPT_SYNC:=1}"

zmodload -i zsh/datetime          # $EPOCHSECONDS, pour ne pas forker `date`
autoload -Uz add-zsh-hook

typeset -gi _jigger_prompt_dirty=0    # le compteur du prompt est connu faux
typeset -gi _jigger_catalog_dirty=0   # la liste des noms connus est connue fausse

# Sous-commandes qui changent ce que `brew outdated` répondra. `update` en fait partie :
# il ne touche à rien d'installé, mais renouvelle le catalogue, donc la liste des
# obsolètes. `cleanup` et `doctor`, eux, n'y changent rien.
typeset -ga _jigger_brew_mutants=(
  install reinstall uninstall remove rm upgrade update tap untap pin unpin
  bundle autoremove
)
# Celles qui changent la liste des *noms* connus. `install` n'en est pas : elle installe
# un paquet que le catalogue connaissait déjà — et les installés, eux, se lisent dans
# Cellar et Caskroom, donc sans cache à refaire.
typeset -ga _jigger_brew_catalog_mutants=( update tap untap )
# Mots qui précèdent une commande sans en être une : la commande brew reste à venir.
typeset -ga _jigger_prefixes=(
  ';' '&' '&&' '||' '|' '|&' '(' ')' '{' '}' '!' then do else elif
  sudo command env exec nohup time arch
)

# _jigger_preexec repère `brew <mutant>` dans la ligne sur le point de s'exécuter. On ne
# lit que des mots — rien n'est évalué — et on n'accepte `brew` qu'en tête de commande :
# « git commit -m 'brew upgrade' » ne déclenche donc rien.
_jigger_preexec() {
  local mot
  local -i debut=1 brew=0
  # ${(z)…} découpe comme le ferait le shell, sans rien exécuter.
  for mot in ${(z)${3:-$1}}; do
    if (( brew )); then
      [[ $mot == -* ]] && continue   # « brew --quiet upgrade » compte pour un upgrade
      (( ${_jigger_brew_mutants[(Ie)$mot]} ))         && _jigger_prompt_dirty=1
      (( ${_jigger_brew_catalog_mutants[(Ie)$mot]} )) && _jigger_catalog_dirty=1
      # Le premier mot nu est la sous-commande : ce qui suit ne nous apprend plus rien.
      brew=0 debut=0
      continue
    fi
    if (( ${_jigger_prefixes[(Ie)$mot]} )); then
      debut=1
      continue
    fi
    # Options d'un préfixe (`arch -arm64 brew …`) et affectations en tête
    # (`HOMEBREW_NO_AUTO_UPDATE=1 brew …`) : la commande n'a toujours pas commencé.
    if (( debut )) && [[ $mot == -* || $mot == *=* ]]; then
      continue
    fi
    if (( debut )) && [[ ${mot:t} == brew ]]; then
      brew=1
      continue
    fi
    debut=0
  done
}

# _jigger_precmd tourne avant chaque prompt, que le bloc oh-my-posh soit activé ou non :
# le catalogue, lui, sert à la complétion.
_jigger_precmd() {
  if (( _jigger_catalog_dirty )); then
    _jigger_catalog_dirty=0
    # Détaché : personne n'attend après un catalogue. Le rendu qui suit se contentera de
    # l'ancien s'il n'est pas encore prêt — c'est toute la règle du réchauffement.
    { command jigger warm --all >/dev/null 2>&1 &! } 2>/dev/null
  fi
  (( JIGGER_PROMPT )) && _jigger_prompt_precmd
  return 0
}

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
#
# Unique exception, assumée : le prompt qui suit une commande brew mutante. Là, le cache
# est *connu* faux, et l'attendre coûte moins cher que d'afficher un mensonge (cf.
# JIGGER_PROMPT_SYNC).

if (( JIGGER_PROMPT )); then
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

  # Exporte <nom>=<valeur>, ou le retire de l'environnement si la valeur est nulle. C'est
  # ce qui permet aux templates de tester la simple présence de la variable.
  _jigger_export_compteur() {
    if (( $2 > 0 )); then
      export $1=$2
    else
      unset $1
    fi
  }

  _jigger_prompt_precmd() {
    local line
    local -a champs
    local -i age=$JIGGER_PROMPT_TTL   # cache absent → réputé périmé

    # Le cache est connu faux : on le refait avant de lire, pas au prochain TTL. --wait
    # parce que le rafraîchissement paresseux d'un autre onglet a fort bien pu partir
    # pendant l'upgrade et tenir encore le verrou : sans attendre, on renoncerait
    # justement dans le cas qu'on cherche à corriger.
    if (( _jigger_prompt_dirty )); then
      _jigger_prompt_dirty=0
      _jigger_last_refresh=$EPOCHSECONDS
      if (( JIGGER_PROMPT_SYNC )); then
        command jigger prompt --refresh --wait >/dev/null 2>&1
      else
        { command jigger prompt --refresh --wait >/dev/null 2>&1 &! } 2>/dev/null
      fi
    fi

    if [[ -r $_jigger_status_file ]] && read -r line < $_jigger_status_file; then
      # Découpage sur tabulation par expansion (pas de `cut`). Le drapeau `p` fait
      # interpréter \t dans le séparateur : pas de tabulation littérale dans ce
      # fichier, qu'un éditeur pourrait convertir en espaces.
      champs=( "${(@ps.\t.)line}" )
      # `<->` = suite de chiffres : une ligne bricolée à la main ne doit pas se
      # retrouver dans une expansion arithmétique.
      if (( $#champs == 4 )) && [[ "$champs[2]$champs[3]$champs[4]" == <-> ]]; then
        export JIGGER_BREW_VERSION="$champs[1]"
        # Un compteur n'est exporté que s'il a quelque chose à signaler : le template
        # oh-my-posh se réduit alors à « {{ if .Env.JIGGER_BREW_FORMULAE }} », sans
        # comparaison de chaînes. Le total reste exposé pour qui préfère un seul chiffre.
        _jigger_export_compteur JIGGER_BREW_FORMULAE $champs[2]
        _jigger_export_compteur JIGGER_BREW_CASKS    $champs[3]
        _jigger_export_compteur JIGGER_BREW_OUTDATED $(( champs[2] + champs[3] ))
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

fi

add-zsh-hook preexec _jigger_preexec
add-zsh-hook precmd _jigger_precmd

# oh-my-posh calcule tout son prompt dans son propre precmd, à partir de l'environnement
# tel qu'il le trouve. Si le nôtre passait après, il exporterait des compteurs qui ne
# seraient affichés qu'au prompt d'après — le bloc aurait toujours un coup de retard,
# exactement le symptôme qu'on corrige. On se place donc en tête, quel que soit l'ordre
# des `source` dans ~/.zshrc.
precmd_functions=( _jigger_precmd ${precmd_functions:#_jigger_precmd} )

# Au premier prompt, le catalogue peut être vieux d'un jour — ou absent, à la première
# installation. On lance le réchauffement tout de suite, détaché : le binaire décide
# lui-même s'il y a quelque chose à faire, et une frappe n'attend jamais après brew.
{ command jigger warm >/dev/null 2>&1 &! } 2>/dev/null
