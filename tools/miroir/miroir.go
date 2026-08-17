// Commande miroir : elle constate l'écart entre le dépôt GitLab et son miroir GitHub.
//
// Le miroir est poussé par GitLab, et il est déjà tombé une fois sans que rien ne le dise :
// jeton révoqué le 17 août, dernier push réussi le 16, deux versions publiées entre-temps
// que GitHub n'a jamais vues. Une panne muette n'est pas rattrapée par la surveillance de
// ce qui l'a causée — le jeton d'aujourd'hui expirera, la protection de secrets de GitHub
// refusera un autre push, quelqu'un désactivera l'entrée de miroir. On surveille donc le
// **symptôme** : les deux dépôts ne portent pas les mêmes références.
//
// Trois usages, tous à partir du même constat :
//
//	miroir                 le verdict sur la sortie standard, code 1 s'il y a un écart
//	miroir -issue          ouvre une issue GitLab s'il y a un écart, la referme sinon
//	miroir -notifier       une bannière macOS, pour le lancement local
//
// Les deux dépôts étant publics, le constat lui-même ne demande aucune authentification.
// Seul -issue a besoin d'un jeton (GARDE_FOU_TOKEN), et il le dit s'il manque.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Etat est ce qu'un dépôt porte, réduit à ce qui doit être identique des deux côtés : la
// tête de la branche par défaut, et les tags. Les branches de travail n'y sont pas — elles
// vont et viennent, et le miroir n'a pas à en rendre compte.
type Etat struct {
	Tete string
	Tags map[string]string
}

// Ecart est une référence qui ne dit pas la même chose des deux côtés. Une chaîne vide
// signifie « absente ici ».
type Ecart struct {
	Quoi   string
	GitLab string
	GitHub string
}

// Comparer rend les écarts, dans un ordre stable : la tête d'abord, puis les tags par
// ordre alphabétique. Deux exécutions à trois jours d'intervalle produisent ainsi deux
// messages comparables à l'œil.
func Comparer(gitlab, github Etat) []Ecart {
	var ecarts []Ecart

	if gitlab.Tete != github.Tete {
		ecarts = append(ecarts, Ecart{Quoi: "main", GitLab: gitlab.Tete, GitHub: github.Tete})
	}

	noms := map[string]bool{}
	for nom := range gitlab.Tags {
		noms[nom] = true
	}
	// Un tag que seul GitHub porte compte aussi : le miroir pousse sans « keep divergent
	// refs », donc rien ne devrait y vivre en propre. S'il y en a un, quelqu'un a poussé
	// à la main — ce qui est exactement ce que j'ai fait le 17 août, et qui mérite d'être
	// vu plutôt que supposé.
	for nom := range github.Tags {
		noms[nom] = true
	}

	tries := make([]string, 0, len(noms))
	for nom := range noms {
		tries = append(tries, nom)
	}
	sort.Strings(tries)

	for _, nom := range tries {
		if gitlab.Tags[nom] != github.Tags[nom] {
			ecarts = append(ecarts, Ecart{
				Quoi:   "tag " + nom,
				GitLab: gitlab.Tags[nom],
				GitHub: github.Tags[nom],
			})
		}
	}

	return ecarts
}

// Resume met les écarts en Markdown, lisible dans une issue comme dans un terminal.
func Resume(ecarts []Ecart) string {
	var b strings.Builder
	for _, e := range ecarts {
		fmt.Fprintf(&b, "- **%s** : GitLab %s, GitHub %s\n", e.Quoi, ou(e.GitLab), ou(e.GitHub))
	}
	return b.String()
}

func ou(sha string) string {
	if sha == "" {
		return "absent"
	}
	return "`" + sha + "`"
}

const (
	// Le libellé sert de mémoire : c'est à lui que la commande reconnaît son issue d'un
	// passage à l'autre, plutôt qu'à un titre qu'une relecture pourrait réécrire.
	libelle = "garde-fou::miroir"
	branche = "main"
)

