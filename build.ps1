# Compiler jigger depuis la ligne de commande — le pendant Windows de build.zsh.
#
#   pwsh -NoProfile -File build.ps1              # compile jigger.exe dans le dépôt
#   pwsh -NoProfile -File build.ps1 --deploy     # compile *et* rend `jigger` appelable
#
# Sans `--deploy`, le script ne touche à rien d'autre qu'au binaire du dépôt : c'est le
# geste de tous les jours, celui qu'on répète entre deux modifications. Le greffon, lui,
# lit `jigger.exe` là où il est — un build suffit donc à voir le changement, à condition
# que `jigger` soit déjà installé en shim (cf. install-windows.ps1).
#
# ── Options ───────────────────────────────────────────────────────────────────────────
#
#   --deploy / -Deploy   installe après la compilation (passe la main à install-windows.ps1)
#   -Methode shim|copie  comment rendre `jigger` appelable — avec --deploy uniquement
#   -Prefixe <dossier>   où copier le binaire — avec --deploy et -Methode copie
#   -Profil              ajoute l'Import-Module au $PROFILE — avec --deploy uniquement
#   -Simuler             montre ce qui serait fait, sans rien faire
#
# Les options longues à deux tirets (`--deploy`, `--profil`, `--simuler`, `--prefixe`,
# `--methode`) sont acceptées à côté des formes PowerShell : le script s'appelle des deux
# côtés avec la même ligne, et build.zsh ne connaît que celles-là.
#
# ── Pourquoi `--deploy` délègue plutôt que d'installer lui-même ───────────────────────
#
# install-windows.ps1 sait déjà compiler puis installer — shim scoop ou copie, PATH
# utilisateur, $PROFILE. Réécrire ces trois étapes ici donnerait deux vérités à tenir à
# jour, et la seconde se serait décalée dès la première correction. Le script lui passe
# donc la main entière, compilation comprise : une seule compilation, pas deux.

[CmdletBinding()]
param(
    [switch]$Deploy,
    [ValidateSet('shim', 'copie')]
    [string]$Methode,
    [string]$Prefixe,
    [switch]$Profil,
    [switch]$Simuler,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Reste
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = $PSScriptRoot
Set-Location $racine

function Aide {
    Write-Host @'
Compiler jigger.

  pwsh -NoProfile -File build.ps1 [--deploy] [options]

  --deploy              installe après la compilation (install-windows.ps1)
  --methode shim|copie  comment rendre `jigger` appelable      (avec --deploy)
  --prefixe <dossier>   où copier le binaire                   (avec --deploy, méthode copie)
  --profil              ajoute l'Import-Module au $PROFILE     (avec --deploy)
  --simuler             montre ce qui serait fait, sans rien faire
  --help                cette aide
'@
}

# ── Les options longues ───────────────────────────────────────────────────────────────
#
# PowerShell ne lie que les formes `-Nom` : tout ce qui commence par deux tirets lui
# arrive comme argument positionnel, donc ici. On les traduit à la main plutôt que de
# laisser passer un `--deploy` silencieusement ignoré — une option ignorée sans un mot,
# c'est un déploiement qu'on croit avoir fait.
if ($Reste) {
    for ($i = 0; $i -lt $Reste.Count; $i++) {
        switch ($Reste[$i]) {
            '--deploy'  { $Deploy = $true }
            '--profil'  { $Profil = $true }
            '--simuler' { $Simuler = $true }
            '--methode' {
                $i++
                if ($i -ge $Reste.Count) { Write-Host '--methode attend « shim » ou « copie »' -ForegroundColor Red; exit 2 }
                $Methode = $Reste[$i]
                if ($Methode -notin @('shim', 'copie')) { Write-Host "méthode inconnue : $Methode" -ForegroundColor Red; exit 2 }
            }
            '--prefixe' {
                $i++
                if ($i -ge $Reste.Count) { Write-Host '--prefixe attend un dossier' -ForegroundColor Red; exit 2 }
                $Prefixe = $Reste[$i]
            }
            { $_ -in @('--help', '-h', '-?') } { Aide; exit 0 }
            default {
                Write-Host "option inconnue : $($Reste[$i])" -ForegroundColor Red
                Aide
                exit 2
            }
        }
    }
}

# `$IsWindows` n'existe pas en Windows PowerShell 5.1 ; l'absence de la variable y vaut
# donc « oui, Windows ».
$surWindows = (-not (Test-Path variable:IsWindows)) -or $IsWindows

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Go est introuvable. Installe-le d'abord : scoop install go" -ForegroundColor Red
    exit 1
}

# ── Déploiement : install-windows.ps1 fait tout, compilation comprise ─────────────────

if ($Deploy) {
    if (-not $surWindows) {
        Write-Host 'Le déploiement PowerShell vise Windows. Hors Windows : ./build.zsh --deploy' -ForegroundColor Red
        exit 2
    }

    $installeur = Join-Path $racine 'install-windows.ps1'
    if (-not (Test-Path $installeur)) {
        Write-Host "install-windows.ps1 est introuvable dans $racine" -ForegroundColor Red
        exit 1
    }

    $options = @{}
    if ($Methode) { $options['Methode'] = $Methode }
    if ($Prefixe) { $options['Prefixe'] = $Prefixe }
    if ($Profil)  { $options['Profil']  = $true }
    if ($Simuler) { $options['Simuler'] = $true }

    Write-Host '→ compilation et installation (install-windows.ps1)' -ForegroundColor Cyan
    & $installeur @options
    $code = if (Test-Path variable:LASTEXITCODE) { $LASTEXITCODE } else { 0 }
    exit $code
}

# ── Compilation seule ─────────────────────────────────────────────────────────────────

$binaire = if ($surWindows) { 'jigger.exe' } else { 'jigger' }
$chemin = Join-Path $racine $binaire

Write-Host "→ go build -o $binaire ." -ForegroundColor Cyan

if ($Simuler) {
    Write-Host "  (simulation) rien n'a été compilé" -ForegroundColor DarkGray
    exit 0
}

$chrono = [Diagnostics.Stopwatch]::StartNew()
& go build -o $binaire '.'
$codeBuild = $LASTEXITCODE
$chrono.Stop()

if ($codeBuild -ne 0) {
    Write-Host 'la compilation a échoué' -ForegroundColor Red
    exit $codeBuild
}

# Un `go build` muet qui ne produit rien est possible (dossier en lecture seule, binaire
# verrouillé par un jigger en cours d'exécution) : on le dit plutôt que d'annoncer un
# succès sur le binaire de la fois d'avant.
if (-not (Test-Path $chemin)) {
    Write-Host "Le binaire n'a pas été produit : $chemin" -ForegroundColor Red
    exit 1
}

$taille = [math]::Round((Get-Item $chemin).Length / 1MB, 1)
$version = (& $chemin --version 2>&1 | Out-String).Trim()

Write-Host "  $version — $taille Mio en $([math]::Round($chrono.Elapsed.TotalSeconds, 1)) s" -ForegroundColor Green
Write-Host "  $chemin"
