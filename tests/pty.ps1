# Suite d'assertions du popup vivant, dans un vrai pseudo-terminal.
#
#   pwsh -NoProfile -File tests/pty.ps1
#
# Chaque cas lance un pwsh dans un ConPTY (cf. tests/conpty), y charge le module, tape
# une séquence de touches, et juge **l'écran** tel qu'on le verrait — pas le flux d'octets.
# C'est le pendant Windows de tests/zpty.zsh.
#
# Le prompt des cas d'essai tient sur deux lignes et l'écran est court : on se place ainsi
# dans la situation d'un terminal en usage, où le prompt occupe la dernière ligne. C'est
# le cas qui compte — et celui qui échouait.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Le harnais rend de l'UTF-8 (cadres, flèches, accents) : sans cela, PowerShell le
# décoderait dans la page de codes du système et les assertions porteraient sur du
# charabia.
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false

$racine = Split-Path -Parent $PSScriptRoot
$binaire = Join-Path $racine 'jigger.exe'
$harnais = Join-Path $racine 'conpty-test.exe'

if (-not (Test-Path $binaire)) {
    Write-Error "compile d'abord : go build -o jigger.exe ."
    exit 1
}
Write-Host 'compilation du harnais…'
& go build -o $harnais ./tests/conpty
if ($LASTEXITCODE -ne 0) { exit 1 }

# rc des cas d'essai : un prompt court sur deux lignes, aucune prédiction (elle
# écrirait par-dessus la ligne et brouillerait les assertions), et le module.
$rc = Join-Path ([IO.Path]::GetTempPath()) 'jigger-pty-rc.ps1'
@"
function global:prompt { "``nPS> " }
Set-PSReadLineOption -PredictionSource None
`$env:JIGGER_BIN = '$binaire'
# Cette suite assère des libellés français en dur : sans l'exporter, le module lirait la
# langue de la machine qui lance les tests.
`$env:JIGGER_LANG = 'fr'
Import-Module '$racine\shell\jigger.psm1' -Force
"@ | Set-Content -Path $rc

$script:Echecs = 0
$script:Total = 0

function ecran([string]$Touches, [int]$Lignes = 16, [int]$Colonnes = 66) {
    # Quelques ⏎ d'abord : ils poussent le prompt en bas de l'écran, là où il vit.
    $out = & $harnais -rc $rc -rows $Lignes -cols $Colonnes -keys ('\r\r\r\r' + $Touches) `
        -settle 3s -pause 250ms -last 1500ms -screen 2>$null
    return ($out -join "`n")
}

function check([string]$Nom, [string]$Ecran, [string]$Aiguille, [bool]$Attendu = $true) {
    $script:Total++
    $trouve = $Ecran.Contains($Aiguille)
    if ($trouve -eq $Attendu) {
        Write-Host "  ok   $Nom"
    } else {
        $quoi = if ($Attendu) { 'absent' } else { 'présent alors qu''il devait avoir disparu' }
        Write-Host "  ÉCHEC $Nom — « $Aiguille » $quoi" -ForegroundColor Red
        Write-Host ($Ecran -split "`n" | ForEach-Object { "         $_" } | Out-String) -ForegroundColor DarkGray
        $script:Echecs++
    }
}

Write-Host "`n→ le popup s'affiche même quand le prompt occupe la dernière ligne"
$e = ecran 'winget ins'
check 'cadre affiché'          $e '╭'
check 'en-tête du contexte'    $e '❯ winget'
check 'candidat install'       $e 'install'
check 'la ligne reste lisible' $e 'PS> winget ins'

Write-Host "`n→ « winget » seul propose les sous-commandes"
$e = ecran 'winget'
check 'sous-commandes listées' $e 'uninstall'
check 'pas de liste vide'      $e 'aucun candidat' $false

Write-Host "`n→ la façade arme le popup, sous ses deux noms"
# Les verbes proposés ici viennent des gestionnaires **disponibles** : ce cas suppose donc
# winget ou scoop installé sur la machine d'essai. C'est le cas de toute machine Windows
# pour winget, et c'est de toute façon la seule où cette suite tourne.
$e = ecran 'jg ins'
check 'cadre affiché'            $e '╭'
check 'en-tête de la façade'     $e '❯ jigger'
check 'candidat install'         $e 'install'
check 'la ligne reste lisible'   $e 'PS> jg ins'
# Le nom en toutes lettres arme le même popup : le relais lit le tampon, où « jg » n'a pas
# été développé — les deux noms doivent donc y être reconnus séparément.
$e = ecran 'jigger ins'
check '« jigger » aussi'         $e '❯ jigger'
# Et rien d'autre : un mot qui *commence* par jg n'est pas la façade.
$e = ecran 'jgit ins'
check 'aucun cadre sur « jgit »' $e '╭' $false

Write-Host "`n→ ⇥ insère le verbe de la façade"
$e = ecran 'jg ins\t'
check 'ligne complétée'        $e 'PS> jg install'

