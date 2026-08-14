# Suite d'assertions du module PowerShell.
#
#   pwsh -NoProfile -File tests/smoke.ps1
#
# Ce que cette suite couvre : tout ce qui ne demande pas de vrai terminal — les touches
# effectivement enregistrées, l'analyse de la sortie de `jigger render`, les séquences
# écrites pour dessiner et effacer le popup, la détection des commandes mutantes, et
# l'export des variables du prompt depuis un cache fabriqué.
#
# Ce qu'elle ne couvre pas : le popup à l'écran. PSReadLine ne se pilote pas sans
# console — c'est justement ce que fait tests/zpty.zsh côté zsh, faute d'équivalent sous
# Windows. Après une modification du module, il reste donc à ouvrir un terminal et à
# taper « winget install fire » pour voir le cadre vivre.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = Split-Path -Parent $PSScriptRoot
$module = Join-Path $racine 'shell/jigger.psm1'
$binaire = Join-Path $racine 'jigger.exe'

if (-not (Test-Path $binaire)) {
    Write-Error "compile d'abord : go build -o jigger.exe ."
    exit 1
}
$env:JIGGER_BIN = $binaire

$script:Echecs = 0
$script:Total = 0

function check([string]$Nom, $Obtenu, $Attendu) {
    $script:Total++
    if ($Obtenu -eq $Attendu) {
        Write-Host "  ok   $Nom"
    } else {
        Write-Host "  ÉCHEC $Nom" -ForegroundColor Red
        Write-Host "         obtenu  : [$Obtenu]" -ForegroundColor Red
        Write-Host "         attendu : [$Attendu]" -ForegroundColor Red
        $script:Echecs++
    }
}

function section([string]$Titre) { Write-Host "`n→ $Titre" }

# ── Chargement ────────────────────────────────────────────────────────────────────────

