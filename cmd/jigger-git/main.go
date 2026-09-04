// Commande jigger-git expose les dépôts git clonés localement comme des paquets, et les
// verbes de jigger comme des opérations git : installer, c'est cloner ; désinstaller,
// c'est supprimer le clone ; mettre à jour, c'est tirer.
//
// C'est un plugin jigger au sens de docs/plans/2026-09-04-plugins-injection.md : un
// binaire autonome, découvert par un descripteur `config.json`, qui dialogue en JSON.
//
//	jigger-git catalog             → {"names":[…],"badges":{…}}
//	jigger-git list                → [{"name":…,"version":…,"kind":…,"source":…}]
//	jigger-git outdated            → [{…,"available":…}]  dépôts en retard sur l'amont
//	jigger-git search <motif>…     → [{…}]  filtrage du catalogue
//	jigger-git run install <nom|url>
//	jigger-git run uninstall <nom> [--force]
//	jigger-git run upgrade [<nom>…]
//
// Les verbes de lecture écrivent un document JSON sur la sortie standard et rien
// d'autre ; les verbes d'écriture relaient git tel quel, terminal compris, pour qu'une
// demande de mot de passe ou une barre de progression arrive jusqu'à l'utilisateur.
//
// Le programme n'importe rien de jigger : un plugin tiers ne le pourrait pas non plus.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Badges. Ils reprennent les deux classes que le popup de jigger sait peindre (cf.
// pm.BadgeRepo et pm.BadgeOther) : le cas ordinaire — un clone rattaché à un dépôt
// distant — et l'autre — un dépôt purement local, qu'on ne peut ni tirer ni recloner.
const (
	badgeSuivi = "R"
	badgeLocal = "X"
)

// profondeurMax borne la descente dans les racines. Deux niveaux couvrent la forme
// courante — ~/git/projet et ~/git/client/projet — sans parcourir un disque entier.
const profondeurMax = 2

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	var err error
	switch args[0] {
	case "catalog":
		err = catalogue()
	case "list":
		err = lister()
	case "outdated":
		err = enRetard(args[1:])
	case "search":
		err = chercher(sansOptions(args[1:]))
	case "run":
		err = executer(args[1:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		err = fmt.Errorf("verbe inconnu %q", args[0])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "jigger-git :", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage : jigger-git <catalog|list|outdated|search|run> …")
	fmt.Fprintln(os.Stderr, "        jigger-git run <install|uninstall|upgrade> …")
}

// sansOptions retire les drapeaux d'une liste d'arguments. jigger peut en ajouter
// (`--json`, par exemple) sans que le plugin ait à les connaître un par un.
func sansOptions(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func aLOption(args []string, nom string) bool {
	for _, a := range args {
		if a == nom {
			return true
		}
	}
	return false
}

// ── le modèle : un dépôt vu comme un paquet ────────────────────────────

// depot est un clone local. `version` porte la branche courante — c'est ce qui distingue
// deux états du même dépôt, donc ce qui tient le mieux le rôle de version.
type depot struct {
	nom     string
	chemin  string
	branche string
	origine string
}

func (d depot) badge() string {
	if d.origine == "" {
		return badgeLocal
	}
	return badgeSuivi
}

// paquet est la ligne normalisée que jigger attend d'un plugin.
type paquet struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Available string `json:"available,omitempty"`
	Kind      string `json:"kind"`
	Source    string `json:"source,omitempty"`
}

func (d depot) paquet() paquet {
	return paquet{Name: d.nom, Version: d.branche, Kind: d.badge(), Source: d.origine}
}

// ── où chercher les dépôts ─────────────────────────────────────────────

// racines rend les répertoires à explorer. JIGGER_GIT_ROOTS a le dernier mot : sans elle,
// on se rabat sur les emplacements usuels, et il n'est pas question de coder en dur le
// dossier d'une machine particulière.
func racines() []string {
	if v := os.Getenv("JIGGER_GIT_ROOTS"); v != "" {
		var out []string
		for _, d := range filepath.SplitList(v) {
			if d = strings.TrimSpace(d); d != "" {
				out = append(out, d)
			}
		}
		return out
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, "git"),
		filepath.Join(home, "Projets"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "src"),
	}
}

// depots parcourt les racines et rend les clones trouvés, triés par nom. Un dépôt vu deux
// fois — deux racines qui se recouvrent — n'est compté qu'une fois.
func depots() []depot {
	var out []depot
	vus := map[string]bool{}

	for _, racine := range racines() {
		for _, chemin := range parcourir(racine, profondeurMax) {
			if vus[chemin] {
				continue
			}
			vus[chemin] = true
			out = append(out, decrire(chemin))
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].nom < out[j].nom })
	return out
}

