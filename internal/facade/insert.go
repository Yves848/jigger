package facade

import "unicode"

// splitInsert découpe le texte que rend pm.Manager.Insert en éléments d'argv,
// quote-aware. Insert écrit pour une ligne de shell (« --cask firealpaca »,
// `"Canon IJ Scan Utility"`) ; l'exécution façade a besoin d'éléments d'argv distincts, et
// un simple strings.Fields couperait un identifiant guillemeté contenant des espaces en
// plusieurs arguments — exactement ce que les guillemets étaient censés empêcher.
//
// Les guillemets ne sont jamais partiels dans ce que rendent les Insert existants : ils
// entourent tout l'identifiant ou ne sont pas là du tout. splitInsert n'a donc besoin que
// de les faire basculer, pas de gérer un mélange guillemeté/non guillemeté dans un même
// mot.
func splitInsert(s string) []string {
	var out []string
	var mot []rune
	dansGuillemets := false

	flush := func() {
		if len(mot) > 0 {
			out = append(out, string(mot))
			mot = mot[:0]
		}
	}

	for _, r := range s {
		switch {
		case r == '"':
			dansGuillemets = !dansGuillemets
		case unicode.IsSpace(r) && !dansGuillemets:
			flush()
		default:
			mot = append(mot, r)
		}
	}
	flush()
	return out
}
