package facade

import "testing"

// splitInsert doit rendre les mêmes éléments d'argv qu'un shell aurait découpés à partir
// du texte que rend pm.Manager.Insert — sans jamais couper un identifiant guillemeté.
func TestSplitInsert(t *testing.T) {
	cas := []struct {
		nom  string
		in   string
		want []string
	}{
		{"cask de brew", "--cask firealpaca", []string{"--cask", "firealpaca"}},
		{"bucket de scoop", "main/flux", []string{"main/flux"}},
		{"identifiant à espace de winget", `"Canon IJ Scan Utility"`, []string{"Canon IJ Scan Utility"}},
		{"nom brut", "git", []string{"git"}},
		{"nom sans correction", "winrar", []string{"winrar"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := splitInsert(c.in)
			if !egauxSlices(got, c.want) {
				t.Fatalf("splitInsert(%q) = %v, attendu %v", c.in, got, c.want)
			}
		})
	}
}

func egauxSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
