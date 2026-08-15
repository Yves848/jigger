//go:build !windows

package main

import "os"

// openTTY ouvre le terminal de contrôle. Un seul descripteur suffit : /dev/tty est
// ouvert en lecture *et* en écriture.
func openTTY() (*TTY, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &TTY{In: f, Out: f}, nil
}

func (t *TTY) Close() { t.In.Close() }
