package winget

import "gitlab.yg-devworks.com/yves/jigger/internal/pm"

// winget est à l'opposé de brew : aucune sortie machine, que des tableaux à largeur fixe
// aux en-têtes traduits. Le découpage aux frontières de colonnes vit déjà dans table.go —
// ces parsers ne font que nommer les colonnes qui les intéressent, via ParseTable et
// Row.Col. colName, colID et colVer sont le tronc commun défini dans table.go ; les
// colonnes suivantes varient selon le tableau et sont posées ici.
const (
	// list, upgrade : « Nom  ID  Version  Disponible  Source ».
	colDisponible = 3
	colSource     = 4

	// source list : « Nom  Argument  Type ». Aucun rapport avec ID/Version, mais même
	// position que colID — la colonne « Argument » (l'URL du dépôt) tombe dessus.
	colArgument = colID
)

func parseOutdated(out []byte) ([]pm.Package, error) {
	lignes := ParseTable(out)
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		id := l.Col(colID)
		if id == "" {
			continue
		}
		rows = append(rows, pm.Package{
			Name:      id,
			Version:   l.Col(colVer),
			Available: l.Col(colDisponible),
			Kind:      pm.BadgeWinget,
			PM:        "winget",
		})
	}
	return rows, nil
}

func parseList(out []byte) ([]pm.Package, error) {
	lignes := ParseTable(out)
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		id := l.Col(colID)
		if id == "" {
			continue
		}
		source := l.Col(colSource)
		badge := pm.BadgeWinget
		if source == "" {
			badge = pm.BadgeOther // installé hors catalogue (ARP/MSIX)
		}
		rows = append(rows, pm.Package{
			Name:    id,
			Version: l.Col(colVer),
			Kind:    badge,
			Source:  source,
			PM:      "winget",
		})
	}
	return rows, nil
}

func parseSearch(out []byte) ([]pm.Package, error) {
	lignes := ParseTable(out)
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		id := l.Col(colID)
		if id == "" {
			continue
		}
		rows = append(rows, pm.Package{
			Name:    id,
			Version: l.Col(colVer),
			Kind:    pm.BadgeWinget,
			PM:      "winget",
		})
	}
	return rows, nil
}

// parseSource : winget source list rend « Nom  Argument  Type », un dépôt par ligne.
func parseSource(out []byte) ([]pm.Package, error) {
	lignes := ParseTable(out)
	rows := make([]pm.Package, 0, len(lignes))
	for _, l := range lignes {
		nom := l.Col(colName)
		if nom == "" {
			continue
		}
		rows = append(rows, pm.Package{Name: nom, Source: l.Col(colArgument), PM: "winget"})
	}
	return rows, nil
}
