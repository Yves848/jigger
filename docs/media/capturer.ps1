<#
.SYNOPSIS
  Produit les enregistrements et les captures de jigger sur Windows.

.DESCRIPTION
  L'équivalent Windows de capturer.sh. Le décor est le même — police, palette,
  colonnes, vitesse de frappe, fréquence, format final — et les fichiers rendus
  sont comparables image pour image à ceux de macOS et d'Omarchy.

  La mécanique, elle, diffère, et il faut savoir pourquoi. Sur zsh, VHS rejoue
  un script de frappe dans un ttyd, lui-même dans tmux. Aucune des deux pièces
  n'existe ici : ttyd n'a pas d'équivalent Windows utilisable, tmux n'existe
  pas. Rien de cela n'est un défaut de jigger — le module PowerShell lit la
  position du curseur par PSReadLine et n'a, lui, aucun besoin de tmux.

  Ce script rejoue donc la même partition avec les outils de la plateforme :

    • Windows Terminal tient lieu de terminal, habillé pour l'occasion ;
    • SendKeys tient lieu de frappe, cadencée sur une horloge absolue ;
    • ffmpeg (gdigrab) tient lieu d'enregistreur, sur le rectangle client exact
      de la fenêtre — sans bordure, sans barre d'onglets, sans curseur de souris.

  Tout ce qu'il modifie sur la machine — les réglages de Windows Terminal — est
  sauvegardé avant et rendu après, y compris si la capture échoue.

.PARAMETER Scenario
  Un ou plusieurs scénarios. Par défaut : les six.

.PARAMETER Preparer
  Ouvre le décor et l'y laisse une minute, sans enregistrer. Sert à vérifier de
  ses yeux la police, la palette et le format avant de tourner.

.EXAMPLE
  pwsh -NoProfile -File docs\media\capturer.ps1
  # les six scénarios, de bout en bout

.EXAMPLE
  pwsh -NoProfile -File docs\media\capturer.ps1 -Scenario 04-regex
  # un seul

.NOTES
  Prérequis : PowerShell 7, Windows Terminal, ffmpeg et ffprobe sur le PATH, le
  binaire jigger sur le PATH, et la police MesloLGL Nerd Font installée.

  ATTENTION : les scénarios 05 et 06 installent et mettent à jour de VRAIS
  paquets scoop (hexyl, hyperfine) — c'est tout leur intérêt, une exécution
  jouée ne prouverait rien. Le script pose l'état attendu avant et le défait
  après : la machine est rendue telle qu'elle était, et aucun paquet qu'elle
  avait déjà n'est touché.

  Voir docs/captures.md.
#>
[CmdletBinding()]
param(
  [ValidateSet('01-gestionnaire-natif','02-jg','03-ssh','04-regex','05-installation','06-upgrade')]
  [string[]]$Scenario = @('01-gestionnaire-natif','02-jg','03-ssh','04-regex','05-installation','06-upgrade'),
  [switch]$Preparer
)

$ErrorActionPreference = 'Stop'

# La conscience du DPI doit être posée AVANT que quoi que ce soit crée une fenêtre.
# Sans elle, GetClientRect rend des unités logiques quand gdigrab filme des pixels
# physiques : sur un écran mis à l'échelle, la capture ne montrerait qu'un coin de
# la fenêtre — et l'erreur est silencieuse.
Add-Type @'
using System; using System.Runtime.InteropServices;
public class Dpi { [DllImport("user32.dll")] public static extern bool SetProcessDPIAware(); }
'@
[void][Dpi]::SetProcessDPIAware()
Add-Type -AssemblyName System.Windows.Forms

$Media = Split-Path -Parent $PSCommandPath
$Repo  = Split-Path -Parent (Split-Path -Parent $Media)
$Out   = Join-Path $Media 'out'
$Travail = Join-Path $env:TEMP 'jigger-capture'

foreach ($outil in 'ffmpeg', 'ffprobe', 'jigger') {
  if (-not (Get-Command $outil -ErrorAction SilentlyContinue)) {
    throw "manquant : $outil sur le PATH"
  }
}
$JiggerExe = (Get-Command jigger).Source

