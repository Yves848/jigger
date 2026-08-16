// Ce fichier écrit les parsers de scoop pour les trois verbes normalisés qu'il expose en
// Native — list, search, source (cf. verbs.go ; outdated, lui, est Direct et n'a pas
// besoin d'un parser).
//
// Vérifiés contre un vrai scoop 0.5.3 sous Windows 10.0.26200 : les jeux d'essai de
// testdata/ sont des captures réelles, prises comme jigger les reçoit — derrière un tuyau
// (cf. tests/captures-scoop.ps1). Deux défauts en sont sortis, et il vaut la peine de les
// garder écrits :
//
//   - **la couleur**. PowerShell colore l'en-tête et la ligne de tirets de ses tableaux
//     même quand la sortie est redirigée. Les données, elles, sont propres. Or c'est la
//     ligne de tirets qui marque l'entrée du tableau : entourée d'échappements ANSI, elle
//     ne ressemblait plus à des tirets, le parser n'entrait jamais dans la table et rendait
//     zéro ligne — sans erreur, donc sans que rien ne le signale. D'où sansAnsi, appliqué
//     à chaque ligne avant tout examen ;
//   - **le format de `search`**. Celui-là était bien écrit contre une sortie obsolète : des
//     sections « 'main' bucket: » suivies de « nom (version) ». scoop rend aujourd'hui un
//     tableau, comme pour list et bucket list. (Ce vieux format survit ailleurs : c'est
//     celui de `scoop --version`, qui liste les commits de chaque bucket.)
package scoop

import (
	"regexp"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
)

// ansi reconnaît les séquences SGR (couleur, gras) que PowerShell insère dans ses en-têtes
// de tableau. On ne retient que cette forme : c'est la seule que scoop émette, et un
// nettoyeur d'échappements trop large risquerait de manger un caractère légitime d'un nom
// de paquet.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func sansAnsi(ligne string) string { return ansi.ReplaceAllString(ligne, "") }

// estSeparateur reconnaît la ligne de tirets que Format-Table pose entre l'en-tête et les
// données (« ----  -------  ------ »).
func estSeparateur(ligne string) bool {
	t := strings.TrimSpace(ligne)
	return t != "" && strings.Trim(t, "- ") == ""
}

// colonnes rend, pour chaque colonne, la position de son premier caractère, lue sur la
// **ligne de tirets**.
//
// C'est elle qui fait autorité, et pas l'en-tête ni l'espacement des données. Format-Table
// remplit chaque colonne à la largeur de sa cellule la plus longue : la ligne la plus large
// d'un tableau n'a donc **qu'un seul espace** avant la colonne suivante. Découper sur « au
// moins deux espaces » — ce que faisait la première version — lisait alors deux champs
// comme un seul, et systématiquement sur la ligne qui donne sa largeur à la colonne :
//
//	git-flow-next               1.1.0     ← deux espaces, découpage correct
//	git-interactive-rebase-tool 2.4.1     ← un seul, nom et version collés
//
// La ligne de tirets, elle, sépare toujours ses groupes, quelle que soit la largeur.
func colonnes(tirets string) []int {
	var debuts []int
	dansGroupe := false
	for i, r := range []rune(tirets) {
		switch {
		case r == '-' && !dansGroupe:
			debuts = append(debuts, i)
			dansGroupe = true
		case r != '-':
			dansGroupe = false
		}
	}
	return debuts
}

// decouper tranche une ligne de données aux positions données. Une colonne absente — la
// ligne s'arrête avant, ce qui arrive quand les dernières cellules sont vides — rend une
// chaîne vide plutôt qu'un champ manquant : l'indice d'une colonne reste ainsi le même
// d'une ligne à l'autre.
func decouper(ligne string, debuts []int) []string {
	runes := []rune(ligne)
	champs := make([]string, len(debuts))
	for i, d := range debuts {
		if d >= len(runes) {
			continue
		}
		fin := len(runes)
		if i+1 < len(debuts) && debuts[i+1] < fin {
			fin = debuts[i+1]
		}
		champs[i] = strings.TrimSpace(string(runes[d:fin]))
	}
	return champs
}

// tableau lit un tableau Format-Table et rend une ligne de champs par enregistrement.
//
// Les trois verbes normalisés de scoop rendent le même gabarit — un titre, un en-tête, une
// ligne de tirets, les données —, seule l'interprétation des colonnes les distingue. Tout
// ce qui précède la ligne de tirets est ignoré : les noms de colonnes ne servent à rien
// tant que scoop ne les traduit pas (contrairement à winget), seule la position compte.
//
// Le découpage des lignes est fait ici plutôt que par pm.SplitLines, qui ôte les blancs de
// tête : une position de colonne ne survivrait pas à ce raccourcissement.
func tableau(out []byte) [][]string {
	var rows [][]string
	var debuts []int
	for _, brute := range strings.Split(string(out), "\n") {
		ligne := sansAnsi(strings.TrimRight(brute, "\r\n"))
		if debuts == nil {
			if estSeparateur(ligne) {
				debuts = colonnes(ligne)
			}
			continue
		}
		champs := decouper(ligne, debuts)
		if len(champs) == 0 || champs[0] == "" {
			continue
		}
		rows = append(rows, champs)
	}
	return rows
}

// champ rend la colonne d'indice i, ou la chaîne vide si la ligne n'en a pas tant.
func champ(champs []string, i int) string {
	if i < len(champs) {
		return champs[i]
	}
	return ""
}

// parseList lit « scoop list » : Name, Version, Source, Updated, Info.
func parseList(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, champs := range tableau(out) {
		if champ(champs, 1) == "" {
			continue
		}
		source := champ(champs, 2)
		rows = append(rows, pm.Package{
			Name:    champs[0],
			Version: champs[1],
			Kind:    badge(source),
			Source:  source,
		})
	}
	return rows, nil
}

// parseSearch lit « scoop search <requête> » : un titre (« Results from local buckets... »),
// puis le même gabarit de tableau que list — Name, Version, Source, Binaries. La colonne
// Binaries est souvent vide, et la ligne s'arrête alors après le bucket : c'est pourquoi
// Source n'est lu que s'il est là.
func parseSearch(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, champs := range tableau(out) {
		if champ(champs, 1) == "" {
			continue
		}
		source := champ(champs, 2)
		rows = append(rows, pm.Package{
			Name:    champs[0],
			Version: champs[1],
			Kind:    badge(source),
			Source:  source,
		})
	}
	return rows, nil
}

// parseSource lit « scoop bucket list » : même gabarit de tableau que list, mais Name est
// le nom du bucket et la deuxième colonne (Source) est l'URL du dépôt — il n'y a pas de
// version à porter.
func parseSource(out []byte) ([]pm.Package, error) {
	var rows []pm.Package
	for _, champs := range tableau(out) {
		rows = append(rows, pm.Package{Name: champs[0], Source: champ(champs, 1)})
	}
	return rows, nil
}
