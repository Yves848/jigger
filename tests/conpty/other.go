//go:build !windows

// Le harnais ConPTY n'a de sens que sous Windows ; ailleurs, c'est tests/zpty.zsh qui
// tient ce rôle. Ce fichier n'existe que pour que `go build ./...` reste vert sur les
// autres plateformes.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "tests/conpty : harnais réservé à Windows (cf. tests/zpty.zsh)")
	os.Exit(2)
}
