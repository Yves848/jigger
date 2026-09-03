<#
.SYNOPSIS
  Produit les enregistrements et les captures de jigger sur Windows.

.DESCRIPTION
  L'équivalent Windows de capturer.sh, mais la mécanique diffère — et il faut
  savoir pourquoi avant de s'étonner qu'il demande une frappe humaine.

  Sur zsh, les captures sont entièrement pilotées : VHS rejoue un script de
  frappe dans tmux. Aucune des deux pièces ne tient son rôle ici. VHS passe par
  ttyd, qui n'a pas d'équivalent Windows utilisable, et tmux n'existe pas. Rien
  de tout cela n'est un défaut de jigger : le module PowerShell lit la position
  du curseur par PSReadLine et n'a, lui, aucun besoin de tmux.

  Ce script fige donc ce qui décide de l'image — dimensions, police, palette,
  prompt, hôtes SSH, langue — et laisse la frappe à l'opérateur. C'est le décor
  qui fait la cohérence entre les trois plateformes, pas le pilotage : les
  lignes à taper sont écrites ici, identiques à celles des tapes zsh.

  L'enregistrement se fait avec la barre de jeu de Windows (Win+Alt+R), présente
  sur toute installation, plutôt qu'avec une dépendance de plus. Le
  post-traitement, lui, est exactement celui d'Unix — mêmes dimensions, même
  fréquence, même instant d'extraction — pour que les fichiers soient
  comparables.

.EXAMPLE
  pwsh -File docs\media\capturer.ps1 -Preparer
  # ouvre le terminal de capture pour le scénario 01

.EXAMPLE
  pwsh -File docs\media\capturer.ps1 -Convertir C:\Users\yves\Videos\Captures
  # transforme les .mp4 enregistrés en .gif + .png dans out\

.NOTES
  Ce script n'a PAS été exécuté par son auteur : le dépôt a été documenté depuis
  un Mac, et aucune machine Windows n'était joignable. Il est écrit d'après le
  module (shell/jigger.psm1) et d'après capturer.sh, dont il reprend les
  constantes. Signaler tout écart plutôt que de le corriger en silence : c'est
  la seule partie du protocole de capture qui n'a jamais tourné.
#>
[CmdletBinding()]
param(
  # Ouvre un terminal au décor figé, prêt à filmer.
  [switch]$Preparer,
  # Le scénario : 01-gestionnaire-natif, 02-jg ou 03-ssh.
  [ValidateSet('01-gestionnaire-natif','02-jg','03-ssh')]
  [string]$Scenario = '01-gestionnaire-natif',
  # Transforme les .mp4 d'un dossier en .gif + .png dans out\.
  [string]$Convertir
)

$ErrorActionPreference = 'Stop'
$Media = Split-Path -Parent $PSCommandPath
$Repo  = Split-Path -Parent (Split-Path -Parent $Media)
$Out   = Join-Path $Media 'out'
New-Item -ItemType Directory -Force -Path $Out | Out-Null

# Les mêmes constantes que generer-tapes.sh. Les changer ici sans les changer
# là-bas ferait diverger Windows des deux autres plateformes en silence.
$Colonnes = 72
$Lignes   = 24
$Police   = 'MesloLGL Nerd Font'
$Taille   = 22

# La ligne à taper, et l'instant où l'image fixe est prise. Identiques aux tapes
# zsh : c'est ce qui fait que les trois plateformes racontent la même chose.
$Scenarios = @{
  '01-gestionnaire-natif' = @{ Ligne = 'winget install fire'; Instant = 4.5 }
  '02-jg'                 = @{ Ligne = 'jg install fd';       Instant = 4.0 }
  '03-ssh'                = @{ Ligne = 'ssh ';                Instant = 3.0 }
}

