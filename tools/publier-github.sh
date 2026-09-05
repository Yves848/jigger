#!/usr/bin/env bash
# Publie une release sur le miroir GitHub, binaires compris.
#
#   tools/publier-github.sh v0.15.0          # depuis le registre GitLab
#   tools/publier-github.sh v0.15.0 dist/    # depuis des fichiers déjà construits
#
# Pourquoi ce script existe : le miroir ne copie que le code. Un visiteur qui arrive
# sur github.com/Yves848/jigger — c'est-à-dire la quasi-totalité de ceux qui
# découvriront le projet — n'y trouve aucun binaire à télécharger, alors que la
# release GitLab en porte quatre. Une release GitHub STOCKE ses fichiers, là où une
# release GitLab ne porte que des liens : on téléverse donc pour de bon.
#
# Idempotent : rejouable sur une release déjà publiée, un fichier déjà présent est
# remplacé plutôt que refusé.
#
# Jeton attendu dans GITHUB_RELEASE_TOKEN (scope `repo`). En local, le script se
# rabat sur `gh auth token` s'il est absent.
set -euo pipefail

TAG="${1:-${CI_COMMIT_TAG:-}}"
SOURCE="${2:-}"
DEPOT="${GITHUB_DEPOT:-Yves848/jigger}"
GITLAB="${GITLAB_API:-https://gitlab.yg-devworks.com/api/v4/projects/25}"

[ -n "$TAG" ] || { echo "usage : $0 <tag> [dossier]" >&2; exit 2; }
VERSION="${TAG#v}"

RACINE="$(cd "$(dirname "$0")/.." && pwd)"
TRAVAIL="$(mktemp -d)"
trap 'rm -rf "$TRAVAIL"' EXIT

# --- le jeton -------------------------------------------------------------
JETON="${GITHUB_RELEASE_TOKEN:-}"
if [ -z "$JETON" ] && command -v gh >/dev/null 2>&1; then
  JETON="$(gh auth token 2>/dev/null || true)"
fi
[ -n "$JETON" ] || {
  echo "GITHUB_RELEASE_TOKEN absent, et « gh auth token » n'a rien rendu." >&2
  echo "Sans jeton on ne peut rien publier : on s'arrête plutôt que de faire semblant." >&2
  exit 1
}
API=(curl --fail --silent --show-error
     --header "Authorization: Bearer $JETON"
     --header "Accept: application/vnd.github+json"
     --header "X-GitHub-Api-Version: 2022-11-28")

# --- le tag doit d'abord être arrivé sur le miroir ------------------------
# Le miroir GitLab → GitHub est ASYNCHRONE : GitLab le déclenche au push, mais il
# s'exécute quand il s'exécute. Ce script, lui, part dans la seconde qui suit le tag.
# GitHub refuse alors en 422 de créer une release sur un tag qu'il ne connaît pas
# encore — c'est ce qui a fait échouer la v0.17.1. La v0.17.0 avait gagné la même
# course six heures plus tôt, ce qui rendait la chaîne fiable en apparence seulement.
#
# On attend donc, au lieu de supposer. Le contrôle est en tête, avant la lecture du
# CHANGELOG : sans le tag, rien de ce qui suit n'a de sens, et l'erreur doit nommer sa
# cause plutôt que de sortir d'un `curl` trois étapes plus loin.
#
# Le plafond est court. Passé une minute, ce n'est plus un décalage de réplication mais
# un miroir en panne, et il vaut mieux le dire que réessayer sans fin — même doctrine
# que le jeton absent : on s'arrête plutôt que de faire semblant.
# Faire passer le miroir, et le VÉRIFIER. Trois correctifs successifs ont échoué ici, et
# toujours pour la même raison de fond : prendre la réponse d'une API pour la preuve de
# l'effet cherché.
#
#   · attendre le tag ne suffit pas — le miroir ne repart pas de lui-même quand une poussée
#     arrive juste après son passage (v0.18.0 : deux secondes de retard, vingt-cinq minutes
#     sans rejouer) ;
#   · CI_JOB_TOKEN ne peut pas le réveiller — mesuré sur la v0.19.0, HTTP 401 : un jeton de
#     job n'a aucun droit sur les réglages du projet ;
#   · et surtout, **`204` ne veut pas dire que le miroir a tourné**. GitLab accepte la
#     demande et l'ignore si elle tombe dans les CINQ MINUTES qui suivent le passage
#     précédent. Mesuré sur la v0.20.0 : réveil accepté à 4 min 53 s, jamais exécuté ; la
#     même demande à 7 min 01 s est passée aussitôt. Sept secondes de trop.
#
# On observe donc `last_update_started_at`, qui est la seule chose qui dise que le miroir a
# réellement démarré, et on insiste jusqu'à ce qu'il avance. Le plafond couvre la fenêtre.
etat_miroir() {
  curl --silent --header "PRIVATE-TOKEN: ${GITLAB_API_TOKEN}" \
       "$GITLAB/remote_mirrors" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)[0]['last_update_started_at'])" 2>/dev/null
}

