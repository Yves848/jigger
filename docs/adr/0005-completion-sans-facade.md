# ADR-0005 — La complétion accueille ce que la façade ne décrit pas

30 août 2026 — **acceptée**

## Contexte

Une demande arrive : proposer les serveurs SSH dès qu'on tape `ssh`, comme jigger propose
déjà les formules dès qu'on tape `brew`. Même popup, mêmes touches, même greffon.

`ssh` n'est pas un gestionnaire de paquets. Il n'installe rien, ne désinstalle rien, n'a
ni catalogue distant ni notion d'« obsolète ». Le vocabulaire de la façade — les douze
verbes de [l'ADR-0002](0002-facade-table-declarative.md) — ne décrit rien de ce qu'il fait.

[A-16](../ameliorations.md) avait déjà posé la question pour npm, et posé la bonne :

> La question à trancher n'est pas « peut-on écrire la table ? » mais « le vocabulaire de
> la façade décrit-il encore la réalité ? ». Si la réponse est non, le dire est un résultat
> d'étude parfaitement valable.

`ssh` pousse cette question un cran plus loin. npm est au moins un gestionnaire ; `ssh`
n'en est pas un du tout.

## Le fait qui décide

L'ADR-0002 a séparé **deux contrats** :

| Contrat | Nature | Ce qu'il exige |
|---|---|---|
| `pm.Manager` | observation — *« ne fait que répondre à des questions »* | `Subcommands`, `Options`, `Load`, `Insert`, `Warm` |
| `pm.Bindings` | action | une table `verbe → liaison` |

Cette séparation n'était pas motivée par `ssh`, qui n'existait pas dans la discussion. Elle
n'en répond pas moins exactement : **un fournisseur peut implémenter `Manager` sans
implémenter `Bindings`.** Rien dans le code ne l'exige, et rien ne le suppose — `All()`
rend des `pm.Manager`, la façade interroge les `Bindings` séparément.

## Décision

> **Le contrat de complétion, `pm.Manager`, n'est pas réservé aux gestionnaires de
> paquets.** Toute commande dont l'argument se choisit dans une liste connue peut être un
> fournisseur de complétion, sans table de verbes et sans jamais rien exécuter.
>
> La façade, elle, reste ce qu'elle est : un vocabulaire de gestion de paquets. Un
> fournisseur sans `Bindings` n'y apparaît pas, et `jg install` ne le concerne pas.

`ssh`, `scp` et `sftp` sont les trois premiers de cette espèce.

## Conséquence sur `complete`

`completeWith` calcule la position ainsi : le mot qui suit immédiatement la commande est
traité comme une **sous-commande**, le catalogue ne venant qu'ensuite. C'est la grammaire
de `brew install firefox` — commande, verbe, opérande.

`ssh archlight` n'a pas de verbe : l'opérande est en deuxième position. Deux issues :

- faire rendre les hôtes par `Subcommands()` — aucun code partagé touché, mais la branche
  `firstWord` construit `Item{Name: s}` **sans badge**, donc sans l'adresse en regard, et
  le modèle mentirait : ce ne sont pas des sous-commandes ;
- **poser la règle générale** : *un fournisseur qui ne déclare aucune sous-commande propose
  son catalogue dès le premier mot.*

La seconde est retenue. Elle n'est pas un cas particulier greffé pour `ssh` : elle énonce
ce que « pas de sous-commande » a toujours voulu dire — l'opérande commence tout de suite.
Elle est vérifiable en une ligne (`len(m.Subcommands()) == 0`), et les trois gestionnaires
existants en déclarent tous, donc aucun ne change de comportement.

## Conséquences

- `pm.Manager` **n'est pas modifié**. C'est le troisième ADR d'affilée à le laisser
  intact, et c'est un signe : le contrat de complétion était bien tracé.
- Un fournisseur sans verbes ne coûte que `Load()` et `Insert()`. Les autres méthodes
  rendent le vide.
- `Cmd()` ne rend qu'un mot. `ssh`, `scp` et `sftp` sont donc **trois fournisseurs**
  partageant une implémentation et un catalogue, plutôt qu'un fournisseur à trois noms —
  ce qui aurait exigé d'élargir l'interface pour un besoin qu'aucun gestionnaire de
  paquets n'a.
- La porte est ouverte à d'autres : `git checkout` sur les branches, `docker` sur les
  conteneurs, `kubectl` sur les contextes. Aucun n'est décidé ici ; ce qui est décidé,
  c'est qu'ils n'auraient pas à se déguiser en gestionnaires de paquets.
- La conception complète est dans
  [la spec du 30 août 2026](../specs/2026-08-30-selecteur-ssh-design.md).