// parcourir descend dans un répertoire à la recherche de dépôts. Un dossier qui porte
// `.git` **est** un dépôt : on ne descend pas dedans, sans quoi chaque sous-module et
// chaque dossier de travail remonterait comme un paquet distinct.
func parcourir(racine string, profondeur int) []string {
	if profondeur <= 0 {
		return nil
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		return nil // racine absente : c'est le cas ordinaire, pas une erreur
	}

	var out []string
	for _, e := range entrees {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		chemin := filepath.Join(racine, e.Name())
		if estDepot(chemin) {
			out = append(out, chemin)
			continue
		}
		out = append(out, parcourir(chemin, profondeur-1)...)
	}
	return out
}

// estDepot dit si un dossier porte un `.git` — dossier pour un clone ordinaire, fichier
// pour un worktree ou un sous-module.
func estDepot(chemin string) bool {
	_, err := os.Stat(filepath.Join(chemin, ".git"))
	return err == nil
}

// decrire lit ce qu'on peut dire d'un dépôt sans jamais échouer : un clone tout juste
// initialisé, sans commit ni remote, reste un paquet valide.
func decrire(chemin string) depot {
	d := depot{nom: filepath.Base(chemin), chemin: chemin}
	d.branche = git(chemin, "rev-parse", "--abbrev-ref", "HEAD")
	if d.branche == "HEAD" {
		// Tête détachée : la révision courte dit davantage que « HEAD ».
		d.branche = git(chemin, "rev-parse", "--short", "HEAD")
	}
	d.origine = git(chemin, "remote", "get-url", "origin")
	return d
}

// git lance une commande git dans un dépôt et rend sa sortie épurée, ou "" si elle
// échoue. Les verbes de lecture ne doivent jamais s'arrêter sur un dépôt bancal.
func git(chemin string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", chemin}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ── le registre : d'où l'on clone ──────────────────────────────────────

// registre lit la table nom → URL posée à côté du descripteur du plugin. Sans elle,
// `install` ne saurait cloner que ce qu'on lui donne sous forme d'URL — et surtout, il ne
// devinerait pas une URL GitHub à partir d'un nom, ce qui clonerait n'importe quoi.
//
// Elle est complétée par les origines retenues des dépôts déjà vus (cf. retenir) : sans
// cela, un dépôt supprimé par `uninstall` ne pourrait plus jamais être recloné par
// `install`, alors que jigger vient d'en afficher l'URL.
func registre() map[string]string {
	table := lireTable(filepath.Join(dossierConfig(), "depots.json"))
	if table == nil {
		table = map[string]string{}
	}
	// Le fichier écrit à la main a le dernier mot : c'est le seul que l'utilisateur
	// contrôle, et il doit pouvoir corriger une origine devenue fausse.
	for nom, url := range lireTable(cheminConnus()) {
		if _, ok := table[nom]; !ok {
			table[nom] = url
		}
	}
	return table
}

func cheminConnus() string { return filepath.Join(dossierConfig(), "connus.json") }

func lireTable(chemin string) map[string]string {
	data, err := os.ReadFile(chemin)
	if err != nil {
		return nil
	}
	var table map[string]string
	if err := json.Unmarshal(data, &table); err != nil {
		fmt.Fprintf(os.Stderr, "jigger-git : %s est illisible : %v\n", chemin, err)
		return nil
	}
	return table
}

// retenir note l'origine des dépôts clonés, pour savoir les recloner plus tard. C'est
// écrit au réchauffement — jamais dans le chemin du rendu —, et un échec n'a rien de
// grave : on perd une commodité, pas une donnée.
func retenir(depots []depot) {
	dossier := dossierConfig()
	if dossier == "" {
		return
	}
	connus := lireTable(cheminConnus())
	if connus == nil {
		connus = map[string]string{}
	}
	change := false
	for _, d := range depots {
		if d.origine != "" && connus[d.nom] != d.origine {
			connus[d.nom] = d.origine
			change = true
		}
	}
	if !change {
		return
	}
	data, err := json.MarshalIndent(connus, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(cheminConnus(), data, 0o644)
}

// dossierConfig rend le dossier du plugin dans la configuration de l'utilisateur.
func dossierConfig() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "jigger", "plugins", "git")
}

// ── verbes de lecture ──────────────────────────────────────────────────

// catalogue rend tout ce que jigger peut proposer : les clones présents, et les dépôts
// que le registre sait cloner. Les seconds sont ce qui rend `jg install <TAB>` utile —
// sans eux, on ne pourrait compléter que ce qui est déjà là.
func catalogue() error {
	noms := []string{}
	badges := map[string]string{}

	clones := depots()
	retenir(clones)
	for _, d := range clones {
		noms = append(noms, d.nom)
		badges[d.nom] = d.badge()
	}
	for nom := range registre() {
		if _, dejaLa := badges[nom]; dejaLa {
			continue
		}
		noms = append(noms, nom)
		badges[nom] = badgeSuivi
	}
	sort.Strings(noms)

	return ecrire(struct {
		Names  []string          `json:"names"`
		Badges map[string]string `json:"badges"`
	}{noms, badges})
}

