# jigger.psm1 — assistance winget et scoop dans PowerShell.
#
# C'est le pendant de jigger.plugin.zsh, avec les mêmes deux modes :
#
#   • popup vivant — dès que la ligne courante est une commande winget ou scoop, le
#     sélecteur s'affiche sous le prompt et suit la frappe, sans rien demander. ↓ (ou ^N)
#     fait entrer dans la liste, ↑ (ou ^P) en ressort et rend l'historique, ⇥ insère le
#     candidat courant, ^G ferme le popup pour la ligne en cours.
#   • Tab classique — quand le popup vivant est éteint (JIGGER_LIVE=0), ⇥ ouvre le
#     sélecteur interactif plein écran (`jigger pick`).
#
# Là où zsh offre un hook appelé à chaque redessin de la ligne (line-pre-redraw),
# PSReadLine n'expose que des touches. On réenregistre donc chaque touche qui modifie la
# ligne — les caractères imprimables, mais aussi Retour arrière, Suppr et les
# déplacements du curseur — derrière un relais qui rappelle la fonction PSReadLine
# d'origine puis redessine le popup. Aucun comportement d'édition n'est réécrit : le
# relais ne fait qu'ajouter le rendu.
#
# L'affichage est écrit directement sur la console, sous la ligne de commande, entre un
# ESC 7 (mémorise le curseur) et un ESC 8 (le remet) : PSReadLine ne s'aperçoit de rien,
# puisque le curseur n'a pas bougé. Quand il n'y a pas la place — le prompt occupe la
# dernière ligne, ce qui est le cas ordinaire d'un terminal en usage —, on la fait en
# poussant l'écran, puis on recale l'ancre de PSReadLine (cf. New-JiggerRoom).
#
# Installation, dans $PROFILE :
#   Import-Module C:\chemin\vers\jigger\shell\jigger.psm1
# (le binaire `jigger` doit être dans le PATH.)

Set-StrictMode -Version Latest

# ── Réglages ──────────────────────────────────────────────────────────────────────────
# Tous se posent dans l'environnement *avant* l'Import-Module, comme les réglages zsh se
# posent avant le `source`.

function Get-JiggerSetting([string]$Name, $Default) {
    $v = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrEmpty($v)) { return $Default }
    return $v
}

$script:Exe        = Get-JiggerSetting 'JIGGER_BIN' 'jigger'
$script:Live       = (Get-JiggerSetting 'JIGGER_LIVE' '1') -ne '0'
$script:Rows       = [int](Get-JiggerSetting 'JIGGER_ROWS' 8)
$script:MinColumns = [int](Get-JiggerSetting 'JIGGER_MIN_COLUMNS' 30)
$script:InsertKey  = Get-JiggerSetting 'JIGGER_KEY' 'Tab'
if ($script:InsertKey -eq '^I') { $script:InsertKey = 'Tab' }   # notation zsh, acceptée telle quelle
$script:Commands   = (Get-JiggerSetting 'JIGGER_COMMANDS' 'winget,scoop') -split '[,\s]+' |
                        Where-Object { $_ }

$script:Prompt     = (Get-JiggerSetting 'JIGGER_PROMPT' '0') -ne '0'
$script:PromptTTL  = [int](Get-JiggerSetting 'JIGGER_PROMPT_TTL' 1800)
$script:PromptSync = (Get-JiggerSetting 'JIGGER_PROMPT_SYNC' '1') -ne '0'

# Même règle que os.UserCacheDir() côté Go, pour que le module lise exactement le fichier
# que le binaire écrit. `jigger prompt --path` donne le chemin exact — mais l'interroger
# coûterait un processus par prompt.
#
# PowerShell 7 tourne aussi sur macOS et Linux, où %LOCALAPPDATA% n'existe pas : le
# `Join-Path` sur une valeur nulle faisait alors échouer l'Import-Module lui-même, avant
# la moindre ligne utile. Les trois plateformes sont donc traitées, comme le fait Go.
function Get-JiggerCacheDefaut {
    # `-or` court-circuite : sous Windows PowerShell 5.1, $IsWindows n'existe pas et le
    # mode strict refuserait de l'évaluer.
    if ($PSVersionTable.PSVersion.Major -lt 6 -or $IsWindows) {
        return Join-Path ([Environment]::GetEnvironmentVariable('LOCALAPPDATA')) 'jigger'
    }
    if ($IsMacOS) {
        return Join-Path $HOME 'Library/Caches/jigger'
    }
    $xdg = [Environment]::GetEnvironmentVariable('XDG_CACHE_HOME')
    if ([string]::IsNullOrEmpty($xdg)) { $xdg = Join-Path $HOME '.cache' }
    return Join-Path $xdg 'jigger'
}

$script:CacheDir   = Get-JiggerSetting 'JIGGER_CACHE_DIR' (Get-JiggerCacheDefaut)
$script:StatusFile = Join-Path $script:CacheDir 'status'

# Touches supplémentaires à relayer, en plus des ASCII imprimables : sur un clavier
# AZERTY, la rangée des chiffres non pressée donne « éèçàù », qui doivent eux aussi
# rafraîchir le popup.
$script:ExtraKeys  = Get-JiggerSetting 'JIGGER_KEYS_EXTRA' 'éèêàçùâîôûëïüö°²µ§£€'

