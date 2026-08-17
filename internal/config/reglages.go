package config

// Les réglages de jigger, déclarés une fois. L'écran, l'export vers les greffons et la
// documentation en dérivent — dans l'esprit de l'ADR-0002 : ce qui est déclaré est
// vérifiable, ce qui est codé en dur ne l'est pas.

// Portee dit qui lit le réglage, donc quand un changement prend effet. C'est la
// distinction qui structure l'écran : huit réglages sur douze appartiennent au greffon et
// ne s'appliquent qu'au prochain shell (spec §2).
type Portee int

const (
	// Binaire : lu à chaque appel de jigger. Un changement prend effet tout de suite.
	Binaire Portee = iota
	// Greffon : lu au chargement du shell. Un changement prend effet au prochain shell.
	Greffon
	// LesDeux : lu des deux côtés.
	LesDeux
)

// Type sert à l'écran pour proposer la bonne saisie, et à la validation.
type Type int

const (
	Texte   Type = iota
	Booleen      // 0 ou 1
	Entier       // un nombre
	Duree        // une durée Go : 24h, 30m…
)

// Reglage déclare un réglage : sa clé, ce qu'il fait, qui le lit, son type et son défaut.
//
// `Cle` est le nom SANS le préfixe JIGGER_ : le fichier écrit « rows = 12 », l'export émet
// « JIGGER_ROWS=12 ». Le préfixe n'a de sens que dans l'environnement.
type Reglage struct {
	Cle     string
	CleI18n string // clé de catalogue pour la description affichée
	Portee  Portee
	Type    Type
	Defaut  string
	// PM, s'il est renseigné, rattache le réglage à un gestionnaire : l'écran groupe alors
	// le réglage sous lui plutôt que dans les réglages généraux.
	PM string
}

// Env rend le nom de la variable d'environnement correspondante.
func (r Reglage) Env() string { return "JIGGER_" + majuscules(r.Cle) }

func majuscules(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		out[i] = c
	}
	return string(out)
}

// Declares est la table des réglages généraux. Les gestionnaires y ajoutent les leurs par
// Declarer, au chargement de leur paquet.
var Declares = []Reglage{
	// ── Ce qui prend effet tout de suite ────────────────────────────────────────────
	{Cle: "pager", CleI18n: "cfg.pager", Portee: Binaire, Type: Booleen, Defaut: "1"},
	{Cle: "lang", CleI18n: "cfg.lang", Portee: LesDeux, Type: Texte, Defaut: ""},
	{Cle: "cache_dir", CleI18n: "cfg.cache_dir", Portee: LesDeux, Type: Texte, Defaut: ""},

	// ── Ce qui prend effet au prochain shell ────────────────────────────────────────
	{Cle: "live", CleI18n: "cfg.live", Portee: LesDeux, Type: Booleen, Defaut: "1"},
	{Cle: "rows", CleI18n: "cfg.rows", Portee: Greffon, Type: Entier, Defaut: "8"},
	{Cle: "key", CleI18n: "cfg.key", Portee: Greffon, Type: Texte, Defaut: "^I"},
	{Cle: "keys_extra", CleI18n: "cfg.keys_extra", Portee: Greffon, Type: Texte, Defaut: ""},
	{Cle: "commands", CleI18n: "cfg.commands", Portee: Greffon, Type: Texte, Defaut: ""},
	{Cle: "min_columns", CleI18n: "cfg.min_columns", Portee: Greffon, Type: Entier, Defaut: "30"},
	{Cle: "prompt", CleI18n: "cfg.prompt", Portee: Greffon, Type: Booleen, Defaut: "0"},
	{Cle: "prompt_ttl", CleI18n: "cfg.prompt_ttl", Portee: Greffon, Type: Entier, Defaut: "1800"},
	{Cle: "bin", CleI18n: "cfg.bin", Portee: Greffon, Type: Texte, Defaut: "jigger"},
}

// Declarer ajoute un réglage à la table. Les gestionnaires l'appellent depuis leur `init`,
// ce qui fait apparaître le réglage dans l'écran, dans l'export et dans la documentation
// sans qu'aucun d'eux n'ait à connaître le gestionnaire.
func Declarer(r Reglage) {
	for _, d := range Declares {
		if d.Cle == r.Cle {
			return // déjà déclaré : un rechargement de paquet ne duplique rien
		}
	}
	Declares = append(Declares, r)
}

// Trouver rend la déclaration d'une clé.
func Trouver(cle string) (Reglage, bool) {
	for _, r := range Declares {
		if r.Cle == cle {
			return r, true
		}
	}
	return Reglage{}, false
}