func main() {
	var (
		apiGitLab = flag.String("gitlab", "https://gitlab.yg-devworks.com/api/v4/projects/25", "racine de l'API du projet GitLab")
		apiGitHub = flag.String("github", "https://api.github.com/repos/Yves848/jigger", "racine de l'API du dépôt GitHub")
		issue     = flag.Bool("issue", false, "ouvrir ou refermer une issue GitLab selon le constat (GARDE_FOU_TOKEN requis)")
		notifier  = flag.Bool("notifier", false, "afficher une notification macOS en cas d'écart")
	)
	flag.Parse()

	gitlab, err := lireGitLab(*apiGitLab)
	if err != nil {
		echouer("lecture de GitLab : %v", err)
	}
	github, err := lireGitHub(*apiGitHub)
	if err != nil {
		echouer("lecture de GitHub : %v", err)
	}

	ecarts := Comparer(gitlab, github)

	if len(ecarts) == 0 {
		fmt.Println("miroir à jour : les deux dépôts portent les mêmes références.")
	} else {
		fmt.Printf("miroir en retard — %d écart(s) :\n%s", len(ecarts), Resume(ecarts))
	}

	if *notifier && len(ecarts) > 0 {
		bannir(ecarts)
	}
	if *issue {
		if err := tenirIssue(*apiGitLab, ecarts); err != nil {
			echouer("issue : %v", err)
		}
	}

	if len(ecarts) > 0 {
		os.Exit(1)
	}
}

func echouer(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "miroir : "+format+"\n", args...)
	os.Exit(2)
}

// --- Lecture des deux dépôts -------------------------------------------------------

var client = &http.Client{Timeout: 20 * time.Second}

func lireJSON(url string, entetes map[string]string, cible any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range entetes {
		req.Header.Set(k, v)
	}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		corps, _ := io.ReadAll(io.LimitReader(r.Body, 400))
		return fmt.Errorf("%s : HTTP %d %s", url, r.StatusCode, strings.TrimSpace(string(corps)))
	}
	return json.NewDecoder(r.Body).Decode(cible)
}

