package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

func (Manager) Load() *pm.Catalog { return catalogue() }

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
// 24 h à tenir. C'est le seul fournisseur de jigger dans ce cas.
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

// Le catalogue est mémorisé et réévalué quand le fichier change. Load() est sur le
// chemin du rendu : le relire à chaque frappe serait gaspiller, et le figer pour la
// session ferait mentir le popup après un `reseau-outil rendre`.
var (
	mémo      *pm.Catalog
	mémoQuand time.Time
	mémoMu    sync.Mutex
)

func catalogue() *pm.Catalog {
	chemin := cheminConfig()
	mémoMu.Lock()
	defer mémoMu.Unlock()

	st, err := os.Stat(chemin)
	if err != nil {
		return pm.NewCatalog()
	}
	if mémo != nil && st.ModTime().Equal(mémoQuand) {
		return mémo
	}
	mémo, mémoQuand = catalogueDe(chemin), st.ModTime()
	return mémo
}

// catalogueDe construit le catalogue d'un fichier donné. Séparé de catalogue() pour être
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
	return cat
}
