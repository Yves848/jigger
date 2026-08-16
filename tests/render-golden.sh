#!/bin/sh
# Banc de non-régression de `jigger render`.
#
#   tests/render-golden.sh --capturer   # écrit la référence (avant le chantier)
#   tests/render-golden.sh --verifier   # compare la sortie courante à la référence
#
# La sortie du popup est ce que le greffon lit à chaque frappe : elle ne doit pas bouger
# d'un octet tant qu'on reste en français. Le banc balaie 480 combinaisons — 12 lignes,
# 5 largeurs, 4 profils de couleur, avec et sans focus.
#
# ── Pourquoi ce banc n'est PAS dans `make test-all` ───────────────────────────────────
#
# La référence est celle d'**une machine et d'une version**, pas celle du dépôt :
#
#   - elle porte 448 fois la bannière « jigger 0.8.0 » — le prochain bump de version fait
#     échouer les 480 combinaisons d'un coup, sans qu'un seul libellé ait bougé ;
#   - les candidats de « brew install fire », « brew inst » et consorts sortent du cache
#     Homebrew **local** — un `brew update` qui ajoute ou retire une formule en « fire* »
#     la fait échouer tout autant, et elle ne passe déjà pas sur une autre machine.
#
# Un banc qui échoue pour ces raisons-là est un banc qu'on finit par ignorer, puis par
# supprimer. Il reste donc lançable à la main, et c'est là qu'il vaut : autour d'un
# chantier qui touche au rendu ou aux libellés, `--capturer` avant, `--verifier` après.
# Il détecte alors exactement ce pour quoi il a été écrit — une chaîne, une couleur, une
# colonne qui bouge —, la version et le catalogue n'ayant pas changé entre les deux.
#
# Recapturer est aussi ce qu'il faut faire après un bump de version ou un `brew update`,
# si l'on tient à garder une référence à jour dans le dépôt.
set -eu

racine=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
reference="$racine/tests/golden/render-fr.txt"
binaire="$racine/jigger"

[ -x "$binaire" ] || { echo "compile d'abord : make build" >&2; exit 1; }

engendrer() {
  for ligne in "brew" "brew inst" "brew install fire" "brew uninstall " \
               "brew list --" "winget install Git." "scoop uninstall 7z" \
               "jg" "jg inst" "jg install fire" "jg source " "git commit"; do
    for cols in 30 48 58 80 120; do
      for couleur in never 16 256 truecolor; do
        for focus in true false; do
          printf '### %s | %s | %s | %s\n' "$ligne" "$cols" "$couleur" "$focus"
          JIGGER_LANG=fr "$binaire" render --line "$ligne" --cols "$cols" \
            --color "$couleur" --focus="$focus" --rows 8 2>&1 || true
        done
      done
    done
  done
}

case "${1:---verifier}" in
  --capturer)
    mkdir -p "$(dirname "$reference")"
    engendrer > "$reference"
    echo "référence écrite : $reference ($(wc -l < "$reference" | tr -d ' ') lignes)"
    ;;
  --verifier)
    [ -f "$reference" ] || { echo "référence absente : lance --capturer" >&2; exit 1; }
    if engendrer | diff -u "$reference" - > /tmp/jigger-golden.diff; then
      echo "aucune différence (480 combinaisons)"
    else
      echo "ÉCART détecté :" >&2
      head -40 /tmp/jigger-golden.diff >&2
      exit 1
    fi
    ;;
  *) echo "usage: $0 [--capturer|--verifier]" >&2; exit 2 ;;
esac
