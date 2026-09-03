package ssh

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Manager implémente pm.Manager pour ssh, scp et sftp.
//
// Il n'implémente PAS pm.Bindings, et c'est le sujet : ces trois commandes n'ont pas de
// verbes à exécuter, jigger se contente de compléter la ligne que l'utilisateur lancera
// lui-même. Voir l'ADR-0005.
type Manager struct{ cmd string }

// New rend le fournisseur pour l'une des trois commandes.
func New(cmd string) Manager { return Manager{cmd: cmd} }

func (m Manager) Cmd() string { return m.cmd }

// Aucune sous-commande : c'est ce vide qui fait proposer le catalogue dès le premier
// mot, par la règle générale de completeWith (ADR-0005). `ssh archlight` n'a pas de
// verbe entre la commande et l'hôte.
func (Manager) Subcommands() []string { return nil }

// Aucune option proposée. La spec l'écarte explicitement : -p, -i, -L et les autres se
// tapent rarement à la main, et les proposer allongerait la liste sans la servir.
func (Manager) Options(string) []string { return nil }

// Sans objet : un hôte n'est ni installé ni absent.
func (Manager) InstalledOnly(string) bool { return false }

// Available dit si la machine a une configuration SSH. Sur une machine qui n'en a pas,
// le fournisseur se tait plutôt que de proposer une liste vide.
func (Manager) Available() bool { return disponible(cheminConfig()) }

// Load relit ~/.ssh/config à chaque appel. Pas de mémoïsation : jigger render tourne
// dans un processus par frappe (cf. shell/jigger.plugin.zsh), donc Load() n'est jamais
// appelée deux fois dans le même processus — un cache de paquet ne survivrait à aucune
// frappe et ne protégerait rien. La spec (§4, « Pas de réchauffement ») l'annonce déjà :
// lire ces quelques fragments coûte une milliseconde.
func (Manager) Load() *pm.Catalog { return catalogueDe(cheminConfig()) }

// Insert colle le deux-points qu'attend scp. `scp fichier archlight /tmp` copierait vers
// un fichier LOCAL nommé archlight, en écrasant peut-être quelque chose — l'erreur est
// silencieuse, d'où la correction ici.
func (m Manager) Insert(_ *pm.Catalog, _, _, name string) string {
	if m.cmd == "scp" && !strings.HasSuffix(name, ":") {
		return name + ":"
	}
	return name
}

// Warm ne fait rien. Lire quelques fragments de configuration coûte une milliseconde :
// il n'y a ni sortie machine à analyser, ni service distant à interroger, ni cache de
// 24 h à tenir. scoop n'a pas plus à réchauffer, pour une autre raison : son catalogue
// vit déjà sur le disque (internal/scoop/scoop.go).
func (Manager) Warm(pm.Scope) error { return nil }

func cheminConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

func disponible(chemin string) bool {
	if chemin == "" {
		return false
	}
	st, err := os.Stat(chemin)
	return err == nil && !st.IsDir()
}

// catalogueDe construit le catalogue d'un fichier donné. Séparé de Load() pour être
// testable sans toucher au ~/.ssh/config réel de la machine.
func catalogueDe(chemin string) *pm.Catalog {
	cat := pm.NewCatalog()
	for _, h := range Lire(chemin) {
		// Badge vide : glyphe() rend alors la puce « • », qui dit « n'appartient à
		// aucune des deux classes de paquets ». C'est exactement le cas d'un hôte.
		cat.Add(h.Nom, "")
		if h.HostName != "" && h.HostName != h.Nom {
			// L'adresse voyage dans Versions parce que c'est le SEUL champ rendu en
			// texte libre à droite de la ligne (internal/ui/frame.go). Le nom du champ
			// ment ; le renommer toucherait les trois gestionnaires, l'UI et les tests
			// de rendu pour un gain de vocabulaire. La spec du 30 août l'assume.
			cat.Versions[h.Nom] = h.HostName
		}
	}
	cat.Sort()
	repousserLesAdresses(cat)
	return cat
}

// repousserLesAdresses pousse après les noms les motifs qui sont eux-mêmes des adresses.
// Un fragment généré peut écrire « Host archlight aquarium 192.168.50.207 » pour que
// `ssh 192.168.50.207` profite du même bloc que `ssh archlight` (cf. le générateur du
// dépôt config, tools/reseau) : le parseur retient les trois motifs, comme la spec le
// demande (§4). Sans ce tri, cat.Sort() range les adresses en tête — les chiffres
// précèdent les lettres — et le popup s'ouvrait sur une poignée d'adresses avant le
// premier nom lisible, qui est ce qu'un humain cherche en général. L'adresse reste un
// candidat à part entière : filtrer sur « 192. » la retrouve toujours, elle ne passe
// simplement plus devant à l'ouverture.
//
// Entre elles, les adresses reviennent à un ordre numérique plutôt qu'alphabétique :
// cat.Sort() placerait 192.168.50.10 avant 192.168.50.8, ce qui saute aux yeux dès qu'on
// parcourt la liste. net.ParseIP().To16() rend une représentation à largeur fixe,
// comparable octet à octet — IPv4 et IPv6 compris, sans expression régulière à tenir.
func repousserLesAdresses(cat *pm.Catalog) {
	noms := make([]string, 0, len(cat.Names))
	adresses := make([]string, 0)
	for _, n := range cat.Names {
		if net.ParseIP(n) != nil {
			adresses = append(adresses, n)
		} else {
			noms = append(noms, n)
		}
	}
	if len(adresses) == 0 {
		return
	}
	sort.Slice(adresses, func(i, j int) bool {
		a, b := net.ParseIP(adresses[i]).To16(), net.ParseIP(adresses[j]).To16()
		return bytes.Compare(a, b) < 0
	})
	cat.Names = append(noms, adresses...)
}
