package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// Les codes de sortie de winget qui parlent de privilèges, relevés dans la table
// officielle du client (`microsoft/winget-cli`, `doc/windows/package-manager/winget/
// returnCodes.md`, consultée le 17 août 2026).
//
// ATTENTION — ces constantes ne sont PAS celles de la colonne « decimal » du tableau
// officiel. Microsoft publie la forme **signée** (0x8A150019 → -1978335207) ; sous
// Windows, `exec.ExitError.ExitCode()` rend le **DWORD non signé** (2316632089). C'est
// mesuré, pas supposé : cf. ADR-0004, « Ce que la mesure a établi ». Recopier la colonne
// décimale du tableau donnerait une comparaison qui n'est jamais vraie — et une panne
// parfaitement muette, jigger ne proposant simplement jamais rien.
const (
	// Command requires administrator privileges to run.
	codeRequiertAdmin = 0x8A150019
	// Loading the module for the configuration unit failed because it requires
	// administrator privileges to run.
	codeConfigRequiertAdmin = 0x8A15C111

	// The installer cannot be run from an administrator context.
	codeInstalleurRefuseElevation = 0x8A150056
	// The requested operation is not permitted from an administrator context on packages
	// installed within the user scope.
	codeActionInterditeEnAdmin = 0x8A15007D
)

// Droits traduit un code de sortie de winget. Fonction pure d'un entier vers un verdict :
// elle s'éprouve sans winget, sans Windows et sans élévation.
//
// Les deux derniers codes disent l'**inverse** du premier. C'est la raison d'être de
// pm.DroitsInterdits, et ce qu'un « code non nul → propose d'élever » aurait manqué.
func (Manager) Droits(code int) pm.Droits {
	switch code {
	case codeRequiertAdmin, codeConfigRequiertAdmin:
		return pm.DroitsRequis
	case codeInstalleurRefuseElevation, codeActionInterditeEnAdmin:
		return pm.DroitsInterdits
	}
	return pm.DroitsRien
}
