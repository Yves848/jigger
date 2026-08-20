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
# console — c'est ce que font tests/zpty.zsh côté zsh et tests/conpty côté Windows.
#
# Elle tourne aussi sur le pwsh de macOS ou de Linux, et c'est voulu : le module se
# développe alors dans la même boucle que le reste, sans démarrer une machine Windows.
# Seul le popup à l'écran, et ce qui touche aux vraies CLI winget et scoop, exige Windows.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = Split-Path -Parent $PSScriptRoot
$module = Join-Path $racine 'shell/jigger.psm1'

# Le binaire porte le nom de sa plateforme : `jigger.exe` sous Windows, `jigger` ailleurs.
$binaire = Join-Path $racine 'jigger.exe'
if (-not (Test-Path $binaire)) { $binaire = Join-Path $racine 'jigger' }

if (-not (Test-Path $binaire)) {
    Write-Error "compile d'abord : make build (ou go build -o jigger .)"
    exit 1
}
$env:JIGGER_BIN = $binaire
# La langue est épinglée pour que rien ici ne dépende de la locale de la machine qui
# lance les tests : le module et le binaire (un sous-processus) parlent tous deux celle
# qu'on leur donne. Les étiquettes de touches, elles, sont des identifiants fixes en
# anglais (cf. Register-JiggerKey) : elles ne bougent pas avec ce réglage.
$env:JIGGER_LANG = 'fr'

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

# Ce que PSReadLine liait *avant* nous : c'est exactement ce que le module doit avoir
# mémorisé comme repli. On le relève au lieu de l'écrire en dur, parce que les valeurs
# dépendent de la plateforme — ⇥ vaut `TabCompleteNext` sous Windows et `Complete`
# ailleurs, Échap vaut `RevertLine` sous Windows et n'est lié nulle part ailleurs. Table
# ordinale, pour la même raison que côté module : `Ctrl+C` et `Ctrl+c` sont deux liaisons.
Import-Module PSReadLine
$avant = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)
foreach ($h in Get-PSReadLineKeyHandler -Bound) { $avant[$h.Key] = $h.Function }

Import-Module $module -Force
$jigger = Get-Module jigger

section 'la langue résolue par le module concorde avec celle du binaire'
# Trois implémentations de la même règle (binaire, greffon zsh, module PowerShell),
# mesurées à la main à la tâche 6 : sans assertion, la prochaine modification les fait
# diverger sans bruit — un popup dans une langue, un module dans l'autre. On compare donc
# à chaque fois $script:Lang à la résolution du binaire, jamais à une valeur en dur :
# c'est la concordance qui est l'exigence, pas la valeur (cf. tests/zpty.zsh, même
# principe côté zsh).
#
# Résolution du binaire : « jigger » sans argument imprime son usage sur stderr, et
# cli.usage1 (internal/i18n/catalogue.go) est la seule chaîne du catalogue qui diffère
# par un simple mot entre les deux langues — « <verb> » contre « <verbe> ». C'est le
# marqueur le plus direct, sans avoir à exposer i18n.Courante() par une commande dédiée.
foreach ($valeur in 'fr', 'FR', 'fr-FR', 'fr_FR.UTF-8', 'ja') {
    $env:JIGGER_LANG = $valeur
    Import-Module $module -Force
    $jigger = Get-Module jigger
    $langModule = & $jigger { $script:Lang }

    $sortie = & $binaire 2>&1 | Out-String
    $langBinaire = if ($sortie -match '<verbe>') { 'fr' } else { 'en' }

    check "JIGGER_LANG=$valeur : module ($langModule) d'accord avec le binaire" $langModule $langBinaire
}

# Le repli d'une langue que jigger ne sait pas traduire, posé explicitement : avec
# JIGGER_LANG=ja, la résolution doit descendre jusqu'à LANG et rendre « fr ». Le cas « ja »
# de la boucle ci-dessus n'exerce ce chemin que si l'hôte est déjà en français — sur une
# machine anglaise, il passerait avec un repli cassé. On pose donc LANG nous-mêmes, et on
# retire LC_ALL et LC_MESSAGES, qui primeraient sur lui.
$avantJa = @{}
foreach ($nom in 'JIGGER_LANG', 'LC_ALL', 'LC_MESSAGES', 'LANG') {
    $avantJa[$nom] = [Environment]::GetEnvironmentVariable($nom)
    Set-Item -Path "Env:$nom" -Value $null -ErrorAction SilentlyContinue
}
$env:JIGGER_LANG = 'ja'
$env:LANG = 'fr_FR.UTF-8'
Import-Module $module -Force
$jigger = Get-Module jigger
$langModule = & $jigger { $script:Lang }
$sortie = & $binaire 2>&1 | Out-String
$langBinaire = if ($sortie -match '<verbe>') { 'fr' } else { 'en' }
check "JIGGER_LANG=ja + LANG=fr_FR.UTF-8 : module ($langModule) d'accord avec le binaire" $langModule $langBinaire
check "JIGGER_LANG=ja + LANG=fr_FR.UTF-8 : le repli descend jusqu'à LANG" $langModule 'fr'
foreach ($nom in @($avantJa.Keys)) {
    Set-Item -Path "Env:$nom" -Value $null -ErrorAction SilentlyContinue
    if ($null -ne $avantJa[$nom]) { Set-Item -Path "Env:$nom" -Value $avantJa[$nom] }
}

