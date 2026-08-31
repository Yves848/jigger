// Package ssh lit ~/.ssh/config pour en tirer les serveurs connus, et les propose
// comme catalogue de complétion.
//
// Ce fournisseur n'exécutera rien : il servira de catalogue de complétion en
// implémentant pm.Manager, jamais pm.Bindings. C'est la décision de l'ADR-0005 — le
// contrat de complétion n'est pas réservé aux gestionnaires de paquets.
package ssh

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Hote est un nom déclaré par un bloc Host, avec l'adresse que HostName lui donne.
type Hote struct {
	Nom      string
	HostName string
}

// Lire rend les hôtes déclarés par le fichier et par tout ce qu'il inclut, triés et
// dédoublonnés. Un fichier absent ou illisible rend une liste vide : une machine sans
// configuration SSH n'est pas une erreur.
func Lire(chemin string) []Hote {
	vus := map[string]string{} // nom -> HostName, première valeur gagnante
	lireDans(chemin, vus, map[string]bool{})

	noms := make([]string, 0, len(vus))
	for n := range vus {
		noms = append(noms, n)
	}
	sort.Slice(noms, func(i, j int) bool {
		return strings.ToLower(noms[i]) < strings.ToLower(noms[j])
	})

	out := make([]Hote, 0, len(noms))
	for _, n := range noms {
		out = append(out, Hote{Nom: n, HostName: vus[n]})
	}
	return out
}

// lireDans analyse un fichier et suit ses Include. `visites` porte les chemins déjà
// ouverts : une configuration fautive qui s'inclut elle-même ne doit pas figer le popup
// pendant la frappe.
func lireDans(chemin string, vus map[string]string, visites map[string]bool) {
	abs, err := filepath.Abs(chemin)
	if err != nil || visites[abs] {
		return
	}
	visites[abs] = true

	f, err := os.Open(abs)
	if err != nil {
		return
	}
	defer f.Close()

	base := filepath.Dir(abs)
	var courants []string // motifs du bloc Host en cours

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// OpenSSH accepte « Host x », « Host=x » et est insensible à la casse.
		ligne := strings.ReplaceAll(strings.TrimSpace(sc.Text()), "=", " ")
		champs := champsUtiles(ligne)
		if len(champs) < 2 {
			continue
		}
		switch strings.ToLower(champs[0]) {
		case "host":
			courants = nil
			for _, motif := range champs[1:] {
				if estMotif(motif) {
					continue
				}
				courants = append(courants, motif)
				if _, déjà := vus[motif]; !déjà {
					vus[motif] = ""
				}
			}
		case "match":
			// Un bloc Match ferme le bloc Host précédent : ses mots-clés ne s'appliquent
			// à aucun des motifs de celui-ci. Sans cette remise à zéro, « Host web »
			// suivi de « Match host *.interne / HostName bastion.example.com » affichait
			// l'adresse du bastion en regard de web, alors que `ssh -G web` répond
			// « hostname web ». L'asymétrie était le défaut : `case "host"` remettait
			// bien `courants` à nil, `match` non.
			//
			// jigger n'évalue PAS les conditions du Match — réimplémenter la logique
			// d'OpenSSH (host, exec, final, canonical…) n'est pas son objet. Il lui
			// suffit de cesser d'attribuer à tort : les mots-clés d'un Match ne
			// concernent aucun candidat du popup.
			courants = nil
		case "hostname":
			for _, n := range courants {
				if vus[n] == "" {
					vus[n] = champs[1]
				}
			}
		case "include":
			for _, motif := range champs[1:] {
				for _, p := range fichiersInclus(motif, base) {
					lireDans(p, vus, visites)
				}
			}
		}
	}
}

// champsUtiles découpe une ligne en mots, amputée de son commentaire de fin de ligne.
//
// OpenSSH ne voit un commentaire que là où le « # » OUVRE un mot : « Host pve  # le
// proxmox du salon » ne déclare que le motif « pve », tandis que « Host a#b » déclare
// bien le motif « a#b » — les deux vérifiés à l'`ssh -G` de la machine (OpenSSH 10.2p1).
// Sans cette coupe, chaque mot du commentaire devenait un candidat du popup, porteur de
// l'adresse du bloc : ⇥ posait alors un nom qui ne résoudrait jamais.
//
// Le cas du commentaire en pleine ligne (« # ceci est un titre ») passe par le même
// chemin : les mots restants sont vides, et l'appelant écarte toute ligne de moins de
// deux mots.
func champsUtiles(ligne string) []string {
	champs := strings.Fields(ligne)
	for i, c := range champs {
		if strings.HasPrefix(c, "#") {
			return champs[:i]
		}
	}
	return champs
}

// estMotif dit si un mot est un gabarit plutôt qu'un serveur. `Host *` est un bloc de
// défauts, pas une machine — le proposer dans le popup n'aurait aucun sens.
func estMotif(s string) bool {
	return strings.ContainsAny(s, "*?!")
}

// fichiersInclus rend les fichiers désignés par un motif Include. OpenSSH résout un
// chemin relatif depuis ~/.ssh/ pour la configuration utilisateur ; on prend le
// répertoire du fichier qui inclut, ce qui revient au même dans le cas ordinaire et
// reste juste pour les tests.
func fichiersInclus(motif, base string) []string {
	if strings.HasPrefix(motif, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			motif = filepath.Join(home, motif[2:])
		}
	}
	if !filepath.IsAbs(motif) {
		motif = filepath.Join(base, motif)
	}
	if trouvés, err := filepath.Glob(motif); err == nil && len(trouvés) > 0 {
		return trouvés
	}
	return []string{motif}
}