tag_sur_le_miroir() {
  "${API[@]}" --output /dev/null "https://api.github.com/repos/$DEPOT/git/refs/tags/$TAG" 2>/dev/null
}

faire_passer_le_miroir() {
  local depart maintenant i
  [ -n "${GITLAB_API_TOKEN:-}" ] || {
    echo "  · GITLAB_API_TOKEN absente — on se contente d'attendre le miroir"
    return 1
  }
  depart="$(etat_miroir)"
  [ -n "$depart" ] || {
    echo "  · état du miroir illisible — on se contente de l'attendre"
    return 1
  }
  echo "→ le tag n'est pas sur $DEPOT ; dernier passage du miroir : $depart"

  # 32 × 15 s = 8 min : la fenêtre de blocage est de 5 min, et on veut de la marge sans
  # attendre indéfiniment. On sort dès que le miroir démarre — le cas courant coûte donc
  # une poignée de secondes, pas huit minutes.
  for i in $(seq 1 "${JIGGER_INSISTANCE_MIROIR:-32}"); do
    curl --silent --output /dev/null --request POST \
         --header "PRIVATE-TOKEN: ${GITLAB_API_TOKEN}" \
         "$GITLAB/remote_mirrors/${GITHUB_MIROIR_ID:-2}/sync"
    sleep 15
    maintenant="$(etat_miroir)"
    if [ -n "$maintenant" ] && [ "$maintenant" != "$depart" ]; then
      echo "→ le miroir a tourné à $maintenant (après $((i * 15)) s d'insistance)"
      return 0
    fi
  done
  echo "  · le miroir n'a pas démarré après $(( ${JIGGER_INSISTANCE_MIROIR:-32} * 15 )) s" >&2
  return 1
}

# Le chemin rapide d'abord : si le miroir a déjà répliqué le tag de lui-même — ce qui arrive
# une fois sur deux — il n'y a rien à faire, et surtout rien à attendre.
if tag_sur_le_miroir; then
  echo "→ le tag est déjà sur $DEPOT"
else
  faire_passer_le_miroir || true
fi

ATTENTE="${JIGGER_ATTENTE_TAG:-60}"
attendu=0
until "${API[@]}" --output /dev/null \
      "https://api.github.com/repos/$DEPOT/git/refs/tags/$TAG" 2>/dev/null; do
  if [ "$attendu" -ge "$ATTENTE" ]; then
    echo "Le tag $TAG n'est pas arrivé sur $DEPOT après ${ATTENTE} s," >&2
    echo "alors que le miroir a été poussé et attendu. Il n'y a rien à publier ici tant" >&2
    echo "qu'il n'est pas passé, et cette fois ce n'est PAS la fenêtre de cinq minutes :" >&2
    echo "elle est traitée plus haut. Regarder l'état réel du miroir avant de relancer :" >&2
    echo "  curl -H \"PRIVATE-TOKEN: \$TOK\" $GITLAB/remote_mirrors" >&2
    echo "  → last_update_started_at doit être postérieur à la date du tag," >&2
    echo "    et last_error dire pourquoi si le miroir a échoué." >&2
    exit 1
  fi
  if [ "$attendu" -eq 0 ]; then
    echo "→ le tag n'est pas encore sur $DEPOT, attente du miroir…"
  fi
  sleep 5
  attendu=$((attendu + 5))
done
if [ "$attendu" -gt 0 ]; then
  echo "  · tag arrivé après ${attendu} s"
fi

# --- les notes de version -------------------------------------------------
# Mêmes notes que la release GitLab : celles du CHANGELOG, écrites avant le tag.
awk -v tag="$TAG" '
  $0 ~ "^## \\[" tag "\\]" { p = 1; next }
  p && /^## \[/ { exit }
  p { print }
' "$RACINE/CHANGELOG.md" > "$TRAVAIL/notes.md"
[ -s "$TRAVAIL/notes.md" ] || {
  echo "Aucune section « ## [$TAG] » dans CHANGELOG.md" >&2; exit 1; }

