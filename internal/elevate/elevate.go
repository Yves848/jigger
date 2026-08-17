// Package elevate rejoue une commande avec les privilèges d'administrateur.
//
// Il n'est appelé qu'après un constat — un code de sortie qui a dit « il faut être
// administrateur » (cf. ADR-0004) — et jamais de sa propre initiative : c'est l'appelant
// qui a demandé un oui explicite à l'utilisateur avant d'arriver ici.
//
// Tout est dans les fichiers par plateforme. Ce fichier-ci ne porte que le vocabulaire
// commun, pour que la façade et `main` s'écrivent sans `//go:build`.
package elevate

import "errors"

var (
	// ErrRefuse : l'utilisateur a refusé l'invite d'élévation du système (UAC annulé).
	// Ce n'est pas une panne, c'est une réponse — et l'appelant doit la distinguer d'un
	// échec pour se taire plutôt que de crier.
	ErrRefuse = errors.New("élévation refusée")
	// ErrIndisponible : la plateforme ne sait pas élever.
	ErrIndisponible = errors.New("élévation indisponible sur cette plateforme")
)

// Voie dit par quel chemin le rejeu passera. Elle se demande **avant** de poser la
// question à l'utilisateur : lui annoncer qu'une autre fenêtre va s'ouvrir, ou que tout
// restera là où il est, fait partie de ce qu'il accepte.
type Voie int

const (
	// VoieAucune : jigger ne sait pas élever sur cette plateforme. C'est le cas partout
	// sauf sous Windows — la moitié Unix d'A-15 reste ouverte, et elle ne se traitera pas
	// par un rejeu (cf. la spec, §7).
	VoieAucune Voie = iota
	// VoieSudo : le `sudo` de Windows est présent et activé. Selon le mode qu'il a reçu,
	// l'élévation peut rester dans la console courante.
	VoieSudo
	// VoieFenetre : `ShellExecuteEx` et le verbe `runas`, c'est-à-dire ce que fait
	// `Start-Process -Verb RunAs`. Windows ouvre une console élevée **séparée** : un
	// processus élevé ne peut pas s'attacher à la console d'un processus qui ne l'est
	// pas. C'est une frontière du système, pas un manque d'application.
	VoieFenetre
)

// Possible est le raccourci dont se sert l'appelant pour savoir s'il a le droit de
// proposer quoi que ce soit.
func Possible() bool { return Prevue() != VoieAucune }
