// Package ssh lit ~/.ssh/config pour en tirer les serveurs connus, et les propose
// comme catalogue de complétion.
//
// Ce fournisseur n'exécute rien : il implémente pm.Manager et jamais pm.Bindings.
// C'est la décision de l'ADR-0005 — le contrat de complétion n'est pas réservé aux
// gestionnaires de paquets.
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
		ligne := strings.TrimSpace(sc.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		// OpenSSH accepte « Host x », « Host=x » et est insensible à la casse.
		ligne = strings.ReplaceAll(ligne, "=", " ")
		champs := strings.Fields(ligne)
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
		case "hostname":
			for _, n := range courants {
				if vus[n] == "" {
					vus[n] = champs[1]
				}
			}
		case "include":
			for _, motif := range champs[1:] {
				for _, p := range résoudreInclude(motif, base) {
					lireDans(p, vus, visites)
				}
			}
		}
	}
}

// estMotif dit si un mot est un gabarit plutôt qu'un serveur. `Host *` est un bloc de
// défauts, pas une machine — le proposer dans le popup n'aurait aucun sens.
func estMotif(s string) bool {
	return strings.ContainsAny(s, "*?!")
}

// résoudreInclude rend les fichiers désignés. OpenSSH résout un chemin relatif depuis
// ~/.ssh/ pour la configuration utilisateur ; on prend le répertoire du fichier qui
// inclut, ce qui revient au même dans le cas ordinaire et reste juste pour les tests.
func résoudreInclude(motif, base string) []string {
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
