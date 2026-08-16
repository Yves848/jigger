# Compiler et installer jigger sous Windows — l'équivalent de `make install`.
#
#   pwsh -NoProfile -File install-windows.ps1
#
# `make install` ne fonctionne pas ici, et il vaut mieux le dire que le laisser découvrir :
# la cible appelle `install(1)`, un outil POSIX que Windows n'a pas, et `make` lui-même n'y
# est pas livré. Trois lignes de PowerShell suffisent — les voici, avec les vérifications
# qui vont avec.
#
# ── Options ───────────────────────────────────────────────────────────────────────────
#
#   -Methode shim|copie  comment rendre `jigger` appelable (défaut : shim si scoop est
#                        installé, copie sinon)
#   -Prefixe <dossier>   où copier le binaire (défaut : $env:USERPROFILE\bin)
#   -Profil              ajoute l'Import-Module au $PROFILE s'il n'y figure pas
#   -Simuler             montre ce qui serait fait, sans rien faire
#
# ── Les deux méthodes, et laquelle choisir ────────────────────────────────────────────
#
#   **shim** — scoop enregistre un raccourci vers le binaire **du dépôt**. Un simple
#   `go build` suffit ensuite à mettre à jour ce que `jigger` exécute : c'est la bonne
#   méthode pour développer. Revers : déplacer le dépôt casse le shim.
#
#   **copie** — le binaire est copié hors du dépôt, et le dossier ajouté au PATH de
#   l'utilisateur. Indépendant du dépôt, mais il faut réinstaller après chaque build.

[CmdletBinding()]
param(
    [ValidateSet('shim', 'copie')]
    [string]$Methode,
    [string]$Prefixe,
    [switch]$Profil,
    [switch]$Simuler
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = $PSScriptRoot
Set-Location $racine

function Faire([string]$Quoi, [scriptblock]$Action) {
    if ($Simuler) {
        Write-Host "  (simulation) $Quoi" -ForegroundColor DarkGray
    } else {
        Write-Host "  $Quoi"
        & $Action
    }
}

# ── 1. Compiler ───────────────────────────────────────────────────────────────────────

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Go est introuvable. Installe-le d'abord : scoop install go" -ForegroundColor Red
    exit 1
}

$binaire = Join-Path $racine 'jigger.exe'
Write-Host "→ compilation" -ForegroundColor Cyan
Faire "go build -o jigger.exe ." { & go build -o 'jigger.exe' '.' ; if ($LASTEXITCODE -ne 0) { throw 'la compilation a échoué' } }

if (-not $Simuler -and -not (Test-Path $binaire)) {
    Write-Host 'Le binaire n''a pas été produit.' -ForegroundColor Red
    exit 1
}

# ── 2. Rendre `jigger` appelable ──────────────────────────────────────────────────────

$scoop = Get-Command scoop -ErrorAction SilentlyContinue
if (-not $Methode) {
    $Methode = if ($scoop) { 'shim' } else { 'copie' }
    Write-Host "→ méthode : $Methode (choisie automatiquement)" -ForegroundColor Cyan
} else {
    Write-Host "→ méthode : $Methode" -ForegroundColor Cyan
}

if ($Methode -eq 'shim') {
    if (-not $scoop) {
        Write-Host "scoop est introuvable : la méthode « shim » demande scoop. Relance avec -Methode copie." -ForegroundColor Red
        exit 1
    }
    # `scoop shim add` refuse un nom déjà pris : on remplace plutôt que d'échouer, pour que
    # relancer ce script après un build soit sans effet de bord.
    $existe = (& scoop shim info jigger 2>&1 | Out-String) -notmatch 'not found|introuvable'
    if ($existe) {
        Faire 'scoop shim rm jigger  (remplacement)' { & scoop shim rm jigger | Out-Null }
    }
    Faire "scoop shim add jigger `"$binaire`"" { & scoop shim add jigger "$binaire" | Out-Null }
    $installe = 'jigger'
} else {
    if (-not $Prefixe) {
        $home2 = $env:USERPROFILE
        if (-not $home2) { $home2 = $HOME }
        $Prefixe = Join-Path $home2 'bin'
    }
    Faire "création de $Prefixe" { New-Item -ItemType Directory -Force -Path $Prefixe | Out-Null }
    Faire "copie du binaire dans $Prefixe" { Copy-Item $binaire (Join-Path $Prefixe 'jigger.exe') -Force }

    # Le PATH de l'utilisateur, pas celui de la session : c'est le seul qui survive à la
    # fermeture du terminal. Il faudra donc rouvrir une fenêtre pour en profiter.
    $pathUtilisateur = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($pathUtilisateur -and ($pathUtilisateur -split ';' | Where-Object { $_.TrimEnd('\') -eq $Prefixe.TrimEnd('\') })) {
        Write-Host "  $Prefixe est déjà dans le PATH utilisateur"
    } else {
        Faire "ajout de $Prefixe au PATH utilisateur" {
            [Environment]::SetEnvironmentVariable('Path', "$pathUtilisateur;$Prefixe", 'User')
        }
        Write-Host '  ⚠ rouvre le terminal : le PATH n''est lu qu''au démarrage' -ForegroundColor Yellow
    }
    $installe = Join-Path $Prefixe 'jigger.exe'
}

# ── 3. Le greffon PowerShell ──────────────────────────────────────────────────────────

$module = Join-Path $racine 'shell\jigger.psm1'
$ligne = "Import-Module $module"

Write-Host '→ greffon PowerShell' -ForegroundColor Cyan
$dejaLa = (Test-Path $PROFILE) -and ((Get-Content $PROFILE -Raw -ErrorAction SilentlyContinue) -match 'jigger\.psm1')

if ($dejaLa) {
    Write-Host "  déjà présent dans $PROFILE"
} elseif ($Profil) {
    Faire "ajout au $PROFILE" {
        New-Item -ItemType File -Force -Path $PROFILE | Out-Null
        Add-Content -Path $PROFILE -Value "`n# jigger`n$ligne"
    }
    Write-Host '  recharge : . $PROFILE' -ForegroundColor Yellow
} else {
    Write-Host "  à ajouter dans $PROFILE (ou relance avec -Profil) :"
    Write-Host "    $ligne" -ForegroundColor Yellow
}

# ── 4. Vérifier ───────────────────────────────────────────────────────────────────────

Write-Host '→ vérification' -ForegroundColor Cyan
if ($Simuler) {
    Write-Host '  (simulation) jigger --version'
} else {
    # Par le chemin complet : le PATH de la session courante n'a pas changé.
    $version = (& $binaire --version 2>&1 | Out-String).Trim()
    Write-Host "  $version"
    Write-Host ''
    Write-Host "Installé — $installe" -ForegroundColor Green
    Write-Host 'Ouvre un terminal neuf et tape « winget ins » sans valider : le cadre doit apparaître.'
    Write-Host 'Note : l''alias « jg » n''existe pas encore côté PowerShell — la façade s''écrit « jigger <verbe> ».'
}
