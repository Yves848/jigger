#!/bin/sh
# La passe macOS en une commande : tester, consigner, publier.
#
#   tests/passe-macos.sh [--banc] [--sans-push] [--message "…"]
#
# Le pendant de tests/passe-windows.ps1, et c'est tout l'intérêt : le même rituel des deux
# côtés. Un rituel qui ne se pratique que d'un côté finit par ne plus se pratiquer du tout,
# et c'est exactement ainsi que les analyseurs scoop ont pu rendre zéro ligne pendant une
# version entière — le code passait tous les tests, sur la seule machine où on les lançait.
#
# Ce qu'il publie : le rapport détaillé `tests/captures/derniers-tests-macos.md` (l'état du
# moment, écrasé à chaque passe) et une entrée dans `docs/tests/journal.md` (ajoutée,
# jamais réécrite, échecs en clair).
#
# Un échec n'empêche pas la publication — c'est l'information la plus utile des deux — mais
# le script sort en code non nul.
#
#   --banc        lance aussi le banc de non-régression du rendu, s'il existe
#   --sans-push   commite sans pousser
#   --message …   message de commit (sinon : un message qui résume la passe)
set -eu

racine=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$racine"

banc=0
push=1
message=""
while [ $# -gt 0 ]; do
  case $1 in
    --banc)      banc=1 ;;
    --sans-push) push=0 ;;
    --message)   shift; message=${1:-} ;;
    *) echo "option inconnue : $1" >&2; exit 2 ;;
  esac
  shift
done

travail=$(mktemp -d) || exit 2
trap 'rm -rf "$travail"' EXIT
: > "$travail/etapes"
n=0

# etape lance une commande, montre sa sortie et retient son code. Rien ne s'arrête sur un
# échec : on veut la passe complète, pas la première mauvaise nouvelle.
#
# La sortie est **diffusée** pendant que l'étape tourne, et capturée en même temps. La
# première version l'écrivait dans un fichier puis l'affichait à la fin : une étape lente
# donnait un silence complet, impossible à distinguer d'un script figé. `tee` fait les
# deux ; le code de retour de la commande, lui, transite par un fichier, sh n'ayant pas
# l'équivalent de PIPESTATUS.
etape() {
  nom=$1
  shift
  n=$((n + 1))
  printf '\n→ %s\n' "$nom"
  debut=$(date +%s)
  { "$@" 2>&1; echo $? > "$travail/code-$n"; } | tee "$travail/sortie-$n"
  code=$(cat "$travail/code-$n")
  duree=$(( $(date +%s) - debut ))
  if [ "$code" -eq 0 ]; then
    printf '  ok — %s s\n' "$duree"
  else
    printf '  ÉCHEC (code %s) — %s s\n' "$code" "$duree"
  fi
  printf '%s\t%s\t%s\t%s\n' "$n" "$code" "$nom" "$duree" >> "$travail/etapes"
}

printf '→ mise à jour depuis origin\n'
git pull --ff-only 2>&1 || printf '  (pull sans effet)\n'

etape 'go build' go build -o jigger .
etape 'go test' go test ./...
etape 'zpty.zsh (vrai pseudo-terminal)' ./tests/zpty.zsh --suite

# La suite PowerShell tourne aussi ici depuis la v0.8.0 : c'est ce qui permet de développer
# le module sans démarrer une machine Windows. Elle ne juge pas le popup à l'écran — ça,
# seul tests/pty.ps1 le fait, et seulement sous Windows.
if command -v pwsh > /dev/null 2>&1; then
  etape 'smoke.ps1' pwsh -NoProfile -File tests/smoke.ps1
else
  printf '\n→ smoke.ps1 : sauté (pwsh absent)\n'
fi

if [ "$banc" -eq 1 ] && [ -x tests/render-golden.sh ]; then
  etape 'banc de rendu (français figé)' ./tests/render-golden.sh --verifier
fi

# ── Verdict ───────────────────────────────────────────────────────────────────────────
echecs=$(awk -F'\t' '$2 != 0' "$travail/etapes" | wc -l | tr -d ' ')
case $echecs in
  0) verdict='tout passe' ;;
  1) verdict='1 étape en échec' ;;
  *) verdict="$echecs étapes en échec" ;;
esac

