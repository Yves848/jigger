package pm

import "testing"

// muet : un gestionnaire qui n'implémente pas Elevateur. C'est le cas ordinaire — brew et
// scoop ne publient aucun code de sortie qui parle de privilèges.
type muet struct{ Manager }

// bavard : un gestionnaire qui sait lire ses codes.
type bavard struct{ Manager }

func (bavard) Droits(code int) Droits {
	if code == 42 {
		return DroitsRequis
	}
	return DroitsRien
}

func TestDroitsDeSeTaitSansContrat(t *testing.T) {
	if got := DroitsDe(muet{}, 42); got != DroitsRien {
		t.Fatalf("%v : un gestionnaire sans le contrat ne doit rien dire", got)
	}
}

func TestDroitsDeInterrogeQuiSait(t *testing.T) {
	if got := DroitsDe(bavard{}, 42); got != DroitsRequis {
		t.Fatalf("%v, attendu DroitsRequis", got)
	}
	if got := DroitsDe(bavard{}, 7); got != DroitsRien {
		t.Fatalf("%v : un autre code ne dit rien des droits", got)
	}
}

// Le succès ne se demande à personne. Sans ce court-circuit, un gestionnaire dont la table
// contiendrait 0 par inadvertance ferait proposer une élévation après une commande qui a
// parfaitement marché.
func TestDroitsDeIgnoreLeSucces(t *testing.T) {
	if got := DroitsDe(bavard{}, 0); got != DroitsRien {
		t.Fatalf("%v : le code 0 ne doit jamais rien déclencher", got)
	}
}