# Séquences d'échappement, nommées une fois pour toutes : écrites en clair dans une
# chaîne PowerShell, « $ESC7 » désignerait une variable nommée ESC7.
$ESC     = [char]27
$SAVE    = "$ESC" + '7'    # DECSC — mémorise la position du curseur
$RESTORE = "$ESC" + '8'    # DECRC — la restitue
$CLREOL  = "$ESC" + '[K'   # efface jusqu'au bord droit
$CLRDOWN = "$ESC" + '[J'   # efface tout ce qui suit à l'écran

# ── État du popup ─────────────────────────────────────────────────────────────────────
$script:Sel       = 0      # candidat courant
$script:SelLine   = ''     # ligne pour laquelle $Sel a un sens
$script:Dismissed = $false # ^G : popup fermé jusqu'à la fin de la ligne
$script:Count     = 0      # candidats du dernier rendu
$script:Left      = ''     # ligne (jusqu'au curseur) après insertion du candidat courant
$script:Frame     = ''     # cadre déjà rendu (évite un processus par redessin)
$script:Key       = ''     # signature du dernier rendu
$script:Shown     = $false # un cadre est-il actuellement à l'écran ?
$script:Failures  = 0      # échecs d'affichage d'affilée
$script:Focused   = $false # le popup a-t-il le clavier ? (cf. Step-JiggerSelection)

# ── Vérifications d'installation ──────────────────────────────────────────────────────

if (-not (Get-Command $script:Exe -ErrorAction SilentlyContinue)) {
    Write-Warning "jigger : binaire « $($script:Exe) » introuvable dans le PATH. Module inactif."
    return
}

