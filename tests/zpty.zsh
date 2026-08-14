#!/usr/bin/env zsh
# Harnais de test du widget zsh : lance un zsh interactif dans un pseudo-terminal,
# tape une séquence de touches, et rend le flux écrit par le terminal.
#
# C'est le seul moyen de tester le popup vivant sans intervention humaine : tout ce qui
# compte (le cadre s'affiche, il disparaît, la ligne est réécrite) ne se voit que dans
# ce que zsh écrit sur le TTY.
#
#   ./tests/zpty.zsh 'brew inst'          → flux brut
#   ./tests/zpty.zsh 'brew inst' --visible → texte visible (séquences ANSI retirées)
#
# Lancer la suite d'assertions : ./tests/zpty.zsh --suite

emulate -L zsh
setopt err_return extendedglob
zmodload zsh/zpty

local root=${0:A:h:h}
local bin=$root/jigger
[[ -x $bin ]] || { print -u2 "compile d'abord : go build -o jigger ."; return 1 }

# rc minimal : pas de ~/.zshrc, uniquement le plugin. JIGGER_TEST_PLUGINS=1 y ajoute
# zsh-autosuggestions et zsh-syntax-highlighting — les deux greffons de ligne d'édition
# les plus répandus, et ceux avec lesquels le popup a le plus de chances de se marcher
# dessus (ils enrobent des widgets et écrivent eux aussi sous le prompt).
_jigger_rc() {
  local rc=$1
  local extra=''
  if (( ${JIGGER_TEST_PLUGINS:-0} )); then
    local as=/opt/homebrew/share/zsh-autosuggestions/zsh-autosuggestions.zsh
    local sh=/opt/homebrew/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh
    [[ -r $as ]] && extra+="source $as"$'\n'
    [[ -r $sh ]] && extra+="source $sh"$'\n'   # doit être chargé en dernier
  fi
  cat > $rc <<RC
PS1='%% '
PATH="$root:\$PATH"
COLORTERM=truecolor
# Les flèches arrivent en trois octets (ESC [ A), et le harnais tape caractère par
# caractère avec des pauses entre eux : sans allonger le délai, zsh conclurait à un
# simple Échap avant d'avoir vu le reste. C'est un artefact du pseudo-terminal, pas du
# plugin — un humain tape ses flèches d'un coup.
KEYTIMEOUT=100
# Stub : la ligne réellement exécutée est la seule chose observable de façon fiable
# (zsh réécrit la ligne en incrémental, le texte complet n'apparaît jamais dans le flux).
brew() { print -r -- "CMD:[\$*]" }
source $root/shell/jigger.plugin.zsh
$extra
RC
}

# rc du bloc oh-my-posh (JIGGER_PROMPT). Trois pièces :
#   • un cache fabriqué, frais et mensonger (10 formulae, 1 cask) ;
#   • un faux binaire `jigger` qui journalise ses appels et, sur --refresh, réécrit le
#     cache à 0/0 — c'est ce qui rend l'invalidation observable. Un vrai fichier
#     exécutable, pas une fonction : le plugin appelle `command jigger`, qui saute les
#     fonctions ;
#   • un PS1 qui affiche le compteur exporté, seul moyen de voir ce que le prompt aurait
#     montré.
# Le popup vivant est éteint : il n'a rien à voir ici, et son DSR brouillerait le flux.
_jigger_prompt_rc() {
  local rc=$1 dir=$2
  mkdir -p $dir/bin
  # printf, pas `print -r` : il faut de vraies tabulations, celles-là mêmes que le hook
  # cherche.
  printf '9.9.9\t10\t1\t%s\n' "$(date +%s)" > $dir/status
  : > $dir/appels   # doit exister même quand rien n'appelle jigger (err_return)

  cat > $dir/bin/jigger <<STUB
#!/bin/sh
printf '%s\n' "\$*" >> $dir/appels
case "\$*" in
  *--refresh*) printf '9.9.9\t0\t0\t%s\n' "\$(date +%s)" > $dir/status ;;
esac
STUB
  chmod +x $dir/bin/jigger

  cat > $rc <<RC
setopt prompt_subst
PS1='jf=\${JIGGER_BREW_FORMULAE:-vide} %% '
# Le pty hérite de l'environnement de celui qui lance les tests — lequel a fort bien pu
# ouvrir son terminal avec le bloc activé. Sans ce nettoyage, on lirait ses compteurs à
# lui et le test passerait sans rien prouver.
unset JIGGER_BREW_VERSION JIGGER_BREW_FORMULAE JIGGER_BREW_CASKS JIGGER_BREW_OUTDATED
PATH="$dir/bin:\$PATH"
brew() { print -r -- "CMD:[\$*]" }
JIGGER_LIVE=0
JIGGER_PROMPT=1
JIGGER_CACHE_DIR=$dir
source $root/shell/jigger.plugin.zsh
RC
}