# GitLab reste la source de vérité : la release GitHub le dit, plutôt que de
# laisser croire que les contributions se font ici.
{
  cat "$TRAVAIL/notes.md"
  printf '\n---\n\n'
  printf 'Built from <https://gitlab.yg-devworks.com/yves/jigger>, which is the source\n'
  printf 'of truth; this repository is a mirror. Checksums for every archive are in\n'
  printf '`SHA256SUMS` below.\n'
} > "$TRAVAIL/corps.md"

# --- les fichiers ---------------------------------------------------------
mkdir -p "$TRAVAIL/dist"
if [ -n "$SOURCE" ]; then
  cp "$SOURCE"/* "$TRAVAIL/dist/"
else
  echo "→ récupération des archives depuis le registre GitLab"
  for nom in "jigger_${VERSION}_darwin_arm64.tar.gz" \
             "jigger_${VERSION}_darwin_amd64.tar.gz" \
             "jigger_${VERSION}_linux_amd64.tar.gz" \
             "jigger_${VERSION}_windows_amd64.zip" \
             SHA256SUMS; do
    curl --fail --silent --show-error --location \
         --output "$TRAVAIL/dist/$nom" \
         "$GITLAB/packages/generic/jigger/$VERSION/$nom"
    printf '  · %s\n' "$nom"
  done
fi

# Le condensat n'est pas recopié de confiance : on le revérifie ici, avant de
# téléverser. Une archive corrompue publiée sous un nom rassurant est pire que
# pas d'archive du tout.
( cd "$TRAVAIL/dist" && sha256sum --check --quiet SHA256SUMS 2>/dev/null \
    || shasum -a 256 --check --quiet SHA256SUMS ) \
  && echo "→ condensats vérifiés"

# --- la release -----------------------------------------------------------
if "${API[@]}" --output "$TRAVAIL/rel.json" \
     "https://api.github.com/repos/$DEPOT/releases/tags/$TAG" 2>/dev/null; then
  ID="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["id"])' "$TRAVAIL/rel.json")"
  echo "→ release existante ($ID) : notes mises à jour"
  python3 -c '
import json,sys
print(json.dumps({"name": sys.argv[1], "body": open(sys.argv[2]).read()}))
' "jigger $VERSION" "$TRAVAIL/corps.md" > "$TRAVAIL/patch.json"
  "${API[@]}" --output /dev/null --request PATCH \
      --data @"$TRAVAIL/patch.json" \
      "https://api.github.com/repos/$DEPOT/releases/$ID"
else
  echo "→ création de la release"
  # make_latest « legacy » : GitHub tranche sur la version sémantique plutôt que
  # sur la date de création. Sans lui, reprendre les anciennes versions APRÈS la
  # récente fait passer la plus vieille pour la dernière — c'est arrivé lors de
  # la reprise du 4 septembre, et il a fallu redresser v0.15.0 à la main.
  python3 -c '
import json,sys
print(json.dumps({"tag_name": sys.argv[1], "name": sys.argv[2],
                  "body": open(sys.argv[3]).read(),
                  "target_commitish": sys.argv[4], "draft": False,
                  "prerelease": False, "make_latest": "legacy"}))
' "$TAG" "jigger $VERSION" "$TRAVAIL/corps.md" "${CI_COMMIT_SHA:-main}" > "$TRAVAIL/post.json"
  "${API[@]}" --output "$TRAVAIL/rel.json" \
      --data @"$TRAVAIL/post.json" \
      "https://api.github.com/repos/$DEPOT/releases"
  ID="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["id"])' "$TRAVAIL/rel.json")"
fi

# --- les fichiers attachés ------------------------------------------------
# GitHub refuse un nom déjà présent : sur un rejeu, on retire avant de reposer.
"${API[@]}" --output "$TRAVAIL/assets.json" \
    "https://api.github.com/repos/$DEPOT/releases/$ID/assets"

for f in "$TRAVAIL"/dist/*; do
  nom="$(basename "$f")"
  ancien="$(python3 -c '
import json,sys
for a in json.load(open(sys.argv[1])):
    if a["name"] == sys.argv[2]:
        print(a["id"]); break
' "$TRAVAIL/assets.json" "$nom")"
  if [ -n "$ancien" ]; then
    "${API[@]}" --output /dev/null --request DELETE \
        "https://api.github.com/repos/$DEPOT/releases/assets/$ancien"
  fi
  "${API[@]}" --output /dev/null \
      --header "Content-Type: application/octet-stream" \
      --data-binary @"$f" \
      "https://uploads.github.com/repos/$DEPOT/releases/$ID/assets?name=$nom"
  printf '  + %s\n' "$nom"
done

echo "Publié : https://github.com/$DEPOT/releases/tag/$TAG"
