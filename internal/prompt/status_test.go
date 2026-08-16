package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
)

// sortie réelle de `brew --version` sur une installation à jour de quelques commits.
const versionAvecCommits = "Homebrew 6.0.17-36-g6cf9b12\n"

// sortie réelle de `brew --version` pile sur un tag.
const versionSurTag = "Homebrew 4.5.4\n"

func TestParseVersion(t *testing.T) {
	cas := []struct {
		nom  string
		out  string
		want string
	}{
		{"commits depuis le tag", versionAvecCommits, "6.0.17"},
		{"pile sur un tag", versionSurTag, "4.5.4"},
		{"lignes suivantes ignorées", "Homebrew 4.5.4\nHomebrew/homebrew-core (git revision abc)\n", "4.5.4"},
		{"espaces superflus", "  Homebrew   6.0.17-1-gabc  \n", "6.0.17"},
		{"sortie vide", "", ""},
		{"sortie inattendue", "command not found\n", ""},
		{"préfixe seul", "Homebrew\n", ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ParseVersion(c.out); got != c.want {
				t.Errorf("ParseVersion(%q) = %q, attendu %q", c.out, got, c.want)
			}
		})
	}
}

// sortie réelle de `brew outdated --json=v2`, réduite à deux formulae et un cask.
const outdatedJSON = `{
  "formulae": [
    {
      "name": "binutils",
      "installed_versions": ["2.46.1"],
      "current_version": "2.47",
      "pinned": false,
      "pinned_version": null
    },
    {
      "name": "ca-certificates",
      "installed_versions": ["2026-07-16"],
      "current_version": "2026-08-13",
      "pinned": false,
      "pinned_version": null
    }
  ],
  "casks": [
    {
      "name": "font-hack-nerd-font",
      "installed_versions": ["3.4.0"],
      "current_version": "3.5.0",
      "pinned": false,
      "pinned_version": null
    }
  ]
}`

func TestParseOutdated(t *testing.T) {
	f, c, err := ParseOutdated([]byte(outdatedJSON))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if f != 2 || c != 1 {
		t.Errorf("ParseOutdated = %d formulae, %d casks ; attendu 2 et 1", f, c)
	}
}

func TestParseOutdatedToutAJour(t *testing.T) {
	f, c, err := ParseOutdated([]byte(`{"formulae":[],"casks":[]}`))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if f != 0 || c != 0 {
		t.Errorf("ParseOutdated = %d/%d, attendu 0/0", f, c)
	}
}

func TestParseOutdatedJSONInvalide(t *testing.T) {
	if _, _, err := ParseOutdated([]byte("Error: brew introuvable")); err == nil {
		t.Error("un JSON invalide doit remonter une erreur")
	}
}

func TestLigneAllerRetour(t *testing.T) {
	s := Status{Version: "6.0.17", Primary: 7, Secondary: 2, At: time.Unix(1755100000, 0)}

	line := s.Line()
	if want := "6.0.17\t7\t2\t1755100000"; line != want {
		t.Fatalf("Line() = %q, attendu %q", line, want)
	}

	got, ok := ParseLine(line)
	if !ok {
		t.Fatal("ParseLine refuse une ligne qu'on vient d'écrire")
	}
	if got.Version != s.Version || got.Primary != s.Primary || got.Secondary != s.Secondary {
		t.Errorf("aller-retour altéré : %+v", got)
	}
	if !got.At.Equal(s.At) {
		t.Errorf("horodatage altéré : %v", got.At)
	}
}

func TestParseLigneRejetteLesMalformees(t *testing.T) {
	cas := map[string]string{
		"vide":                "",
		"tronquée":            "6.0.17\t7",
		"compteur non entier": "6.0.17\tsept\t2\t1755100000",
		"horodatage absent":   "6.0.17\t7\t2\t",
		"champ en trop":       "6.0.17\t7\t2\t1755100000\t?",
	}
	for nom, line := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, ok := ParseLine(line); ok {
				t.Errorf("ParseLine(%q) devrait échouer", line)
			}
		})
	}
}