# _drain lit tout ce que le pty a à dire (non bloquant) et répond aux interrogations du
# curseur (DSR 6n) : personne n'est derrière ce pseudo-terminal pour le faire, et le
# widget refuse d'afficher tant qu'il ignore où est le curseur. C'est le seul endroit où
# le harnais doit imiter un vrai terminal.
: ${JIGGER_TEST_ROW:=3}
_drain() {
  local chunk all=''
  while zpty -r -t z chunk 2>/dev/null; do
    all+=$chunk
    [[ $chunk == *$'\e[6n'* ]] && zpty -w -n z $'\e['$JIGGER_TEST_ROW';1R'
  done
  print -rn -- $all
}

# jigger_type tape une chaîne (caractère par caractère, comme un humain) et rend le flux.
# Un rc déjà écrit peut être passé en second argument (cf. _jigger_prompt_rc).
jigger_type() {
  local keys=$1 rc=${2:-}
  if [[ -z $rc ]]; then
    rc=${TMPDIR:-/tmp}/jigger-zpty-rc.zsh
    _jigger_rc $rc
  fi

  zpty -b z zsh -f -i
  sleep 0.6; _drain >/dev/null
  zpty -w z "source $rc"
  sleep 0.6; _drain >/dev/null

  # Vider le pty au fur et à mesure est vital : un cadre fait quelques kilo-octets, le
  # tampon du pseudo-terminal sature en deux frappes et zsh se bloque en écriture — la
  # session se fige alors sans le moindre message.
  local out=''
  for (( i = 1; i <= ${#keys}; i++ )); do
    zpty -w -n z ${keys[i]}
    # Scruter souvent : le widget n'attend la réponse au DSR que 150 ms, et il faut la
    # lui donner dans ce délai sous peine de tester un popup qui ne s'affiche jamais.
    for (( j = 0; j < 12; j++ )); do
      out+=$(_drain)
      sleep 0.02
    done
  done
  sleep 0.4
  out+=$(_drain)
  zpty -d z 2>/dev/null
  print -r -- $out
}

# visible retire les séquences ANSI : les assertions portent sur ce que l'œil verrait.
visible() {
  # setopt local : `emulate -L zsh` (en tête de script) rend les fonctions définies
  # ensuite « collantes » à cette émulation — les options posées au niveau du script
  # sont réinitialisées à l'entrée de la fonction, extendedglob compris.
  setopt local_options extendedglob
  local s=$1
  s=${s//$'\e'\[[0-9;?]#[a-zA-Z]/}   # séquences CSI
  print -r -- ${s//$'\e'[78]/}         # sauvegarde/restauration du curseur (DECSC/DECRC)
}

# ---------------------------------------------------------------- suite d'assertions

local -i failed=0
# Le « haystack » se passe entre guillemets, toujours : une expansion nue et vide est
# purement élidée par zsh, les arguments glissent d'un cran, et l'assertion se met à
# comparer n'importe quoi — en passant, de préférence.
check() {
  local label=$1 haystack=$2 needle=$3 want=${4:-oui}
  local hit=non
  [[ $haystack == *$needle* ]] && hit=oui
  if [[ $hit == $want ]]; then
    print -r -- "  ok   $label"
  else
    (( failed += 1 ))
    if [[ $want == oui ]]; then
      print -r -- "  FAIL $label — « $needle » absent"
    else
      print -r -- "  FAIL $label — « $needle » présent alors qu'il devait avoir disparu"
    fi
  fi
}

# Les sous-commandes brew sont une liste figée dans le code : « brew u » donne toujours
# uninstall, upgrade, uses… dans cet ordre. Les tests de navigation s'appuient dessus
# plutôt que sur le catalogue local, qui varie d'une machine à l'autre.
suite() {
  local out

  print -r -- "→ le popup apparaît en tapant une commande brew"
  out=$(visible "$(jigger_type 'brew inst')")
  check "cadre affiché"                 "$out" '╭─'
  check "en-tête du contexte"           "$out" '❯ brew'
  check "candidat install"              "$out" 'install'

  print -r -- "→ une ligne qui n'est pas brew n'affiche rien"
  out=$(visible "$(jigger_type 'echo hi')")
  check "aucun cadre"                   "$out" '╭─' non

  print -r -- "→ ⇥ insère le candidat courant"
  out=$(visible "$(jigger_type $'brew u\t\n')")
  check "uninstall exécuté"             "$out" 'CMD:[uninstall]'

  print -r -- "→ ^N descend d'un candidat avant l'insertion"
  out=$(visible "$(jigger_type $'brew u\x0e\t\n')")
  check "upgrade exécuté"               "$out" 'CMD:[upgrade]'

  print -r -- "→ ^P remonte, et ne dépasse pas le premier"
  out=$(visible "$(jigger_type $'brew u\x0e\x10\x10\t\n')")
  check "retour sur uninstall"          "$out" 'CMD:[uninstall]'

  print -r -- "→ ↓ fait entrer dans la liste, ↑ en ressort"
  out=$(visible "$(jigger_type $'brew u\e[B\t\n')")
  check "↓ descend d'un candidat"       "$out" 'CMD:[upgrade]'
  out=$(visible "$(jigger_type $'brew u\e[B\e[A\e[A\t\n')")
  check "↑ remonte, puis rend le clavier" "$out" 'CMD:[uninstall]'

  print -r -- "→ tant que le popup n'a pas le clavier, ↑ reste l'historique"
  # La commande précédente doit revenir dans la ligne, popup ouvert ou non : c'est toute
  # la raison d'être du focus.
  out=$(visible "$(jigger_type $'brew leaves\nbrew u\e[A\n')")
  check "commande précédente rappelée"  "${out#*CMD:\[leaves\]}" 'CMD:[leaves]'

  print -r -- "→ le pied dit où ira la prochaine flèche"
  out=$(visible "$(jigger_type 'brew u')")
  check "invite à entrer dans la liste" "$out" '↓  parcourir'
  out=$(visible "$(jigger_type $'brew u\e[B')")
  check "puis à naviguer"               "$out" '↑↓  naviguer'

  print -r -- "→ ^G ferme le popup et laisse la ligne intacte"
  out=$(visible "$(jigger_type $'brew u\x07\n')")
  check "ligne exécutée telle quelle"   "$out" 'CMD:[u]'

  print -r -- "→ ⏎ efface le cadre avant la sortie de la commande"
  out=$(visible "$(jigger_type $'brew u\t\n')")
  check "aucun cadre après la sortie"   "${out##*CMD:\[uninstall\]}" '╭─' non

  # ── Bloc oh-my-posh ────────────────────────────────────────────────────────────────
  # Un dossier neuf par cas : le journal des appels au faux jigger ne doit rien traîner
  # d'un cas à l'autre.
  local dir rc appels

  print -r -- "→ bloc oh-my-posh : le compteur est exporté depuis le cache"
  dir=$(mktemp -d); rc=$dir/rc.zsh; _jigger_prompt_rc $rc $dir
  out=$(visible "$(jigger_type $'echo hi\n' $rc)")
  appels=$(cat $dir/appels)
  check "compteur exporté"              "$out" 'jf=10'
  check "cache frais, aucun appel"      "$appels" 'refresh' non

  print -r -- "→ une sous-commande brew inoffensive ne rafraîchit rien"
  dir=$(mktemp -d); rc=$dir/rc.zsh; _jigger_prompt_rc $rc $dir
  out=$(visible "$(jigger_type $'brew list\n' $rc)")
  appels=$(cat $dir/appels)
  check "brew list ne force rien"       "$appels" 'refresh' non
  check "compteur inchangé"             "$out" 'jf=10'

  print -r -- "→ après un brew upgrade, le compteur est juste dès le prompt suivant"
  dir=$(mktemp -d); rc=$dir/rc.zsh; _jigger_prompt_rc $rc $dir
  out=$(visible "$(jigger_type $'brew upgrade\n' $rc)")
  appels=$(cat $dir/appels)
  check "rafraîchissement forcé"        "$appels" 'prompt --refresh --wait'
  check "compteur remis à jour"         "${out##*CMD:\[upgrade\]}" 'jf=vide'
  check "l'ancien compteur a disparu"   "${out##*CMD:\[upgrade\]}" 'jf=10' non

  print -r -- "→ la détection voit brew derrière un préfixe, et pas dans une citation"
  dir=$(mktemp -d); rc=$dir/rc.zsh; _jigger_prompt_rc $rc $dir
  out=$(visible "$(jigger_type $'HOMEBREW_NO_AUTO_UPDATE=1 brew install jq\n' $rc)")
  check "affectation en tête"           "$(cat $dir/appels)" 'prompt --refresh --wait'

  dir=$(mktemp -d); rc=$dir/rc.zsh; _jigger_prompt_rc $rc $dir
  out=$(visible "$(jigger_type $'echo brew upgrade\n' $rc)")
  check "brew cité ne compte pas"       "$(cat $dir/appels)" 'refresh' non

  if (( failed )); then
    print -r -- "\n$failed assertion(s) en échec"
    return 1
  fi
  print -r -- ""
  print -r -- "tout passe"
}

case ${1:-} in
  (--suite) suite ;;
  ('')      print -u2 "usage: $0 <touches> [--visible] | --suite"; return 2 ;;
  (*)
    local out=$(jigger_type "$1")
    if [[ ${2:-} == --visible ]]; then visible "$out"; else print -r -- "$out"; fi
    ;;
esac
