package main

import "os"

// openTTY ouvre la console. Windows n'a pas de /dev/tty : l'entrée et la sortie sont
// deux objets distincts, CONIN$ et CONOUT$, qu'on ouvre nommément pour retrouver le
// clavier et l'écran même quand stdout est capturé par le widget.
//
// CONIN$ peut être refusé (session sans console attachée, ConPTY particulier) ; stdin,
// lui, est resté le clavier dans le cas qui nous intéresse — le shell ne redirige que
// notre sortie. On s'en contente alors plutôt que de renoncer au sélecteur.
func openTTY() (*TTY, error) {
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		in = os.Stdin
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		if in != os.Stdin {
			in.Close()
		}
		return nil, err
	}
	return &TTY{In: in, Out: out}, nil
}

func (t *TTY) Close() {
	if t.In != os.Stdin {
		t.In.Close()
	}
	t.Out.Close()
}