function Invoke-Preparer {
  $s = $Scenarios[$Scenario]

  $env:JIGGER_REPO = $Repo
  $env:JIGGER_LANG = if ($env:JIGGER_LANG) { $env:JIGGER_LANG } else { 'en' }

  # Le sélecteur SSH lit le ~/.ssh/config du profil, sans surcharge possible
  # (internal/ssh/manager.go lit os.UserHomeDir, soit %USERPROFILE% ici). On lui
  # donne le HOME de fixture : la capture montre les serveurs inventés du dépôt,
  # les mêmes que sur macOS et Omarchy, et jamais l'infrastructure réelle.
  if ($Scenario -eq '03-ssh') {
    $env:USERPROFILE = Join-Path $Media 'fixtures\home'
    $env:HOME        = $env:USERPROFILE
  }

  Write-Host ''
  Write-Host "  Scénario  : $Scenario"    -ForegroundColor Cyan
  Write-Host "  À taper   : $($s.Ligne)"  -ForegroundColor Yellow
  Write-Host "  Puis      : ↓  ↓  ⇥      (naviguer, puis insérer)" -ForegroundColor Yellow
  Write-Host ''
  Write-Host '  1. Win+Alt+R  démarre l''enregistrement de la fenêtre.'
  Write-Host '  2. Taper la ligne SANS se presser, laisser le popup se poser.'
  Write-Host '  3. ↓ ↓ ⇥, attendre deux secondes.'
  Write-Host '  4. Win+Alt+R  arrête. Le .mp4 va dans Vidéos\Captures.'
  Write-Host ''
  Write-Host "  Puis : pwsh -File docs\media\capturer.ps1 -Convertir `"$env:USERPROFILE\Videos\Captures`""
  Write-Host ''

  $profil = Join-Path $Media 'fixtures\profile.ps1'
  # -NoProfile : le $PROFILE de la machine ne doit pas entrer dans la capture.
  # Les dimensions sont posées après le démarrage, une fenêtre ne pouvant être
  # dimensionnée en cellules à la ligne de commande.
  $amorce = @"
`$Host.UI.RawUI.WindowTitle = 'jigger — capture $Scenario'
mode con: cols=$Colonnes lines=$Lignes
. '$profil'
"@
  $tmp = Join-Path $env:TEMP "jigger-capture-$Scenario.ps1"
  Set-Content -Path $tmp -Value $amorce -Encoding UTF8

  Write-Host "  Police à choisir dans les réglages du terminal : $Police $Taille pt" -ForegroundColor DarkGray
  Write-Host '  Palette : Catppuccin Mocha (docs/captures.md donne les valeurs).' -ForegroundColor DarkGray
  Write-Host ''

  wt.exe --title "jigger — capture $Scenario" `
         new-tab --size "$Colonnes,$Lignes" `
         pwsh -NoExit -NoProfile -File $tmp
}

function Invoke-Convertir {
  if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
    throw 'ffmpeg est requis pour la conversion (winget install Gyan.FFmpeg).'
  }
  $sources = Get-ChildItem -Path $Convertir -Filter '*.mp4' | Sort-Object LastWriteTime
  if (-not $sources) { throw "aucun .mp4 dans $Convertir" }

  Write-Host "  $($sources.Count) enregistrement(s) à convertir." -ForegroundColor Cyan
  Write-Host '  Associer chacun à son scénario :' -ForegroundColor Cyan
  $i = 0
  foreach ($src in $sources) {
    $i++
    Write-Host "   [$i] $($src.Name)  ($([math]::Round($src.Length/1MB,1)) Mo)"
  }
  foreach ($cle in $Scenarios.Keys | Sort-Object) {
    $rep = Read-Host "  Numéro de l'enregistrement pour « windows-$cle » (vide pour passer)"
    if (-not $rep) { continue }
    $src = $sources[[int]$rep - 1]
    $nom = "windows-$cle"
    $gif = Join-Path $Out "$nom.gif"
    $mp4 = Join-Path $Out "$nom.mp4"
    $png = Join-Path $Out "$nom.png"

    # La même échelle et la même fréquence que les tapes VHS, pour que les GIF
    # des trois plateformes aient le même poids et la même fluidité.
    ffmpeg -y -loglevel error -i $src.FullName `
      -vf "fps=24,scale=1000:-1:flags=lanczos,split[a][b];[a]palettegen[p];[b][p]paletteuse" $gif
    ffmpeg -y -loglevel error -i $src.FullName -vf 'scale=1000:-1' -an $mp4
    ffmpeg -y -loglevel error -ss $Scenarios[$cle].Instant -i $gif -vframes 1 $png
    Write-Host "   → $nom.gif  $nom.mp4  $nom.png" -ForegroundColor Green
  }
}

if ($Convertir) { Invoke-Convertir } elseif ($Preparer) { Invoke-Preparer } else { Get-Help $PSCommandPath }
