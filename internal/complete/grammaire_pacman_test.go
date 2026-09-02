package complete

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pacman"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// pacman est le seul fournisseur dont les « sous-commandes » commencent par un tiret. Le
// moteur route sur `strings.HasPrefix(word, "-")` : ces tests gardent le fait que les deux
// chemins — celui des sous-commandes et celui des options — mènent au même endroit tant
// qu'aucune opération n'est posée, et divergent ensuite.

func catPacman() *pm.Catalog {
	return pacman.NewCatalog(
		[]string{"core acl 2.4.0-1", "extra firefox 142.0-1", "extra fd 10.2.0-1"},
		[]string{"firefox-nightly"},
		[]string{"acl 2.4.0-1"},
	)
}

func contient(liste []string, mot string) bool {
	for _, m := range liste {
		if m == mot {
			return true
		}
	}
	return false
}

// « pacman ⇥ » : le mot est vide, donc le moteur passe par les sous-commandes.
func TestPacmanPremierMotDonneLesOperations(t *testing.T) {
	res := CompleteWith("pacman ", pacman.New("pacman"), catPacman())
	if !contient(names(res.Items), "-S") {
		t.Fatalf("« pacman ⇥ » devait proposer -S, obtenu %v", names(res.Items))
	}
}

// « pacman -⇥ » : le mot commence par un tiret, donc le moteur passe par les OPTIONS —
// mais ce que l'utilisateur tape est bien une opération. C'est le cas que le fournisseur
// règle en déclarant la même liste des deux côtés.
func TestPacmanTiretDonneAussiLesOperations(t *testing.T) {
	res := CompleteWith("pacman -", pacman.New("pacman"), catPacman())
	if !contient(names(res.Items), "-S") {
		t.Fatalf("« pacman -⇥ » devait proposer -S, obtenu %v", names(res.Items))
	}
}

// Le moteur minuscule le mot courant. Sans le repli symétrique côté sous-commandes,
// « -S » ne se retrouverait jamais derrière un « -s ».
func TestPacmanFiltreSansEgardALaCasse(t *testing.T) {
	res := CompleteWith("pacman -S", pacman.New("pacman"), catPacman())
	if !contient(names(res.Items), "-S") {
		t.Fatalf("« pacman -S » devait se proposer lui-même, obtenu %v", names(res.Items))
	}
	if !contient(names(res.Items), "-Syu") {
		t.Fatalf("« pacman -S » devait proposer -Syu, obtenu %v", names(res.Items))
	}
}

// Une opération posée, le catalogue vient.
func TestPacmanApresOperationDonneLesPaquets(t *testing.T) {
	res := CompleteWith("pacman -S f", pacman.New("pacman"), catPacman())
	got := names(res.Items)
	if !contient(got, "fd") || !contient(got, "firefox") {
		t.Fatalf("attendu fd et firefox, obtenu %v", got)
	}
	if res.Sub != "-s" {
		t.Errorf("sous-commande retenue %q, attendue « -s » (minusculée par le moteur)", res.Sub)
	}
}

// Une opération de la famille -R ne puise que dans les installés.
func TestPacmanRetraitNeProposeQueLesInstalles(t *testing.T) {
	res := CompleteWith("pacman -Rns ", pacman.New("pacman"), catPacman())
	got := names(res.Items)
	if !contient(got, "acl") {
		t.Fatalf("acl est installé et devait être proposé, obtenu %v", got)
	}
	if contient(got, "firefox") {
		t.Fatalf("firefox n'est pas installé et ne devait pas être proposé, obtenu %v", got)
	}
}

// Une opération posée, un tiret rend les drapeaux secondaires — plus les opérations.
func TestPacmanDrapeauxSecondaires(t *testing.T) {
	res := CompleteWith("pacman -S --n", pacman.New("pacman"), catPacman())
	got := names(res.Items)
	if !contient(got, "--needed") || !contient(got, "--nodeps") {
		t.Fatalf("attendu --needed et --nodeps, obtenu %v", got)
	}
}

// yay complète l'AUR, pacman non : c'est la seule différence visible dans le popup.
func TestYayCompleteLAUR(t *testing.T) {
	res := CompleteWith("yay -S firefox-n", pacman.New("yay"), catPacman())
	if !contient(names(res.Items), "firefox-nightly") {
		t.Fatalf("yay devait proposer le paquet AUR, obtenu %v", names(res.Items))
	}
	if res.Items[0].Badge != pm.BadgeAUR {
		t.Errorf("badge %q, attendu %q", res.Items[0].Badge, pm.BadgeAUR)
	}
}

// Le cadre annonce la ligne qu'on écrit, pas la clé minusculée avec laquelle jigger
// interroge ses tables : « pacman -Rns » est une commande, « pacman -rns » n'en est pas
// une.
func TestTitreGardeLaCasseTapee(t *testing.T) {
	res := CompleteWith("pacman -Rns fi", pacman.New("pacman"), catPacman())
	if got := res.Title(); got != "pacman -Rns" {
		t.Errorf("titre %q, attendu « pacman -Rns »", got)
	}
	// Les gestionnaires à verbes minuscules ne changent pas de comportement.
	if got := brewComplete("brew install f", testCatalog()).Title(); got != "brew install" {
		t.Errorf("titre brew %q, attendu « brew install »", got)
	}
}
