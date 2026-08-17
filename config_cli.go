package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"gitlab.yg-devworks.com/yves/jigger/internal/config"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
	"gitlab.yg-devworks.com/yves/jigger/internal/managers"
	"gitlab.yg-devworks.com/yves/jigger/internal/ui"
)

// runConfig sert la sous-commande `jigger config`.
//
//	jigger config --export [--shell zsh|powershell]   lignes à évaluer par le greffon
//	jigger config --path                              chemin du fichier
//	jigger config --list                              tous les réglages, valeur et provenance
//	jigger config                                     l'écran (à venir)
//
// L'export est ce que les greffons évaluent au chargement : c'est le seul chemin par
// lequel un réglage du fichier atteint le shell (ADR-0003).
func runConfig(argv []string) int {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	export := fs.Bool("export", false, i18n.T("cli.flag_export"))
	shell := fs.String("shell", "", i18n.T("cli.flag_shell"))
	chemin := fs.Bool("path", false, i18n.T("cli.flag_config_path"))
	lister := fs.Bool("list", false, i18n.T("cli.flag_list"))
	if err := fs.Parse(argv); err != nil {
		return 2
	}

	if *chemin {
		p, err := config.Chemin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(p)
		return 0
	}

	// Un fichier illisible ne doit jamais empêcher un shell de s'ouvrir : on repart d'une
	// configuration vide, les défauts s'appliquent, et on le dit sur stderr — que le
	// greffon n'évalue pas.
	fic, err := config.Charger()
	if err != nil {
		fmt.Fprint(os.Stderr, i18n.Tf("cfg.unreadable", err))
		fic = config.Nouveau()
	}

	switch {
	case *export:
		sh := config.Zsh
		if strings.EqualFold(*shell, "powershell") || strings.EqualFold(*shell, "pwsh") {
			sh = config.PowerShell
		}
		fmt.Print(config.Export(fic, sh))
		return 0

	case *lister:
		afficherReglages(fic)
		return 0
	}

	// Sans drapeau : l'écran, s'il y a un terminal pour le porter. Sinon la liste —
	// `jigger config | grep` doit rester utilisable.
	if !sortieEstTerminal() {
		afficherReglages(fic)
		return 0
	}
	return ecranConfig(fic)
}

// ecranConfig ouvre l'écran, puis enregistre ce qu'il a changé. L'écran ne touche jamais au
// fichier lui-même : il rend des modifications, et c'est ici qu'elles sont écrites.
func ecranConfig(fic *config.Fichier) int {
	tty, err := openTTY()
	if err != nil {
		afficherReglages(fic)
		return 0
	}
	defer tty.Close()
	lipgloss.SetColorProfile(termenv.NewOutput(tty.Out).Profile)

	modele := ui.NouvelleConfiguration(groupesConfig(fic))
	prog := tea.NewProgram(modele, tea.WithInput(tty.In), tea.WithOutput(tty.Out), tea.WithAltScreen())
	final, err := prog.Run()
	if err != nil {
		return 1
	}

	c, ok := final.(ui.Configuration)
	if !ok || (len(c.Modifs) == 0 && len(c.Retraits) == 0) {
		return 0
	}
	for cle, valeur := range c.Modifs {
		fic.Poser(cle, valeur)
	}
	for _, cle := range c.Retraits {
		fic.Retirer(cle)
	}
	if err := fic.Ecrire(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	chemin, _ := config.Chemin()
	fmt.Print(i18n.Tf("cfg.written", chemin))
	return 0
}

// groupesConfig compose les trois groupes de la spec §2. Ils correspondent à trois
// natures — ce qui prend effet tout de suite, ce qui attend le prochain shell, et ce que
// jigger observe sans le posséder — et non à un classement esthétique.
func groupesConfig(fic *config.Fichier) []ui.GroupeConfig {
	var maintenant, shell []ui.LigneConfig
	for _, r := range config.Declares {
		valeur, prov := config.Resoudre(os.Getenv(r.Env()), fic.Valeur(r.Cle), r.Defaut)
		if valeur == "" {
			valeur = "—"
		}
		ligne := ui.LigneConfig{
			Cle: r.Cle, Env: r.Env(), Valeur: valeur,
			Provenance: provenanceLisible(prov), Description: i18n.T(r.CleI18n),
		}
		if r.Portee == config.Greffon {
			shell = append(shell, ligne)
		} else {
			maintenant = append(maintenant, ligne)
		}
	}

	return []ui.GroupeConfig{
		{Titre: i18n.T("cfg.grp_now"), Note: i18n.T("cfg.now"), Lignes: maintenant},
		{Titre: i18n.T("cfg.grp_shell"), Note: i18n.T("cfg.next_shell"), Lignes: shell},
		{Titre: i18n.T("cfg.grp_seen"), Note: i18n.T("cfg.readonly"), Lignes: observations()},
	}
}

// observations rend ce que jigger voit de l'installation, en lecture seule : ces valeurs
// appartiennent aux gestionnaires, pas à jigger. Les proposer à la modification serait
// mentir sur ce qu'un changement produirait.
func observations() []ui.LigneConfig {
	var out []ui.LigneConfig
	for _, v := range []string{"SCOOP", "SCOOP_GLOBAL", "HOMEBREW_PREFIX"} {
		valeur := os.Getenv(v)
		if valeur == "" {
			valeur = "—"
		}
		out = append(out, ui.LigneConfig{Env: v, Valeur: valeur, Provenance: "env", Fige: true})
	}
	var noms []string
	for _, m := range managers.Available() {
		noms = append(noms, m.Cmd())
	}
	dispo := strings.Join(noms, ", ")
	if dispo == "" {
		dispo = "—"
	}
	out = append(out, ui.LigneConfig{Env: "managers", Valeur: dispo, Provenance: "auto", Fige: true})
	return out
}

// afficherReglages imprime chaque réglage, sa valeur effective et sa provenance. La
// provenance n'est pas décorative : sans elle, on lirait une valeur du fichier pendant que
// la machine en applique une autre, venue de l'environnement (ADR-0003).
func afficherReglages(fic *config.Fichier) {
	var lignes [][]string
	for _, r := range config.Declares {
		valeur, prov := config.Resoudre(os.Getenv(r.Env()), fic.Valeur(r.Cle), r.Defaut)
		if valeur == "" {
			valeur = "—"
		}
		lignes = append(lignes, []string{r.Env(), valeur, provenanceLisible(prov), i18n.T(r.CleI18n)})
	}

	largeurs := make([]int, 3)
	for _, l := range lignes {
		for i := range 3 {
			if n := len([]rune(l[i])); n > largeurs[i] {
				largeurs[i] = n
			}
		}
	}
	for _, l := range lignes {
		fmt.Printf("%-*s  %-*s  %-*s  %s\n",
			largeurs[0], l[0], largeurs[1], l[1], largeurs[2], l[2], l[3])
	}
}

func provenanceLisible(p config.Provenance) string {
	switch p {
	case config.DeLEnvironnement:
		return i18n.T("cfg.from_env")
	case config.DuFichier:
		return i18n.T("cfg.from_file")
	default:
		return i18n.T("cfg.from_default")
	}
}
