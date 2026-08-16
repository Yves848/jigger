# Passe macOS — derniers résultats

*Engendré par `tests/passe-macos.sh`. Ne pas modifier à la main.*

**Verdict : tout passe.**

| Étape | Code | Durée |
|---|---|---|
| go build | ok | 1 s |
| go test | ok | 0 s |
| zpty.zsh (vrai pseudo-terminal) | ok | 142 s |
| smoke.ps1 | ok | 1 s |
| banc de rendu (français figé) | ok | 4 s |

## Contexte

```
date    : 2026-08-16 11:18:37
macOS 26.5.2 · zsh 5.9 · go1.26.6
commit  : eba2d59
```

## go build

```
```

## go test

```
ok  	gitlab.yg-devworks.com/yves/jigger	0.006s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/brew	0.008s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/complete	0.039s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/facade	0.007s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/i18n	0.008s
?   	gitlab.yg-devworks.com/yves/jigger/internal/managers	[no test files]
ok  	gitlab.yg-devworks.com/yves/jigger/internal/pm	(cached)
ok  	gitlab.yg-devworks.com/yves/jigger/internal/prompt	0.524s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/scoop	0.016s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/ui	0.014s
ok  	gitlab.yg-devworks.com/yves/jigger/internal/winget	0.005s
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

## banc de rendu (français figé)

```
aucune différence (480 combinaisons)
```