# ── Le décor figé, mot pour mot celui de generer-tapes.sh ──────────────────────
#
# Une seule grandeur manque à cette liste, et c'est le nombre de lignes : personne
# ne le déclare, ni ici ni dans les tapes. VHS le déduit de la hauteur demandée et
# de la cellule que xterm.js donne à la police ; on fait la même déduction, mais
# il faut d'abord MESURER la cellule — elle dépend de la police, du DPI et de la
# version de Windows Terminal (voir Capturer).
$Colonnes = 72
$Police   = 'MesloLGL Nerd Font'
$Taille   = 22
$Freq     = 24
$MsParCar = 90
$LargeurFinale = 1000 ; $HauteurFinale = 530 ; $Marge = 24
$RatioContenu  = ($LargeurFinale - 2*$Marge) / ($HauteurFinale - 2*$Marge)

# ── Les scénarios, en étapes ───────────────────────────────────────────────────
#
# Une étape est l'une de ces cinq choses :
#
#   @{ Taper   = '…'   }   frappe, un caractère toutes les 90 ms
#   @{ Touche  = '…'   }   une touche, syntaxe SendKeys ({TAB}, {DOWN}, ^r…)
#   @{ Pause   = 1500  }   attente, en millisecondes
#   @{ Photo   = $true }   l'instant où l'image fixe est prise
#   @{ Attente = 60000 }   attente d'une commande qui tourne (voir « Où couper »)
#
# Les trois premiers scénarios reprennent, milliseconde pour milliseconde, les
# Sleep du tape zsh du même nom : ce ne sont pas les mêmes d'un scénario à
# l'autre, et toucher un tape demande de revoir ces étapes-ci.
#
# « Preparer » et « Ranger » encadrent la capture, hors caméra : ils posent
# l'état que le scénario suppose, puis le défont. Sans eux, un second passage
# filmerait « déjà installé » au lieu d'une installation.
$Scenarios = [ordered]@{

  # 01 · Le gestionnaire natif. La démonstration qu'on attend en premier : on tape
  # la commande qu'on tapait déjà, et le cadre arrive tout seul.
  '01-gestionnaire-natif' = @{
    Etapes = @(
      @{ Taper = 'winget install fire' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 1000 }
      @{ Touche = '{DOWN}' }, @{ Pause = 1000 }
      @{ Touche = '{DOWN}' }, @{ Pause = 1000 }
      @{ Touche = '{TAB}'  }, @{ Pause = 2500 }
    )
  }

  # 02 · La syntaxe unique. « node » plutôt que le « fd » des tapes zsh : sous
  # Windows, « fd » n'est connu que d'un seul catalogue et le cadre n'aurait
  # qu'une ligne. « node » en donne quatre, répartis entre scoop et winget — la
  # colonne de droite le dit, et c'est la démonstration attendue de jg.
  '02-jg' = @{
    Etapes = @(
      @{ Taper = 'jg install node' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 1000 }
      @{ Touche = '{DOWN}' }, @{ Pause = 1200 }
      @{ Touche = '{TAB}'  }, @{ Pause = 2500 }
    )
  }

  # 03 · Le sélecteur SSH. Une commande sans verbe : le catalogue vient dès
  # l'espace. Les hôtes sont ceux du fixture, donc les mêmes partout.
  '03-ssh' = @{
    Etapes = @(
      @{ Taper = 'ssh ' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 500 }
      @{ Touche = '{DOWN}' }, @{ Pause = 1000 }
      @{ Touche = '{DOWN}' }, @{ Pause = 1000 }
      @{ Touche = '{TAB}'  }, @{ Pause = 2500 }
    )
  }

  # 04 · La recherche par expression régulière. Le scénario montre les DEUX modes
  # dans la même prise : la même ligne, filtrée en texte simple puis en regex.
  # « fire » donne vingt-et-un candidats ; « ^R » bascule, le titre du cadre
  # affiche « [regex] », et « (bird|blade) » n'en garde que quatre — une
  # alternance qu'aucune recherche par préfixe ne sait exprimer.
  '04-regex' = @{
    Etapes = @(
      @{ Taper = 'winget install fire' }
      @{ Pause = 2500 }
      @{ Touche = '^r' }, @{ Pause = 1500 }
      @{ Taper = '(bird|blade)' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 1000 }
      @{ Touche = '{TAB}' }, @{ Pause = 2500 }
    )
  }

  # 05 · Une installation, pour de vrai. Le cadre complète, ⇥ insère, ⏎ part — et
  # à partir de là jigger ne fait plus rien : la sortie est celle de scoop,
  # relayée telle quelle, barre de progression comprise.
  #
  # « hexy » et non « hex » : le second laisse quatorze candidats, dont le premier
  # est hex-editor-neo — la capture installerait alors un éditeur graphique de
  # 30 Mo au lieu de l'outil de deux méga-octets qu'on voulait montrer. Une
  # complétion qui exécute se choisit sans ambiguïté.
  '05-installation' = @{
    Etapes = @(
      @{ Taper = 'jg install hexy' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 1500 }
      @{ Touche = '{TAB}'   }, @{ Pause = 1200 }
      @{ Touche = '{ENTER}' }, @{ Attente = 60000 }
    )
    Preparer = { scoop uninstall hexyl 2>&1 | Out-Null }
    Ranger   = { scoop uninstall hexyl 2>&1 | Out-Null }
  }

  # 06 · Une mise à jour, pour de vrai. Même geste, autre verbe. La préparation
  # installe une vieille version de hyperfine pour qu'il y ait quelque chose à
  # mettre à jour, et le rangement la retire : aucun paquet que la machine avait
  # déjà n'est touché.
  '06-upgrade' = @{
    Etapes = @(
      @{ Taper = 'jg upgrade hyperf' }
      @{ Pause = 2000 }, @{ Photo = $true }, @{ Pause = 1500 }
      @{ Touche = '{TAB}'   }, @{ Pause = 1200 }
      @{ Touche = '{ENTER}' }, @{ Attente = 60000 }
    )
    Preparer = {
      scoop uninstall hyperfine 2>&1 | Out-Null
      scoop install hyperfine@1.16.1 2>&1 | Out-Null
      # scoop range une version demandée à la main sous « auto-generated » et cesse
      # alors de la comparer au bucket : « jg upgrade » répondrait « latest version »
      # et le scénario ne montrerait rien du tout — c'est ce qu'a filmé la première
      # prise. On rattache donc l'application à son bucket d'origine.
      $j = Join-Path $env:USERPROFILE 'scoop\apps\hyperfine\current\install.json'
      $o = Get-Content $j -Raw | ConvertFrom-Json
      $o | Add-Member -NotePropertyName bucket -NotePropertyValue 'main' -Force
      $o.PSObject.Properties.Remove('url')
      $o | ConvertTo-Json -Compress | Set-Content $j -Encoding utf8
    }
    Ranger   = { scoop uninstall hyperfine 2>&1 | Out-Null }
  }
}

