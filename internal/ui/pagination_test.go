package ui

import "testing"

// Le tableau des cas qui comptent. Chacun correspond à une ligne de la spec §2 : si
// l'un d'eux change, c'est un changement de contrat, pas un détail d'implémentation.
func TestDoitPaginer(t *testing.T) {
	cas := []struct {
		nom    string
		ctx    Contexte
		attend bool
	}{
		{"contenu long sur un terminal",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 200}, true},

		{"contenu qui tient à l'écran",
			Contexte{EstTTY: true, Hauteur: 40, NbLignes: 6}, false},

		{"contenu juste au ras de l'écran",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 24 - chromeTableau}, false},

		{"une ligne de trop",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 24 - chromeTableau + 1}, true},

		// Le cas qui protège tous les scripts du monde.
		{"sortie redirigée",
			Contexte{EstTTY: false, Hauteur: 24, NbLignes: 500}, false},

		{"--json ne se pagine jamais",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 500, EnJSON: true}, false},

		// --json l'emporte même sur --select : un contrat machine ne se négocie pas.
		{"--json l'emporte sur --select",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 500, EnJSON: true, Force: true}, false},

		{"JIGGER_PAGER=0 désarme",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 500, Pager: "0"}, false},

		// --select est un geste explicite : il l'emporte sur un réglage d'ambiance.
		{"--select l'emporte sur JIGGER_PAGER=0",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 500, Pager: "0", Force: true}, true},

		{"--select force même un contenu court",
			Contexte{EstTTY: true, Hauteur: 40, NbLignes: 3, Force: true}, true},

		// Sans terminal, --select n'a nulle part où dessiner : main doit le refuser.
		{"--select sans terminal",
			Contexte{EstTTY: false, NbLignes: 500, Force: true}, false},

		{"aucune ligne à montrer",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 0}, false},

		{"hauteur inconnue : on pagine plutôt que de deviner",
			Contexte{EstTTY: true, Hauteur: 0, NbLignes: 500}, true},

		{"JIGGER_PAGER à autre chose que 0 ne désarme pas",
			Contexte{EstTTY: true, Hauteur: 24, NbLignes: 500, Pager: "1"}, true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := DoitPaginer(c.ctx); got != c.attend {
				t.Fatalf("DoitPaginer = %v, attendu %v", got, c.attend)
			}
		})
	}
}
