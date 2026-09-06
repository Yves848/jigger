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
	TypeTexte   Type = iota
	TypeBooleen      // 0 ou 1
	TypeEntier       // un nombre
	TypeDuree        // une durée Go : 24h, 30m…
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

	// Choix propose des valeurs à l'écran, et Ferme dit si la liste est exhaustive.
	//
	// Un seul mécanisme pour trois besoins, et c'est voulu :
	//   Ferme + Choix   → une liste de choix : la valeur se prend dans la liste, un champ
	//                     libre n'aurait aucun sens (`lang` vaut en, fr, ou rien).
	//   Choix seul      → un combo : on pioche ou on écrit. Une durée de cache a des
	//                     valeurs usuelles sans être bornée à elles.
	//   AideI18n seul   → un champ à saisir, mais jamais à l'aveugle.
	// Les trois se rendent avec le même code d'affichage ; seule la saisie diffère.
	Choix []string
	Ferme bool
	// AideI18n : clé de catalogue d'une ligne d'aide montrée pendant la saisie. Elle dit le
	// FORMAT attendu, là où CleI18n dit à quoi sert le réglage — deux questions distinctes,
	// et c'est la seconde qu'on se pose au moment de taper.
	AideI18n string
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
	{Cle: "pager", CleI18n: "cfg.pager", Portee: Binaire, Type: TypeBooleen, Defaut: "1"},
	// La langue est un ensemble FINI : « fr », « en », ou rien — auquel cas jigger suit la
	// locale. Un champ libre y laisserait écrire « francais » sans jamais dire pourquoi
	// c'est refusé.
	{Cle: "lang", CleI18n: "cfg.lang", Portee: LesDeux, Type: TypeTexte, Defaut: "",
		Choix: []string{"", "en", "fr"}, Ferme: true},
	{Cle: "cache_dir", CleI18n: "cfg.cache_dir", Portee: LesDeux, Type: TypeTexte, Defaut: "",
		AideI18n: "cfg.aide_cache_dir"},

	// ── Ce qui prend effet au prochain shell ────────────────────────────────────────
	{Cle: "live", CleI18n: "cfg.live", Portee: LesDeux, Type: TypeBooleen, Defaut: "1"},
	{Cle: "rows", CleI18n: "cfg.rows", Portee: Greffon, Type: TypeEntier, Defaut: "8",
		Choix: []string{"5", "8", "12", "20"}},
	{Cle: "key", CleI18n: "cfg.key", Portee: Greffon, Type: TypeTexte, Defaut: "^I",
		Choix: []string{"^I", "^ ", "^X^F"}, AideI18n: "cfg.aide_key"},
	{Cle: "keys_extra", CleI18n: "cfg.keys_extra", Portee: Greffon, Type: TypeTexte, Defaut: "",
		AideI18n: "cfg.aide_keys_extra"},
	// Le défaut déclaré est vide parce que la vraie valeur vit dans le greffon ; la proposer
	// ici est ce qui la rend visible, et surtout modifiable sans la deviner — la liste est
	// écrasée, pas complétée, et l'ignorer coûtait les six commandes d'un coup.
	{Cle: "commands", CleI18n: "cfg.commands", Portee: Greffon, Type: TypeTexte, Defaut: "",
		Choix:    []string{"brew pacman yay ssh scp sftp", "brew pacman yay", "winget scoop"},
		AideI18n: "cfg.aide_commands"},
	{Cle: "min_columns", CleI18n: "cfg.min_columns", Portee: Greffon, Type: TypeEntier, Defaut: "30",
		Choix: []string{"30", "40", "60", "80"}},
	{Cle: "prompt", CleI18n: "cfg.prompt", Portee: Greffon, Type: TypeBooleen, Defaut: "0"},
	{Cle: "prompt_ttl", CleI18n: "cfg.prompt_ttl", Portee: Greffon, Type: TypeEntier, Defaut: "1800",
		Choix: []string{"300", "900", "1800", "3600"}, AideI18n: "cfg.aide_secondes"},
	{Cle: "bin", CleI18n: "cfg.bin", Portee: Greffon, Type: TypeTexte, Defaut: "jigger",
		Choix: []string{"jigger"}, AideI18n: "cfg.aide_bin"},
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