# ── Fenêtres : trouver celle qu'on vient d'ouvrir, et la mesurer ───────────────
Add-Type @'
using System; using System.Text; using System.Runtime.InteropServices; using System.Collections.Generic;
public class Fenetres {
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool ClientToScreen(IntPtr h, ref POINT p);
  [DllImport("user32.dll")] public static extern int GetClassName(IntPtr h, StringBuilder s, int m);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr p);
  [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
  public delegate bool EnumProc(IntPtr h, IntPtr p);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
  [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X, Y; }
  // La classe de fenêtre de Windows Terminal. Chercher par titre serait fragile :
  // le titre suit le shell, et plusieurs terminaux peuvent être ouverts.
  public static List<IntPtr> Terminaux() {
    var l = new List<IntPtr>();
    EnumWindows((h, p) => {
      if (!IsWindowVisible(h)) return true;
      var sb = new StringBuilder(256); GetClassName(h, sb, 256);
      if (sb.ToString() == "CASCADIA_HOSTING_WINDOW_CLASS") l.Add(h);
      return true;
    }, IntPtr.Zero);
    return l;
  }
}
'@

# ── Le clavier : de l'Unicode, jamais une disposition ─────────────────────────
#
# SendKeys traduit chaque caractère en touche selon la disposition COURANTE. Sur
# un clavier AZERTY, « | » est un AltGr+6 qu'il ne sait pas former : il l'avale
# en silence, et « fire(bird|blade) » devient « fire(birdblade) » — donc « no
# matches ». SendInput en mode KEYEVENTF_UNICODE envoie le caractère lui-même,
# sans passer par une touche : la capture ne dépend plus de la disposition de la
# machine qui la produit. Les touches de navigation, elles, restent à SendKeys —
# ⇥, ↓, ⏎ et ^R sont des touches virtuelles, pas des caractères.
Add-Type @'
using System; using System.Runtime.InteropServices;
public class Clavier {
  [StructLayout(LayoutKind.Explicit, Size = 40)]
  public struct INPUT {
    [FieldOffset(0)]  public uint type;
    [FieldOffset(8)]  public ushort wVk;
    [FieldOffset(10)] public ushort wScan;
    [FieldOffset(12)] public uint dwFlags;
    [FieldOffset(16)] public uint time;
    [FieldOffset(24)] public IntPtr dwExtraInfo;
  }
  [DllImport("user32.dll")] static extern uint SendInput(uint n, INPUT[] p, int taille);
  public static void Caractere(char c) {
    var i = new INPUT[2];
    i[0].type = 1; i[0].wScan = c; i[0].dwFlags = 0x0004;            // KEYEVENTF_UNICODE
    i[1].type = 1; i[1].wScan = c; i[1].dwFlags = 0x0004 | 0x0002;   // + KEYUP
    SendInput(2, i, Marshal.SizeOf(typeof(INPUT)));
  }
}
'@

