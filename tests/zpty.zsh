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
# Stub : la ligne réellement exécutée est la seule chose observable de façon fiable
# (zsh réécrit la ligne en incrémental, le texte complet n'apparaît jamais dans le flux).
brew() { print -r -- "CMD:[\$*]" }
source $root/shell/jigger.plugin.zsh
$extra
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
jigger_type() {
  local keys=$1 rc=${TMPDIR:-/tmp}/jigger-zpty-rc.zsh
  _jigger_rc $rc

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
  check "cadre affiché"                 $out '╭─'
  check "en-tête du contexte"           $out '❯ brew'
  check "candidat install"              $out 'install'

  print -r -- "→ une ligne qui n'est pas brew n'affiche rien"
  out=$(visible "$(jigger_type 'echo hi')")
  check "aucun cadre"                   $out '╭─' non

  print -r -- "→ ⇥ insère le candidat courant"
  out=$(visible "$(jigger_type $'brew u\t\n')")
  check "uninstall exécuté"             $out 'CMD:[uninstall]'

  print -r -- "→ ^N descend d'un candidat avant l'insertion"
  out=$(visible "$(jigger_type $'brew u\x0e\t\n')")
  check "upgrade exécuté"               $out 'CMD:[upgrade]'

  print -r -- "→ ^P remonte, et ne dépasse pas le premier"
  out=$(visible "$(jigger_type $'brew u\x0e\x10\x10\t\n')")
  check "retour sur uninstall"          $out 'CMD:[uninstall]'

  print -r -- "→ ^G ferme le popup et laisse la ligne intacte"
  out=$(visible "$(jigger_type $'brew u\x07\n')")
  check "ligne exécutée telle quelle"   $out 'CMD:[u]'

  print -r -- "→ ⏎ efface le cadre avant la sortie de la commande"
  out=$(visible "$(jigger_type $'brew u\t\n')")
  check "aucun cadre après la sortie"   ${out##*CMD:\[uninstall\]} '╭─' non

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
