package facade

import (
	"encoding/json"
	"fmt"
	"strings"

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
		return "rien à signaler\n"
	}

	avecPM := plusieursPM(rows)
	avecDispo := false
	avecSource := false
	for _, r := range rows {
		if r.Available != "" {
			avecDispo = true
		}
		if r.Source != "" {
			avecSource = true
		}
	}

	entete := []string{"PAQUET", "ACTUEL"}
	if avecDispo {
		entete = append(entete, "DISPO")
	}
	if avecSource {
		entete = append(entete, "SOURCE")
	}
	if avecPM {
		entete = append(entete, "PM")
	}

	table := [][]string{entete}
	for _, r := range rows {
		ligne := []string{r.Name, r.Version}
		if avecDispo {
			ligne = append(ligne, r.Available)
		}
		if avecSource {
			ligne = append(ligne, r.Source)
		}
		if avecPM {
			ligne = append(ligne, r.PM)
		}
		table = append(table, ligne)
	}
	return aligner(table)
}

func plusieursPM(rows []pm.Package) bool {
	vu := ""
	for _, r := range rows {
		if r.PM == "" {
			continue
		}
		if vu == "" {
			vu = r.PM
		} else if vu != r.PM {
			return true
		}
	}
	return false
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