func TestOutdatedEstLaSomme(t *testing.T) {
	if got := (Status{Primary: 7, Secondary: 2}).Outdated(); got != 9 {
		t.Errorf("Outdated() = %d, attendu 9", got)
	}
}

func TestReadCacheAbsent(t *testing.T) {
	if _, ok := Read(t.TempDir()); ok {
		t.Error("Read doit échouer proprement quand le cache n'existe pas")
	}
}

func TestReadIgnoreUnCacheCorrompu(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(File(dir), []byte("n'importe quoi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read(dir); ok {
		t.Error("un cache illisible doit être traité comme absent")
	}
}

// fauxBrew renvoie un Runner qui imite `brew --version` et `brew outdated --json=v2`.
func fauxBrew(version, outdated string, err error) Runner {
	return func(args ...string) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		if len(args) > 0 && args[0] == "--version" {
			return []byte(version), nil
		}
		return []byte(outdated), nil
	}
}

func TestRefreshEcritLeCacheEtLeRelit(t *testing.T) {
	dir := t.TempDir()

	s, err := RefreshWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil))
	if err != nil {
		t.Fatalf("RefreshWith : %v", err)
	}
	if s.Version != "6.0.17" || s.Primary != 2 || s.Secondary != 1 {
		t.Fatalf("état inattendu : %+v", s)
	}
	if time.Since(s.At) > time.Minute {
		t.Errorf("horodatage non renseigné : %v", s.At)
	}

	relu, ok := Read(dir)
	if !ok {
		t.Fatal("le cache écrit n'est pas relisible")
	}
	if relu.Version != s.Version || relu.Outdated() != s.Outdated() {
		t.Errorf("relecture divergente : %+v vs %+v", relu, s)
	}
}

func TestRefreshNeLaissePasDeFichierTemporaire(t *testing.T) {
	dir := t.TempDir()
	if _, err := RefreshWith(dir, fauxBrew(versionSurTag, `{"formulae":[],"casks":[]}`, nil)); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(File(dir)) {
			t.Errorf("résidu dans le cache : %s", e.Name())
		}
	}
}

func TestRefreshPreserveLeCacheQuandBrewEchoue(t *testing.T) {
	dir := t.TempDir()
	if _, err := RefreshWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil)); err != nil {
		t.Fatal(err)
	}

	if _, err := RefreshWith(dir, fauxBrew("", "", errors.New("brew absent"))); err == nil {
		t.Error("l'échec de brew doit remonter une erreur")
	}

	// Le prompt continue d'afficher la dernière valeur connue.
	s, ok := Read(dir)
	if !ok || s.Version != "6.0.17" || s.Outdated() != 3 {
		t.Errorf("le cache précédent a été perdu : %+v (ok=%v)", s, ok)
	}
}

func TestRefreshRenonceSiUnVerrouEstDejaPris(t *testing.T) {
	dir := t.TempDir()

	libere, ok := lock(dir)
	if !ok {
		t.Fatal("le premier verrou devrait être obtenu")
	}
	defer libere()

	if _, err := RefreshWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil)); !errors.Is(err, ErrVerrouille) {
		t.Errorf("erreur = %v, attendu ErrVerrouille", err)
	}
	if _, ok := Read(dir); ok {
		t.Error("aucun cache ne doit avoir été écrit")
	}
}

// ErrVerrouille ressort à l'utilisateur derrière cli.prompt_failed : elle doit parler sa
// langue — « jigger prompt --refresh: rafraîchissement déjà en cours » était la phrase
// hybride d'avant — sans cesser d'être la sentinelle que errors.Is compare.
func TestErrVerrouilleParleLaLangueEtResteComparable(t *testing.T) {
	t.Cleanup(i18n.Recharger)

	dir := t.TempDir()
	libere, ok := lock(dir)
	if !ok {
		t.Fatal("le premier verrou devrait être obtenu")
	}
	defer libere()

	for langue, attendu := range map[string]string{
		"en": "a refresh is already running",
		"fr": "rafraîchissement déjà en cours",
	} {
		t.Setenv("JIGGER_LANG", langue)
		i18n.Recharger()

		_, err := RefreshWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil))
		if !errors.Is(err, ErrVerrouille) {
			t.Fatalf("%s : erreur = %v, attendu ErrVerrouille", langue, err)
		}
		if err.Error() != attendu {
			t.Errorf("%s : message = %q, attendu %q", langue, err.Error(), attendu)
		}
	}
}

