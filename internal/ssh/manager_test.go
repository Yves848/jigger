package ssh

import (
	"path/filepath"
	"strings"
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

func TestManagerImplementePmManager(t *testing.T) {
	// La preuve tient dans l'affectation : si l'interface n'est pas satisfaite, ça ne
	// compile pas.
	var _ pm.Manager = New("ssh")
}

func TestPasDeSousCommandeNiOption(t *testing.T) {
	// C'est ce vide qui declenche la regle generale de completeWith (ADR-0005).
	m := New("ssh")
	if got := m.Subcommands(); len(got) != 0 {
		t.Errorf("Subcommands() = %v, attendu vide", got)
	}
	if got := m.Options(""); len(got) != 0 {
		t.Errorf("Options() = %v, attendu vide", got)
	}
	if m.InstalledOnly("") {
		t.Error("InstalledOnly() = true, attendu false")
	}
	if err := m.Warm(pm.ScopeAll); err != nil {
		t.Errorf("Warm() = %v, attendu nil", err)
	}
}

func TestCmdRendLeMotDemande(t *testing.T) {
	for _, c := range []string{"ssh", "scp", "sftp"} {
		if got := New(c).Cmd(); got != c {
			t.Errorf("Cmd() = %q, attendu %q", got, c)
		}
	}
}

func TestCatalogueDepuisUnFichier(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host pve\n    HostName 192.168.50.8\n\nHost solo\n")
	cat := catalogueDe(p)

	egal(t, cat.Names, []string{"pve", "solo"})
	// L'adresse voyage dans Versions : c'est le seul champ rendu en texte libre a
	// droite de la ligne. Le nom du champ ment, la spec dit pourquoi.
	if got := cat.Version("pve"); got != "192.168.50.8" {
		t.Errorf("Version(pve) = %q, attendu 192.168.50.8", got)
	}
	// Un hote sans HostName n'affiche rien a droite plutot que de repeter son nom.
	if got := cat.Version("solo"); got != "" {
		t.Errorf("Version(solo) = %q, attendu vide", got)
	}
	// Aucun badge : le glyphe par defaut est la puce, qui convient — un hote
	// n'appartient a aucune des deux classes de paquets.
	if got := cat.Badge("pve"); got != "" {
		t.Errorf("Badge(pve) = %q, attendu vide", got)
	}
	// MarkInstalled poserait Installed[nom] = true sans toucher ni Badge ni
	// Version : la substitution passerait toutes les autres assertions, et
	// chaque hote gagnerait une pastille. Un serveur n'est ni installe ni
	// absent — cette garde est la pour ca.
	if len(cat.Installed) != 0 {
		t.Errorf("Installed = %v, attendu vide", cat.Installed)
	}
}

func TestLesNomsPassentAvantLesAdresses(t *testing.T) {
	d := t.TempDir()
	// Trois motifs sur la meme ligne Host, comme le genere le fragment reseau du
	// depot config : le nom, un alias, et l'adresse elle-meme en motif.
	p := ecrire(t, d, "config", "Host zzz archlight 192.168.50.207\n")
	cat := catalogueDe(p)

	// Sans repousserLesAdresses, cat.Sort() (alphabetique) placerait l'adresse en
	// tete : les chiffres precedent les lettres. « archlight » et « zzz » d'abord,
	// dans leur ordre alphabetique habituel ; l'adresse ensuite.
	egal(t, cat.Names, []string{"archlight", "zzz", "192.168.50.207"})
}

func TestLesAdressesSontTrieesNumeriquement(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host a 192.168.50.10\nHost b 192.168.50.8\n")
	cat := catalogueDe(p)

	// Un tri lexicographique placerait .10 avant .8 (« 1 » < « 8 »). Ce test rate si
	// le tri numerique est absent ou inverse.
	egal(t, cat.Names, []string{"a", "b", "192.168.50.8", "192.168.50.10"})
}

func TestUneAdresseResteUnCandidat(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host archlight 192.168.50.207\n")
	cat := catalogueDe(p)

	for _, n := range cat.Names {
		if strings.HasPrefix(n, "192.") {
			return
		}
	}
	t.Errorf("aucune adresse dans %v, attendu 192.168.50.207 conservee comme candidat", cat.Names)
}

func TestInsertColleUnDeuxPointsPourScp(t *testing.T) {
	// « scp fichier archlight /tmp » copierait vers un FICHIER LOCAL nomme archlight,
	// en ecrasant peut-etre quelque chose. Le deux-points fait partie du candidat.
	cat := pm.NewCatalog()
	if got := New("scp").Insert(cat, "", "", "archlight"); got != "archlight:" {
		t.Errorf("scp Insert = %q, attendu archlight:", got)
	}
	for _, c := range []string{"ssh", "sftp"} {
		if got := New(c).Insert(cat, "", "", "archlight"); got != "archlight" {
			t.Errorf("%s Insert = %q, attendu archlight", c, got)
		}
	}
}

func TestInsertNeDoublePasLeDeuxPoints(t *testing.T) {
	// L'utilisateur a deja tape « archlight: » et complete le chemin : ne pas en
	// remettre un.
	cat := pm.NewCatalog()
	if got := New("scp").Insert(cat, "", "", "archlight:"); got != "archlight:" {
		t.Errorf("Insert = %q, attendu archlight:", got)
	}
}

func TestAvailableSuitLExistenceDuFichier(t *testing.T) {
	d := t.TempDir()
	absent := filepath.Join(d, "rien")
	if disponible(absent) {
		t.Error("disponible() = true sur un fichier absent")
	}
	présent := ecrire(t, d, "config", "Host x\n")
	if !disponible(présent) {
		t.Error("disponible() = false sur un fichier présent")
	}
}