# Le module et le binaire vont par paire : le module passe à `jigger render` des options
# qu'un binaire plus ancien ne connaît pas. Celui-ci sortirait alors en erreur, et le
# popup ne s'afficherait jamais — sans un mot, ce qui est la pire façon de tomber en
# panne. Un appel au démarrage (~25 ms) suffit à le dire.
$script:VersionRequise = [version]'0.6.0'
$version = $null
try {
    $brut = @(& $script:Exe --version 2>$null)[0]
    $version = [version](($brut -split '\s+')[-1])
} catch { }
if ($version -and $version -lt $script:VersionRequise) {
    $chemin = (Get-Command $script:Exe).Source
    Write-Warning ("jigger : le binaire $chemin est en $version, or ce module en demande " +
        "$($script:VersionRequise). Recompile-le (« make install ») ou remplace celui du PATH. Module inactif.")
    return
}
if (-not (Get-Module PSReadLine)) {
    Import-Module PSReadLine -ErrorAction SilentlyContinue
}
if (-not (Get-Module PSReadLine)) {
    Write-Warning 'jigger : PSReadLine est absent. Module inactif.'
    return
}
# En mode Vi, les caractères imprimables ne s'insèrent pas toujours : les relayer
# casserait la navigation en mode commande. On garde alors le seul mode ⇥.
if ($script:Live -and (Get-PSReadLineOption).EditMode -eq 'Vi') {
    Write-Warning 'jigger : popup vivant désactivé en mode Vi (⇥ ouvre le sélecteur).'
    $script:Live = $false
}
# Les cadres sont dessinés en caractères semi-graphiques : sans console en UTF-8, ils
# arriveraient en charabia. PowerShell 7 s'en charge, pas 5.1.
try {
    if ([Console]::OutputEncoding.CodePage -ne 65001) {
        [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false
    }
} catch {
    # Pas de console (session redirigée, tâche planifiée) : il n'y a alors pas de popup
    # à dessiner non plus. Le reste du module — le bloc de prompt — fonctionne quand même.
}

# ── Rendu du popup ────────────────────────────────────────────────────────────────────

# Le profil couleur ne peut pas être deviné par jigger : sa sortie est capturée. C'est
# donc le shell, qui connaît son terminal, qui tranche.
function Get-JiggerColor {
    if ($env:WT_SESSION -or $env:COLORTERM -match 'truecolor|24bit') { return 'truecolor' }
    if ($env:TERM -match '256color') { return '256' }
    if ($env:TERM -eq 'dumb') { return 'never' }
    return '256'   # conhost de Windows 10 et suivants : 256 couleurs au minimum
}
$script:Color = Get-JiggerColor

# Get-JiggerBuffer rend la ligne courante jusqu'au curseur — l'équivalent du $LBUFFER de
# zsh : c'est ce qui est à gauche du curseur qui définit le contexte de complétion.
function Get-JiggerBuffer {
    $line = $null
    $cursor = 0
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    if ($null -eq $line) { return '' }
    return $line.Substring(0, $cursor)
}

# Test-JiggerLine : la ligne est-elle celle d'un gestionnaire connu ?
function Test-JiggerLine([string]$Buffer) {
    foreach ($c in $script:Commands) {
        if ($Buffer -eq $c -or $Buffer.StartsWith("$c ")) { return $true }
    }
    return $false
}

# Test-JiggerActive : un popup est-il effectivement à l'écran, avec des candidats ?
function Test-JiggerActive {
    if (-not $script:Live) { return $false }
    if ($script:Dismissed) { return $false }
    if (-not (Test-JiggerLine (Get-JiggerBuffer))) { return $false }
    return $script:Count -gt 0
}

# Get-JiggerFrame appelle le binaire et met à jour l'état. Rend $false s'il n'y a rien à
# afficher.
function Get-JiggerFrame([string]$Buffer, [int]$Rows, [int]$Columns) {
    $focus = if ($script:Focused) { 'true' } else { 'false' }
    $out = & $script:Exe render --line $Buffer --sel $script:Sel --cols $Columns `
                --rows $Rows --color $script:Color --focus=$focus 2>$null
    if ($LASTEXITCODE -ne 0 -or $null -eq $out) { return $false }
    $lines = @($out)
    if ($lines.Count -lt 2) { return $false }

    # Métadonnées : des champs clé=valeur séparés par des tabulations, `left=` en dernier
    # car c'est le seul qui peut contenir des espaces.
    $meta = $lines[0]
    $i = $meta.IndexOf("`tleft=")
    if ($i -lt 0) { return $false }
    $script:Left = $meta.Substring($i + 6)

    $champs = @{}
    foreach ($kv in $meta.Substring(0, $i) -split "`t") {
        $k, $v = $kv -split '=', 2
        $champs[$k] = $v
    }
    $script:Count = [int]$champs['count']
    $script:Sel   = [int]$champs['sel']    # jigger a ramené l'index dans les bornes
    $script:Frame = ($lines[1..($lines.Count - 1)]) -join "`n"
    return $true
}

# Format-JiggerDraw compose ce qui part vers la console : mémorisation du curseur
# (ESC 7), descente d'une ligne, le cadre — chaque ligne effacée jusqu'au bord (ESC [ K)
# pour ne rien laisser d'un cadre plus large —, puis l'effacement de tout ce qui suit
# (ESC [ J) et le retour du curseur (ESC 8). PSReadLine ne s'aperçoit de rien : de son
# point de vue, le curseur n'a pas bougé.
#
# Fonction à part, et sans effet : c'est la seule façon de vérifier ces séquences sans
# console (cf. tests/smoke.ps1).
function Format-JiggerDraw([string]$Frame) {
    if ([string]::IsNullOrEmpty($Frame)) {
        return $SAVE + "`r`n" + $CLRDOWN + $RESTORE
    }
    $corps = $Frame -replace "`n", ($CLREOL + "`r`n")
    return $SAVE + "`r`n" + $corps + $CLREOL + $CLRDOWN + $RESTORE
}

# La liste de prédictions de PSReadLine (PredictionViewStyle ListView) se dessine
# exactement là où le popup se dessine : sous la ligne de commande. Les deux se disputent
# alors les mêmes lignes à chaque frappe — et l'écran scintille.
#
# Tant que le popup est là, on ramène donc la prédiction à sa forme en ligne, qui
# n'occupe aucune ligne à elle seule (le texte grisé à la suite du curseur). Elle n'est
# pas perdue, juste rangée ; PSReadLine retrouve sa vue dès que le cadre s'efface.
$script:VuePrediction = ''

function Suspend-JiggerPrediction {
    if ($script:VuePrediction) { return }   # déjà rangée
    try {
        $vue = "$((Get-PSReadLineOption).PredictionViewStyle)"
        if ($vue -ne 'ListView') { return }
        $script:VuePrediction = $vue
        Set-PSReadLineOption -PredictionViewStyle InlineView
    } catch { }
}

function Resume-JiggerPrediction {
    if (-not $script:VuePrediction) { return }
    try { Set-PSReadLineOption -PredictionViewStyle $script:VuePrediction } catch { }
    $script:VuePrediction = ''
}

# Show-JiggerPopup écrit le cadre sous la ligne de commande.
function Show-JiggerPopup {
    Suspend-JiggerPrediction
    [Console]::Write((Format-JiggerDraw $script:Frame))
    $script:Shown = $true
}

# Hide-JiggerPopup efface tout ce qui est sous la ligne de commande.
function Hide-JiggerPopup {
    Resume-JiggerPrediction
    if (-not $script:Shown) { return }
    [Console]::Write((Format-JiggerDraw ''))
    $script:Shown = $false
}

# New-JiggerRoom pousse l'écran de $Lignes pour dégager la place du popup, puis remet le
# curseur sur la ligne de commande — qui a suivi le mouvement.
#
# Sans cela, le popup ne s'afficherait pour ainsi dire jamais : dans un terminal en
# usage, le prompt occupe la dernière ligne de l'écran, et il n'y a rien en dessous. On
# fait donc ce que fait n'importe quel sélecteur en incrustation (fzf --height, entre
# autres) : on défile.
#
# Reste que PSReadLine retient la ligne où commence sa saisie, et qu'un défilement qu'il
# n'a pas provoqué la lui fait mentir de $Lignes — sa ligne se réafficherait alors plus
# bas, précédée d'autant de vide. On corrige donc son ancre. C'est la seule chose que
# jigger touche à l'intérieur de PSReadLine, et un échec y est sans conséquence : on
# renonce simplement au défilement.
function New-JiggerRoom([int]$Lignes) {
    if ($Lignes -le 0) { return $true }
    if (-not (Move-JiggerAnchor $Lignes)) { return $false }

    $bas = [Console]::WindowTop + [Console]::WindowHeight
    [Console]::Write($SAVE + "$ESC[$bas;1H" + ("`n" * $Lignes) + $RESTORE + "$ESC[${Lignes}A")
    return $true
}

# Move-JiggerAnchor recule de $Lignes la ligne d'ancrage de PSReadLine.
function Move-JiggerAnchor([int]$Lignes) {
    try {
        $type = [Microsoft.PowerShell.PSConsoleReadLine]
        $drapeaux = [System.Reflection.BindingFlags]'NonPublic, Instance'
        $singleton = $type.GetField('_singleton', [System.Reflection.BindingFlags]'NonPublic, Static').GetValue($null)
        $ancre = $type.GetField('_initialY', $drapeaux)
        $ancre.SetValue($singleton, [int]$ancre.GetValue($singleton) - $Lignes)
        return $true
    } catch {
        return $false
    }
}

# Update-JiggerPopup fait vivre le popup : appelé après chaque touche qui modifie la
# ligne ou déplace le curseur. On ne relance le binaire que si la ligne, la sélection ou
# la géométrie ont bougé — le reste du temps, on réaffiche le cadre déjà rendu.
function Update-JiggerPopup {
    if (-not $script:Live) { return }
    try {
        $buffer = Get-JiggerBuffer
        if ($script:Dismissed -or -not (Test-JiggerLine $buffer)) {
            Hide-JiggerPopup
            return
        }

        $largeur = [Console]::WindowWidth
        if ($largeur -lt $script:MinColumns) { Hide-JiggerPopup; return }

        # Où est le curseur ? De sa position dépend la place disponible sous la ligne de
        # commande. 5 lignes de décor : 2 bordures, l'en-tête, la respiration, le pied.
        $sousLeCurseur = [Console]::WindowHeight - ([Console]::CursorTop - [Console]::WindowTop) - 1
        $voulu = $script:Rows + 5
        if ($sousLeCurseur -lt $voulu) {
            # Pas la place : on la fait. Si le défilement n'est pas possible, on se
            # contente de ce qu'il reste — et de rien du tout s'il ne reste rien.
            if (New-JiggerRoom ($voulu - $sousLeCurseur)) {
                $sousLeCurseur = $voulu
            }
        }
        $tiennent = $sousLeCurseur - 5
        if ($tiennent -lt 1) { Hide-JiggerPopup; return }
        $rows = [Math]::Min($script:Rows, $tiennent)

        # Changer la ligne change la liste : le candidat courant redevient le premier.
        if ($buffer -ne $script:SelLine) {
            $script:Sel = 0
            $script:SelLine = $buffer
        }

        $signature = "$buffer|$($script:Sel)|$($script:Focused)|$largeur|$rows"
        if ($signature -ne $script:Key) {
            $script:Key = $signature
            if (-not (Get-JiggerFrame $buffer $rows $largeur)) {
                $script:Frame = ''
                $script:Count = 0
                Hide-JiggerPopup
                return
            }
        }
        if ($script:Frame) { Show-JiggerPopup }
        $script:Failures = 0
    }
    catch {
        # Une console qui refuse de dire où elle en est, un binaire qui disparaît : on
        # n'a pas le droit de faire échouer une frappe. Au deuxième échec d'affilée, on
        # éteint le popup pour la session (⇥ reste disponible).
        $script:Failures++
        if ($script:Failures -ge 2) { $script:Live = $false }
        try { Hide-JiggerPopup } catch { }
    }
}

# Reset-JiggerLine : remise à zéro à chaque nouvelle ligne. ^G ne vaut que pour la ligne
# où il a été frappé, et l'écran a défilé : il n'y a plus rien à effacer.
function Reset-JiggerLine {
    Resume-JiggerPrediction   # filet : la vue de PSReadLine ne doit jamais rester rangée
    $script:Sel = 0
    $script:SelLine = ''
    $script:Focused = $false
    $script:Dismissed = $false
    $script:Count = 0
    $script:Frame = ''
    $script:Key = ''
    $script:Shown = $false
}

# Step-JiggerSelection décide de ce que fait une flèche, et rend $true si le popup l'a
# prise. C'est là que vit le focus, et la règle tient en deux phrases :
#
#   • ↓ fait entrer dans la liste (et y descend). Le popup a dès lors le clavier.
#   • ↑ y remonte, et rend la main dès qu'on est déjà sur le premier candidat.
#
# Hors focus, les deux flèches restent l'historique du shell — c'est le point : on ne
# perd pas l'historique parce qu'un popup est ouvert. Le cadre montre où en est le focus
# (ligne courante soulignée ou au repos), sans quoi la règle serait invisible.
#
# La fonction ne lit pas la ligne elle-même : `$Actif` lui est passé, ce qui la rend
# vérifiable hors console (cf. tests/smoke.ps1).
function Step-JiggerSelection([int]$Sens, [bool]$Actif) {
    if (-not $Actif) { return $false }

    if ($Sens -gt 0) {
        $script:Focused = $true
        $script:Sel++
    } else {
        if (-not $script:Focused) { return $false }   # popup ouvert, mais pas au clavier
        if ($script:Sel -gt 0) { $script:Sel-- } else { $script:Focused = $false }
    }
    $script:SelLine = Get-JiggerBuffer
    Update-JiggerPopup
    return $true
}

# ── Touches ───────────────────────────────────────────────────────────────────────────

# Les fonctions de PSReadLine prennent toutes ($key, $arg) : un relais peut donc rappeler
# celle qui tenait la touche, quelle qu'elle soit, avant de redessiner. Aucun
# comportement d'édition n'est réécrit ici.
#
# Les replis d'origine sont mémorisés dans une variable globale : réimporter le module ne
# doit pas enregistrer un relais qui rappellerait le relais précédent.
# Une table **ordinale** : `Ctrl+C` (copier) et `Ctrl+c` (abandonner la ligne) sont deux
# liaisons distinctes, or les tables de hachage de PowerShell ignorent la casse par
# défaut — les deux touches se confondraient.
function New-JiggerTable {
    return [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)
}

if (-not (Get-Variable -Name JiggerFallbacks -Scope Global -ErrorAction SilentlyContinue)) {
    $global:JiggerFallbacks = New-JiggerTable
}

# Liaisons PSReadLine telles qu'elles sont *avant* qu'on y touche, relevées d'un coup :
# les redemander touche par touche coûterait un appel de cmdlet par caractère imprimable.
$script:Bindings = New-JiggerTable
foreach ($h in Get-PSReadLineKeyHandler -Bound) {
    if ($h.Key -and -not $script:Bindings.ContainsKey($h.Key)) {
        $script:Bindings[$h.Key] = $h.Function
    }
}

# Get-JiggerFallback rend la fonction PSReadLine qu'un relais devra rappeler pour une
# touche : celle mémorisée lors d'un import précédent, sinon celle qui la tient
# aujourd'hui, sinon le défaut donné. La disposition du clavier, le mode d'édition et les
# autres modules varient trop pour qu'on décide à leur place.
function Get-JiggerFallback([string]$Chord, [string]$Defaut) {
    if ($global:JiggerFallbacks.ContainsKey($Chord)) {
        return $global:JiggerFallbacks[$Chord]
    }
    $fallback = ''
    if ($script:Bindings.ContainsKey($Chord)) { $fallback = $script:Bindings[$Chord] }
    # Un relais d'un import précédent n'est pas un repli : on remonterait la boucle.
    if (-not $fallback -or $fallback.StartsWith('jigger:')) { $fallback = $Defaut }
    if ($fallback) { $global:JiggerFallbacks[$Chord] = $fallback }
    return $fallback
}

# Invoke-JiggerFallback rappelle, pour une touche, la fonction PSReadLine qu'elle avait
# avant nous. Le nom est cherché à l'exécution : c'est ce qui permet aux relais d'être des
# blocs de script **simples** (cf. plus bas).
function Invoke-JiggerFallback([string]$Chord, $key, $arg, [string]$Defaut = '') {
    $f = $global:JiggerFallbacks[$Chord]
    if (-not $f) { $f = $Defaut }
    if ($f) { [Microsoft.PowerShell.PSConsoleReadLine]::$f($key, $arg) }
}

# Un relais doit être un bloc de script **simple**, écrit ici, dans le module. Ni
# `.GetNewClosure()` ni `[scriptblock]::Create()` ne conviennent : tous deux rattachent le
# bloc à un module dynamique, d'où les fonctions de jigger ne sont plus visibles — le
# relais échouerait à la première frappe sur « Update-JiggerPopup introuvable ».
#
# Conséquence : un relais ne peut rien recevoir de son enregistrement, et doit donc nommer
# sa touche en clair. D'où un relais unique pour tous les caractères imprimables — qui
# retrouve sa touche dans le caractère frappé, lequel *est* le chord sous lequel elle a
# été enregistrée — et un relais écrit par touche d'édition.
$script:RelaisTexte = {
    param($key, $arg)
    Invoke-JiggerFallback ([string]$key.KeyChar) $key $arg 'SelfInsert'
    Update-JiggerPopup
}

# Flèches et ^N/^P : au popup s'il a (ou prend) le clavier, à l'historique sinon.
$script:RelaisNavigation = @{
    'DownArrow' = { param($key, $arg)
        if (-not (Step-JiggerSelection 1 (Test-JiggerActive))) {
            Invoke-JiggerFallback 'DownArrow' $key $arg 'NextHistory'
        } }
    'UpArrow'   = { param($key, $arg)
        if (-not (Step-JiggerSelection -1 (Test-JiggerActive))) {
            Invoke-JiggerFallback 'UpArrow' $key $arg 'PreviousHistory'
        } }
    'Ctrl+n'    = { param($key, $arg)
        if (-not (Step-JiggerSelection 1 (Test-JiggerActive))) {
            Invoke-JiggerFallback 'Ctrl+n' $key $arg 'NextHistory'
        } }
    'Ctrl+p'    = { param($key, $arg)
        if (-not (Step-JiggerSelection -1 (Test-JiggerActive))) {
            Invoke-JiggerFallback 'Ctrl+p' $key $arg 'PreviousHistory'
        } }
}

$script:RelaisEdition = @{
    'Backspace'      = { param($key, $arg) Invoke-JiggerFallback 'Backspace'      $key $arg; Update-JiggerPopup }
    'Delete'         = { param($key, $arg) Invoke-JiggerFallback 'Delete'         $key $arg; Update-JiggerPopup }
    'LeftArrow'      = { param($key, $arg) Invoke-JiggerFallback 'LeftArrow'      $key $arg; Update-JiggerPopup }
    'RightArrow'     = { param($key, $arg) Invoke-JiggerFallback 'RightArrow'     $key $arg; Update-JiggerPopup }
    'Home'           = { param($key, $arg) Invoke-JiggerFallback 'Home'           $key $arg; Update-JiggerPopup }
    'End'            = { param($key, $arg) Invoke-JiggerFallback 'End'            $key $arg; Update-JiggerPopup }
    'Ctrl+LeftArrow' = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+LeftArrow' $key $arg; Update-JiggerPopup }
    'Ctrl+RightArrow'= { param($key, $arg) Invoke-JiggerFallback 'Ctrl+RightArrow' $key $arg; Update-JiggerPopup }
    'Ctrl+Backspace' = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+Backspace' $key $arg; Update-JiggerPopup }
    'Ctrl+Delete'    = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+Delete'    $key $arg; Update-JiggerPopup }
    'Ctrl+w'         = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+w'         $key $arg; Update-JiggerPopup }
    'Ctrl+u'         = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+u'         $key $arg; Update-JiggerPopup }
    'Ctrl+v'         = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+v'         $key $arg; Update-JiggerPopup }
    'Ctrl+y'         = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+y'         $key $arg; Update-JiggerPopup }
    'Ctrl+z'         = { param($key, $arg) Invoke-JiggerFallback 'Ctrl+z'         $key $arg; Update-JiggerPopup }
    'Escape'         = { param($key, $arg) Invoke-JiggerFallback 'Escape'         $key $arg; Update-JiggerPopup }
}

# ⏎ et ^C mettent fin à la ligne : ce qui suivait le prompt va être recouvert par la
# sortie de la commande, ou par le prompt suivant. On efface le cadre *avant* de leur
# rendre la main, sinon il resterait à l'écran, coupé en deux.
$script:RelaisFin = @{
    'Enter'  = { param($key, $arg) Hide-JiggerPopup; Reset-JiggerLine
                 Invoke-JiggerFallback 'Enter'  $key $arg 'AcceptLine' }
    'Ctrl+c' = { param($key, $arg) Hide-JiggerPopup; Reset-JiggerLine
                 Invoke-JiggerFallback 'Ctrl+c' $key $arg 'CopyOrCancelLine' }
}

# Register-JiggerKey mémorise le repli de la touche puis lui pose le relais donné. Une
# touche qui ne fait rien aujourd'hui et n'a pas de défaut est laissée libre : lui
# inventer un comportement ne nous regarde pas.
function Register-JiggerKey([string]$Chord, [scriptblock]$Relais, [string]$Defaut, [string]$Etiquette) {
    $fallback = Get-JiggerFallback $Chord $Defaut
    if (-not $fallback -or $fallback -eq 'Ignore') { return }
    if (-not $Etiquette) { $Etiquette = "jigger:$fallback" }
    Set-PSReadLineKeyHandler -Chord $Chord -ScriptBlock $Relais `
        -BriefDescription $Etiquette -Description 'Relaie la touche, puis rafraîchit le popup jigger.'
}

if ($script:Live) {
    # Caractères imprimables : ils s'insèrent, puis le popup se refiltre.
    $imprimables = 33..126 | ForEach-Object { [char]$_ }
    foreach ($c in @($imprimables) + @($script:ExtraKeys.ToCharArray())) {
        Register-JiggerKey ([string]$c) $script:RelaisTexte 'SelfInsert'
    }
    Register-JiggerKey 'Spacebar' $script:RelaisTexte 'SelfInsert'

    # Touches qui modifient la ligne ou déplacent le curseur sans rien insérer : le
    # contexte de complétion change tout autant.
    foreach ($k in $script:RelaisEdition.Keys) {
        Register-JiggerKey $k $script:RelaisEdition[$k] ''
    }

    # ↑↓ (et ^N/^P, qui restent) : au popup s'il a le clavier, à l'historique sinon.
    Register-JiggerKey 'DownArrow' $script:RelaisNavigation['DownArrow'] 'NextHistory'     'jigger:descendre'
    Register-JiggerKey 'UpArrow'   $script:RelaisNavigation['UpArrow']   'PreviousHistory' 'jigger:monter'
    Register-JiggerKey 'Ctrl+n'    $script:RelaisNavigation['Ctrl+n']    'NextHistory'     'jigger:suivant'
    Register-JiggerKey 'Ctrl+p'    $script:RelaisNavigation['Ctrl+p']    'PreviousHistory' 'jigger:précédent'

    # ^G : ferme le popup jusqu'à la fin de la ligne.
    Set-PSReadLineKeyHandler -Chord 'Ctrl+g' -BriefDescription 'jigger:fermer' `
        -Description 'Ferme le popup jigger pour la ligne en cours.' -ScriptBlock {
        if (Test-JiggerActive) {
            $script:Dismissed = $true
            $script:Focused = $false
            $script:Frame = ''
            Hide-JiggerPopup
        }
    }

    Register-JiggerKey 'Enter'  $script:RelaisFin['Enter']  'AcceptLine'       'jigger:valider'
    Register-JiggerKey 'Ctrl+c' $script:RelaisFin['Ctrl+c'] 'CopyOrCancelLine' 'jigger:annuler'
}

# ⇥ : insère le candidat courant si le popup est là ; sinon sélecteur classique sur une
# ligne de gestionnaire (ou popup refermé), et complétion PowerShell normale partout
# ailleurs.
$null = Get-JiggerFallback $script:InsertKey 'TabCompleteNext'   # mémorise ce que ⇥ faisait
Set-PSReadLineKeyHandler -Chord $script:InsertKey -BriefDescription 'jigger:insérer' `
    -Description 'Insère le candidat jigger, ou complète normalement.' -ScriptBlock {
    param($key, $arg)

    $line = $null
    $cursor = 0
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    $buffer = if ($null -eq $line) { '' } else { $line.Substring(0, $cursor) }

    if (Test-JiggerActive) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $cursor, $script:Left)
        $script:Key = ''                       # la ligne a changé : le cadre est à refaire
        $script:SelLine = $script:Left
        $script:Sel = 0
        Update-JiggerPopup
        return
    }

    # Ailleurs que sur une ligne de gestionnaire, ⇥ garde le comportement qu'il avait —
    # complétion classique, menu, ou ce que l'utilisateur y a mis.
    if (-not (Test-JiggerLine $buffer)) {
        Invoke-JiggerFallback $script:InsertKey $key $arg 'TabCompleteNext'
        return
    }

    if ($script:Live) {
        # Popup fermé par ^G : ⇥ le rouvre.
        if ($script:Dismissed) {
            $script:Dismissed = $false
            $script:Key = ''
            Update-JiggerPopup
            return
        }
        # Popup allumé mais sans rien à proposer — aucun candidat, ou pas la place de
        # l'afficher. ⇥ rend alors la main à la complétion du shell. Surtout pas au
        # sélecteur plein écran : il dessinerait par-dessus le prompt et attendrait une
        # touche, ce qui n'a rien à voir avec ce qu'on vient de demander.
        Invoke-JiggerFallback $script:InsertKey $key $arg 'TabCompleteNext'
        return
    }

    # JIGGER_LIVE=0 : le sélecteur plein écran, celui qu'on a explicitement choisi. Il
    # possède le terminal le temps du choix.
    Hide-JiggerPopup
    $out = & $script:Exe pick $buffer
    $code = $LASTEXITCODE
    [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
    # 2 = annulé, ou sortie vide : on ne touche à rien.
    if ($code -eq 2 -or [string]::IsNullOrEmpty($out)) { return }

    [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $cursor, ($out -join ''))
    # 10 = la commande est complète → on l'exécute directement (↩ dans le sélecteur).
    if ($code -eq 10) { [Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine() }
}

# ── Bloc de prompt (oh-my-posh) ───────────────────────────────────────────────────────
#
# Exporte à chaque prompt JIGGER_WINGET_VERSION, JIGGER_WINGET_OUTDATED et
# JIGGER_SCOOP_OUTDATED, qu'un segment `text` d'oh-my-posh affiche (cf.
# shell/oh-my-posh/windows.segment.json).
#
# Règle qui gouverne tout ce qui suit : **aucun processus dans le chemin normal**.
# Compter les mises à niveau coûte une bonne seconde à winget ; ça tourne en tâche de
# fond (`jigger prompt --refresh`) et le résultat atterrit dans un fichier d'une ligne,
# que ce hook relit avec les seules API .NET. Un prompt ne doit rien attendre.
#
# Unique exception, assumée : le prompt qui suit une commande mutante. Là, le cache est
# *connu* faux, et l'attendre coûte moins cher que d'afficher un mensonge (cf.
# JIGGER_PROMPT_SYNC).

$script:Dirty       = $false   # une commande mutante vient de tourner
$script:LastRefresh = 0
$script:BasePrompt  = $null    # fonction `prompt` que la nôtre enveloppe

# Sous-commandes qui changent ce que « les mises à niveau disponibles » répondront.
$script:Mutants = @{
    winget = @('install', 'uninstall', 'upgrade', 'update', 'import', 'repair', 'source',
               'pin', 'add', 'remove', 'rm')
    scoop  = @('install', 'uninstall', 'update', 'bucket', 'hold', 'unhold', 'import',
               'reset', 'cleanup')
}

function Start-JiggerBackground([string[]]$Arguments) {
    try {
        $psi = [System.Diagnostics.ProcessStartInfo]::new((Get-Command $script:Exe).Source)
        foreach ($a in $Arguments) { [void]$psi.ArgumentList.Add($a) }
        $psi.UseShellExecute = $false
        $psi.CreateNoWindow = $true
        # Le fils ne doit pas écrire par-dessus le prompt : ses deux sorties partent dans
        # des tuyaux que personne ne lit (il n'en émet qu'une ligne).
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        [void][System.Diagnostics.Process]::Start($psi)
    } catch { }
}

# Test-JiggerMutating repère `winget install`, `scoop update`… dans une ligne sur le
# point de s'exécuter. L'analyse passe par l'AST de PowerShell : on ne lit que des noms
# de commandes, rien n'est évalué, et `git commit -m 'winget upgrade'` ne déclenche donc
# rien — le texte y est un argument, pas une commande.
function Test-JiggerMutating([string]$Line) {
    if ([string]::IsNullOrWhiteSpace($Line)) { return $false }
    $ast = [System.Management.Automation.Language.Parser]::ParseInput($Line, [ref]$null, [ref]$null)
    $commandes = $ast.FindAll({ param($n)
        $n -is [System.Management.Automation.Language.CommandAst] }, $true)

    foreach ($c in $commandes) {
        $nom = $c.GetCommandName()
        if (-not $nom) { continue }
        $nom = [IO.Path]::GetFileNameWithoutExtension($nom).ToLowerInvariant()
        if (-not $script:Mutants.ContainsKey($nom)) { continue }

        foreach ($e in $c.CommandElements | Select-Object -Skip 1) {
            $mot = "$e"
            if ($mot.StartsWith('-')) { continue }   # « winget --nowarn upgrade » reste un upgrade
            if ($script:Mutants[$nom] -contains $mot.ToLowerInvariant()) { return $true }
            break                                     # le premier mot nu est la sous-commande
        }
    }
    return $false
}

# Set-JiggerCounter exporte <nom>=<valeur>, ou retire la variable de l'environnement si
# la valeur est nulle. C'est ce qui permet aux gabarits oh-my-posh de tester la simple
# présence de la variable, sans comparaison de chaînes.
function Set-JiggerCounter([string]$Name, [int]$Value) {
    if ($Value -gt 0) {
        [Environment]::SetEnvironmentVariable($Name, "$Value")
    } else {
        [Environment]::SetEnvironmentVariable($Name, $null)
    }
}

# Update-JiggerPrompt est appelée avant chaque prompt. Exportée : qui n'utilise pas
# oh-my-posh peut l'appeler depuis sa propre fonction `prompt`.
function Update-JiggerPrompt {
    Reset-JiggerLine

    $maintenant = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $age = $script:PromptTTL   # cache absent → réputé périmé

    # Une commande mutante vient de tourner : la liste des paquets installés est fausse.
    # Sans cela, le paquet qu'on vient d'installer manquerait à la complétion de
    # `uninstall` jusqu'à la péremption du cache. On la refait donc tout de suite, en
    # tâche de fond — que le bloc de prompt soit activé ou non, car cela ne le regarde pas.
    if ($script:Dirty) {
        $script:Dirty = $false
        Start-JiggerBackground @('warm', '--installed')

        # Le compteur du prompt, lui, est connu faux : on le refait avant de lire, pas au
        # prochain TTL. --wait parce qu'un rafraîchissement paresseux d'un autre onglet a
        # fort bien pu partir pendant l'upgrade et tenir encore le verrou : sans attendre,
        # on renoncerait justement dans le cas qu'on cherche à corriger.
        if ($script:Prompt) {
            $script:LastRefresh = $maintenant
            if ($script:PromptSync) {
                & $script:Exe prompt --refresh --wait *> $null
            } else {
                Start-JiggerBackground @('prompt', '--refresh', '--wait')
            }
        }
    }

    if (-not $script:Prompt) { return }

    if ([IO.File]::Exists($script:StatusFile)) {
        try {
            $ligne = ([IO.File]::ReadAllText($script:StatusFile) -split "`r?`n")[0]
            $champs = $ligne -split "`t"
            if ($champs.Count -eq 4 -and
                $champs[1] -match '^\d+$' -and $champs[2] -match '^\d+$' -and $champs[3] -match '^\d+$') {
                [Environment]::SetEnvironmentVariable('JIGGER_WINGET_VERSION', $champs[0])
                Set-JiggerCounter 'JIGGER_WINGET_OUTDATED' ([int]$champs[1])
                Set-JiggerCounter 'JIGGER_SCOOP_OUTDATED'  ([int]$champs[2])
                Set-JiggerCounter 'JIGGER_OUTDATED'        ([int]$champs[1] + [int]$champs[2])
                $age = $maintenant - [long]$champs[3]
            }
        } catch { }
    }

    # Rafraîchissement paresseux : on affiche toujours la valeur connue, et on prépare la
    # suivante. Le garde-fou évite qu'une rafale de prompts (ou dix onglets) ne relance
    # winget en boucle ; le verrou côté Go couvre le cas inter-processus.
    if ($age -ge $script:PromptTTL -and ($maintenant - $script:LastRefresh) -gt 60) {
        $script:LastRefresh = $maintenant
        Start-JiggerBackground @('prompt', '--refresh')
    }
}

