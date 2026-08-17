# Les textes d'annonce

Quatre messages, un par canal, rédigés d'après
[la conception de la diffusion](../specs/2026-08-16-diffusion-design.md).

**Ils sont publiés par Yves, jamais en son nom par quelqu'un d'autre.** Ce sont ses
comptes et sa réputation.

## L'ordre, et pourquoi

Du public le plus indulgent au plus exposé, pour que les premiers retours corrigent le tir
avant le passage qui ne se rejoue pas.

| Rang | Fichier | Canal | Attendre |
|---|---|---|---|
| 1 | [`reddit-commandline.md`](reddit-commandline.md) | r/commandline | — |
| 2 | [`reddit-zsh.md`](reddit-zsh.md) | r/zsh | 2–3 jours |
| 3 | [`reddit-powershell.md`](reddit-powershell.md) | r/PowerShell | 2–3 jours |
| 4 | [`show-hn.md`](show-hn.md) | Hacker News | 2–3 jours |

Poster le même jour partout se voit, et prive des retours qui auraient amélioré le texte
suivant.

## Quel dépôt citer

**Les trois messages Reddit pointent GitLab**, pas le miroir GitHub. Deux raisons, dans cet
ordre :

1. **C'est le lien exact.** GitLab est le dépôt de référence, GitHub n'en est qu'un miroir
   poussé. Le dire ainsi n'est pas un détour.
2. **r/commandline fait auditer les liens GitHub par un bot** (« GitHub Guard »), qui note
   un dépôt de moins de 30 jours 0 sur 1 et commente le post en conséquence. Le miroir a
   été créé le 16 août 2026 : il passe la barre vers le **15 septembre**.

Ne pas poster sans lien du tout pour éviter le bot, puis le déposer en commentaire : c'est
le motif que les modérateurs guettent, et il coûte plus cher que le commentaire du bot. Si
le sujet revient, le modmail tranche en une minute.

`show-hn.md` garde GitHub — Hacker News n'a pas ce garde-fou, et son public y est habitué.

## Avant de poster, à chaque fois

- [ ] Un début de journée ouvrable côté Amérique du Nord. **Jamais un vendredi ni un
      week-end.**
- [ ] Deux heures devant soi. C'est le vrai facteur : ce qui décide du sort d'un message
      n'est pas son texte mais la présence de son auteur dans l'heure qui suit.
- [ ] Les liens cliqués une dernière fois — le site, le dépôt, le guide.
- [ ] `brew install jigger` et `scoop install jigger` fonctionnent toujours.

## Pendant

Répondre à tout, **surtout aux critiques**. Un bogue signalé pendant l'annonce et corrigé
le jour même est la meilleure réponse possible à un premier passage : la chaîne de release
permet de publier en une commande.

Ne pas défendre un choix contesté par principe. Si trois personnes butent sur la même
chose, c'est la chose qui est en cause, pas les trois personnes.

## Après

Ce qu'on regarde : les issues ouvertes par des inconnus, et ce que les gens comprennent de
travers dans le README. **Toute question posée deux fois est un défaut de documentation.**
