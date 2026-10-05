package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/complete"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/pm"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

// Les six mots réservés ne doivent jamais devenir des verbes de façade. Contrainte
// permanente : aucune sous-commande interne future ne peut porter le nom d'un verbe
// canonique (cf. spec §1).
func TestMotsReserves(t *testing.T) {
	attendus := []string{"pick", "render", "complete", "prompt", "warm", "demo"}
	for _, m := range attendus {
		if !motsReserves[m] {
			t.Errorf("« %s » doit être réservé", m)
		}
	}
	// Un verbe de la façade ne doit surtout pas y figurer.
	for _, v := range []string{"install", "list", "outdated", "search", "info"} {
		if motsReserves[v] {
			t.Errorf("« %s » est un verbe de façade, il ne peut pas être réservé", v)
		}
	}
}

func TestSeparerDrapeaux(t *testing.T) {
	verbe, args, o, err := separerDrapeaux(
		[]string{"install", "--pm", "scoop", "fd", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if verbe != "install" {
		t.Fatalf("verbe = %q", verbe)
	}
	if len(args) != 1 || args[0] != "fd" {
		t.Fatalf("args = %v, attendu [fd]", args)
	}
	if o.PM != "scoop" {
		t.Fatalf("PM = %q, attendu scoop", o.PM)
	}
	if !o.Yes {
		t.Fatal("--yes non pris en compte")
	}
}

func TestSeparerDrapeauxJSON(t *testing.T) {
	_, _, o, err := separerDrapeaux([]string{"outdated", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.JSON {
		t.Fatal("--json non pris en compte")
	}
}

// Un drapeau destiné au gestionnaire ne doit pas être avalé par jigger.
func TestDrapeauxInconnusPassentAuGestionnaire(t *testing.T) {
	_, args, _, err := separerDrapeaux([]string{"install", "--cask", "firefox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "--cask" || args[1] != "firefox" {
		t.Fatalf("args = %v, attendu [--cask firefox]", args)
	}
}

func TestPMSansValeur(t *testing.T) {
	if _, _, _, err := separerDrapeaux([]string{"install", "--pm"}); err == nil {
		t.Fatal("attendu une erreur : --pm sans valeur")
	}
}

// capturerStdout rend ce que f a imprimé sur la sortie standard.
func capturerStdout(t *testing.T, f func()) string {
	t.Helper()
	out, err := capturerStdoutErr(f)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// capturerStdoutErr fait le travail sans *testing.T, pour rester appelable depuis une
// goroutine. La distinction n'est pas cosmétique : t.Fatal hors de la goroutine de test
// n'interrompt pas le test, il l'égare — FailNow ne fait Goexit que sur la goroutine
// appelante, si bien que l'appelant attendrait un résultat qui ne viendra jamais, et un
// t.Fatal arrivé après la fin du test fait paniquer le binaire entier.
func capturerStdoutErr(f func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Le tube est drainé PENDANT que f écrit, et non après : son tampon vaut 4096 octets
	// sous Windows, et une écriture qui le dépasse bloque jusqu'à ce que quelqu'un lise.
	// Lire après coup était donc un interblocage dès que la sortie grossissait — ce qu'elle
	// a fait avec l'aperçu coloré de runDemo et ses 6043 octets, qui envoyait `go test ./...`
	// au timeout de dix minutes. Sous Linux le tampon fait 64 Ko, si bien que la CI ne
	// pouvait pas voir le défaut.
	type lecture struct {
		texte string
		err   error
	}
	lu := make(chan lecture, 1)
	go func() {
		out, err := io.ReadAll(r)
		lu <- lecture{string(out), err}
	}()

	ancien := os.Stdout
	os.Stdout = w
	f()
	os.Stdout = ancien

	// Fermer AVANT d'attendre : sans fin de flux, io.ReadAll ne rendrait jamais la main.
	// La fermeture a lieu même quand elle échoue, et le résultat est reçu dans tous les
	// cas : sortir sans lire laisserait la goroutine bloquée sur le tube pour la durée du
	// binaire de test.
	errFermeture := w.Close()
	res := <-lu

	if res.err != nil {
		return "", res.err
	}
	return res.texte, errFermeture
}

// Le tube d'os.Pipe fait 4096 octets sous Windows. capturerStdout écrivait tout avant de
// lire : au-delà du tampon, l'écriture bloquait et le lecteur qui l'aurait débloquée
// n'était jamais atteint — il venait après. `go test ./...` partait alors au timeout de
// dix minutes, sur l'aperçu coloré de runDemo et ses 6043 octets.
//
// Ce test lit sous délai plutôt que d'attendre : une régression doit ÉCHOUER, et non
// reproduire le blocage de dix minutes qu'on cherche précisément à supprimer.
func TestCapturerStdoutNeBloquePasSurUneGrosseSortie(t *testing.T) {
	// Au-delà du tampon de tube de TOUTES les plateformes : 4 Ko sous Windows, 16 Ko sous
	// macOS, 64 Ko sous Linux. Dimensionner sur le seul Windows rendrait ce test muet sur
	// la CI, qui tourne sous Linux — soit très exactement l'asymétrie qui a laissé le
	// défaut passer jusqu'ici.
	const taille = 1 << 17 // 128 Ko

	// os.Stdout est repris ici pour pouvoir le rendre si l'attente expire. En cas de
	// blocage, la capture le laisse branché sur un tube plein : le t.Fatalf qui suit, et
	// la ligne « FAIL » du framework, s'y bloqueraient à leur tour. Le test qui dénonce
	// l'interblocage le reproduirait au lieu de le signaler.
	ancien := os.Stdout

	type resultat struct {
		texte string
		err   error
	}
	fait := make(chan resultat, 1)
	go func() {
		texte, err := capturerStdoutErr(func() {
			fmt.Print(strings.Repeat("x", taille))
		})
		fait <- resultat{texte, err}
	}()

	select {
	case res := <-fait:
		if res.err != nil {
			t.Fatalf("la capture a échoué : %v", res.err)
		}
		if len(res.texte) != taille {
			t.Errorf("la capture a rendu %d octets, attendu %d", len(res.texte), taille)
		}
	case <-time.After(10 * time.Second):
		// Écriture concurrente assumée : la goroutine bloquée détient encore os.Stdout.
		// Ne pas le rendre coûterait le message d'échec lui-même, ce qui est pire que la
		// course — et en cas de vrai interblocage la goroutine ne le rendra jamais.
		os.Stdout = ancien
		t.Fatalf("la capture s'est bloquée sur %d octets — le tube n'est pas drainé", taille)
	}
}

// TestRenderSeTaitSansConfigurationSSH vérifie le protocole du silence, et non seulement
// le drapeau qui le porte : une sortie d'UNE SEULE ligne. C'est ce que les deux greffons
// traitent comme « rien à afficher » — `_jigger_fetch` exige deux lignes, `Get-JiggerFrame`
// aussi — et qui les fait effacer le cadre resté à l'écran. Émettre un cadre vide, comme
// avant, faisait clignoter une boîte « aucun candidat » sous chaque frappe d'une ligne ssh
// sur une machine neuve.
func TestRenderSeTaitSansConfigurationSSH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir() sous Windows

	sortie := capturerStdout(t, func() {
		runRender([]string{"--line", "ssh serv", "--color", "never"})
	})
	lignes := strings.Split(strings.TrimRight(sortie, "\n"), "\n")
	if len(lignes) != 1 {
		t.Fatalf("render a émis %d lignes, attendu la seule ligne de métadonnées :\n%s", len(lignes), sortie)
	}
	if !strings.HasPrefix(lignes[0], "count=0\t") {
		t.Errorf("métadonnées = %q", lignes[0])
	}
}

// ⏎ ne pose le candidat courant que si quelque chose l'a choisi : un mot commencé, ou le
// focus. Au mot vide sans le focus, le premier candidat par ordre alphabétique n'est la
// réponse à rien — `brew uninstall ␣⏎` désinstallait le premier paquet venu. ⇥, lui,
// insère toujours : `left` ne bouge pas.
func TestRenderEnterAuMotVide(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".ssh", "config")
	if err := os.WriteFile(cfg, []byte("Host serveur\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cas := []struct {
		ligne, focus, enter string
	}{
		{"ssh ", "false", "enter=0"},  // mot vide, sans focus : rien n'est choisi
		{"ssh ", "true", "enter=1"},   // le focus est un choix
		{"ssh s", "false", "enter=1"}, // un mot commencé aussi
	}
	for _, c := range cas {
		sortie := capturerStdout(t, func() {
			runRender([]string{"--line", c.ligne, "--focus=" + c.focus, "--color", "never"})
		})
		meta, _, _ := strings.Cut(sortie, "\n")
		if !strings.Contains(meta, "\t"+c.enter+"\t") {
			t.Errorf("%q focus=%s : métadonnées %q, attendu %s", c.ligne, c.focus, meta, c.enter)
		}
		if !strings.HasSuffix(meta, "left=ssh serveur") {
			t.Errorf("%q focus=%s : ⇥ doit toujours pouvoir insérer, métadonnées %q", c.ligne, c.focus, meta)
		}
	}
}

// Le pendant : dès que la configuration existe, le cadre revient. Sans lui, faire taire
// jigger en toutes circonstances passerait pour une correction.
func TestRenderDessineUnCadreQuandLaConfigurationExiste(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".ssh", "config")
	if err := os.WriteFile(cfg, []byte("Host serveur\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	sortie := capturerStdout(t, func() {
		runRender([]string{"--line", "ssh serv", "--color", "never"})
	})
	lignes := strings.Split(strings.TrimRight(sortie, "\n"), "\n")
	if len(lignes) < 2 {
		t.Fatalf("render n'a émis que %d ligne(s), attendu un cadre :\n%s", len(lignes), sortie)
	}
	if !strings.HasPrefix(lignes[0], "count=1\t") {
		t.Errorf("métadonnées = %q, attendu un candidat", lignes[0])
	}
}

// L'aperçu doit parler du gestionnaire qu'on lui nomme, et de lui seul.
//
// Le défaut que ce test ferme : runDemo testait `runtime.GOOS == "windows"` et retombait
// sur brew partout ailleurs. `jigger demo` annonçait donc « brew install », et listait des
// formules Homebrew, sur une machine Arch qui n'a pas de brew — le module pacman l'a rendu
// faux sans jamais le toucher, et rien ne le disait.
//
// L'assertion porte sur la CONCORDANCE — le titre commence par le mot de commande, les
// pastilles appartiennent au gestionnaire — et jamais sur une valeur choisie d'avance :
// c'est ce qui la rend vraie pour les trois plateformes, depuis n'importe laquelle.
func TestApercuSuitLeGestionnaire(t *testing.T) {
	pastilles := map[string][]string{
		"brew":   {pm.BadgeFormula, pm.BadgeCask},
		"winget": {pm.BadgeWinget, pm.BadgeOther},
		"pacman": {pm.BadgeRepo, pm.BadgeAUR},
		"yay":    {pm.BadgeRepo, pm.BadgeAUR},
	}
	for cmd, permises := range pastilles {
		titre, items := apercu(cmd)
		if !strings.HasPrefix(titre, cmd+" ") {
			t.Errorf("apercu(%q) : titre = %q, attendu qu'il commence par le mot de commande", cmd, titre)
		}
		if len(items) == 0 {
			t.Errorf("apercu(%q) : aucun candidat", cmd)
			continue
		}
		for _, it := range items {
			ok := false
			for _, b := range permises {
				if it.Badge == b {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("apercu(%q) : « %s » porte la pastille %q, étrangère à ce gestionnaire",
					cmd, it.Name, it.Badge)
			}
		}
	}
}

// Un mot que l'aperçu ne connaît pas ne doit pas rendre un cadre vide : brew reste le
// repli, comme chez managers.Default().
func TestApercuReplieSurBrew(t *testing.T) {
	titre, items := apercu("ssh")
	if titre != "brew install" || len(items) == 0 {
		t.Errorf("apercu(\"ssh\") = %q, %d candidat(s) ; attendu le repli brew", titre, len(items))
	}
}

// Le câblage, qui est l'endroit exact où le défaut se trouvait : runDemo doit montrer le
// gestionnaire de LA MACHINE. Comparé à managers.Default() et non à « brew » ou
// « pacman » — la valeur dépend de la machine qui lance les tests, la concordance non.
func TestDemoMontreLeGestionnaireDeLaMachine(t *testing.T) {
	attendu, _ := apercu(managers.Default().Cmd())
	sortie := capturerStdout(t, runDemo)
	if !strings.Contains(sortie, attendu) {
		t.Errorf("demo n'annonce pas « %s » :\n%s", attendu, sortie)
	}
}

// Les bannières des cadres doivent annoncer la version du binaire.
//
// Elles sont treize, dans six fichiers, et elles ont fait mentir la documentation quatre
// versions de suite : les deux guides et le site sont restés à « jigger 0.10.0 » de la
// v0.11.0 à la v0.14.1, l'image Open Graph à « jigger 0.9.0 ». Le retard était connu et
// signalé à chaque release ; il n'a jamais fait échouer quoi que ce soit, alors il a duré.
// C'est exactement ce qu'un contrôle vaut mieux qu'une bonne intention.
//
// Le test ne recopie pas le numéro : il le lit dans `version`, l'unique source de vérité,
// quelques centaines de lignes plus haut. Poser une nouvelle version dans main.go sans
// repasser sur les cadres fait donc échouer `make test`, en nommant chaque fichier resté
// en arrière.
func TestLesBannieresSuiventLaVersion(t *testing.T) {
	fichiers := []string{
		"README.md", "README.fr.md",
		"docs/getting-started.md", "docs/fr/getting-started.md",
		"website/index.html", "website/og.html",
	}
	// Le motif attrape n'importe quel numéro, pas seulement le bon : c'est ce qui permet
	// de DIRE ce qu'on a trouvé plutôt que de rendre un « 0 occurrence » muet.
	banniere := regexp.MustCompile(`jigger \d+\.\d+\.\d+`)
	attendu := "jigger " + version

	for _, f := range fichiers {
		contenu, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s : %v", f, err)
			continue
		}
		trouvees := banniere.FindAllString(string(contenu), -1)
		if len(trouvees) == 0 {
			// Pas une broutille : un fichier qui perd ses cadres perd la capture qui
			// montre à quoi jigger ressemble, et le test ne le dirait plus jamais.
			t.Errorf("%s : plus aucune bannière de cadre", f)
			continue
		}
		for _, v := range trouvees {
			if v != attendu {
				t.Errorf("%s : « %s » au lieu de « %s » — capture à reprendre", f, v, attendu)
			}
		}
	}
}

// La décision d'écrire, éprouvée sans terminal : c'est elle qui distingue « q » d'« esc »,
// et l'écran ne peut pas être piloté de façon fiable dans un pseudo-terminal (cf. aEcrire).
func TestAEcrire(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		ecran  ui.Configuration
		attend bool
	}{
		{"abandon avec des modifications : rien n'est écrit",
			ui.Configuration{Abandon: true, Modifs: map[string]string{"rows": "12"}}, false},
		{"abandon avec des remises : rien n'est écrit",
			ui.Configuration{Abandon: true, Retraits: []string{"rows"}}, false},
		{"enregistrement avec une modification",
			ui.Configuration{Modifs: map[string]string{"rows": "12"}}, true},
		{"enregistrement avec une remise",
			ui.Configuration{Retraits: []string{"rows"}}, true},
		{"enregistrement sans rien changer : inutile d'écrire",
			ui.Configuration{}, false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			if got := aEcrire(cas.ecran); got != cas.attend {
				t.Errorf("aEcrire = %v, attendu %v", got, cas.attend)
			}
		})
	}
}

// Un cadre vide dit ce qui a été cherché, et comment s'en sortir. « aucun candidat » ne
// disait ni l'un ni l'autre — et le disait aussi d'un motif regex qui ne compile pas.
// ^R n'est proposé que là où il agit : parmi les noms d'un catalogue, jamais pour un
// verbe ou une option, qui gardent leur préfixe dans les deux modes.
func TestEtatVide(t *testing.T) {
	t.Setenv("JIGGER_LANG", "fr")
	i18n.Recharger()
	defer i18n.Recharger()

	cas := []struct {
		nom     string
		res     complete.Result
		regex   bool
		message string
		touche  string // libellé de ^R attendu ; "" = aucune touche
	}{
		{"préfixe sans résultat", complete.Result{Word: "zzqxw", Catalogue: true}, false,
			"rien ne commence par « zzqxw »", "chercher partout"},
		{"regex sans résultat", complete.Result{Word: "(bird|fx)z", Catalogue: true}, true,
			"rien ne correspond à « (bird|fx)z »", "chercher depuis le début"},
		{"regex fautive", complete.Result{Word: "fire(", Catalogue: true}, true,
			"motif invalide : « fire( »", "chercher depuis le début"},
		{"option : pas de ^R", complete.Result{Word: "--zz"}, true,
			"rien ne commence par « --zz »", ""},
		{"mot vide", complete.Result{Catalogue: true}, false, "rien à proposer", ""},
		{"note du gestionnaire", complete.Result{Word: "fd", Catalogue: true, Note: "catalogue en préparation…"}, false,
			"catalogue en préparation…", ""},
	}
	for _, c := range cas {
		message, touches := etatVide(c.res, c.regex)
		if message != c.message {
			t.Errorf("%s : %q, attendu %q", c.nom, message, c.message)
		}
		switch {
		case c.touche == "" && len(touches) != 0:
			t.Errorf("%s : touches %v, attendu aucune", c.nom, touches)
		case c.touche != "" && (len(touches) != 1 || touches[0].Key != "^R" || touches[0].Label != c.touche):
			t.Errorf("%s : touches %v, attendu ^R %s", c.nom, touches, c.touche)
		}
	}
}
