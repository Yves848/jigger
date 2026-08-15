package brew

import (
	"encoding/json"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// brew est le seul des trois à offrir une vraie sortie machine : --json=v2 dispense de
// tout découpage de colonnes.
type sortieOutdated struct {
	Formulae []entreeOutdated `json:"formulae"`
	Casks    []entreeOutdated `json:"casks"`
}

type entreeOutdated struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

func parseOutdated(out []byte) ([]pm.Package, error) {
	var s sortieOutdated
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	rows := make([]pm.Package, 0, len(s.Formulae)+len(s.Casks))
	for _, groupe := range []struct {
		entrees []entreeOutdated
		badge   string
	}{
		{s.Formulae, pm.BadgeFormula},
		{s.Casks, pm.BadgeCask},
	} {
		for _, e := range groupe.entrees {
			installee := ""
			if len(e.InstalledVersions) > 0 {
				installee = e.InstalledVersions[len(e.InstalledVersions)-1]
			}
			rows = append(rows, pm.Package{
				Name:      e.Name,
				Version:   installee,
				Available: e.CurrentVersion,
				Kind:      groupe.badge,
				PM:        "brew",
			})
		}
	}
	return rows, nil
}

// parseList lit « nom version [version…] », une ligne par formula ou cask.
func parseList(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		champs := strings.Fields(ligne)
		if len(champs) < 2 {
			continue
		}
		rows = append(rows, pm.Package{
			Name:    champs[0],
			Version: champs[len(champs)-1], // la plus récente des versions gardées
			Kind:    pm.BadgeFormula,
			PM:      "brew",
		})
	}
	return rows, nil
}

// parseSearch : brew search rend un nom par ligne, sans version.
func parseSearch(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		nom := strings.TrimSpace(ligne)
		// Les en-têtes de section (« ==> Formulae ») ne sont pas des paquets.
		if nom == "" || strings.HasPrefix(nom, "==>") {
			continue
		}
		rows = append(rows, pm.Package{Name: nom, Kind: pm.BadgeFormula, PM: "brew"})
	}
	return rows, nil
}

// parseSource : brew tap rend un tap par ligne.
func parseSource(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, ligne := range pm.SplitLines(out) {
		rows = append(rows, pm.Package{Name: strings.TrimSpace(ligne), PM: "brew"})
	}
	return rows, nil
}
