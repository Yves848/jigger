package facade

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// Formater rend la sortie d'un verbe normalisé. Les colonnes s'adaptent aux données :
// PM n'apparaît que si plusieurs gestionnaires ont contribué, DISPO que si au moins une
// ligne porte une version disponible. Une colonne toujours vide n'apprend rien.
func Formater(v pm.Verb, rows []pm.Package, enJSON bool) string {
	if enJSON {
		if rows == nil {
			rows = []pm.Package{}
		}
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return "[]"
		}
		return string(data)
	}

	if len(rows) == 0 {
		return i18n.T("facade.nothing")
	}

	entete, cellules := Colonnes(rows)
	return aligner(append([][]string{entete}, cellules...))
}

// Colonnes rend l'en-tête et les cellules d'un jeu de lignes, selon la règle adaptative :
// une colonne toujours vide n'apprend rien, donc elle n'apparaît pas. PM ne sort que si
// plusieurs gestionnaires ont contribué ; ACTUEL que si une version est renseignée — le
// cas de list/outdated, pas de search/source.
//
// C'est la SEULE source des colonnes : la table brute et la vue paginée l'appellent
// toutes les deux, sans quoi elles divergeraient le jour où l'une des deux évoluerait
// (spec §1).
func Colonnes(rows []pm.Package) (entete []string, cellules [][]string) {
	avecPM := plusieursPM(rows)
	avecActuel := false
	avecDispo := false
	avecSource := false
	for _, r := range rows {
		if r.Version != "" {
			avecActuel = true
		}
		if r.Available != "" {
			avecDispo = true
		}
		if r.Source != "" {
			avecSource = true
		}
	}

	entete = []string{i18n.T("table.package")}
	if avecActuel {
		entete = append(entete, i18n.T("table.current"))
	}
	if avecDispo {
		entete = append(entete, i18n.T("table.available"))
	}
	if avecSource {
		entete = append(entete, i18n.T("table.source"))
	}
	if avecPM {
		entete = append(entete, i18n.T("table.pm"))
	}

	cellules = make([][]string, 0, len(rows))
	for _, r := range rows {
		ligne := []string{r.Name}
		if avecActuel {
			ligne = append(ligne, r.Version)
		}
		if avecDispo {
			ligne = append(ligne, r.Available)
		}
		if avecSource {
			ligne = append(ligne, r.Source)
		}
		if avecPM {
			ligne = append(ligne, r.PM)
		}
		cellules = append(cellules, ligne)
	}
	return entete, cellules
}

// plusieursPM délègue à pm.PlusieursPM : c'est la même règle que la colonne PM du popup
// (cf. ui.Frame.avecPM), une seule fois — pas deux critères qui finissent par diverger.
func plusieursPM(rows []pm.Package) bool {
	pms := make([]string, len(rows))
	for i, r := range rows {
		pms[i] = r.PM
	}
	return pm.PlusieursPM(pms)
}

// aligner pose les colonnes à largeur fixe, deux espaces de gouttière. Le même principe
// que les tableaux de winget — à ceci près qu'ici, c'est nous qui les écrivons.
func aligner(table [][]string) string {
	if len(table) == 0 {
		return ""
	}
	largeurs := make([]int, len(table[0]))
	for _, ligne := range table {
		for i, cell := range ligne {
			if n := len([]rune(cell)); n > largeurs[i] {
				largeurs[i] = n
			}
		}
	}

	var b strings.Builder
	for _, ligne := range table {
		for i, cell := range ligne {
			if i == len(ligne)-1 {
				b.WriteString(cell)
			} else {
				fmt.Fprintf(&b, "%-*s  ", largeurs[i], cell)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
