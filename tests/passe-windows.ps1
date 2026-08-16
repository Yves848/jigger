# La passe Windows en une commande : capturer, tester, consigner, publier.
#
#   pwsh -NoProfile -File tests/passe-windows.ps1
#
# À lancer depuis la racine du dépôt, sur la machine Windows. Le script enchaîne ce qui
# ne peut se faire que là — les vraies CLI winget et scoop, et la vraie console — puis
# commite et pousse le tout.
#
# ── Ce qu'il publie, et pourquoi ──────────────────────────────────────────────────────
#
# Le rapport `tests/captures/derniers-tests.md` est commité **avec** les captures. C'est
# le point : une fois poussé, la session qui travaille sur le Mac lit les résultats
# directement dans le dépôt, sans que personne ait à recopier une sortie de terminal —
# et sans qu'une erreur de copier-coller vienne s'ajouter à celles qu'on cherche.
#
# Les tests qui échouent **n'empêchent pas** la publication : un échec est une
# information, et c'est même la plus utile des deux. Le rapport le dit, le message de
# commit le dit, et le script sort en code non nul pour que tu le voies passer.
#
# ── Options ───────────────────────────────────────────────────────────────────────────
#
#   -SansCaptures   ne refait pas les captures scoop/winget (tests seuls)
#   -SansPush       commite sans pousser
#   -Message "…"    message de commit (sinon : un message qui résume la passe)