function Rect-Client([IntPtr]$h) {
  $r = New-Object Fenetres+RECT ; [void][Fenetres]::GetClientRect($h, [ref]$r)
  $p = New-Object Fenetres+POINT ; [void][Fenetres]::ClientToScreen($h, [ref]$p)
  [pscustomobject]@{ X = $p.X; Y = $p.Y; W = $r.Right - $r.Left; H = $r.Bottom - $r.Top }
}

# ── Windows Terminal : on habille le profil PAR DÉFAUT, puis on le rend ────────
#
# Ajouter un profil dédié serait plus propre — et c'est ce qu'on a essayé d'abord.
# Windows Terminal 1.23 relit bien settings.json à chaud, mais ne retient pas un
# profil apparu après son démarrage : « --profile jigger-capture » retombe alors
# silencieusement sur le profil par défaut, et la capture sort avec la police et
# la transparence de l'utilisateur. On habille donc le profil par défaut lui-même,
# sauvegarde à l'appui, et on le rend dans un « finally ».
$WtSettings = Join-Path $env:LOCALAPPDATA 'Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json'
$WtSauvegarde = Join-Path $Travail 'wt-settings.sauvegarde.json'

function Habiller-Terminal {
  $cfg = Get-Content $WtSauvegarde -Raw | ConvertFrom-Json

  # Catppuccin Mocha, les mêmes valeurs que le thème JSON des tapes.
  $palette = [ordered]@{
    name='Jigger Mocha'; background='#1E1E2E'; foreground='#CDD6F4'; cursorColor='#F5E0DC'
    selectionBackground='#585B70'
    black='#45475A'; red='#F38BA8'; green='#A6E3A1'; yellow='#F9E2AF'; blue='#89B4FA'
    purple='#F5C2E7'; cyan='#94E2D5'; white='#BAC2DE'
    brightBlack='#585B70'; brightRed='#F38BA8'; brightGreen='#A6E3A1'; brightYellow='#F9E2AF'
    brightBlue='#89B4FA'; brightPurple='#F5C2E7'; brightCyan='#94E2D5'; brightWhite='#A6ADC8'
  }
  $cfg.schemes = @($cfg.schemes | Where-Object { $_.name -ne 'Jigger Mocha' }) + [pscustomobject]$palette

  # Le mode « focus » retire la barre d'onglets : le rectangle client n'est plus
  # que la grille de caractères, et la mesure n'a plus rien à en retrancher.
  $cfg | Add-Member -NotePropertyName launchMode -NotePropertyValue 'focus' -Force

  $def = $cfg.profiles.list | Where-Object guid -eq $cfg.defaultProfile
  # « adjustIndistinguishableColors » n'est pas un détail : laissé à « always », il
  # retouche les couleurs de jigger pour les rendre lisibles, et la capture ne
  # montre plus la palette de la charte.
  foreach ($kv in @{
      colorScheme='Jigger Mocha'; padding='0'; opacity=100; useAcrylic=$false
      scrollbarState='hidden'; cursorShape='filledBox'; antialiasingMode='grayscale'
      adjustIndistinguishableColors='never'; historySize=100
    }.GetEnumerator()) { $def | Add-Member -NotePropertyName $kv.Key -NotePropertyValue $kv.Value -Force }
  $def | Add-Member -NotePropertyName font `
                    -NotePropertyValue ([pscustomobject]@{ face=$Police; size=$Taille }) -Force

  # Écriture par fichier temporaire puis renommage : Windows Terminal surveille le
  # fichier, et une écriture en place peut lui présenter un JSON à moitié écrit.
  $tmp = "$WtSettings.jigger-tmp"
  $cfg | ConvertTo-Json -Depth 32 | Set-Content $tmp -Encoding utf8
  Move-Item $tmp $WtSettings -Force
  Start-Sleep -Seconds 5      # le temps qu'il relise
}

function Rendre-Terminal { Copy-Item $WtSauvegarde $WtSettings -Force }

# ── Ouvrir un terminal au décor figé ───────────────────────────────────────────
$Wt = Join-Path $env:LOCALAPPDATA 'Microsoft\WindowsApps\wt.exe'

function Ouvrir-Terminal([int]$Lignes, [string]$Amorce) {
  $avant = [Fenetres]::Terminaux()
  Start-Process $Wt -ArgumentList @(
    '--window','new','--pos','120,120','--size',"$Colonnes,$Lignes",
    'new-tab','pwsh','-NoProfile','-NoExit','-File',$Amorce)
  for ($i = 0; $i -lt 100; $i++) {
    Start-Sleep -Milliseconds 200
    $neuves = [Fenetres]::Terminaux() | Where-Object { $avant -notcontains $_ }
    if ($neuves) { return @($neuves)[0] }
  }
  throw 'fenêtre Windows Terminal introuvable'
}

function Fermer-Terminal([IntPtr]$h) {
  [void][Fenetres]::PostMessage($h, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero)  # WM_CLOSE
  Start-Sleep -Seconds 2
}

# ── L'amorce jouée dans le terminal capturé ────────────────────────────────────
function Ecrire-Amorce([string]$Nom) {
  $lignes = @(
    "`$env:JIGGER_REPO = '$Repo'"
    "`$env:JIGGER_BIN  = '$JiggerExe'"
    "`$env:JIGGER_LANG = if (`$env:JIGGER_LANG) { `$env:JIGGER_LANG } else { 'en' }"
  )
  # Le sélecteur SSH lit le ~/.ssh/config du profil, sans surcharge possible
  # (internal/ssh/manager.go passe par os.UserHomeDir, soit %USERPROFILE% ici). On
  # lui donne la copie locale du HOME de fixture : la capture montre les serveurs
  # inventés du dépôt, les mêmes que sur macOS et Omarchy, jamais l'infrastructure
  # de la machine. Les scénarios qui installent, eux, gardent le vrai profil —
  # c'est là que scoop range ses paquets.
  if ($Nom -eq '03-ssh') {
    $lignes += "`$env:USERPROFILE = '$Travail\home'"
    $lignes += "`$env:HOME        = '$Travail\home'"
  }
  $lignes += ". '$Media\fixtures\profile.ps1'"
  $fichier = Join-Path $Travail "amorce-$Nom.ps1"
  Set-Content -Path $fichier -Value ($lignes -join "`n") -Encoding utf8
  return $fichier
}

