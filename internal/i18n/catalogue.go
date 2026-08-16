// Le catalogue des chaînes visibles, une ligne par clé, toutes les langues côte à côte.
//
//	{EN, FR}
//
// L'anglais est l'original : il n'est jamais vide. Le français peut l'être — le repli joue.
// Les clés sont préfixées par la surface où la chaîne s'affiche (popup, facade, cli), pour
// qu'une relecture de traduction soit possible sans lire le code.
package i18n

var catalogue = map[string][nbLangues]string{}
