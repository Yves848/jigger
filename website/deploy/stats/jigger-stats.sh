#!/bin/sh
# Régénère le rapport GoAccess des visites du site jigger.
#
# Aucun script côté client, aucun cookie, aucun tiers : tout est calculé sur le serveur à
# partir des journaux que nginx écrit déjà. Les IP sont anonymisées et les robots écartés.
#
# Écart assumé avec la recette de cocktails, dont ce fichier descend : cocktails ne lit que
# le journal courant et sa première rotation — deux jours. Ici on lit TOUTES les rotations,
# soit une quinzaine de jours, parce qu'un site à faible trafic n'a presque rien à montrer
# sur deux jours. Mesuré le 2026-09-08 : 10 631 lignes sur l'ensemble, contre quelques
# centaines sur les deux derniers jours.
#
# D'où le `zcat -f` : GoAccess 1.7 ne sait PAS ouvrir un journal compressé — il n'échoue
# même pas franchement, il rend un rapport vide. Éprouvé sur l'hôte avant d'écrire ceci.
# `-f` laisse passer les fichiers non compressés sans y toucher, ce qui permet de traiter
# le courant, la rotation `.1` et les `.gz` par un seul tuyau.
set -u

LOGS=/var/log/nginx
SORTIE=/var/www/jigger-stats

# La liste est construite fichier par fichier plutôt que par un glob passé tel quel : sans
# cela, un motif `*.gz` qui ne correspond à rien est transmis littéralement à zcat, qui
# s'interrompt. Le premier jour d'un serveur neuf n'a aucune rotation.
set --
for f in "$LOGS"/jigger.access.log.*.gz "$LOGS"/jigger.access.log.1 "$LOGS"/jigger.access.log; do
  [ -f "$f" ] && set -- "$@" "$f"
done
[ "$#" -gt 0 ] || exit 0

mkdir -p "$SORTIE"
zcat -f "$@" | goaccess - \
  --log-format=COMBINED --ignore-crawlers --anonymize-ip \
  -o "$SORTIE/index.html" --html-report-title="jigger — visites"
