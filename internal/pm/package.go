package pm

// Package est une ligne de sortie normalisée : ce que jigger sait dire d'un paquet, quel
// que soit le gestionnaire qui l'a produit. Un seul type sert les quatre verbes
// normalisés — list, outdated, search et source.
type Package struct {
	Name      string `json:"name"`      // identifiant natif : « fd », « Git.Git »
	Version   string `json:"version"`   // version installée ; vide si non installé
	Available string `json:"available"` // version disponible ; vide si à jour ou inconnue
	Kind      string `json:"kind"`      // badge Badge* — le popup l'affiche déjà
	Source    string `json:"source"`    // provenance fine : « main », « extras », « homebrew/core »
	PM        string `json:"pm"`        // « brew », « winget », « scoop »
}
