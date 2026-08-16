# Passe macOS — derniers résultats

*Engendré par `tests/passe-macos.sh`. Ne pas modifier à la main.*

**Verdict : tout passe.**

| Étape | Code | Durée |
|---|---|---|
| go build | ok | 0 s |
| go test | ok | 1 s |
| zpty.zsh (vrai pseudo-terminal) | ok | 131 s |
| smoke.ps1 | ok | 1 s |

## Contexte

```
date    : 2026-08-16 11:15:01
macOS 26.5.2 · zsh 5.9 · go1.26.6
commit  : 5a54e20
```

## go build

```
```

## go test

```
ok  	gitlab.yg-devworks.com/yves/jigger	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/brew	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/complete	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/facade	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/i18n	(cached)
?   	gitlab.yg-devworks.com/yves/jigger/internal/managers	[no test files]
ok  	gitlab.yg-devworks.com/yves/jigger/internal/pm	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/prompt	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/scoop	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/ui	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/winget	(cached)
?   	gitlab.yg-devworks.com/yves/jigger/tests/conpty	[no test files]
```

## zpty.zsh (vrai pseudo-terminal)

```
  ok   puis à naviguer
→ ^G ferme le popup et laisse la ligne intacte
  ok   ligne exécutée telle quelle
→ ⏎ efface le cadre avant la sortie de la commande
  ok   aucun cadre après la sortie
→ bloc oh-my-posh : le compteur est exporté depuis le cache
  ok   compteur exporté
  ok   cache frais, aucun appel
→ une sous-commande brew inoffensive ne rafraîchit rien
  ok   brew list ne force rien
  ok   compteur inchangé
→ après un brew upgrade, le compteur est juste dès le prompt suivant
  ok   rafraîchissement forcé
  ok   compteur remis à jour
  ok   l'ancien compteur a disparu
→ le catalogue est réchauffé au chargement, et après un brew tap
  ok   réchauffement au chargement
  ok   puis après le tap
→ un brew install ne réchauffe pas le catalogue
  ok   aucun réchauffement inutile
→ la détection voit brew derrière un préfixe, et pas dans une citation
  ok   affectation en tête
  ok   brew cité ne compte pas

tout passe
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

tout passe (70 assertions)
```
