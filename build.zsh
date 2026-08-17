#!/usr/bin/env zsh
# Compiler jigger depuis la ligne de commande — le pendant zsh de build.ps1.
#
#   ./build.zsh              # compile le binaire `jigger` dans le dépôt
#   ./build.zsh --deploy     # compile *et* l'installe dans $PREFIXE/bin
#
# Sans `--deploy`, le script ne touche à rien d'autre qu'au binaire du dépôt : c'est le
# geste de tous les jours, celui qu'on répète entre deux modifications. Avec `--deploy`,
# il fait ce que fait `make install` — le binaire dans `~/.local/bin`, et la ligne du
# greffon rappelée (ou ajoutée au ~/.zshrc avec `--profil`).
#
# ── Options ───────────────────────────────────────────────────────────────────────────
#
#   --deploy             installe le binaire après la compilation
#   --prefixe <dossier>  où l'installer (défaut : $HOME/.local, donc .../bin/jigger)
#   --profil             ajoute le `source …/jigger.plugin.zsh` au ~/.zshrc s'il manque
#   --simuler            montre ce qui serait fait, sans rien faire
#   --help               l'aide
#
# Les mêmes noms qu'install-windows.ps1 et build.ps1, à un tiret près : le rituel est le
# même des deux côtés, et un rituel qui change de vocabulaire d'une plateforme à l'autre
# est un rituel qu'on relit à chaque fois au lieu de le taper.

set -eu

racine=${0:A:h}
cd "$racine"

deployer=0
profil=0
simuler=0
prefixe=${PREFIX:-$HOME/.local}

if [[ -t 1 ]]; then
  cyan=$'\e[36m'; vert=$'\e[32m'; rouge=$'\e[31m'; jaune=$'\e[33m'; gris=$'\e[90m'; net=$'\e[0m'
else
  cyan=''; vert=''; rouge=''; jaune=''; gris=''; net=''
fi

aide() {
  cat <<'FIN'
Compiler jigger.

  ./build.zsh [--deploy] [options]

  --deploy             installe le binaire après la compilation
  --prefixe <dossier>  où l'installer (défaut : $HOME/.local → .../bin/jigger)
  --profil             ajoute la ligne du greffon au ~/.zshrc   (avec --deploy)
  --simuler            montre ce qui serait fait, sans rien faire
  --help               cette aide
FIN
}

while (( $# > 0 )); do
  case $1 in
    --deploy)  deployer=1 ;;
    --profil)  profil=1 ;;
    --simuler) simuler=1 ;;
    --prefixe)
      shift
      if (( $# == 0 )); then print -u2 -- "--prefixe attend un dossier"; exit 2; fi
      prefixe=$1
      ;;
    --help|-h) aide; exit 0 ;;
    *) print -u2 -- "option inconnue : $1"; aide >&2; exit 2 ;;
  esac
  shift
done

# `faire` exécute, ou raconte sans exécuter. Une seule porte pour --simuler : une étape
# qui l'oublierait installerait pour de bon pendant qu'on croit regarder une simulation.
faire() {
  local quoi=$1
  shift
  if (( simuler )); then
    print -- "  ${gris}(simulation) $quoi${net}"
  else
    print -- "  $quoi"
    "$@"
  fi
}

# ── 1. Compiler ───────────────────────────────────────────────────────────────────────

if ! command -v go >/dev/null 2>&1; then
  print -u2 -- "${rouge}Go est introuvable. Installe-le d'abord : brew install go${net}"
  exit 1
fi

binaire=$racine/jigger

print -- "${cyan}→ compilation${net}"
debut=$SECONDS
faire "go build -o jigger ." go build -o jigger .
duree=$(( SECONDS - debut ))

if (( ! simuler )); then
  # Un `go build` muet qui ne produit rien est possible (dossier en lecture seule,
  # binaire occupé) : on le dit, plutôt que d'annoncer un succès sur le binaire de la
  # fois d'avant.
  if [[ ! -x $binaire ]]; then
    print -u2 -- "${rouge}Le binaire n'a pas été produit : $binaire${net}"
    exit 1
  fi
  version=$("$binaire" --version 2>&1 | head -1)
  octets=$(wc -c < "$binaire" | tr -d ' ')
  print -- "  ${vert}${version} — $(( octets / 1048576 )) Mio en ${duree} s${net}"
  print -- "  $binaire"
fi

if (( ! deployer )); then
  exit 0
fi

# ── 2. Installer ──────────────────────────────────────────────────────────────────────
#
# Les mêmes deux lignes que `make install`, et le même défaut de PREFIX : ce script et le
# Makefile doivent poser le binaire au même endroit, sinon `which jigger` désigne l'un
# pendant qu'on recompile l'autre.

destination=$prefixe/bin

print -- "${cyan}→ installation dans $destination${net}"
faire "install -d $destination" install -d "$destination"
faire "install -m 0755 jigger $destination/jigger" install -m 0755 "$binaire" "$destination/jigger"

case ":${PATH}:" in
  *":${destination}:"*) : ;;
  *) print -- "  ${jaune}⚠ $destination n'est pas dans le PATH${net}" ;;
esac

# ── 3. Le greffon zsh ─────────────────────────────────────────────────────────────────

greffon=$racine/shell/jigger.plugin.zsh
ligne="source $greffon"
zshrc=${ZDOTDIR:-$HOME}/.zshrc

print -- "${cyan}→ greffon zsh${net}"
if [[ -f $zshrc ]] && grep -q 'jigger\.plugin\.zsh' "$zshrc"; then
  print -- "  déjà présent dans $zshrc"
elif (( profil )); then
  # `mkdir -p` d'abord : avec un $ZDOTDIR qui pointe vers un dossier encore absent, la
  # redirection échoue et le script s'arrête après avoir installé le binaire — à mi-chemin,
  # donc, ce qui est le pire endroit où s'arrêter.
  ajouter_au_zshrc() { mkdir -p "${zshrc:h}" && printf '\n# jigger\n%s\n' "$ligne" >> "$zshrc"; }
  faire "ajout au $zshrc" ajouter_au_zshrc
  print -- "  ${jaune}recharge : source $zshrc${net}"
else
  print -- "  à ajouter dans $zshrc (ou relance avec --profil) :"
  print -- "    ${jaune}${ligne}${net}"
fi

# ── 4. Vérifier ───────────────────────────────────────────────────────────────────────

if (( simuler )); then
  print -- "  ${gris}(simulation) $destination/jigger --version${net}"
  exit 0
fi

print -- "${cyan}→ vérification${net}"
print -- "  $("$destination/jigger" --version 2>&1 | head -1)"
print -- ""
print -- "${vert}Installé — $destination/jigger${net}"
print -- "Ouvre un terminal neuf et tape « brew ins » sans valider : le cadre doit apparaître."