Write-Host "`n→ une ligne qui n'est pas un gestionnaire n'affiche rien"
$e = ecran 'echo bonjour'
check 'aucun cadre'            $e '╭' $false

Write-Host "`n→ ⇥ insère le candidat courant dans la ligne"
$e = ecran 'winget ins\t'
check 'ligne complétée'        $e 'PS> winget install'

Write-Host "`n→ ⏎ complète la dernière partie, et exécute dans la même frappe"
# La frappe économisée : ⇥ n'est plus nécessaire avant ⏎. La ligne échoue (aucun paquet
# n'est nommé) et c'est voulu — ⏎ dit « pars », la ligne part. C'est la trace d'exécution
# qu'on assère, pas la réussite.
$e = ecran 'jg ins\r'
check 'ligne complétée'        $e 'jg install'
check 'ligne exécutée'         $e 'jigger'
check 'cadre effacé'           $e '╭' $false

Write-Host "`n→ ⏎ reste ⏎ là où il n'y a rien à compléter"
# L'autre moitié de la règle : hors ligne de gestionnaire, aucun popup, donc la touche
# rend la main à PSReadLine et la commande s'exécute. Sans elle, plus rien ne partirait.
$e = ecran 'echo bonjour\r'
check 'commande exécutée'      $e 'bonjour'

Write-Host "`n→ ⇥ sans candidat rend la main au shell, sans ouvrir de sélecteur"
$e = ecran 'winget zzz\t'
check 'ligne intacte'          $e 'PS> winget zzz'
# Le sélecteur plein écran se reconnaît à sa ligne de filtre et à ses touches.
check 'aucun sélecteur ouvert' $e 'filtrer' $false
check 'aucun « annuler »'      $e 'annuler' $false

Write-Host "`n→ les flèches prennent le clavier, puis le rendent"
$e = ecran 'winget install Git.'
check 'sans focus : ↓ parcourir' $e '↓  parcourir'
$e = ecran 'winget install Git.\x1b[B'
check 'après ↓ : ↑↓ naviguer'    $e '↑↓  naviguer'
# Un premier ↑ revient sur le premier candidat, un second rend le clavier au shell : on
# ne retombe donc jamais dans l'historique par inadvertance.
$e = ecran 'winget install Git.\x1b[B\x1b[A\x1b[A'
check 'deux ↑ rendent le clavier' $e '↓  parcourir'
$e = ecran 'winget install Git.\x1b[B\x1b[B\t'
check '⇥ insère le candidat visé' $e 'PS> winget install Git.MinGit'

Write-Host "`n→ la liste de prédictions de PSReadLine cède la place au popup"
# ListView se dessine exactement là où le cadre se dessine : les deux se disputeraient les
# mêmes lignes à chaque frappe, et l'écran scintillerait. jigger range la vue le temps du
# popup, et la rend ensuite.
$rcListe = Join-Path ([IO.Path]::GetTempPath()) 'jigger-pty-rc-liste.ps1'
@"
function global:prompt { "``nPS> " }
Set-PSReadLineOption -PredictionSource History -PredictionViewStyle ListView
[Microsoft.PowerShell.PSConsoleReadLine]::AddToHistory('winget install Git.Git')
[Microsoft.PowerShell.PSConsoleReadLine]::AddToHistory('winget install Mozilla.Firefox')
`$env:JIGGER_BIN = '$binaire'
`$env:JIGGER_LANG = 'fr'
Import-Module '$racine\shell\jigger.psm1' -Force
"@ | Set-Content -Path $rcListe

$avecListe = (& $harnais -rc $rcListe -rows 20 -cols 70 -keys '\r\r\rwinget install G' `
    -settle 3s -pause 250ms -last 1500ms -screen 2>$null) -join "`n"
check 'le cadre est bien là'        $avecListe '╭'
check 'la liste a cédé la place'    $avecListe '[History]' $false
# Hors ligne de gestionnaire, elle reste chez elle.
$sansPopup = (& $harnais -rc $rcListe -rows 20 -cols 70 -keys '\r\r\recho G' `
    -settle 3s -pause 250ms -last 1500ms -screen 2>$null) -join "`n"
check 'aucun cadre sur une autre ligne' $sansPopup '╭' $false
Remove-Item -Force $rcListe -ErrorAction SilentlyContinue

Write-Host "`n→ ^C efface le cadre avant de rendre la ligne"
$e = ecran 'winget ins\x03'
check 'cadre effacé'           $e '╭' $false

Remove-Item -Force $harnais -ErrorAction SilentlyContinue
Remove-Item -Force $rc -ErrorAction SilentlyContinue

Write-Host ''
if ($script:Echecs -gt 0) {
    Write-Host "$($script:Echecs) assertion(s) en échec sur $($script:Total)" -ForegroundColor Red
    exit 1
}
Write-Host "tout passe ($($script:Total) assertions)" -ForegroundColor Green
