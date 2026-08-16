# Mettre en page et paginer les sorties — conception

16 août 2026 — état : validé, implémentation directe demandée (pas de plan intermédiaire).

## Objet

Rendre navigables les quatre verbes tabulaires de la façade — `list`, `outdated`,
`search`, `source` — sans rien casser de ce qui les consomme aujourd'hui.

C'est l'entrée **A-10** du tableau des améliorations.

## Les quatre décisions

1. **Lire, avec sélection multiple.** On parcourt, on filtre, on coche, on sort avec les
   noms choisis. La vue n'exécute rien : pas d'installation depuis le plein écran, donc
   pas de question sur ce qui arrive quand une commande réclame un mot de passe au milieu
   d'un affichage plein écran.
2. **La table brute reste le comportement hors terminal**, à l'octet près. `jg list | grep`
   dans un script ne doit jamais ouvrir une interface qui attend une touche. Pour choisir
   dans un tube, un drapeau explicite : `--select`.
3. **Un cœur commun** plutôt qu'un second dispositif de navigation.
4. **Sous-chaîne par défaut, `^R` bascule en regex**, et le mode courant est affiché. Le
   comportement actuel de `jigger pick` ne change pas.

## §1 — Ce qu'on construit

**Une correction à la conception initiale, faite en lisant le code.** Le rendu (`Frame`,
`internal/ui/frame.go`) est intimement lié à `complete.Item` et sert **le popup** — le
chemin le plus critique du projet, couvert par le banc d'essai golden. On ne le généralise
pas.

Le cœur partagé est donc le **modèle**, pas le rendu :

| Fichier | Responsabilité |
|---|---|
| `internal/ui/liste.go` *(nouveau)* | Données, filtre, curseur, défilement, sélection. Aucun Bubble Tea, aucun style |
| `internal/ui/picker.go` | Inchangé de l'extérieur ; s'appuie sur `Liste` à l'intérieur |
| `internal/ui/tableau.go` *(nouveau)* | La vue tabulaire : modèle Bubble Tea + son rendu à colonnes |
| `internal/facade/format.go` | Le calcul des colonnes en sort pour être **partagé** |

Un tableau à cinq colonnes et un popup à une ligne se dessinent différemment : deux rendus
sont honnêtes, deux comportements de navigation ne le seraient pas.

**Les colonnes ont une seule source.** La règle adaptative — `DISPO` n'apparaît que si une
ligne en porte une — est extraite de `Formater` et employée par la table brute **et** par la
vue paginée. Sans ça, les deux divergeront le jour où l'une évoluera.

## §2 — Quand la vue s'arme

| Situation | Comportement |
|---|---|
| Terminal, contenu plus haut que l'écran | Vue paginée |
| Terminal, contenu qui tient à l'écran | Table brute — ouvrir un plein écran pour six lignes est une nuisance |
| Sortie redirigée | Table brute, à l'octet près |
| `--json` | Jamais paginé : c'est un contrat machine |
| `--select` | Vue forcée, dessinée sur `/dev/tty`, noms choisis imprimés sur la sortie standard |
| `JIGGER_PAGER=0` | Désarme, comme `JIGGER_LIVE=0` désarme le popup |

`--select` imprime **un nom par ligne**, ce qui rend `jg install $(jg search fd --select)`
naturel et compatible avec `xargs`.

## §3 — Les touches

Le champ de filtre a le focus en permanence : on tape pour filtrer, sans rien presser
d'abord. **`Espace` est donc une lettre**, pas un raccourci, et `^A`, `^E`, `^K`, `^U`,
`^W` appartiennent à l'édition du champ.

| Touche | Effet |
|---|---|
| `⇥` | coche / décoche — libre ici, puisqu'il n'y a rien à insérer |
| `↵` | valide : imprime les lignes cochées, ou la ligne courante si aucune |
| `^G`, `esc` | annule sans rien imprimer |
| `↑` `↓`, `^P` `^N` | se déplacer |
| `PgPréc` `PgSuiv`, `^B` `^F` | page par page |
| `^R` | bascule sous-chaîne ↔ regex |

Aucun chiffre, aucun symbole décalé : tout se fait d'une main sur un clavier AZERTY. Le
pied du cadre affiche les touches **et le mode de filtre courant**.

## §4 — Comment on saura que ça marche

1. **Les tests du sélecteur passent sans une modification.** Si l'extraction du cœur les
   casse, elle n'est pas fidèle. C'est le seul juge honnête de la refactorisation.
2. **La décision d'armer est une fonction pure** — `DoitPaginer(...) bool` — testable sans
   terminal, sur les cas qui comptent : tube, JSON, contenu court, `JIGGER_PAGER=0`,
   `--select`.
3. **Un test croisé sur les colonnes** : pour un même jeu de lignes, l'en-tête de la table
   brute et celui de la vue paginée sont identiques.
4. **Le filtre a ses cas limites écrits** : motif regex invalide, `c++`, et `.` qui ne doit
   pas se comporter comme un joker tant qu'on est en mode sous-chaîne.

## Non-buts

- **Pas d'exécution depuis la vue.** Elle rend des noms, elle ne lance rien.
- **Pas de tri des colonnes** — c'est A-12. Le cœur connaît les colonnes, donc rien ne
  l'empêchera ensuite.
- **Pas de coloration des versions obsolètes** — c'est A-13. La vue aura le point
  d'accroche, pas la règle.
- **Pas de regex dans le popup** — c'est A-11, qui réemploiera le filtre né ici.
- **`Frame` n'est pas généralisé.** Le popup garde son rendu.

## Risques

| Risque | Parade |
|---|---|
| L'extraction casse le sélecteur | Ses tests existants font foi, sans être modifiés |
| La vue paginée et la table brute divergent | Colonnes calculées par la même fonction, avec un test croisé |
| Un script casse | La table brute reste le comportement par défaut hors terminal |
| `--select` inutilisable dans un tube | La vue se dessine sur `/dev/tty`, jamais sur la sortie standard |
| Une touche inconfortable sur AZERTY | Aucune touche n'est un chiffre ni un symbole décalé |

## Décisions liées

- `docs/ameliorations.md` — A-10 (ce document), A-11, A-12, A-13, qui en dépendent.
