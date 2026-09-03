#!/usr/bin/env bash
# Génère les tapes VHS des deux plateformes qui savent jouer VHS : macOS et Omarchy.
#
# Windows n'en a pas. VHS y passerait par ttyd, qui n'existe pas, et par tmux, qui
# n'existe pas non plus ; un tape « windows-*.tape » ne pourrait jamais être joué.
# Le décor et les gestes y sont tenus par docs/media/capturer.ps1, qui reprend les
# constantes ci-dessous et les Sleep des tapes, scénario par scénario.
#
# Le préambule — police, taille, dimensions, thème, vitesse de frappe — est écrit
# ici UNE fois et recopié tel quel dans chaque tape. C'est ce qui garantit que la
# capture de macOS et celle d'Omarchy sont comparables : rien de ce qui décide de
# l'image ne peut diverger par recopie manuelle.
#
# Les tapes produits sont pour autant AUTONOMES : aucune directive « Source », rien
# à installer d'autre que VHS. On copie un fichier sur la machine cible et on le
# lance. C'est le sens de « instructions réutilisables » : le script est
# l'instruction.
#
#   ./docs/media/generer-tapes.sh      # réécrit docs/media/tapes/*.tape
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p tapes

# Catppuccin Mocha, la palette de la charte. Posée en JSON dans chaque tape plutôt
# que choisie parmi les 348 thèmes intégrés de VHS : la liste varie d'une version à
# l'autre, la palette écrite ne varie pas.
THEME='{ "background": "#1e1e2e", "foreground": "#cdd6f4", "cursor": "#f5e0dc", "selection": "#585b70", "black": "#45475a", "red": "#f38ba8", "green": "#a6e3a1", "yellow": "#f9e2af", "blue": "#89b4fa", "magenta": "#f5c2e7", "cyan": "#94e2d5", "white": "#bac2de", "brightBlack": "#585b70", "brightRed": "#f38ba8", "brightGreen": "#a6e3a1", "brightYellow": "#f9e2af", "brightBlue": "#89b4fa", "brightMagenta": "#f5c2e7", "brightCyan": "#94e2d5", "brightWhite": "#a6adc8" }'

preambule() { # $1 = nom de sortie
  cat <<EOF
Output out/$1.gif
Output out/$1.mp4

# --- décor figé : identique sur macOS et Omarchy (et repris par capturer.ps1) ---
Set FontFamily "MesloLGL Nerd Font"
Set FontSize 22
Set Width 1000
Set Height 530
Set Padding 24
Set TypingSpeed 90ms
Set Framerate 24
Set Theme $THEME
EOF
}

# Le shell capturé tourne dans tmux — sans quoi le popup ne s'affiche pas du tout
# sous un enregistreur (DSR, cf. docs/captures.md).
amorce_zsh() {
  cat <<'EOF'
Set Shell "zsh"
Hide
Type "tmux -L jiggercap -f $JIGGER_MEDIA/fixtures/tmux.conf new-session -A -s capture zsh" Enter
Sleep 2s
Type "clear" Enter
Sleep 1s
Show
Sleep 800ms
EOF
}
ecrire() { # $1=plateforme  $2=nom  $3=amorce  $4=corps
  local f="tapes/$1-$2.tape"
  { preambule "$1-$2"; echo; $3; echo; printf '%s\n' "$4"; } > "$f"
  echo "  $f"
}

echo "Tapes générés :"

# --- 01 · le gestionnaire natif ---------------------------------------------
# La démonstration que l'utilisateur attend en premier : on tape la commande
# qu'on tapait déjà, et le popup arrive tout seul. Aucune touche pressée.
ecrire macos   01-gestionnaire-natif amorce_zsh  'Type "brew install fire"
Sleep 3s
Down
Sleep 1s
Down
Sleep 1s
Tab
Sleep 2500ms'

ecrire omarchy 01-gestionnaire-natif amorce_zsh  'Type "yay -S visual-studio"
Sleep 3s
Down
Sleep 1s
Down
Sleep 1s
Tab
Sleep 2500ms'


# --- 02 · la syntaxe unique « jg » ------------------------------------------
# Le même geste, mais sans avoir à savoir quel gestionnaire connaît le paquet.
ecrire macos   02-jg amorce_zsh  'Type "jg install fd"
Sleep 3s
Down
Sleep 1200ms
Tab
Sleep 2500ms'

ecrire omarchy 02-jg amorce_zsh  'Type "jg install fd"
Sleep 3s
Down
Sleep 1200ms
Tab
Sleep 2500ms'


# --- 03 · le sélecteur SSH ---------------------------------------------------
# Une commande sans verbe : le catalogue vient dès l'espace. Les hôtes sont ceux
# du fixture, donc les mêmes sur les trois plateformes.
ecrire macos   03-ssh amorce_zsh  'Type "ssh "
Sleep 2500ms
Down
Sleep 1s
Down
Sleep 1s
Tab
Sleep 2500ms'

ecrire omarchy 03-ssh amorce_zsh  'Type "ssh "
Sleep 2500ms
Down
Sleep 1s
Down
Sleep 1s
Tab
Sleep 2500ms'


# --- 04 · la bascule regex ---------------------------------------------------
# Le seul scénario où l'on presse une touche pour changer quelque chose, et le
# seul qui montre le mode regex : le reste de la documentation l'énonce, aucune
# image ne le montrait hors de Windows.
#
# La bascule est jouée sur une saisie DÉJÀ filtrée, parce que c'est là qu'elle
# apprend le plus : « fire » en préfixe ne retient que les noms qui commencent
# par « fire », en regex il retient aussi « arrayfire » — le motif n'est pas
# ancré. Une capture qui basculerait sur une ligne vide ne montrerait que le
# « [regex] » du titre, pas ce qu'il change.
#
# Vient ensuite l'alternance, qu'aucune recherche par préfixe ne sait exprimer.
ecrire macos   04-regex amorce_zsh  'Type "brew install fire"
Sleep 3s
Ctrl+R
Sleep 2s
Type "(bird|fly)"
Sleep 3s
Tab
Sleep 2500ms'

ecrire omarchy 04-regex amorce_zsh  'Type "yay -S fire"
Sleep 3s
Ctrl+R
Sleep 2s
Type "(bird|fly)"
Sleep 3s
Tab
Sleep 2500ms'

