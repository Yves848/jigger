package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

// ecrire pose un fichier et rend son chemin.
func ecrire(t *testing.T, dir, nom, contenu string) string {
	t.Helper()
	p := filepath.Join(dir, nom)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func noms(hotes []Hote) []string {
	out := make([]string, 0, len(hotes))
	for _, h := range hotes {
		out = append(out, h.Nom)
	}
	return out
}

func egal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLireNomsEtHostName(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host pve
    HostName 192.168.50.8
    User root

Host archlight
    HostName 192.168.50.207
`)
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"archlight", "pve"})
	if hotes[1].HostName != "192.168.50.8" {
		t.Errorf("HostName de pve = %q, attendu 192.168.50.8", hotes[1].HostName)
	}
}

func TestLireUnBlocAPlusieursMotifs(t *testing.T) {
	// « Host archlight aquarium 192.168.50.207 » declare trois facons valides de
	// designer la meme machine : les trois sont des candidats.
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host archlight aquarium 192.168.50.207\n    HostName 192.168.50.207\n")
	egal(t, noms(Lire(p)), []string{"192.168.50.207", "aquarium", "archlight"})
}

func TestLireEcarteLesMotifs(t *testing.T) {
	// `Host *` n'est pas un serveur : c'est un bloc de defauts.
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host *
    AddKeysToAgent yes

Host web-?
    User deploy

Host !prod
    User dev

Host reel
    HostName 10.0.0.1
`)
	egal(t, noms(Lire(p)), []string{"reel"})
}

func TestLireSuitUnInclude(t *testing.T) {
	d := t.TempDir()
	ecrire(t, d, "config.d/homelab.conf", "Host archlight\n    HostName 192.168.50.207\n")
	p := ecrire(t, d, "config", "Include config.d/homelab.conf\n\nHost pve\n    HostName 192.168.50.8\n")
	egal(t, noms(Lire(p)), []string{"archlight", "pve"})
}

func TestLireResoutUnIncludeGlob(t *testing.T) {
	// OpenSSH accepte les jokers dans Include ; une configuration en fragments s'en sert.
	d := t.TempDir()
	ecrire(t, d, "config.d/a.conf", "Host aaa\n")
	ecrire(t, d, "config.d/b.conf", "Host bbb\n")
	p := ecrire(t, d, "config", "Include config.d/*.conf\n")
	egal(t, noms(Lire(p)), []string{"aaa", "bbb"})
}

func TestLireNeBouclePasSurUnIncludeCirculaire(t *testing.T) {
	// Une configuration fautive ne doit pas figer le popup pendant la frappe.
	d := t.TempDir()
	ecrire(t, d, "b.conf", "Include a.conf\nHost dansB\n")
	p := ecrire(t, d, "a.conf", "Include b.conf\nHost dansA\n")
	egal(t, noms(Lire(p)), []string{"dansA", "dansB"})
}

