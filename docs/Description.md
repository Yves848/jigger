## A quoi sert le module `Jigger`
C'est un module helper pour différents packages managers connus.
Il fait partie de la suite `Cocktails` initialement dédiée à Homebrew sous macO

## Direction
`Jigger`va devenir un `proxy`, un `orm` pour les différents packages managers connus.
Une seule syntaxe, un seul jeu de commandes et d'options à retenir.  `Jigger` fera le lien avec la couche inférieure qui sera le package manager cible.

## Portée
Dans sa phase 1, `Jigger`va se concentrer sur Homebrew (macOS), Winget (Windows) et Scoop (Windows).  
D'autres PM seront ajoutés par la suite.

## Architecture
C'est la phase critique.  
Elle doit être modulaire.  On utilisera le lazyload pour charger la couche correspondant au PM sélectionné.
On utilisera une gestion par ADR pour avancer dans le projet.
Je laisse `Claude` structurer l'analyse et l'arborescence sur disque.
Les documents seront stockés dans Obsidian.

## Technologie
A trancher dès le début.
Pour des raisons de portabilité, on va limiter le choix entre 3 stacks techniques :
- C# (+ Avalonia pour le visuel)
- Go (+ BubbleTea / LipGloss pour le visuel) (version acuelle)
- Rust (manifestement pour l'efficacité, aucune idée de la stack visuelle à ce point)