# PSReadLine appelle AddToHistoryHandler juste avant d'exécuter la ligne : c'est le seul
# « preexec » de PowerShell. On s'y greffe sans déloger un gestionnaire existant — et le
# gestionnaire d'origine est mémorisé *globalement*, pour qu'un réimport du module
# n'aille pas s'enchaîner sur le nôtre.
if (-not (Get-Variable -Name JiggerHistoryHandler -Scope Global -ErrorAction SilentlyContinue)) {
    $global:JiggerHistoryHandler = (Get-PSReadLineOption).AddToHistoryHandler
}
$script:HistoryHandlerPrecedent = $global:JiggerHistoryHandler
Set-PSReadLineOption -AddToHistoryHandler {
    param([string]$line)
    try {
        if (-not $script:Dirty -and (Test-JiggerMutating $line)) { $script:Dirty = $true }
    } catch { }
    # Le gestionnaire précédent est rendu par PSReadLine sous forme de délégué, pas de
    # bloc de script : on l'appelle donc par Invoke, qui vaut pour les deux.
    if ($script:HistoryHandlerPrecedent) {
        try { return $script:HistoryHandlerPrecedent.Invoke($line) } catch { return $true }
    }
    return $true
}

# `prompt` est le seul « precmd » de PowerShell : on enveloppe celui qui est en place —
# celui d'oh-my-posh, le plus souvent. Notre code tourne donc *avant* que le prompt ne
# soit calculé, sans quoi les compteurs auraient toujours un coup de retard. L'ordre des
# imports dans $PROFILE n'a qu'une exigence : importer jigger après oh-my-posh.
# Le prompt d'origine est mémorisé globalement : réimporter le module doit réinstaller
# *notre* enveloppe autour du prompt d'origine, jamais autour de l'enveloppe précédente.
$marqueur = 'jigger:prompt-hook'
if (-not (Get-Variable -Name JiggerBasePrompt -Scope Global -ErrorAction SilentlyContinue)) {
    $promptActuel = (Get-Command prompt -CommandType Function -ErrorAction SilentlyContinue)
    $global:JiggerBasePrompt = if ($promptActuel -and "$($promptActuel.ScriptBlock)" -notmatch $marqueur) {
        $promptActuel.ScriptBlock
    } else { $null }
}
$script:BasePrompt = $global:JiggerBasePrompt
Set-Item -Path function:global:prompt -Value {
    # jigger:prompt-hook
    # Quoi qu'il arrive ici, le prompt doit s'afficher : une erreur de jigger ne peut pas
    # se mettre à barrer l'écran de rouge à chaque ligne.
    try { Update-JiggerPrompt } catch { }
    if ($script:BasePrompt) { & $script:BasePrompt }
    else { "PS $($ExecutionContext.SessionState.Path.CurrentLocation)$('>' * ($NestedPromptLevel + 1)) " }
}

# Au premier prompt, les catalogues peuvent être vieux d'un jour (ou absents, à la
# première installation) : on lance le réchauffement tout de suite, détaché. Le binaire
# décide lui-même s'il y a quelque chose à faire.
Start-JiggerBackground @('warm')

Export-ModuleMember -Function Update-JiggerPrompt, Test-JiggerMutating