[CmdletBinding()]
param(
    [switch]$SansCaptures,
    [switch]$SansPush,
    [string]$Message
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$racine = Split-Path -Parent $PSScriptRoot
Set-Location $racine

$utf8 = [System.Text.UTF8Encoding]::new($false)
$etapes = [System.Collections.Generic.List[object]]::new()

# Etape lance une commande dans un **processus séparé** et retient son code de sortie.
#
# Le processus séparé n'est pas une précaution de style : tests/smoke.ps1 réenregistre les
# touches de PSReadLine. Le lancer dans la session courante détruirait le clavier du
# terminal depuis lequel tu l'as lancé.
function Etape([string]$Nom, [string]$Exe, [string[]]$Argv) {
    Write-Host "`n→ $Nom" -ForegroundColor Cyan
    $sortie = & $Exe @Argv 2>&1 | Out-String
    $code = $LASTEXITCODE
    Write-Host $sortie.TrimEnd()
    if ($code -eq 0) {
        Write-Host "  ok" -ForegroundColor Green
    } else {
        Write-Host "  ÉCHEC (code $code)" -ForegroundColor Red
    }
    $etapes.Add([pscustomobject]@{ Nom = $Nom; Code = $code; Sortie = $sortie.TrimEnd() })
    return $code
}

try {
    # Se remettre à jour d'abord : rien n'est plus pénible qu'une passe complète suivie
    # d'un push refusé pour non-fast-forward.
    Write-Host "→ mise à jour depuis origin" -ForegroundColor Cyan
    & git pull --ff-only 2>&1 | Write-Host

    if (-not $SansCaptures) {
        Etape 'captures scoop et winget' 'pwsh' @('-NoProfile', '-File', 'tests/captures-scoop.ps1') | Out-Null
    }

    # Le binaire d'abord, et **nommé** : `go build ./...` compile sans rien produire, or
    # smoke.ps1 et pty.ps1 lancent l'exécutable pour de vrai. Sans cette étape, les deux
    # échouent sur une machine fraîchement clonée — pour une raison qui n'a rien à voir
    # avec ce qu'on teste. (Constaté en essayant ce script avant de le livrer.)
    Etape 'go build'   'go'   @('build', '-o', 'jigger.exe', '.')                 | Out-Null
    Etape 'go test'    'go'   @('test', './...')                                 | Out-Null
    Etape 'smoke.ps1'  'pwsh' @('-NoProfile', '-File', 'tests/smoke.ps1')         | Out-Null

    # Le seul harnais qui juge ce que l'utilisateur voit vraiment : un pwsh dans une vraie
    # console (ConPTY), qui tape des touches et rend l'écran. Il n'existe que sous Windows.
    if (Test-Path 'tests/pty.ps1') {
        Etape 'pty.ps1 (vraie console)' 'pwsh' @('-NoProfile', '-File', 'tests/pty.ps1') | Out-Null
    }

    # ── Le rapport ────────────────────────────────────────────────────────────────────
    $echecs = @($etapes | Where-Object { $_.Code -ne 0 })
    $verdict = switch ($echecs.Count) {
        0       { 'tout passe' }
        1       { '1 étape en échec' }
        default { "$($echecs.Count) étapes en échec" }
    }

    $lignes = [System.Collections.Generic.List[string]]::new()
    $lignes.Add('# Passe Windows — derniers résultats')
    $lignes.Add('')
    $lignes.Add("*Engendré par ``tests/passe-windows.ps1``. Ne pas modifier à la main.*")
    $lignes.Add('')
    $lignes.Add("**Verdict : $verdict.**")
    $lignes.Add('')
    $lignes.Add("| Étape | Code |")
    $lignes.Add("|---|---|")
    foreach ($e in $etapes) {
        $etat = if ($e.Code -eq 0) { 'ok' } else { "ÉCHEC ($($e.Code))" }
        $lignes.Add("| $($e.Nom) | $etat |")
    }
    $lignes.Add('')
    $lignes.Add('## Contexte')
    $lignes.Add('')
    $lignes.Add('```')
    $lignes.Add("date       : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
    $lignes.Add("PowerShell : $($PSVersionTable.PSVersion)")
    $lignes.Add("système    : $([System.Environment]::OSVersion.VersionString)")
    $lignes.Add("go         : $((& go version 2>&1 | Out-String).Trim())")
    $lignes.Add("commit     : $((& git rev-parse --short HEAD 2>&1 | Out-String).Trim())")
    $lignes.Add('```')

    foreach ($e in $etapes) {
        $lignes.Add('')
        $lignes.Add("## $($e.Nom)")
        $lignes.Add('')
        $lignes.Add('```')
        # Les sorties vertes sont longues et sans intérêt : on n'en garde que la fin.
        # Une sortie en échec est gardée en entier — c'est elle qu'on vient lire.
        if ($e.Code -eq 0) {
            $texte = ($e.Sortie -split "`r?`n" | Select-Object -Last 25) -join "`n"
        } else {
            $texte = $e.Sortie
        }
        $lignes.Add($texte)
        $lignes.Add('```')
    }

    New-Item -ItemType Directory -Force -Path 'tests/captures' | Out-Null
    [System.IO.File]::WriteAllText(
        (Join-Path $racine 'tests/captures/derniers-tests-windows.md'),
        (($lignes -join "`r`n") + "`r`n"), $utf8)

    # ── Le journal des passes ─────────────────────────────────────────────────────────
    #
    # Le rapport ci-dessus est écrasé à chaque passe : il dit l'état du moment. Le journal,
    # lui, s'ajoute et ne se réécrit pas — c'est lui qui permettra de savoir, plus tard, si
    # tel commit avait été éprouvé ici. Les échecs y figurent **en clair** : on doit pouvoir
    # juger une passe sans ouvrir un autre fichier.
    $commit = (& git rev-parse --short HEAD 2>&1 | Out-String).Trim()
    $entree = [System.Collections.Generic.List[string]]::new()
    $entree.Add("## $(Get-Date -Format 'yyyy-MM-dd HH:mm') — Windows — ``$commit`` — $verdict")
    $entree.Add('')
    $capt = if ($SansCaptures) { 'captures inchangées' } else { 'captures rafraîchies' }
    $entree.Add("$([System.Environment]::OSVersion.VersionString) · pwsh $($PSVersionTable.PSVersion) · $((& go version 2>&1 | Out-String).Trim()) · $capt")
    $entree.Add('')
    $verts = @($etapes | Where-Object { $_.Code -eq 0 } | ForEach-Object { $_.Nom })
    if ($verts) { $entree.Add("- **ok** — $($verts -join ' · ')") }
    foreach ($e in $echecs) {
        # On remonte les lignes que les harnais impriment pour dire ce qui a lâché : c'est
        # ce qu'un lecteur veut voir, pas la centaine de lignes vertes qui les entourent.
        $motifs = ($e.Sortie -split "`r?`n" |
            Where-Object { $_ -match 'ÉCHEC|--- FAIL|FAIL\s|assertion' } |
            Select-Object -First 6 | ForEach-Object { $_.Trim() })
        $entree.Add("- **échec — $($e.Nom)** (code $($e.Code))")
        foreach ($m in $motifs) { $entree.Add("  - $m") }
        if (-not $motifs) { $entree.Add('  - (aucune ligne d''échec reconnue — voir le rapport détaillé)') }
    }
    $entree.Add('')

    $journal = Join-Path $racine 'docs/tests/journal.md'
    $marqueur = '<!-- nouvelles passes ici -->'
    if (Test-Path $journal) {
        $texte = [System.IO.File]::ReadAllText($journal)
        if ($texte.Contains($marqueur)) {
            $texte = $texte.Replace($marqueur, $marqueur + "`r`n`r`n" + ($entree -join "`r`n"))
            [System.IO.File]::WriteAllText($journal, $texte, $utf8)
        } else {
            Write-Host "  (marqueur absent de docs/tests/journal.md — entrée non ajoutée)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  (docs/tests/journal.md absent — entrée non ajoutée)" -ForegroundColor Yellow
    }

    # ── Publication ───────────────────────────────────────────────────────────────────
    Write-Host "`n→ publication" -ForegroundColor Cyan
    & git add 'internal/scoop/testdata' 'tests/captures' 'docs/tests/journal.md' 2>&1 | Write-Host

    $enAttente = (& git diff --cached --name-only 2>&1 | Out-String).Trim()
    if (-not $enAttente) {
        Write-Host '  rien de neuf à commiter (captures et rapport identiques)'
    } else {
        Write-Host "  fichiers : $($enAttente -replace "`r?`n", ', ')"
        if (-not $Message) {
            $Message = "Passe Windows : $verdict"
        }
        & git commit -m $Message 2>&1 | Write-Host
        if ($SansPush) {
            Write-Host '  (push sauté)'
        } else {
            & git push 2>&1 | Write-Host
        }
    }

    # D'autres fichiers modifiés ? On ne les commite pas à l'aveugle, mais on le dit.
    $reste = (& git status --short 2>&1 | Out-String).Trim()
    if ($reste) {
        Write-Host "`n  non commité (à toi de voir) :" -ForegroundColor Yellow
        Write-Host $reste
    }

    Write-Host "`n$verdict." -ForegroundColor $(if ($echecs.Count -eq 0) { 'Green' } else { 'Red' })
    exit $(if ($echecs.Count -eq 0) { 0 } else { 1 })
}
catch {
    Write-Host "`nLe script s'est arrêté : $($_.Exception.Message)" -ForegroundColor Red
    Write-Host $_.ScriptStackTrace
    exit 2
}