// lister rend les dépôts clonés — les « installés » au sens de jigger.
func lister() error {
	depots := depots()
	out := make([]paquet, 0, len(depots))
	for _, d := range depots {
		out = append(out, d.paquet())
	}
	return ecrire(out)
}

// enRetard rend les dépôts dont la branche courante est derrière son amont. `available`
// dit de combien de commits — c'est la colonne que `jg outdated` affiche en face de la
// version installée.
//
// Le retard se lit sur la référence de suivi (`origin/main`), qui ne bouge qu'au fetch :
// sans réseau, on rend donc ce que la dernière synchronisation sait, et l'on peut très
// bien répondre « rien » d'un dépôt qui a pris dix commits entre-temps. `--fetch` va le
// demander au distant. Ce n'est pas le défaut, et c'est délibéré : `outdated` peut porter
// sur des dizaines de dépôts, et une lecture ne doit pas partir en réseau sans qu'on
// l'ait demandé.
func enRetard(args []string) error {
	fetch := aLOption(args, "--fetch") || os.Getenv("JIGGER_GIT_FETCH") == "1"

	out := []paquet{}
	for _, d := range depots() {
		if d.origine == "" {
			continue // rien à rattraper sans amont
		}
		if fetch {
			// --quiet : la sortie du fetch polluerait le document JSON attendu.
			_ = exec.Command("git", "-C", d.chemin, "fetch", "--quiet").Run()
		}
		retard := git(d.chemin, "rev-list", "--count", "HEAD..@{upstream}")
		if retard == "" || retard == "0" {
			continue // pas d'amont configuré, ou déjà à jour
		}
		p := d.paquet()
		p.Available = retard + " commit(s)"
		out = append(out, p)
	}
	return ecrire(out)
}

// chercher filtre le catalogue sur un motif, sans distinction de casse.
func chercher(motifs []string) error {
	depots := depots()
	reg := registre()

	out := []paquet{}
	for _, d := range depots {
		if correspond(d.nom, motifs) {
			out = append(out, d.paquet())
		}
	}
	for nom, url := range reg {
		if !correspond(nom, motifs) || estClone(depots, nom) {
			continue
		}
		out = append(out, paquet{Name: nom, Kind: badgeSuivi, Source: url})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return ecrire(out)
}

// correspond dit si un nom contient l'un des motifs. Sans motif, tout correspond — c'est
// ce que fait `jg search` seul.
func correspond(nom string, motifs []string) bool {
	if len(motifs) == 0 {
		return true
	}
	bas := strings.ToLower(nom)
	for _, m := range motifs {
		if strings.Contains(bas, strings.ToLower(m)) {
			return true
		}
	}
	return false
}

func estClone(depots []depot, nom string) bool {
	for _, d := range depots {
		if d.nom == nom {
			return true
		}
	}
	return false
}

// ecrire sérialise une réponse sur la sortie standard. Un seul document, rien d'autre :
// c'est tout le contrat de lecture du protocole.
func ecrire(v any) error {
	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("sérialisation : %w", err)
	}
	fmt.Println(string(out))
	return nil
}

// ── verbes d'écriture ──────────────────────────────────────────────────

func executer(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run attend un verbe")
	}
	verbe, reste := args[0], args[1:]

	switch verbe {
	case "install":
		return installer(reste)
	case "uninstall":
		return desinstaller(reste)
	case "upgrade":
		return mettreAJour(reste)
	default:
		return fmt.Errorf("verbe non supporté : %q", verbe)
	}
}

// installer clone un dépôt. L'argument est soit une URL — la façon la plus directe —,
// soit un nom que le registre sait traduire. Rien n'est deviné : sans l'un ni l'autre,
// on le dit plutôt que de cloner une adresse construite au jugé.
func installer(args []string) error {
	noms := sansOptions(args)
	if len(noms) == 0 {
		return fmt.Errorf("install attend un nom de dépôt ou une URL")
	}

	for _, arg := range noms {
		if err := clonerUn(arg); err != nil {
			return err
		}
	}
	return nil
}

func clonerUn(arg string) error {
	url, nom, err := resoudre(arg)
	if err != nil {
		return err
	}

	dest, err := destination(nom)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s est déjà cloné dans %s", nom, dest)
	}

	fmt.Printf("git clone %s %s\n", url, dest)
	return relayer("clone", url, dest)
}