# Le cas qui a débusqué la rupture de parité : sans lui, une résolution qui s'arrête à la
# première variable non vide, reconnue ou pas (l'ancien code), ne se voit jamais — les cinq
# valeurs ci-dessus posent toutes JIGGER_LANG. On retire ici les quatre variables
# entièrement (`$env:X = $null` les enlève pour de bon, y compris pour les sous-processus —
# vérifié : ce n'est pas juste les vider), pour que ce cas soit « aucune variable » pour de
# vrai, pas seulement sous la locale qui a lancé cette suite.
$avantLocale = @{}
foreach ($nom in 'JIGGER_LANG', 'LC_ALL', 'LC_MESSAGES', 'LANG') {
    $avantLocale[$nom] = [Environment]::GetEnvironmentVariable($nom)
    Set-Item -Path "Env:$nom" -Value $null -ErrorAction SilentlyContinue
}
Import-Module $module -Force
$jigger = Get-Module jigger
$langModule = & $jigger { $script:Lang }
$sortie = & $binaire 2>&1 | Out-String
$langBinaire = if ($sortie -match '<verbe>') { 'fr' } else { 'en' }
check "aucune variable de locale posée : module ($langModule) d'accord avec le binaire" $langModule $langBinaire
foreach ($nom in $avantLocale.Keys) {
    if ($null -ne $avantLocale[$nom]) { Set-Item -Path "Env:$nom" -Value $avantLocale[$nom] }
}

# Le reste de la suite travaille langue épinglée : on remet JIGGER_LANG=fr et on
# réimporte, pour que les sections suivantes retrouvent l'état attendu.
$env:JIGGER_LANG = 'fr'
Import-Module $module -Force
$jigger = Get-Module jigger

section 'le module s''installe dans PSReadLine'
$handlers = Get-PSReadLineKeyHandler -Bound
check 'Tab est repris'        (($handlers | Where-Object { $_.Key -eq 'Tab' }).Function) 'jigger:insert'
check 'Ctrl+n est repris'     (($handlers | Where-Object { $_.Key -eq 'Ctrl+n' }).Function) 'jigger:next'
check 'Ctrl+g est repris'     (($handlers | Where-Object { $_.Key -eq 'Ctrl+g' }).Function) 'jigger:close'
check 'Entrée est reprise'    (($handlers | Where-Object { $_.Key -eq 'Enter' }).Function) 'jigger:accept'
check 'les lettres aussi'     (($handlers | Where-Object { $_.Key -eq 'g' }).Function) 'jigger:SelfInsert'
check 'Retour arrière aussi'  (($handlers | Where-Object { $_.Key -eq 'Backspace' }).Function) 'jigger:BackwardDeleteChar'
check 'le prompt est enveloppé' ((Get-Command prompt).ScriptBlock -match 'jigger:prompt-hook') $true

check 'Ctrl+c est repris'     (($handlers | Where-Object { $_.Key -ceq 'Ctrl+c' }).Function) 'jigger:cancel'
check 'Ctrl+r est repris'     (($handlers | Where-Object { $_.Key -ceq 'Ctrl+r' }).Function) 'jigger:regex'

# Le repli d'une touche doit être **une** fonction PSReadLine, pas deux collées : `Ctrl+C`
# (copier) et `Ctrl+c` (abandonner la ligne) sont deux liaisons distinctes, et le `-eq` de
# PowerShell ignore la casse.
check 'repli de Ctrl+c'       $global:JiggerFallbacks['Ctrl+c'] $avant['Ctrl+c']
check 'repli d''Échap'        $global:JiggerFallbacks['Escape'] $avant['Escape']
check 'repli de ⇥'            $global:JiggerFallbacks['Tab'] $avant['Tab']

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

section 'les flèches sont reprises, sans perdre l''historique'
check '↓ est reprise'  (($handlers | Where-Object { $_.Key -ceq 'DownArrow' }).Function) 'jigger:down'
check '↑ est reprise'  (($handlers | Where-Object { $_.Key -ceq 'UpArrow' }).Function) 'jigger:up'
# Le repli doit être ce que la flèche faisait avant nous — l'historique, sous une forme
# ou une autre selon les réglages —, jamais un relais de jigger.
foreach ($k in 'UpArrow', 'DownArrow') {
    $repli = $global:JiggerFallbacks[$k]
    check "repli de $k" ($repli -and -not $repli.StartsWith('jigger:')) $true
}

