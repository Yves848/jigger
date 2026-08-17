package config

import (
	"os"
	"sync"
	"time"
)

// cacheLu évite de relire le fichier à chaque appel : un gestionnaire demande sa durée
// au chargement de son catalogue, ce qui peut arriver plusieurs fois dans un processus,
// et le fichier ne change pas en cours de route.
var (
	uneFois sync.Once
	fichier *Fichier
)

func chargerUneFois() *Fichier {
	uneFois.Do(func() {
		f, err := Charger()
		if err != nil {
			f = Nouveau()
		}
		fichier = f
	})
	return fichier
}

// Duree rend la valeur d'un réglage de type durée, ou le défaut donné si la clé n'est pas
// réglée ou si sa valeur ne s'analyse pas.
//
// Une valeur illisible ne fait pas échouer : elle est ignorée, et le défaut s'applique.
// Un catalogue qui refuserait de se charger parce qu'une durée est mal écrite serait une
// punition disproportionnée pour une faute de frappe — d'autant que la valeur passe par
// le chemin de la frappe.
func Duree(cle string, defaut time.Duration) time.Duration {
	r, ok := Trouver(cle)
	if !ok {
		return defaut
	}
	brut, _ := Resoudre(os.Getenv(r.Env()), chargerUneFois().Valeur(cle), "")
	if brut == "" {
		return defaut
	}
	d, err := time.ParseDuration(brut)
	if err != nil || d < 0 {
		return defaut
	}
	return d
}

// Recharger vide le cache de lecture. Les tests s'en servent ; le binaire, jamais — il ne
// vit que le temps d'une commande.
func Recharger() {
	uneFois = sync.Once{}
	fichier = nil
}