// resoudre traduit l'argument en (URL, nom du dossier).
func resoudre(arg string) (url, nom string, err error) {
	if estURL(arg) {
		return arg, nomDepuisURL(arg), nil
	}
	if u, ok := registre()[arg]; ok {
		return u, arg, nil
	}
	return "", "", fmt.Errorf(
		"%s est inconnu : donnez son URL, ou inscrivez-le dans %s",
		arg, filepath.Join(dossierConfig(), "depots.json"))

}

// estURL reconnaît les formes que git sait cloner : https://…, git@hôte:chemin, file://,
// ssh://… Un simple nom, lui, doit passer par le registre.
func estURL(s string) bool {
	if strings.Contains(s, "://") {
		return true
	}
	// Forme scp : user@hôte:chemin — le « : » doit précéder le premier « / ».
	if i := strings.Index(s, ":"); i > 0 && strings.Contains(s[:i], "@") {
		return true
	}
	return false
}

// nomDepuisURL tire le nom du dossier d'une URL de clonage.
func nomDepuisURL(url string) string {
	s := strings.TrimSuffix(strings.TrimRight(url, "/"), ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// destination choisit où cloner : la première racine qui existe, créée au besoin s'il
// n'y en a aucune.
func destination(nom string) (string, error) {
	rs := racines()
	if len(rs) == 0 {
		return "", fmt.Errorf("aucune racine de dépôts : posez JIGGER_GIT_ROOTS")
	}
	for _, r := range rs {
		if fi, err := os.Stat(r); err == nil && fi.IsDir() {
			return filepath.Join(r, nom), nil
		}
	}
	if err := os.MkdirAll(rs[0], 0o755); err != nil {
		return "", fmt.Errorf("création de %s : %w", rs[0], err)
	}
	return filepath.Join(rs[0], nom), nil
}

// desinstaller supprime un clone — et refuse de le faire tant qu'il porte du travail que
// la suppression perdrait pour de bon. `--force` lève la garde, mais il faut l'écrire.
func desinstaller(args []string) error {
	noms := sansOptions(args)
	force := aLOption(args, "--force") || aLOption(args, "-f")
	if len(noms) == 0 {
		return fmt.Errorf("uninstall attend un nom de dépôt")
	}

	tous := depots()
	for _, nom := range noms {
		d, ok := trouver(tous, nom)
		if !ok {
			return fmt.Errorf("%s n'est pas cloné", nom)
		}
		if !force {
			if raison := travailEnPeril(d); raison != "" {
				return fmt.Errorf("%s : %s — relancez avec --force pour le supprimer quand même", nom, raison)
			}
		}
		fmt.Printf("suppression de %s\n", d.chemin)
		if err := os.RemoveAll(d.chemin); err != nil {
			return fmt.Errorf("suppression de %s : %w", d.chemin, err)
		}
	}
	return nil
}

// travailEnPeril dit ce qu'une suppression ferait perdre, et "" s'il n'y a rien à perdre.
// Un clone se refait ; des modifications non validées ou des commits non poussés, non.
func travailEnPeril(d depot) string {
	if git(d.chemin, "status", "--porcelain") != "" {
		return "des modifications ne sont pas validées"
	}
	if d.origine == "" {
		return "ce dépôt n'a pas de distant, il n'existe qu'ici"
	}
	if git(d.chemin, "log", "--branches", "--not", "--remotes", "--oneline") != "" {
		return "des commits ne sont pas poussés"
	}
	return ""
}

// mettreAJour tire les dépôts nommés, ou tous s'il n'y en a aucun. `--ff-only` est
// délibéré : une mise à jour ne doit pas fabriquer un commit de fusion dans le dos de
// l'utilisateur, ni le laisser au milieu d'un conflit.
func mettreAJour(args []string) error {
	noms := sansOptions(args)
	tous := depots()

	cibles := tous
	if len(noms) > 0 {
		cibles = nil
		for _, nom := range noms {
			d, ok := trouver(tous, nom)
			if !ok {
				return fmt.Errorf("%s n'est pas cloné", nom)
			}
			cibles = append(cibles, d)
		}
	}

	var echecs []string
	for _, d := range cibles {
		if d.origine == "" {
			continue // rien à tirer
		}
		fmt.Printf("── %s\n", d.nom)
		if err := relayer("-C", d.chemin, "pull", "--ff-only"); err != nil {
			echecs = append(echecs, d.nom)
		}
	}
	if len(echecs) > 0 {
		return fmt.Errorf("mise à jour en échec : %s", strings.Join(echecs, ", "))
	}
	return nil
}

func trouver(depots []depot, nom string) (depot, bool) {
	for _, d := range depots {
		if d.nom == nom {
			return d, true
		}
	}
	return depot{}, false
}

// relayer lance git en lui laissant le terminal : c'est ce qui fait qu'une demande de
// phrase de passe ou une barre de progression arrive jusqu'à l'utilisateur.
func relayer(args ...string) error {
	c := exec.Command("git", args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}