func TestLireIgnoreLaCasseDesMotsCles(t *testing.T) {
	// OpenSSH est insensible a la casse sur ses mots-cles.
	d := t.TempDir()
	p := ecrire(t, d, "config", "HOST pve\n    hostname 192.168.50.8\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"pve"})
	if hotes[0].HostName != "192.168.50.8" {
		t.Errorf("HostName = %q", hotes[0].HostName)
	}
}

func TestLireIgnoreCommentairesEtLignesVides(t *testing.T) {
	d := t.TempDir()
	p := ecrire(t, d, "config", "# un commentaire\n\n   # indente\nHost pve\n")
	egal(t, noms(Lire(p)), []string{"pve"})
}

func TestLireDedoublonne(t *testing.T) {
	// Le meme nom declare deux fois (fragment + fichier principal) ne doit sortir
	// qu'une fois : le popup afficherait sinon deux lignes identiques.
	d := t.TempDir()
	ecrire(t, d, "f.conf", "Host pve\n    HostName 10.0.0.1\n")
	p := ecrire(t, d, "config", "Include f.conf\nHost pve\n    HostName 192.168.50.8\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"pve"})
	// La premiere valeur rencontree gagne, comme le fait OpenSSH lui-meme.
	if hotes[0].HostName != "10.0.0.1" {
		t.Errorf("HostName = %q, attendu celui du fragment inclus en premier", hotes[0].HostName)
	}
}

func TestLireUnFichierAbsentRendVide(t *testing.T) {
	// Une machine sans configuration SSH n'est pas une erreur.
	if got := Lire(filepath.Join(t.TempDir(), "inexistant")); len(got) != 0 {
		t.Errorf("got %v, attendu vide", got)
	}
}

func TestLireHostSansHostName(t *testing.T) {
	// Un bloc peut n'avoir que son nom : c'est valide, HostName reste alors vide,
	// et c'est au consommateur de décider quoi en faire.
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host solo\n    User root\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"solo"})
	if hotes[0].HostName != "" {
		t.Errorf("HostName = %q, attendu vide", hotes[0].HostName)
	}
}

func TestLireCoupeUnCommentaireDeFinDeLigne(t *testing.T) {
	// « Host pve  # le proxmox du salon » ne déclare qu'un motif : OpenSSH traite le
	// reste comme un commentaire (vérifié à l'`ssh -G`, qui répond « hostname salon »
	// pour `salon`, c'est-à-dire qu'aucun bloc ne correspond). Sans la coupe, le popup
	// proposait « # », « du », « le », « proxmox » et « salon », tous porteurs de
	// l'adresse du bloc.
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host pve  # le proxmox du salon\n    HostName 192.168.50.8\n")
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"pve"})
	if hotes[0].HostName != "192.168.50.8" {
		t.Errorf("HostName = %q, attendu 192.168.50.8", hotes[0].HostName)
	}
}

func TestLireGardeUnDieseAuMilieuDUnMot(t *testing.T) {
	// Le « # » n'ouvre un commentaire que là où il ouvre un mot. « Host a#b c » déclare
	// bien les deux motifs — mesuré à l'`ssh -G` : `a#b` et `c` tombent tous deux sur le
	// bloc. Couper au premier « # » rencontré, où qu'il soit, perdrait « c ».
	d := t.TempDir()
	p := ecrire(t, d, "config", "Host a#b c\n    HostName 10.0.0.1\n")
	egal(t, noms(Lire(p)), []string{"a#b", "c"})
}

func TestLireUnMatchNAttribuePasSonHostNameAuBlocPrecedent(t *testing.T) {
	// Un Match ferme le bloc Host précédent. `ssh -G web` répond « hostname web » : le
	// HostName du Match ne concerne pas web, et l'afficher en regard annoncerait une
	// machine pour une autre — précisément l'information que le popup a ajoutée pour
	// aider à choisir.
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host web
    User deploy

Match host *.interne
    HostName bastion.example.com
`)
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"web"})
	if hotes[0].HostName != "" {
		t.Errorf("HostName de web = %q, attendu vide : il vient d'un bloc Match", hotes[0].HostName)
	}
}

func TestLireReprendApresUnMatch(t *testing.T) {
	// Le Match n'éteint pas le parseur : le bloc Host qui le suit retrouve ses motifs et
	// son adresse.
	d := t.TempDir()
	p := ecrire(t, d, "config", `
Host web
    HostName 10.0.0.1

Match final host *
    ProxyJump bastion

Host db
    HostName 10.0.0.2
`)
	hotes := Lire(p)
	egal(t, noms(hotes), []string{"db", "web"})
	if hotes[0].HostName != "10.0.0.2" || hotes[1].HostName != "10.0.0.1" {
		t.Errorf("adresses = %q / %q, attendu 10.0.0.2 / 10.0.0.1", hotes[0].HostName, hotes[1].HostName)
	}
}