func lireGitLab(racine string) (Etat, error) {
	var b struct {
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if err := lireJSON(racine+"/repository/branches/"+branche, nil, &b); err != nil {
		return Etat{}, err
	}

	var tags []struct {
		Nom    string `json:"name"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if err := lireJSON(racine+"/repository/tags?per_page=100", nil, &tags); err != nil {
		return Etat{}, err
	}

	e := Etat{Tete: court(b.Commit.ID), Tags: map[string]string{}}
	for _, t := range tags {
		e.Tags[t.Nom] = court(t.Commit.ID)
	}
	return e, nil
}

func lireGitHub(racine string) (Etat, error) {
	// L'API publique de GitHub est limitée à 60 appels par heure et par adresse. Deux
	// appels par passage : de quoi tourner toutes les heures sans jamais s'en approcher.
	entetes := map[string]string{"Accept": "application/vnd.github+json"}

	var c struct {
		SHA string `json:"sha"`
	}
	if err := lireJSON(racine+"/commits/"+branche, entetes, &c); err != nil {
		return Etat{}, err
	}

	var tags []struct {
		Nom    string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := lireJSON(racine+"/tags?per_page=100", entetes, &tags); err != nil {
		return Etat{}, err
	}

	e := Etat{Tete: court(c.SHA), Tags: map[string]string{}}
	for _, t := range tags {
		e.Tags[t.Nom] = court(t.Commit.SHA)
	}
	return e, nil
}

func court(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// --- L'issue GitLab ----------------------------------------------------------------

type issueGitLab struct {
	IID   int    `json:"iid"`
	Titre string `json:"title"`
	URL   string `json:"web_url"`
}

// tenirIssue ouvre une issue s'il y a un écart et qu'aucune n'est ouverte, la referme si
// l'écart a disparu. Elle ne rouvre pas la même issue : une panne qui revient est une
// panne neuve, et son message doit porter les sha du jour.
func tenirIssue(racine string, ecarts []Ecart) error {
	jeton := os.Getenv("GARDE_FOU_TOKEN")
	if jeton == "" {
		return fmt.Errorf("GARDE_FOU_TOKEN absent — un jeton d'API est nécessaire pour tenir l'issue")
	}

	ouvertes, err := issuesOuvertes(racine, jeton)
	if err != nil {
		return err
	}

	switch {
	case len(ecarts) == 0 && len(ouvertes) > 0:
		for _, i := range ouvertes {
			if err := fermerIssue(racine, jeton, i.IID); err != nil {
				return err
			}
			fmt.Printf("issue #%d refermée : le miroir est reparti.\n", i.IID)
		}
	case len(ecarts) > 0 && len(ouvertes) == 0:
		i, err := ouvrirIssue(racine, jeton, ecarts)
		if err != nil {
			return err
		}
		fmt.Printf("issue #%d ouverte : %s\n", i.IID, i.URL)
	case len(ecarts) > 0:
		fmt.Printf("issue #%d déjà ouverte, rien à signaler de neuf.\n", ouvertes[0].IID)
	}
	return nil
}

func issuesOuvertes(racine, jeton string) ([]issueGitLab, error) {
	var issues []issueGitLab
	u := racine + "/issues?state=opened&labels=" + url.QueryEscape(libelle)
	err := lireJSON(u, map[string]string{"PRIVATE-TOKEN": jeton}, &issues)
	return issues, err
}

func ouvrirIssue(racine, jeton string, ecarts []Ecart) (issueGitLab, error) {
	corps := "Les deux dépôts ne portent plus les mêmes références.\n\n" + Resume(ecarts) + `
Le constat ne dit pas la cause. À regarder, dans cet ordre :

1. **L'état du miroir** — Settings → Repository → Mirroring repositories. Un jeton expiré
   ou révoqué s'y voit immédiatement, c'est la panne du 17 août.
2. **La protection de secrets de GitHub** — elle refuse un push entier si un commit porte
   ce qu'elle prend pour un secret, et le miroir n'en dit rien de plus qu'« échec ».
3. **Un push local** — un tag que seul GitHub porte vient de quelqu'un qui a poussé à la
   main, pas d'une panne.

Cette issue a été ouverte par ` + "`tools/miroir`" + `, et elle se refermera d'elle-même
au premier passage où les deux dépôts se seront rejoints.`

	champs := url.Values{}
	champs.Set("title", "Le miroir GitHub a décroché")
	champs.Set("description", corps)
	champs.Set("labels", libelle+",type::interne")

	return ecrireIssue(http.MethodPost, racine+"/issues", jeton, champs)
}

// La création est un POST sur la collection, la fermeture un PUT sur l'issue : GitLab ne
// définit pas de POST sur une issue existante, et y poster rend un 404 qui ressemble à
// « issue introuvable » alors que c'est la méthode qui est fausse.
func fermerIssue(racine, jeton string, iid int) error {
	champs := url.Values{}
	champs.Set("state_event", "close")
	_, err := ecrireIssue(http.MethodPut, fmt.Sprintf("%s/issues/%d", racine, iid), jeton, champs)
	return err
}

func ecrireIssue(methode, u, jeton string, champs url.Values) (issueGitLab, error) {
	req, err := http.NewRequest(methode, u, strings.NewReader(champs.Encode()))
	if err != nil {
		return issueGitLab{}, err
	}
	req.Header.Set("PRIVATE-TOKEN", jeton)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r, err := client.Do(req)
	if err != nil {
		return issueGitLab{}, err
	}
	defer r.Body.Close()
	if r.StatusCode < 200 || r.StatusCode > 299 {
		corps, _ := io.ReadAll(io.LimitReader(r.Body, 400))
		return issueGitLab{}, fmt.Errorf("%s : HTTP %d %s", u, r.StatusCode, strings.TrimSpace(string(corps)))
	}

	var i issueGitLab
	return i, json.NewDecoder(r.Body).Decode(&i)
}

// --- La bannière macOS -------------------------------------------------------------

// bannir affiche une notification macOS. Le texte passe par osascript, qui interprète les
// guillemets doubles : ils sont donc échappés, faute de quoi un sha ne s'affiche pas et,
// pire, le script se termine là.
func bannir(ecarts []Ecart) {
	texte := fmt.Sprintf("%d référence(s) ne sont pas arrivées sur GitHub", len(ecarts))
	script := fmt.Sprintf(
		`display notification %s with title %s sound name "Basso"`,
		citer(texte), citer("jigger : le miroir a décroché"))

	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "miroir : notification impossible (%v)\n", err)
	}
}

func citer(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
