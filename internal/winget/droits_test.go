package winget

import (
	"testing"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// La table des codes, verdict par verdict. Fonction pure d'un entier : ce test tourne
// sous macOS comme sous Windows, sans winget et sans élévation.
func TestDroitsTraduitLesCodes(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		code    int
		attendu pm.Droits
	}{
		{"COMMAND_REQUIRES_ADMIN", 0x8A150019, pm.DroitsRequis},
		{"CONFIG_UNIT_IMPORT_MODULE_ADMIN", 0x8A15C111, pm.DroitsRequis},

		// Les deux contre-cas. Ce sont eux qui justifient une troisième valeur : proposer
		// d'élever ici reviendrait à refaire, élevé, ce qui vient d'échouer *pour cause
		// d'élévation*.
		{"INSTALLER_PROHIBITS_ELEVATION", 0x8A150056, pm.DroitsInterdits},
		{"ADMIN_CONTEXT_ACTION_PROHIBITED", 0x8A15007D, pm.DroitsInterdits},

		// Des échecs ordinaires, qui ne parlent pas de droits. Le premier est le plus
		// fréquent de tous (« No packages found ») : s'il proposait d'élever, la
		// proposition perdrait tout son sens.
		{"NO_APPLICATIONS_FOUND", 0x8A150014, pm.DroitsRien},
		{"DOWNLOAD_FAILED", 0x8A150008, pm.DroitsRien},
		{"échec quelconque", 1, pm.DroitsRien},
		{"succès", 0, pm.DroitsRien},
	} {
		if got := (Manager{}).Droits(cas.code); got != cas.attendu {
			t.Errorf("%s (0x%X) : %v, attendu %v", cas.nom, uint32(cas.code), got, cas.attendu)
		}
	}
}

// Le piège que ce test existe pour tenir : la table officielle de Microsoft publie ces
// codes sous leur forme **signée**, tandis que `exec.ExitError.ExitCode()` rend sous
// Windows le DWORD **non signé** — mesuré, cf. ADR-0004.
//
// Sans cette assertion, une relecture qui « corrigerait » les constantes en recopiant la
// colonne décimale du tableau officiel casserait la fonction en silence : jigger ne
// proposerait plus jamais rien, et rien ne le signalerait.
func TestDroitsIgnoreLaFormeSigneeDesCodes(t *testing.T) {
	const signe = -1978335207 // ce que le tableau de Microsoft imprime pour 0x8A150019

	if got := (Manager{}).Droits(signe); got != pm.DroitsRien {
		t.Fatalf("la forme signée rend %v : les constantes ont été recopiées du tableau "+
			"officiel au lieu de la forme non signée que Go rend (cf. ADR-0004)", got)
	}
	if got := (Manager{}).Droits(0x8A150019); got != pm.DroitsRequis {
		t.Fatalf("la forme non signée rend %v, attendu DroitsRequis", got)
	}
}