# ── Le minutage d'un scénario, déduit de ses étapes ────────────────────────────
function Minutage($Etapes) {
  $t = 0.0 ; $photo = 0.0 ; $attente = $false
  foreach ($e in $Etapes) {
    if ($e.Photo)   { $photo = $t ; continue }
    if ($e.Taper)   { $t += $e.Taper.Length * $MsParCar / 1000.0 ; continue }
    if ($e.Pause)   { $t += $e.Pause / 1000.0 ; continue }
    if ($e.Attente) { $t += $e.Attente / 1000.0 ; $attente = $true ; continue }
    # une touche est instantanée
  }
  [pscustomobject]@{ Duree = $t; Photo = $photo; Attente = $attente }
}

# ── Une capture ────────────────────────────────────────────────────────────────
function Capturer([string]$Nom) {
  $sc     = $Scenarios[$Nom]
  $amorce = Ecrire-Amorce $Nom
  $m      = Minutage $sc.Etapes

  if ($sc.Preparer) {
    Write-Host '  mise en place…' -ForegroundColor DarkGray
    & $sc.Preparer
  }

  # Première passe : mesurer la cellule. Sa taille dépend de la police, du DPI et
  # de la version de Windows Terminal — on ne peut pas la supposer, et c'est elle
  # qui décide du nombre de lignes, donc du format de l'image.
  $hwnd = Ouvrir-Terminal 24 $amorce
  Start-Sleep -Seconds 6
  $r = Rect-Client $hwnd
  $cellW = $r.W / $Colonnes ; $cellH = $r.H / 24
  $lignes = [math]::Floor(($Colonnes * $cellW) / ($RatioContenu * $cellH))
  Write-Host ("  cellule {0} x {1} px → {2} lignes" -f [math]::Round($cellW,2), [math]::Round($cellH,2), $lignes)
  Fermer-Terminal $hwnd

  # Seconde passe : celle qu'on filme.
  $hwnd = Ouvrir-Terminal $lignes $amorce
  Start-Sleep -Seconds 6
  [void][Fenetres]::SetForegroundWindow($hwnd)
  Start-Sleep -Milliseconds 600
  $r = Rect-Client $hwnd

  if ($Preparer) {
    Write-Host "  décor ouvert — $($r.W) x $($r.H) px. Une minute pour le regarder."
    Start-Sleep -Seconds 60
    Fermer-Terminal $hwnd
    return
  }

  # ffmpeg s'arrête tout seul, sur « -t ». Lui envoyer « q » par un tuyau ne
  # fonctionne pas — il ne lit le clavier que depuis une vraie console — et le
  # tuer laisse un conteneur inachevé que ffprobe refuse ensuite de mesurer.
  $duree = [math]::Round(0.8 + $m.Duree + 1.0, 2)
  $brut  = Join-Path $Travail "brut-$Nom.mkv"
  Remove-Item $brut -ErrorAction SilentlyContinue

  # Un pixel de bordure arrondie de Windows Terminal déborde sur la grille : on le
  # retire des quatre côtés plutôt que de le retrouver dans l'image finale.
  $psi = [System.Diagnostics.ProcessStartInfo]::new('ffmpeg')
  $psi.Arguments = "-y -hide_banner -f gdigrab -framerate $Freq -draw_mouse 0 " +
                   "-offset_x $($r.X + 1) -offset_y $($r.Y + 1) " +
                   "-video_size $($r.W - 2)x$($r.H - 2) -i desktop " +
                   "-t $duree -c:v libx264 -preset ultrafast -qp 0 -pix_fmt yuv444p `"$brut`""
  $psi.UseShellExecute = $false ; $psi.RedirectStandardError = $true
  $ff = [System.Diagnostics.Process]::Start($psi)

  # Attendre que ffmpeg filme VRAIMENT. Entre le lancement du processus et la
  # première image il s'écoule une seconde qu'on ne peut pas deviner, et tout
  # décalage ici décale d'autant l'instant de l'image fixe. La première ligne
  # d'état (« frame= ») dit que l'encodage a commencé.
  for ($k = 0; $k -lt 200; $k++) {
    $l = $ff.StandardError.ReadLine()
    if ($null -eq $l -or $l -match 'frame=') { break }
  }

  [void][Fenetres]::SetForegroundWindow($hwnd)
  Start-Sleep -Milliseconds 800          # le « Sleep 800ms » qui suit Show dans les tapes

  # La frappe est cadencée sur une horloge ABSOLUE. SendKeys coûte plusieurs
  # dizaines de millisecondes par caractère : un simple « Start-Sleep 90ms » dérive
  # de moitié sur une ligne de vingt caractères, et l'image fixe tombe alors en
  # pleine frappe — c'est exactement ce qu'a montré la première passe.
  $horloge = [System.Diagnostics.Stopwatch]::StartNew()
  function Attendre([double]$ms) { while ($horloge.ElapsedMilliseconds -lt $ms) { Start-Sleep -Milliseconds 5 } }

  $t = 0.0
  foreach ($e in $sc.Etapes) {
    if ($e.Photo) { continue }
    if ($e.Taper) {
      foreach ($c in $e.Taper.ToCharArray()) {
        Attendre $t
        [Clavier]::Caractere($c)
        $t += $MsParCar
      }
      continue
    }
    if ($e.Touche)  { Attendre $t ; [System.Windows.Forms.SendKeys]::SendWait($e.Touche) ; continue }
    if ($e.Pause)   { $t += $e.Pause   ; continue }
    if ($e.Attente) { $t += $e.Attente ; continue }
  }
  Attendre $t

  if (-not $ff.WaitForExit(180000)) { $ff.Kill() }
  Fermer-Terminal $hwnd
  if ($sc.Ranger) { Write-Host '  rangement…' -ForegroundColor DarkGray ; & $sc.Ranger }

  # ── Où couper ───────────────────────────────────────────────────────────────
  #
  # Un scénario qui exécute une commande ne peut pas savoir d'avance combien de
  # temps elle durera : la sienne télécharge. On lui laisse donc une minute, puis
  # on coupe la queue morte — ffmpeg sait dire quand l'image cesse de bouger, et
  # le dernier de ces instants est la fin de la commande. Les scénarios qui ne
  # font qu'ouvrir un cadre gardent, eux, leur minutage exact.
  $coupe = $duree
  if ($m.Attente) {
    $journal = (& ffmpeg -hide_banner -nostats -i $brut -vf 'freezedetect=n=-60dB:d=1.5' `
                  -map 0:v -f null - 2>&1) -join "`n"
    $gels = @([regex]::Matches($journal, 'freeze_start:\s*([0-9.]+)') |
              ForEach-Object { [double]$_.Groups[1].Value })
    if ($gels.Count -gt 0) { $coupe = [math]::Min($gels[-1] + 2.0, $duree) }
    Write-Host ("  commande terminée vers {0} s (sur {1} s filmées)" -f [math]::Round($coupe,1), $duree)
  }

  # ── Les trois fichiers, au format exact des captures VHS ────────────────────
  New-Item -ItemType Directory -Force -Path $Out | Out-Null
  $sortie = "windows-$Nom"
  # 952 px de contenu, puis 24 px de marge de chaque côté : 1000 × 530, comme VHS.
  $filtre = "fps=$Freq,scale=$($LargeurFinale - 2*$Marge):-2:flags=lanczos," +
            "pad=${LargeurFinale}:${HauteurFinale}:(ow-iw)/2:(oh-ih)/2:color=0x1E1E2E"
  ffmpeg -y -loglevel error -t $coupe -i $brut `
    -vf "$filtre,split[a][b];[a]palettegen[p];[b][p]paletteuse" (Join-Path $Out "$sortie.gif")
  ffmpeg -y -loglevel error -t $coupe -i $brut -vf $filtre -pix_fmt yuv420p -an (Join-Path $Out "$sortie.mp4")

  # L'image fixe est prise à l'étape « Photo » : cadre ouvert, complet, aucune
  # touche de navigation encore pressée. Elle est extraite du GIF, comme sur Unix
  # — l'image fixe ne peut donc pas montrer autre chose que l'enregistrement.
  $instant = [math]::Round(0.8 + $m.Photo, 2)
  ffmpeg -y -loglevel error -ss $instant -i (Join-Path $Out "$sortie.gif") `
    -vframes 1 (Join-Path $Out "$sortie.png")
  Write-Host "  → $sortie.gif  $sortie.mp4  $sortie.png  (image fixe à $instant s)"
}

# ── Le déroulé ─────────────────────────────────────────────────────────────────
New-Item -ItemType Directory -Force -Path $Travail | Out-Null
Copy-Item $WtSettings $WtSauvegarde -Force

# Les fixtures sont recopiées en local : un HOME sur un partage réseau ferait payer
# sa latence à chaque frappe, et la capture s'en verrait.
Remove-Item (Join-Path $Travail 'home') -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $Travail 'home') | Out-Null
Copy-Item "$Media\fixtures\home\*" (Join-Path $Travail 'home') -Recurse -Force

Habiller-Terminal
try {
  foreach ($nom in $Scenario) {
    Write-Host "── windows-$nom" -ForegroundColor Cyan
    Capturer $nom
  }
}
finally {
  Rendre-Terminal
  Write-Host 'réglages de Windows Terminal rendus.' -ForegroundColor DarkGray
}
