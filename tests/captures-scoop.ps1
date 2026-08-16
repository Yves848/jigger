# Capture les sorties réelles de scoop (et deux aides de winget) sur une machine Windows.
#
#   pwsh -NoProfile -File tests/captures-scoop.ps1
#
# À lancer depuis la racine du dépôt, sur une machine où scoop est installé. Le script
# n'installe rien, ne modifie rien : il ne fait que lire et écrire des fichiers texte dans
# le dépôt.
#
# ── Pourquoi un script, plutôt que trois redirections à la main ────────────────────────
#
# scoop formate ses tableaux avec Format-Table de PowerShell, dont la largeur dépend de ce
# à quoi le processus écrit : la largeur de la fenêtre s'il écrit dans une console, une
# valeur par défaut s'il écrit dans un tuyau. Or jigger capture toujours la sortie
# (cf. internal/facade.lancerReel) : il ne voit donc jamais la version « console ».
# Capturer depuis un terminal large produirait un jeu d'essai que jigger ne rencontrera
# jamais — et un parser accordé dessus resterait faux.
#
# Chaque commande est donc lancée ici avec sa sortie capturée dans une variable, c'est-à-
# dire derrière un tuyau, exactement comme le fait jigger.
#
# Deuxième précaution : la commande est résolue en **exécutable**, comme le ferait Go
# (exec.Command), et non en fonction ou en alias PowerShell — qui n'existeraient pas pour
# jigger.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = Split-Path -Parent $PSScriptRoot
$fixtures  = Join-Path $racine 'internal/scoop/testdata'
$reference = Join-Path $racine 'tests/captures'

New-Item -ItemType Directory -Force -Path $fixtures, $reference | Out-Null

# Sans cela, une sortie UTF-8 lue par une console en cp1252 arriverait avec des accents
# cassés — et les aides de winget sont traduites.
$encodageInitial = [Console]::OutputEncoding
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# UTF-8 **sans** BOM : les fichiers sont lus par des tests Go, qui ne l'attendent pas.
$utf8 = [System.Text.UTF8Encoding]::new($false)

function Trouver-Exe([string]$Nom) {
    $c = Get-Command $Nom -CommandType Application -ErrorAction SilentlyContinue |
            Select-Object -First 1
    if ($c) { return $c.Source }
    return $null
}

# Capturer lance la commande, sa sortie derrière un tuyau, et l'écrit telle quelle.
# Les fins de ligne restent celles de Windows (CRLF) : elles font partie de ce qu'on
# teste, et .gitattributes empêche git de les normaliser.
function Capturer([string]$Exe, [string[]]$Argv, [string]$Fichier) {
    $nom = [IO.Path]::GetFileName($Fichier)
    try {
        $texte = (& $Exe @Argv 2>&1 | Out-String)
    } catch {
        Write-Host "  ÉCHEC $nom — $($_.Exception.Message)" -ForegroundColor Red
        return
    }
    if ([string]::IsNullOrWhiteSpace($texte)) {
        Write-Host "  VIDE  $nom — la commande n'a rien rendu" -ForegroundColor Yellow
        return
    }
    [System.IO.File]::WriteAllText($Fichier, $texte, $utf8)
    $lignes = ($texte -split "`r?`n").Count
    Write-Host ("  ok    {0}  ({1} lignes)" -f $nom, $lignes)
}

try {
    $scoop = Trouver-Exe 'scoop'
    if (-not $scoop) {
        Write-Host @'
scoop est introuvable en tant qu'exécutable.

S'il fonctionne dans ton terminal, c'est qu'il n'y existe que comme fonction ou alias
PowerShell — et jigger, qui lance un processus, ne le trouvera pas davantage. C'est alors
un diagnostic à part entière, et non un défaut des analyseurs.
'@ -ForegroundColor Red
        exit 1
    }
    Write-Host "scoop : $scoop"

    # ── Les trois jeux d'essai des parsers ────────────────────────────────────────────
    #
    # Les noms de fichiers sont ceux qu'attend internal/scoop/parse_test.go ; source.txt
    # porte la sortie de `bucket list`, le verbe `source` de la façade.
    Write-Host "`n→ jeux d'essai des analyseurs (internal/scoop/testdata)"
    Capturer $scoop @('list')            (Join-Path $fixtures 'list.txt')
    Capturer $scoop @('bucket', 'list')  (Join-Path $fixtures 'source.txt')
    Capturer $scoop @('search', 'git')   (Join-Path $fixtures 'search.txt')

    # ── De quoi vérifier la table des verbes (tâche 1b, point 1) ──────────────────────
    #
    # Ces captures-là ne sont pas des jeux d'essai : elles servent à confronter
    # internal/scoop/verbs.go et internal/winget/verbs.go aux vraies CLI, dont les
    # colonnes n'ont jamais été vérifiées ailleurs que sur le papier.
    Write-Host "`n→ références pour la table des verbes (tests/captures)"
    Capturer $scoop @('help')             (Join-Path $reference 'scoop-help.txt')
    Capturer $scoop @('update', '--help') (Join-Path $reference 'scoop-update-help.txt')

    $winget = Trouver-Exe 'winget'
    if ($winget) {
        Write-Host "winget : $winget"
        Capturer $winget @('pin', '--help')    (Join-Path $reference 'winget-pin-help.txt')
        Capturer $winget @('source', '--help') (Join-Path $reference 'winget-source-help.txt')
    } else {
        Write-Host "  (winget introuvable — les deux captures winget sont sautées)"
    }

    # ── Le contexte, pour que ces captures restent lisibles dans six mois ─────────────
    $contexte = @(
        "Captures faites par tests/captures-scoop.ps1"
        "PowerShell      : $($PSVersionTable.PSVersion)"
        "Système         : $([System.Environment]::OSVersion.VersionString)"
        "scoop           : $((& $scoop --version 2>&1 | Out-String).Trim())"
        "winget          : $(if ($winget) { (& $winget --version 2>&1 | Out-String).Trim() } else { 'absent' })"
        "Sortie capturée : derrière un tuyau, comme jigger (jamais la largeur de la console)"
    ) -join "`r`n"
    [System.IO.File]::WriteAllText((Join-Path $reference 'contexte.txt'), $contexte + "`r`n", $utf8)
    Write-Host "  ok    contexte.txt"
}
finally {
    [Console]::OutputEncoding = $encodageInitial
}

Write-Host @'

Terminé. Reste à publier les captures :

  git add internal/scoop/testdata tests/captures
  git commit -m "Captures reelles de scoop et winget, machine Windows"
  git push

Les analyseurs seront réécrits contre ces fichiers.
'@
