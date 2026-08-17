//go:build !windows

package elevate

// Hors Windows, jigger ne sait pas rejouer une commande élevée — et ce n'est pas un
// manque à combler ici. Aucun gestionnaire Unix ne publie de code de sortie qui dise « il
// fallait être root » ; la moitié Unix d'A-15 devra donc s'instruire autrement, en
// *anticipant* (`sudo -v` avant de lancer) plutôt qu'en constatant. Cf. la spec, §7.
//
// Les deux fonctions existent quand même, avec la même signature : la façade et `main`
// s'écrivent ainsi sans un seul `//go:build`.

// Prevue rend VoieAucune : il n'y a pas de chemin.
func Prevue() Voie { return VoieAucune }

// Rejouer n'est jamais appelé — Possible() ayant rendu false, l'appelant n'a rien proposé.
// Il rend une erreur plutôt que de faire semblant.
func Rejouer(cmd string, argv []string) (int, error) { return 1, ErrIndisponible }
