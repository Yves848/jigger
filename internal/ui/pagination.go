package ui

// Contexte rassemble tout ce dont la décision d'armer la vue paginée dépend. La
// décision est ainsi une fonction pure : elle se teste sans terminal, sur les cas qui
// comptent — tube, JSON, contenu court, JIGGER_PAGER=0, --select (spec §4).
type Contexte struct {
	EstTTY   bool   // la sortie standard est-elle un terminal ?
	Hauteur  int    // hauteur de l'écran en lignes (0 = inconnue)
	NbLignes int    // lignes de données à afficher, en-tête non compris
	EnJSON   bool   // --json
	Force    bool   // --select
	Pager    string // valeur de JIGGER_PAGER ("" = non posée)
}

// chromeTableau est ce que la vue consomme en plus des données : titre, filtre,
// en-tête de colonnes, pied. En dessous de ce seuil, la table brute suffit.
const chromeTableau = tableauChrome

// DoitPaginer dit si la vue paginée doit s'armer.
//
// L'ordre des règles est l'ordre des priorités, et il se lit comme un engagement :
//
//  1. --json ne se pagine jamais : c'est un contrat machine, et rien ne le rompt.
//  2. Sans terminal — tube, script, CI — jamais rien d'interactif. Un `jg list | grep`
//     qui attendrait une touche casserait tous les scripts.
//  3. --select est explicite : il l'emporte sur JIGGER_PAGER=0, qui ne règle que le
//     comportement automatique.
//  4. JIGGER_PAGER=0 désarme, comme JIGGER_LIVE=0 désarme le popup.
//  5. Un contenu qui tient à l'écran s'imprime tel quel : ouvrir un plein écran pour
//     six lignes est une nuisance, pas un service.
func DoitPaginer(c Contexte) bool {
	if c.EnJSON {
		return false
	}
	if !c.EstTTY {
		return false
	}
	if c.Force {
		return true
	}
	if c.Pager == "0" {
		return false
	}
	if c.NbLignes == 0 {
		return false
	}
	// Hauteur inconnue : on ne devine pas, on pagine. Le terminal existe (EstTTY), donc
	// la vue est utilisable ; et un contenu long sans pagination serait le pire des deux.
	if c.Hauteur <= 0 {
		return true
	}
	return c.NbLignes+chromeTableau > c.Hauteur
}
