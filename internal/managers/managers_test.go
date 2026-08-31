package managers

import "testing"

func TestAllContientLesTroisCommandesSSH(t *testing.T) {
	vus := map[string]bool{}
	for _, m := range All() {
		vus[m.Cmd()] = true
	}
	for _, c := range []string{"ssh", "scp", "sftp"} {
		if !vus[c] {
			t.Errorf("%q absent de All()", c)
		}
	}
	// Les trois gestionnaires de paquets restent là.
	for _, c := range []string{"brew", "winget", "scoop"} {
		if !vus[c] {
			t.Errorf("%q disparu de All()", c)
		}
	}
}

func TestLesFournisseursSSHNeDeclarentAucunVerbe(t *testing.T) {
	for _, m := range All() {
		switch m.Cmd() {
		case "ssh", "scp", "sftp":
			if len(m.Subcommands()) != 0 {
				t.Errorf("%q déclare des sous-commandes", m.Cmd())
			}
		}
	}
}
