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