commit=$(git rev-parse --short HEAD)
contexte="macOS $(sw_vers -productVersion 2>/dev/null || uname -r) · zsh $(zsh --version 2>/dev/null | cut -d' ' -f2) · $(go version | cut -d' ' -f3)"

# ── Le rapport détaillé, écrasé à chaque passe ────────────────────────────────────────
mkdir -p tests/captures
rapport=tests/captures/derniers-tests-macos.md
{
  printf '# Passe macOS — derniers résultats\n\n'
  printf '*Engendré par `tests/passe-macos.sh`. Ne pas modifier à la main.*\n\n'
  printf '**Verdict : %s.**\n\n' "$verdict"
  printf '| Étape | Code | Durée |\n|---|---|---|\n'
  while IFS="$(printf '\t')" read -r i code nom duree; do
    if [ "$code" -eq 0 ]; then printf '| %s | ok | %s s |\n' "$nom" "$duree"; else printf '| %s | ÉCHEC (%s) | %s s |\n' "$nom" "$code" "$duree"; fi
  done < "$travail/etapes"
  printf '\n## Contexte\n\n```\ndate    : %s\n%s\ncommit  : %s\n```\n' \
    "$(date '+%Y-%m-%d %H:%M:%S')" "$contexte" "$commit"
  while IFS="$(printf '\t')" read -r i code nom duree; do
    printf '\n## %s\n\n```\n' "$nom"
    # Une sortie verte est longue et sans intérêt : on n'en garde que la fin. Une sortie
    # en échec est gardée entière — c'est elle qu'on vient lire.
    if [ "$code" -eq 0 ]; then tail -25 "$travail/sortie-$i"; else cat "$travail/sortie-$i"; fi
    printf '```\n'
  done < "$travail/etapes"
} > "$rapport"

# ── Le journal, ajouté et jamais réécrit ──────────────────────────────────────────────
journal=docs/tests/journal.md
marqueur='<!-- nouvelles passes ici -->'
if [ -f "$journal" ] && grep -qF "$marqueur" "$journal"; then
  {
    printf '## %s — macOS — `%s` — %s\n\n' "$(date '+%Y-%m-%d %H:%M')" "$commit" "$verdict"
    printf '%s\n\n' "$contexte"
    verts=$(awk -F'\t' '$2 == 0 {printf "%s · ", $3}' "$travail/etapes" | sed 's/ · $//')
    [ -n "$verts" ] && printf -- '- **ok** — %s\n' "$verts"
    printf -- '- durée totale : %s s\n' "$(awk -F'\t' '{t += $4} END {print t}' "$travail/etapes")"
    while IFS="$(printf '\t')" read -r i code nom duree; do
      [ "$code" -eq 0 ] && continue
      printf -- '- **échec — %s** (code %s)\n' "$nom" "$code"
      # Les lignes que les harnais impriment pour dire ce qui a lâché — c'est ce qu'un
      # lecteur veut voir, pas les cent lignes vertes autour.
      grep -E 'ÉCHEC|--- FAIL|^FAIL' "$travail/sortie-$i" | head -6 | sed 's/^[[:space:]]*/  - /'
    done < "$travail/etapes"
    printf '\n'
  } > "$travail/entree"
  awk -v marqueur="$marqueur" -v fichier="$travail/entree" '
    { print }
    index($0, marqueur) { print ""; while ((getline l < fichier) > 0) print l }
  ' "$journal" > "$travail/journal" && mv "$travail/journal" "$journal"
else
  printf '  (docs/tests/journal.md absent ou sans marqueur — entrée non ajoutée)\n'
fi

# ── Publication ───────────────────────────────────────────────────────────────────────
printf '\n→ publication\n'
git add tests/captures docs/tests/journal.md 2>/dev/null || true
if [ -z "$(git diff --cached --name-only)" ]; then
  printf '  rien de neuf à commiter\n'
else
  printf '  fichiers : %s\n' "$(git diff --cached --name-only | tr '\n' ' ')"
  [ -n "$message" ] || message="Passe macOS : $verdict"
  git commit -m "$message"
  if [ "$push" -eq 1 ]; then git push; else printf '  (push sauté)\n'; fi
fi

reste=$(git status --short)
if [ -n "$reste" ]; then
  printf '\n  non commité (à toi de voir) :\n%s\n' "$reste"
fi

printf '\n%s.\n' "$verdict"
[ "$echecs" -eq 0 ] || exit 1
