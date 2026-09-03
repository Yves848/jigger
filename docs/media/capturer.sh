#!/usr/bin/env bash
# Produit les enregistrements et les captures de jigger — macOS et Omarchy.
#
#   ./docs/media/capturer.sh                    # tout ce qui concerne cette machine
#   ./docs/media/capturer.sh macos-03-ssh       # un seul scénario
#
# La plateforme est déduite du système : sur macOS on ne peut produire que les
# tapes « macos-* », sur Arch que les « omarchy-* ». Windows a son propre script
# (capturer.ps1) : ni tmux ni VHS n'y tiennent le même rôle.
#
# Chaque scénario rend TROIS fichiers dans out/ :
#   <nom>.gif   l'enregistrement, celui qu'on met dans la documentation
#   <nom>.mp4   le même, pour le site
#   <nom>.png   une image fixe prise au moment où le popup est ouvert et au repos
#
# Prérequis : vhs, ffmpeg, tmux, et le binaire jigger sur le PATH.
set -euo pipefail

MEDIA="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$MEDIA/../.." && pwd)"
cd "$MEDIA"
mkdir -p out

for outil in vhs ffmpeg tmux; do
  command -v "$outil" >/dev/null || { echo "manquant : $outil" >&2; exit 1; }
done
command -v jigger >/dev/null || { echo "manquant : jigger sur le PATH" >&2; exit 1; }

case "$(uname -s)" in
  Darwin) PLATEFORME=macos   ;;
  Linux)  PLATEFORME=omarchy ;;
  *) echo "plateforme non gérée ici — voir capturer.ps1 pour Windows" >&2; exit 1 ;;
esac

# L'instant où l'on prend l'image fixe, scénario par scénario : le popup y est
# ouvert, complet, et personne n'a encore appuyé sur une flèche. Calculé, pas
# deviné — c'est la somme des Sleep et du temps de frappe du tape (90 ms par
# caractère), et c'est pourquoi toucher un tape demande de revoir la valeur.
instant() {
  case "$1" in
    *-01-gestionnaire-natif) echo 4.5 ;;
    *-02-jg)                 echo 4.0 ;;
    *-03-ssh)                echo 3.0 ;;
    *)                       echo 3.0 ;;
  esac
}

capturer() {
  local nom=$1 tape="tapes/$1.tape"
  [[ -f $tape ]] || { echo "tape absent : $tape" >&2; return 1; }
  echo "── $nom"

  # Le sélecteur SSH lit $HOME/.ssh/config, sans surcharge possible. On lui donne
  # donc un HOME de fixture : la capture montre des serveurs inventés, les mêmes
  # partout, et jamais l'infrastructure de la machine qui l'a produite.
  local home_capture=$HOME
  [[ $nom == *-03-ssh ]] && home_capture="$MEDIA/fixtures/home"

  tmux -L jiggercap kill-server 2>/dev/null || true
  env HOME="$home_capture" \
      ZDOTDIR="$MEDIA/fixtures/zdotdir" \
      JIGGER_REPO="$REPO" \
      JIGGER_MEDIA="$MEDIA" \
      JIGGER_LANG="${JIGGER_LANG:-en}" \
      vhs "$tape"
  tmux -L jiggercap kill-server 2>/dev/null || true

  ffmpeg -y -loglevel error -ss "$(instant "$nom")" -i "out/$nom.gif" \
         -vframes 1 "out/$nom.png"
  echo "   out/$nom.gif  out/$nom.mp4  out/$nom.png"
}

if (( $# )); then
  for nom in "$@"; do capturer "$nom"; done
else
  for tape in tapes/"$PLATEFORME"-*.tape; do
    capturer "$(basename "$tape" .tape)"
  done
fi

echo
echo "Terminé. Les fichiers sont dans $MEDIA/out/."
