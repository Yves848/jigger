# Passe Windows — derniers résultats

*Engendré par `tests/passe-windows.ps1`. Ne pas modifier à la main.*

**Verdict : tout passe.**

| Étape | Code |
|---|---|
| captures scoop et winget | ok |
| go build | ok |
| go test | ok |
| smoke.ps1 | ok |
| pty.ps1 (vraie console) | ok |

## Contexte

```
date       : 2026-08-16 08:51:25
PowerShell : 7.6.3
système    : Microsoft Windows NT 10.0.26200.0
go         : go version go1.26.5 windows/amd64
commit     : eb3d131
```

## captures scoop et winget

```
scoop : C:\Users\yvesg\scoop\shims\scoop.cmd

→ jeux d'essai des analyseurs (internal/scoop/testdata)
  ok    list.txt  (10 lignes)
  ok    source.txt  (8 lignes)
  ok    search.txt  (80 lignes)

→ références pour la table des verbes (tests/captures)
  ok    scoop-help.txt  (38 lignes)
  ok    scoop-update-help.txt  (17 lignes)
winget : C:\Users\yvesg\AppData\Local\Microsoft\WindowsApps\winget.exe
  ok    winget-pin-help.txt  (27 lignes)
  ok    winget-source-help.txt  (30 lignes)
  ok    contexte.txt

Terminé. Reste à publier les captures :

  git add internal/scoop/testdata tests/captures
  git commit -m "Captures reelles de scoop et winget, machine Windows"
  git push

Les analyseurs seront réécrits contre ces fichiers.
```

## go build

```

```

## go test

```
ok  	gitlab.yg-devworks.com/yves/jigger	0.661s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/brew	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/complete	1.249s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/facade	0.747s
?   	gitlab.yg-devworks.com/yves/jigger/internal/managers	[no test files]
ok  	gitlab.yg-devworks.com/yves/jigger/internal/pm	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/prompt	1.245s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/scoop	0.899s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/ui	0.743s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/winget	0.598s
?   	gitlab.yg-devworks.com/yves/jigger/tests/conpty	[no test files]
```

## smoke.ps1

```
  ok   « winget --nowarn upgrade »
  ok   « scoop update * »
  ok   « scoop bucket add extras »
  ok   « winget search git »
  ok   « scoop status »
  ok   « winget list »
  ok   « git commit -m "winget upgrade" »
  ok   « echo scoop install 7zip »
  ok   « ls; winget install jq »

→ le prompt lit le cache sans lancer de processus
  ok   version exportée
  ok   compteur winget exporté
  ok   compteur scoop absent
  ok   total exporté
  ok   plus rien à signaler
  ok   la version reste
  ok   cache corrompu ignoré

→ une commande mutante fait refaire la liste des installés
  ok   les installés sont refaits
  ok   sans bloc de prompt, rien de plus
  ok   aucune commande mutante, aucun appel

tout passe (62 assertions)
```

## pty.ps1 (vraie console)

```
  ok   aucun cadre

→ ⇥ insère le candidat courant dans la ligne
  ok   ligne complétée

→ ⇥ sans candidat rend la main au shell, sans ouvrir de sélecteur
  ok   ligne intacte
  ok   aucun sélecteur ouvert
  ok   aucun « annuler »

→ les flèches prennent le clavier, puis le rendent
  ok   sans focus : ↓ parcourir
  ok   après ↓ : ↑↓ naviguer
  ok   deux ↑ rendent le clavier
  ok   ⇥ insère le candidat visé

→ la liste de prédictions de PSReadLine cède la place au popup
  ok   le cadre est bien là
  ok   la liste a cédé la place
  ok   aucun cadre sur une autre ligne

→ ^C efface le cadre avant de rendre la ligne
  ok   cadre effacé

tout passe (19 assertions)
```
