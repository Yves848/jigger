# ADR-0002 — Façade multi-gestionnaires par table déclarative

15 août 2026 — **acceptée**

## Contexte

[Description.md](../Description.md) veut faire de jigger un « proxy, un ORM pour les
différents packages managers » : une seule syntaxe, un seul jeu de commandes à retenir.

jigger est aujourd'hui **passif** : il observe la ligne qu'on tape et la complète, il
n'émet jamais de commande. `pm.Manager` est un **contrat de complétion** — `Subcommands`,
`Options`, `InstalledOnly`, `Load`, `Insert`, `Warm` — dont le commentaire de tête dit
explicitement qu'une implémentation *« ne fait que répondre à des questions »*.

Ajouter la façade, c'est ajouter une nature d'**action** à côté de cette nature
d'**observation**. Trois formes étaient possibles.

## Options considérées

**A — Interface `Executor`, symétrique de `Manager`.** Chaque gestionnaire écrit du code
Go méthode par verbe (`Install`, `Uninstall`, `Outdated`, `Supports`…).

**B — Table de verbes déclarative + moteur générique.** Chaque gestionnaire déclare une
table `verbe → liaison` ; un moteur unique construit l'argv, exécute et normalise.

**C — Élargir `pm.Manager` en place.** Ajouter les méthodes d'exécution à l'interface
existante.

## Décision

**Option B.** Un second contrat, `pm.Bindings`, indépendant de `pm.Manager` :

```go
type Verb string   // « install », « source add » — un membre de phrase entier

type Binding struct {
    Native []string                              // gabarit d'argv — le cas ordinaire
    Build  func(args []string) []string          // argv calculé — le cas rétif
    Direct func(args []string) ([]Package, error) // sans sous-processus du tout

    Pool  Pool   // Catalogue | Installés | Aucun
    Parse Parser // nil → relais brut
}

type Parser func(out []byte) ([]Package, error)

type Bindings interface{ Verbs() map[Verb]Binding }
```

## Justification

- **Une capacité déclarée dans du code impératif est une convention qu'on oublie de
  tenir ; une capacité déclarée dans une table est vérifiable.** Le modèle de capacités
  tombe tout seul : un verbe absent de la table est un verbe non supporté, sans drapeau à
  maintenir.
- **La table de correspondance devient un artefact réel** — lisible d'un bloc, diffable en
  merge request, testable sans lancer aucun gestionnaire. C'est le document qui manquait à
  [Description.md](../Description.md) ; ici, il *est* le code.
- **Le popup s'étend presque gratuitement.** Les clés de table sont le vocabulaire :
  `jg ⇥` complète les verbes, `jg source ⇥` complète `add` et `rm`, sans donnée
  supplémentaire.
- **Un gestionnaire ajouté coûte une table**, pas dix méthodes.

L'option A a été écartée pour la raison inverse : dix méthodes × trois gestionnaires,
presque toutes réduites à « construire un argv et lancer », et une table de correspondance
qui reste **implicite**, éparpillée dans trente fonctions — alors que « que sait faire
scoop ? » est exactement la question que le modèle de capacités doit rendre triviale.

L'option C a été écartée parce qu'elle casse la frontière citée plus haut : elle forcerait
les trois paquets à implémenter de l'exécution jusque dans les tests, et rendrait
`pm.Manager` inutilisable comme contrat de complétion seule. Le gain — un type de moins —
ne vaut pas ça.

## Conséquences

- `pm.Manager` **n'est pas modifié**. Les deux contrats vivent dans le paquet `pm` mais
  dans des fichiers séparés (`pm.go` fait déjà 341 lignes, et ils ne se lisent pas
  ensemble).
- `Build` est une **porte de sortie, pas la règle** : un gabarit d'argv ne dira jamais tout
  (`brew tap` / `untap` sont deux mots sans rapport).
- `Direct` existe parce que **scoop sait déjà répondre sans lancer scoop** :
  `internal/scoop/outdated.go` compare les manifestes sur le disque. Passer par un
  sous-processus pour redemander ce que jigger sait déjà serait absurde.
- `InstalledOnly() bool` est promu en `Pool` à trois valeurs, pour couvrir « ce verbe ne
  prend pas de paquet du tout ».
- La conception complète qui découle de cet ADR est dans
  [la spec du 15 août 2026](../specs/2026-08-15-facade-multi-gestionnaires-design.md).
