package pm

// Droits est ce que le code de sortie d'un gestionnaire dit des privilèges. C'est un
// **constat**, pas une interception : jigger laisse la commande tourner relayée — invites,
// barres de progression et UAC compris — et ne lit que le code qu'elle rend
// (cf. ADR-0004).
type Droits int

const (
	// DroitsRien : le code ne parle pas de privilèges. C'est le cas de l'immense
	// majorité des échecs, et le seul verdict d'un gestionnaire qui ne sait pas lire ses
	// codes.
	DroitsRien Droits = iota
	// DroitsRequis : la commande exige d'être relancée élevée.
	DroitsRequis
	// DroitsInterdits : la commande exige l'inverse — être relancée SANS élévation.
	// Trois valeurs et non deux, précisément pour ce cas : deux des quatre codes de
	// winget qui parlent de droits disent cela, et les confondre avec DroitsRien ferait
	// taire jigger là où il a le plus à dire. Proposer d'élever y serait nuisible : ce
	// serait refaire, élevé, ce qui vient d'échouer *pour cause d'élévation*.
	DroitsInterdits
)

// Elevateur est le gestionnaire qui sait ce que ses propres codes de sortie disent des
// privilèges. Contrat **optionnel**, comme Bindings : celui qui ne l'implémente pas ne dit
// rien, et jigger ne propose rien. Même modèle de capacités que `cleanup` ou `doctor` —
// ce qui n'est pas déclaré n'existe pas.
type Elevateur interface {
	Droits(code int) Droits
}

// DroitsDe interroge un gestionnaire s'il sait répondre, et rend DroitsRien sinon. Un seul
// endroit fait le test de type : les appelants n'ont pas à connaître le contrat.
func DroitsDe(m Manager, code int) Droits {
	if code == 0 {
		return DroitsRien
	}
	e, ok := m.(Elevateur)
	if !ok {
		return DroitsRien
	}
	return e.Droits(code)
}
