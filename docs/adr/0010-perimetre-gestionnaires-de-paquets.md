# ADR-0010 — jigger assiste les gestionnaires de paquets, pas les commandes quelconques

6 septembre 2026 — **acceptée** · remplace l'[ADR-0009](0009-viviers-de-plugin-par-verbe.md)

## Contexte

L'[ADR-0009](0009-viviers-de-plugin-par-verbe.md) posait qu'« un plugin existe pour rendre
commode une commande **existante** », et en tirait tout son mécanisme : viviers nommés par
verbe, régime `direct` interrogé à la frappe, options par verbe. Elle a été appliquée, et le
plugin `git` réécrit en helper conforme à cette définition.

Le résultat marchait. `git ⇥` proposait les dix-sept vraies sous-commandes, `git checkout ⇥`
les branches avec leur retard et leur âge, `git add ⇥` les fichiers modifiés du répertoire
courant. Et il a été retiré — c'est ce retrait qui rend cette décision nécessaire, parce
qu'il ne s'explique par aucun défaut technique.

**Ce que l'usage a montré :**

- **Le shell complète déjà git, et il le fait bien.** La valeur ajoutée du helper se
  réduisait à la colonne de contexte — le retard d'une branche, l'âge d'une étiquette. Réel,
  mais mince au regard de ce qu'il fallait porter pour l'obtenir.
- **Un helper sans binaire est borné par ce que la commande assistée sait produire.**
  Mesuré : `git remote -v` met l'URL au champ du badge et rien ne l'en déplace,
  `git status` ne sait pas se formater. Deux verbes sur dix-sept restaient sans contexte,
  et les rendre complets aurait exigé un binaire — donc de renoncer à la propriété qui
  faisait l'élégance de la chose.
- **Le coût, lui, ne se réduisait pas.** Le régime `direct` met un sous-processus dans le
  chemin du rendu à chaque frappe, et fait tourner le binaire d'un tiers dans le répertoire
  courant sans que l'utilisateur ait rien lancé. L'ADR-0009 l'assumait pour un besoin qui
  n'existe plus.
- **Le périmètre n'était écrit nulle part.** C'est ce qui a permis d'écrire un plugin `git`
  sans que rien ne s'y oppose, deux fois.

## Options pesées

| Option | Ce qu'elle coûte |
|---|---|
| **A. Garder les helpers dans le périmètre** — la position de l'ADR-0009 | Met jigger en concurrence avec les complétions natives des shells, qui ont vingt ans d'avance sur chaque commande. Chaque helper demande de connaître une commande de plus, et le jour où l'une d'elles change, c'est jigger qui a tort. La surface à tenir n'a pas de borne. |
| **B. Restreindre aux gestionnaires de paquets** *(retenue)* | Ferme une capacité qui vient d'être construite et éprouvée : le régime `direct`, les viviers par verbe, les options par verbe partent avec. Quelqu'un qui espérait écrire un helper pour sa commande devra chercher ailleurs — et la documentation doit le lui dire franchement plutôt que de le laisser découvrir. |
| **C. Garder le mécanisme sans livrer de helper** | La pire des trois : une machinerie sans utilisateur, qu'il faut maintenir et documenter, et une surface de sécurité — sous-processus tiers à chaque frappe — que plus aucun besoin ne justifie. Une invitation qu'il faudrait ensuite retirer. |

L'option A était celle que je défendais deux heures plus tôt, et la mesure de ce qu'un helper
apportait vraiment est ce qui l'a écartée.

## Décision

**jigger assiste les gestionnaires de paquets. Un plugin fait connaître à jigger un
gestionnaire de plus — celui d'un langage, un gestionnaire maison, un dépôt d'entreprise —
et rien d'autre : assister une commande quelconque est hors périmètre.**

Le mécanisme de plugins garde donc ce qu'un gestionnaire réclame — découverte, verbes,
exécution par le chemin natif, caches réchauffés — et perd ce qui n'avait été ajouté que pour
les helpers.

## Conséquences

**Ce que ça coûte.** Le travail de l'ADR-0009 est retiré : `pm.Viviers`, les viviers nommés,
le régime `direct` et les options par verbe. Trois heures de conception et de tests partent
avec, et c'est le prix d'avoir tranché le périmètre **après** avoir construit plutôt
qu'avant.

**Ce que ça ferme.** Un plugin ne peut plus proposer de candidats contextuels calculés à la
frappe. Un gestionnaire de paquets n'en avait pas besoin — son catalogue est gros, lent et
stable, exactement ce que le cache réchauffé sert — mais si l'un d'eux en réclamait un un
jour, il faudrait rouvrir cette décision, et non contourner.

**Ce que ça garde.** Tout ce qui précédait : la découverte des plugins, l'exécution par le
chemin natif (ADR-0008), le silence sur les verbes non déclarés, l'armement du mot dans les
greffons. Un gestionnaire tiers reste écrivable, et c'est le seul usage promis.

**Ce qui la remplacerait.** Un gestionnaire de paquets dont les candidats seraient
véritablement contextuels — dépendants du répertoire courant plutôt que d'un catalogue — et
que le cache rendrait faux. On saurait alors que la frontière tracée ici passe au mauvais
endroit.