func TestRefreshWaitAttendQueLeVerrouSeLibere(t *testing.T) {
	dir := t.TempDir()

	// Le cas réel : un rafraîchissement paresseux est parti pendant `brew upgrade` et
	// tient encore le verrou quand le shell, upgrade terminé, force la mise à jour.
	libere, ok := lock(dir)
	if !ok {
		t.Fatal("le premier verrou devrait être obtenu")
	}
	go func() {
		time.Sleep(3 * intervalleVerrou)
		libere()
	}()

	s, err := RefreshWaitWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil), AttenteVerrou)
	if err != nil {
		t.Fatalf("RefreshWaitWith : %v", err)
	}
	if s.Primary != 2 || s.Secondary != 1 {
		t.Errorf("état inattendu : %+v", s)
	}
	if _, ok := Read(dir); !ok {
		t.Error("le cache doit avoir été écrit")
	}
}

func TestRefreshWaitAbandonneApresSonDelai(t *testing.T) {
	dir := t.TempDir()

	libere, ok := lock(dir)
	if !ok {
		t.Fatal("le premier verrou devrait être obtenu")
	}
	defer libere()

	debut := time.Now()
	_, err := RefreshWaitWith(dir, fauxBrew(versionAvecCommits, outdatedJSON, nil), 2*intervalleVerrou)
	if !errors.Is(err, ErrVerrouille) {
		t.Errorf("erreur = %v, attendu ErrVerrouille", err)
	}
	// Il a bien attendu — sans quoi le test précédent passerait pour de mauvaises raisons.
	if ecoule := time.Since(debut); ecoule < 2*intervalleVerrou {
		t.Errorf("abandon au bout de %v, sans avoir attendu son délai", ecoule)
	}
	if _, ok := Read(dir); ok {
		t.Error("aucun cache ne doit avoir été écrit")
	}
}

func TestLeVerrouSeLibere(t *testing.T) {
	dir := t.TempDir()

	libere, ok := lock(dir)
	if !ok {
		t.Fatal("premier verrou refusé")
	}
	libere()

	if _, ok := lock(dir); !ok {
		t.Error("le verrou libéré doit pouvoir être repris")
	}
}

func TestUnVerrouPerimeEstIgnore(t *testing.T) {
	dir := t.TempDir()

	// Un `--refresh` tué laisse son verrou derrière lui : au-delà de peremptionVerrou,
	// il ne doit plus bloquer le rafraîchissement pour toujours.
	if _, ok := lock(dir); !ok {
		t.Fatal("premier verrou refusé")
	}
	vieux := time.Now().Add(-peremptionVerrou - time.Minute)
	if err := os.Chtimes(verrou(dir), vieux, vieux); err != nil {
		t.Fatal(err)
	}

	if _, ok := lock(dir); !ok {
		t.Error("un verrou périmé doit être ignoré")
	}
}

func TestRefreshToleraUneVersionIllisible(t *testing.T) {
	// brew peut répondre n'importe quoi à --version ; le compteur, lui, reste utile.
	dir := t.TempDir()
	s, err := RefreshWith(dir, fauxBrew("???\n", outdatedJSON, nil))
	if err != nil {
		t.Fatalf("RefreshWith : %v", err)
	}
	if s.Version != "" || s.Outdated() != 3 {
		t.Errorf("état inattendu : %+v", s)
	}
	if _, ok := Read(dir); !ok {
		t.Error("le cache doit tout de même être écrit")
	}
}

func TestFileEstDansLeDossierDonne(t *testing.T) {
	dir := t.TempDir()
	if got := File(dir); !strings.HasPrefix(got, dir+string(os.PathSeparator)) {
		t.Errorf("File(%q) = %q, hors du dossier", dir, got)
	}
}
