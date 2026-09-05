package pm

// Viviers est le gestionnaire qui sait rendre ses candidats **verbe par verbe**, plutôt que
// depuis l'unique catalogue que Load() construit. Contrat **optionnel**, comme Bindings,
// Elevateur ou Exhaustif : celui qui ne l'implémente pas garde le comportement d'avant.
//
// Un gestionnaire de paquets n'en a pas besoin : `install` et `uninstall` puisent dans le
// même catalogue, à la moitié près, et c'est ce que dit InstalledOnly. Un **helper de
// commande** en a besoin partout : les branches derrière `checkout`, les fichiers modifiés
// derrière `add`, les distants derrière `push` n'ont rien à voir entre eux, et rien à voir
// avec un catalogue (ADR-0009).
type Viviers interface {
	// Candidats rend le vivier de ce verbe, et false si le verbe n'en déclare pas.
	//
	// Le booléen distingue « pas de vivier propre » — la complétion retombe alors sur le
	// catalogue — de « vivier vide », qui doit afficher zéro candidat. Les confondre
	// ferait proposer tout le catalogue derrière un verbe dont le vivier est
	// légitimement vide.
	Candidats(sub string) (*Catalog, bool)
}

// CandidatsDe interroge un gestionnaire s'il sait répondre, et rend false sinon. Un seul
// endroit fait le test de type : les appelants n'ont pas à connaître le contrat.
func CandidatsDe(m Manager, sub string) (*Catalog, bool) {
	v, ok := m.(Viviers)
	if !ok {
		return nil, false
	}
	return v.Candidats(sub)
}
