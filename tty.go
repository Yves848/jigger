package main

import "os"

// TTY est le terminal, ouvert directement plutôt que par les flux standard : le
// sélecteur y dessine et y lit les touches, ce qui laisse stdout au seul résultat — la
// nouvelle ligne de commande, que le widget du shell récupère. C'est la mécanique de
// fzf, et la seule qui permette d'être appelé depuis une substitution de commande.
type TTY struct {
	In, Out *os.File
}