$cache = Join-Path ([IO.Path]::GetTempPath()) ("jigger-test-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $cache | Out-Null
$env:JIGGER_CACHE_DIR = $cache
$env:JIGGER_PROMPT = '1'
$env:JIGGER_PROMPT_SYNC = '0'   # aucun test ne doit attendre après winget

Import-Module $module -Force
$jigger = Get-Module jigger

section 'le module s''installe dans PSReadLine'
$handlers = Get-PSReadLineKeyHandler -Bound
check 'Tab est repris'        (($handlers | Where-Object { $_.Key -eq 'Tab' }).Function) 'jigger:insérer'
check 'Ctrl+n est repris'     (($handlers | Where-Object { $_.Key -eq 'Ctrl+n' }).Function) 'jigger:suivant'
check 'Ctrl+g est repris'     (($handlers | Where-Object { $_.Key -eq 'Ctrl+g' }).Function) 'jigger:fermer'
check 'Entrée est reprise'    (($handlers | Where-Object { $_.Key -eq 'Enter' }).Function) 'jigger:valider'
check 'les lettres aussi'     (($handlers | Where-Object { $_.Key -eq 'g' }).Function) 'jigger:SelfInsert'
check 'Retour arrière aussi'  (($handlers | Where-Object { $_.Key -eq 'Backspace' }).Function) 'jigger:BackwardDeleteChar'
check 'le prompt est enveloppé' ((Get-Command prompt).ScriptBlock -match 'jigger:prompt-hook') $true

check 'Ctrl+c est repris'     (($handlers | Where-Object { $_.Key -ceq 'Ctrl+c' }).Function) 'jigger:annuler'

# Le repli d'une touche doit être **une** fonction PSReadLine, pas deux collées : `Ctrl+C`
# (copier) et `Ctrl+c` (abandonner la ligne) sont deux liaisons distinctes, et le `-eq` de
# PowerShell ignore la casse.
check 'repli de Ctrl+c'       $global:JiggerFallbacks['Ctrl+c'] 'CopyOrCancelLine'
check 'repli d''Échap'        $global:JiggerFallbacks['Escape'] 'RevertLine'
check 'repli de ⇥'            $global:JiggerFallbacks['Tab'] 'TabCompleteNext'

# Un relais doit être un bloc de script rattaché au **module** : `.GetNewClosure()` et
# `[scriptblock]::Create()` le rattachent à un module dynamique, d'où Update-JiggerPopup
# n'est plus visible — la première frappe échouerait, et rien ici ne s'en apercevrait.
$modules = & $jigger { @{
    texte   = $script:RelaisTexte.Module.Name
    edition = $script:RelaisEdition['Backspace'].Module.Name
    fin     = $script:RelaisFin['Enter'].Module.Name
} }
check 'le relais de texte voit le module'   $modules.texte 'jigger'
check 'le relais d''édition voit le module' $modules.edition 'jigger'
check 'le relais de fin voit le module'     $modules.fin 'jigger'

# Réimporter ne doit pas empiler les enveloppes : le repli mémorisé reste la fonction
# d'origine de PSReadLine, jamais notre propre relais.
Import-Module $module -Force
check 'réimport : repli intact' $global:JiggerFallbacks['Backspace'] 'BackwardDeleteChar'

section 'la ligne est reconnue (ou non) comme celle d''un gestionnaire'
foreach ($cas in @(
    @('winget install', $true), @('scoop ', $true), @('winget', $true),
    @('git commit', $false), @('', $false), @('echo winget install', $false))) {
    check "« $($cas[0]) »" (& $jigger { param($l) Test-JiggerLine $l } $cas[0]) $cas[1]
}

section 'la sortie de `jigger render` est décodée'
# Catalogue fabriqué : la suite ne dépend ni de winget, ni de ce qui est installé sur la
# machine. C'est le même fichier que `jigger warm` écrirait.
Set-Content -Path (Join-Path $cache 'winget-catalog') -Value "Git.Git`nGit.MinGit`nGitHub.cli"
Set-Content -Path (Join-Path $cache 'winget-installed') -Value "Git.Git`t2.55.0"
$etat = & $jigger {
    $script:Sel = 1
    $ok = Get-JiggerFrame 'winget install Git.' 5 80
    [PSCustomObject]@{
        Ok     = $ok
        Count  = $script:Count
        Sel    = $script:Sel
        Left   = $script:Left
        Lignes = ($script:Frame -split "`n").Count
    }
}
check 'le rendu réussit'           $etat.Ok $true
check 'les candidats sont comptés' $etat.Count 2
# --sel 1 : c'est le deuxième candidat qui part dans la ligne.
check 'la ligne complétée revient' $etat.Left 'winget install Git.MinGit'
check 'le cadre a ses bordures'    ($etat.Lignes -ge 5) $true

section 'le dessin sauve et restaure le curseur'
$ESC = [char]27
$save = "$ESC" + '7'; $restore = "$ESC" + '8'
$clreol = "$ESC" + '[K'; $clrdown = "$ESC" + '[J'
$dessin = & $jigger { param($f) Format-JiggerDraw $f } "ligne1`nligne2"
check 'mémorise le curseur'   $dessin.StartsWith($save + "`r`n") $true
check 'le remet à la fin'     $dessin.EndsWith($restore) $true
check 'efface sous le cadre'  $dessin.Contains($clrdown) $true
check 'efface chaque ligne'   $dessin.Contains("ligne1$clreol`r`nligne2") $true
$efface = & $jigger { Format-JiggerDraw '' }
check 'l''effacement n''écrit rien' $efface ($save + "`r`n" + $clrdown + $restore)

section 'les commandes mutantes sont repérées, et elles seules'
foreach ($cas in @(
    @('winget install Git.Git', $true),
    @('winget upgrade --all', $true),
    @('winget --nowarn upgrade', $true),      # une option avant la sous-commande
    @('scoop update *', $true),
    @('scoop bucket add extras', $true),
    @('winget search git', $false),
    @('scoop status', $false),
    @('winget list', $false),
    @('git commit -m "winget upgrade"', $false),   # cité : ce n'est pas une commande
    @('echo scoop install 7zip', $false),
    @('ls; winget install jq', $true))) {         # une commande mutante en seconde position
    check "« $($cas[0]) »" (Test-JiggerMutating $cas[0]) $cas[1]
}

section 'le prompt lit le cache sans lancer de processus'
# Un cache fabriqué, frais et mensonger : c'est le seul moyen de vérifier ce que le
# prompt aurait affiché.
$maintenant = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
Set-Content -Path (Join-Path $cache 'status') -Value "1.2.3`t7`t0`t$maintenant" -NoNewline
Update-JiggerPrompt
check 'version exportée'          $env:JIGGER_WINGET_VERSION '1.2.3'
check 'compteur winget exporté'   $env:JIGGER_WINGET_OUTDATED '7'
# Un compteur nul n'est pas exporté : le gabarit oh-my-posh se réduit à « {{ if … }} ».
check 'compteur scoop absent'     ([string]::IsNullOrEmpty($env:JIGGER_SCOOP_OUTDATED)) $true
check 'total exporté'             $env:JIGGER_OUTDATED '7'

Set-Content -Path (Join-Path $cache 'status') -Value "1.2.3`t0`t0`t$maintenant" -NoNewline
Update-JiggerPrompt
check 'plus rien à signaler'      ([string]::IsNullOrEmpty($env:JIGGER_OUTDATED)) $true
check 'la version reste'          $env:JIGGER_WINGET_VERSION '1.2.3'

# Un cache corrompu est traité comme absent, jamais comme à moitié valide.
Set-Content -Path (Join-Path $cache 'status') -Value "n'importe quoi" -NoNewline
[Environment]::SetEnvironmentVariable('JIGGER_WINGET_VERSION', $null)
Update-JiggerPrompt
check 'cache corrompu ignoré'     ([string]::IsNullOrEmpty($env:JIGGER_WINGET_VERSION)) $true

Remove-Item -Recurse -Force $cache -ErrorAction SilentlyContinue

Write-Host ''
if ($script:Echecs -gt 0) {
    Write-Host "$($script:Echecs) assertion(s) en échec sur $($script:Total)" -ForegroundColor Red
    exit 1
}
Write-Host "tout passe ($($script:Total) assertions)" -ForegroundColor Green
