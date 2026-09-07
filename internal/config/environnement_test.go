package config

import "testing"

// neutraliseEnvironnement met hors jeu, pour la durée du test, toutes les variables
// d'environnement que ce paquet déclare.
//
// Sans elle, le verdict d'un test dépend du shell qui lance `go test`. Ce n'est pas une
// hypothèse : `JIGGER_ROWS=8` est un réglage parfaitement normal, présent dans le `.zshrc`
// du poste de développement — et il suffisait à faire échouer
// TestExportNEmetQueLeFichier sur un dépôt sain. Le développeur en tirait la seule
// conclusion possible et la mauvaise : « j'ai cassé l'export ». (#186)
//
// Le paquet a beau tester la PRÉSÉANCE de l'environnement sur le fichier, ses tests ne
// doivent rien devoir à l'environnement réel : c'est celui qu'ils posent eux-mêmes qui
// compte. D'où l'ordre à respecter — neutraliser d'abord, poser sa propre valeur ensuite,
// `t.Setenv` le plus tardif l'emportant.
//
// Le parcours de `Declares` plutôt qu'une liste écrite à la main : le prochain réglage
// ajouté au catalogue sera couvert sans que personne ait à y penser. C'est le seul moyen
// pour qu'une protection de ce genre ne se périme pas en silence.
//
// Chaîne vide et non `os.Unsetenv` : `Resoudre` traite les deux de la même façon (une
// valeur vide n'est pas une valeur), et `t.Setenv` restaure l'état d'origine à la fin du
// test — ce qu'un `Unsetenv` nu ne ferait pas. Incompatible avec `t.Parallel`, qu'aucun
// test de ce paquet n'utilise.
func neutraliseEnvironnement(t *testing.T) {
	t.Helper()
	for _, r := range Declares {
		t.Setenv(r.Env(), "")
	}
}
