//go:build !windows

package i18n

// cultureSysteme n'a rien à ajouter hors Windows : les variables POSIX ont déjà été
// consultées par resoudre.
func cultureSysteme() string { return "" }