section 'la bascule regex est offerte au shell'
# Le relais ^R du module ne suffit pas : un profil qui lie ^R *après* notre import — un
# fzf d'historique, par exemple — reprend la touche, et la bascule devient injoignable.
# Elle est donc aussi une fonction exportée, que ce profil-là appelle avant son propre
# repli. Le greffon zsh obtient la même chose autrement, en relevant le widget lié avant
# lui : dans un sens comme dans l'autre, la touche n'est confisquée par personne (A-19).
check 'la bascule est exportée' ($jigger.ExportedFunctions.ContainsKey('Invoke-JiggerRegex')) $true

# Hors ligne de gestionnaire, elle ne bascule rien et rend $false : à l'appelant de faire
# ce qu'il aurait fait sans nous. Sur une ligne surveillée, elle prend la touche et le dit.
# La ligne est passée en argument — sans console, PSReadLine n'a pas de tampon à lire.
$bascule = & $jigger {
    $etat = @{ regex = $script:Regex; live = $script:Live; echecs = $script:Failures }
    $hors = Invoke-JiggerRegex 'git status'
    $regexHors = $script:Regex
    $sur = Invoke-JiggerRegex 'scoop install fi'
    $regexSur = $script:Regex
    $script:Regex = $etat.regex; $script:Live = $etat.live; $script:Failures = $etat.echecs
    @{ hors = $hors; regexHors = $regexHors; sur = $sur; regexSur = $regexSur }
}
check 'hors gestionnaire : la touche est rendue' $bascule.hors $false
check 'hors gestionnaire : le mode ne bouge pas' $bascule.regexHors $false
check 'sur gestionnaire : la touche est prise'   $bascule.sur $true
check 'sur gestionnaire : le mode bascule'       $bascule.regexSur $true

section 'le redessin du popup est offert au shell'
# Même raison, autres touches : ^U, ^D, ^E sont souvent reliées après nous à quelque chose
# qui prend l'écran puis rend la ligne — un explorateur de fichiers, un sélecteur de
# lecteur. Notre relais est alors écrasé, et le cadre reste derrière, périmé. Un appel en
# fin de relais le remet d'accord avec la ligne : c'est tout ce qu'il manquait.
check 'le redessin est exporté' ($jigger.ExportedFunctions.ContainsKey('Update-JiggerPopup')) $true

# Et il s'appelle sans précaution : sans console, PSReadLine n'a pas de tampon et tout
# échoue — mais rien ne remonte, une frappe ne peut pas se mettre à barrer l'écran de
# rouge. Ce que ce cas vérifie, c'est le silence.
$redessin = & $jigger {
    $etat = @{ live = $script:Live; echecs = $script:Failures }
    $leve = $false
    try { Update-JiggerPopup } catch { $leve = $true }
    $script:Live = $etat.live; $script:Failures = $etat.echecs
    $leve
}
check 'sans console, il ne lève pas' $redessin $false

section 'le focus décide qui prend les flèches'
# La règle : ↓ fait entrer dans la liste, ↑ en ressort et rend l'historique. Hors focus,
# les deux flèches ne sont pas prises — c'est ce que dit `pris=$false`.
$transitions = & $jigger {
    $script:Sel = 0
    $script:Focused = $false
    $etapes = @()
    foreach ($sens in -1, 1, 1, -1, -1, -1) {
        $pris = Step-JiggerSelection $sens $true
        $etapes += "$pris/$($script:Sel)/$($script:Focused)"
    }
    $etapes += "$(Step-JiggerSelection 1 $false)/$($script:Sel)/$($script:Focused)"
    $etapes
}
check '↑ sur un popup sans focus : à l''historique' $transitions[0] 'False/0/False'
check '↓ entre dans la liste'                       $transitions[1] 'True/1/True'
check '↓ descend'                                   $transitions[2] 'True/2/True'
check '↑ remonte'                                   $transitions[3] 'True/1/True'
check '↑ atteint le premier candidat'               $transitions[4] 'True/0/True'
check '↑ au premier candidat rend le clavier'       $transitions[5] 'True/0/False'
check 'sans popup, rien n''est pris'                $transitions[6] 'False/0/False'

