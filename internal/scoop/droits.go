package scoop

// Pourquoi scoop n'implémente pas pm.Elevateur — et pourquoi ce fichier ne contient rien
// d'autre que cette explication.
//
// L'issue #169 tenait scoop pour « le trou le moins cher à combler » : il tourne sur la
// seule plateforme où internal/elevate fonctionne, et tout le mécanisme d'A-15 est déjà
// écrit pour winget. Il ne manquait, croyait-on, que la table des codes. La mesure a dit
// l'inverse — même discipline que l'ADR-0004, où deux mesures sur trois ont retourné le
// résultat attendu.
//
// Mesuré le 6 septembre 2026 sur Windows 11, scoop 0.5.3, depuis une tâche planifiée en
// `RunLevel Limited` — donc un jeton réellement non élevé, vérifié par `whoami` dans la
// même tâche :
//
//	scoop install -g <app>   « you need admin rights to install global apps »   → 0
//	scoop install            « <app> missing »                                  → 0
//	scoop pas-une-commande   « isn't a scoop command »                           → 1
//	cmd /c exit 3            témoin du dispositif de mesure                      → 3
//
// Le témoin n'est pas décoratif : sans lui, les deux premiers 0 ne prouveraient rien de
// plus qu'une mesure fausse.
//
// La cause est dans le lanceur de scoop (`bin/scoop.ps1`) : il répartit les sous-commandes
// par `exec $subCommand`, et l'`exit 1` d'`abort` (`lib/core.ps1`) s'exécute alors dans la
// portée du script fils. Le lanceur poursuit son `switch` et se termine à 0. Le seul
// `exit` de sa propre portée est la branche `default`, celle du nom de sous-commande
// inconnu — d'où le 1 isolé de la troisième ligne, qui est ce qui rend l'explication
// vérifiable plutôt que plausible.
//
// Trois conséquences, dans l'ordre où elles ferment la question :
//
//  1. Il n'y a rien à constater. Le défaut de droits ne se distingue d'aucun autre échec,
//     ni même d'une réussite. Un `Droits(code int)` ne pourrait que mentir, et il mentirait
//     en silence — exactement la panne muette que le piège du DWORD avait failli produire
//     chez winget.
//  2. Il n'y a pas de contre-cas. Rien dans scoop ne refuse l'élévation, là où deux des
//     quatre codes de winget le font. pm.DroitsInterdits serait inatteignable ici.
//  3. Le cas ne se présente même pas par la façade : Verbs n'émet jamais `-g` ni
//     `--global`. Seule une ligne tapée à la main peut le produire, et le chemin popup
//     n'élève sur aucune plateforme (#167, #169).
//
// Que le code observé soit toujours 0, y compris sur les échecs sans rapport avec les
// droits, dépasse la question de l'élévation : c'est l'issue #171.
//
// Si scoop venait un jour à propager ses codes, c'est cette mesure qu'il faudrait refaire
// avant d'écrire quoi que ce soit ici — pas ce commentaire qu'il faudrait croire.
