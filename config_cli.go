package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gitlab.yg-devworks.com/yves/jigger/internal/config"
	"gitlab.yg-devworks.com/yves/jigger/internal/i18n"
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

	// Sans drapeau : la liste, tant que l'écran n'est pas là.
	afficherReglages(fic)
	return 0
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