section '⏎ complète la dernière partie, puis exécute'
# La règle vaut à tous les niveaux de l'arbre — sous-commande, option, paquet — parce
# qu'elle ne connaît que deux choses : y a-t-il un candidat désigné, et apporte-t-il
# quelque chose à la ligne. Les cas ci-dessous prennent un niveau chacun.
#
# $false ne veut pas dire « la ligne ne part pas » : elle part toujours. Il veut dire
# « rien à poser dedans avant » — ligne déjà complète, aucun candidat, ou pas de popup.
foreach ($cas in @(
    @('une sous-commande à finir', $true,  'winget li',            'winget list',            $true),
    @('une option à finir',        $true,  'scoop install --g',    'scoop install --global', $true),
    @('un paquet à finir',         $true,  'winget install Git.M', 'winget install Git.MinGit', $true),
    @('la ligne est complète',     $true,  'winget list',          'winget list',            $false),
    # La casse compte : c'est elle que la complétion corrige le plus souvent, et winget
    # résout ses identifiants au caractère près.
    @('seule la casse diffère',    $true,  'winget install git.mingit', 'winget install Git.MinGit', $true),
    @('aucun candidat',            $true,  'winget install zzzz',  '',                       $false),
    @('pas de popup',              $false, 'winget li',            'winget list',            $false))) {
    check "« $($cas[2]) » : $($cas[0])" `
        (& $jigger { param($a, $b, $l) Test-JiggerCompletion $a $b $l } $cas[1] $cas[2] $cas[3]) $cas[4]
}

section 'la ligne est reconnue (ou non) comme celle d''un gestionnaire'
# « jg » et « jigger » sont de la partie : la façade arme le popup au même titre que les
# gestionnaires. Et c'est « jg » **tel qu'il est tapé** qui doit être reconnu — le relais
# lit le tampon de PSReadLine, où aucun alias n'a été développé. « jgx » et « jgit » sont
# là pour cela : reconnaître un préfixe au lieu du mot entier armerait le popup sur la
# moitié des commandes d'un poste de développement.
foreach ($cas in @(
    @('winget install', $true), @('scoop ', $true), @('winget', $true),
    @('jg install f', $true), @('jg ', $true), @('jg', $true),
    @('jigger search fd', $true), @('jigger', $true),
    @('git commit', $false), @('', $false), @('echo winget install', $false),
    @('jgx foo', $false), @('jgit status', $false), @('echo jg install', $false))) {
    check "« $($cas[0]) »" (& $jigger { param($l) Test-JiggerLine $l } $cas[0]) $cas[1]
}

section 'l''alias jg est posé par le module'
# L'autre moitié du mécanisme : la liste ci-dessus fait apparaître le popup sur « jg … »,
# l'alias fait que la ligne s'exécute. Les deux sont indépendants — il faut les deux.
#
# Les deux champs sont relevés sous condition : en mode strict, lire une propriété sur un
# $null lève, et une suite qui s'interrompt au premier échec cache tout ce qui la suit.
$alias = Get-Alias jg -ErrorAction SilentlyContinue
$def = if ($alias) { $alias.Definition } else { '' }
$src = if ($alias) { $alias.Source } else { '' }
check 'jg existe'                       ($null -ne $alias) $true
# Il suit JIGGER_BIN, et non le mot « jigger » : sans quoi, sur un poste où le PATH porte
# un autre jigger, le popup et la commande parleraient de deux binaires différents.
check 'jg désigne le binaire du module' $def $env:JIGGER_BIN
check 'jg vient bien du module'         $src 'jigger'

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

section 'une commande mutante fait refaire la liste des installés'
# Le cas : `winget install X` puis `winget uninstall X` — sans rafraîchissement, le paquet
# fraîchement installé manque à la complétion jusqu'à la péremption du cache. Et cela ne
# dépend pas du bloc de prompt, qui est désactivé ici comme il l'est par défaut.
$env:JIGGER_PROMPT = '0'
Import-Module $module -Force
$jigger = Get-Module jigger
& $jigger {
    # Doublure : on relève ce que le module demande au binaire, sans rien lancer.
    $script:Appels = @()
    # `script:` est indispensable : sans lui, la doublure ne vivrait que le temps de
    # ce bloc, et le module continuerait d'appeler la vraie.
    function script:Start-JiggerBackground([string[]]$Arguments) { $script:Appels += ($Arguments -join ' ') }
}
& $jigger { $script:Dirty = $true }
Update-JiggerPrompt
$appels = @(& $jigger { $script:Appels })
check 'les installés sont refaits'  ($appels -contains 'warm --installed') $true
check 'sans bloc de prompt, rien de plus' $appels.Count 1

& $jigger { $script:Appels = @(); $script:Dirty = $false }
Update-JiggerPrompt
check 'aucune commande mutante, aucun appel' @(& $jigger { $script:Appels }).Count 0

Remove-Item -Recurse -Force $cache -ErrorAction SilentlyContinue

Write-Host ''
if ($script:Echecs -gt 0) {
    Write-Host "$($script:Echecs) assertion(s) en échec sur $($script:Total)" -ForegroundColor Red
    exit 1
}
Write-Host "tout passe ($($script:Total) assertions)" -ForegroundColor Green
