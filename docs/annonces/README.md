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

**Les quatre messages citent GitHub**, et c'est le fruit d'un essai raté plutôt que d'une
préférence.

Le 17 août, le lien du code a été basculé sur GitLab — le dépôt de référence, GitHub n'en
étant qu'un miroir poussé — pour éviter le bot d'audit de r/commandline. **Le post a été
rejeté sans motif.** Un domaine personnel sans réputation (`gitlab.yg-devworks.com`) tombe
sous les filtres d'automod, quand ce n'est pas sous celui de Reddit lui-même, et cela se
fait en silence.

Le choix est donc entre deux désagréments inégaux :

| Lien | Ce qui arrive | Coût |
|---|---|---|
| GitHub | « GitHub Guard » commente : 0 sur 1, dépôt de moins de 30 jours | cosmétique, le post tient |
| GitLab auto-hébergé | suppression automatique, sans explication | le post entier |

Le miroir GitHub a été créé le 16 août 2026 : il passe la barre des 30 jours vers le
**15 septembre**, après quoi la question ne se pose plus.

Deux choses à savoir avant de retenter GitLab :

- **Les commandes d'installation portent le domaine de toute façon.** `brew tap` et
  `scoop bucket add` pointent `gitlab.yg-devworks.com` dans le corps des messages r/zsh et
  r/PowerShell. Si un automod filtre le domaine où qu'il apparaisse, changer le lien du code
  ne suffit pas.
- **Ne pas poster sans lien pour le déposer ensuite en commentaire.** C'est le motif que les
  modérateurs guettent, et il coûte plus cher que le commentaire d'un bot. Le modmail, lui,
  tranche en une minute et lève souvent une suppression automatique.

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
