# Fixture de capture — PowerShell
#
# L'équivalent Windows de fixtures/zdotdir/.zshrc : le décor figé dans lequel les
# captures sont tournées. Il n'est jamais chargé à la place du $PROFILE de la
# machine — capturer.ps1 le passe explicitement par « pwsh -NoProfile -File ».

# Le même prompt d'un caractère que sur zsh, dans le même bleu. Sans lui, la
# capture Windows porterait un chemin, donc un nom d'utilisateur.
function prompt { "$([char]27)[38;2;137;180;250m❯$([char]27)[0m " }

# La langue de la capture ; l'anglais est celle des documents de référence.
if (-not $env:JIGGER_LANG) { $env:JIGGER_LANG = 'en' }

# Les commandes surveillées, écrites en toutes lettres plutôt que laissées au
# défaut : une capture ne doit pas dépendre d'un défaut qui pourrait changer.
$env:JIGGER_COMMANDS = 'winget,scoop,ssh,scp,sftp'

# Le module pris dans le dépôt, et non dans l'installation de la machine : une
# capture doit montrer le code qu'on documente.
if (-not $env:JIGGER_REPO) { throw 'JIGGER_REPO doit pointer sur la racine du dépôt' }
Import-Module (Join-Path $env:JIGGER_REPO 'shell/jigger.psm1')

# L'historique de PSReadLine est la principale source de non-déterminisme : ↑ y
# rejouerait ce que la machine a tapé avant.
Set-PSReadLineOption -HistorySaveStyle SaveNothing
Set-PSReadLineOption -PredictionSource None
Clear-Host
