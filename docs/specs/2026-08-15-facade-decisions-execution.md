# Façade, phase 1 — décisions prises pendant l'exécution

15 août 2026 — compagnon de [la spec](2026-08-15-facade-multi-gestionnaires-design.md) et du
[plan](../plans/2026-08-15-facade-phase-1.md)

## Pourquoi ce document

L'exécution du plan a demandé vingt décisions qui n'étaient pas dans la spec : des défauts du
plan découverts en codant, des promesses qu'aucune tâche ne tenait, des tests qui ne
prouvaient pas ce qu'ils annonçaient. Chacune a été prise sur le moment pour ne pas bloquer,
et chacune est révisable. Ce document les rassemble pour qu'elles soient discutables plutôt
qu'enfouies dans un historique git.

Elles sont classées par gravité, pas par ordre chronologique.

## Trois décisions qui ont changé du code que le plan disait intouchable

**H — `internal/scoop/outdated.go` a été modifié, par extraction pure.** Le plan supposait
une fonction rendant la liste des applications obsolètes ; le fichier n'exposait que des
compteurs, tout en calculant puis jetant le nom, la version installée, la version du manifeste
et le bucket. Ajout de `AppObsolete` et `OutdatedAppsIn` ; `OutdatedIn` se réduit à un `len`.
Les signatures publiques sont préservées à l'identique, et `scoop_test.go` passe sans
modification — c'est la preuve de non-régression.
*Si c'est faux :* le bloc oh-my-posh dépend d'`Outdated()`. Le test inchangé le couvre.

**P — `Manager.Insert` n'était branché nulle part dans la façade.** La spec le promettait en
note de bas de table. Sans lui, `jg install <un cask>` lançait `brew install <cask>`, que brew
refuse. Une tâche 17, absente du plan, a été ajoutée : un adaptateur traduit la sortie de
`Insert` — un texte de ligne shell — en éléments d'argv, conscient des guillemets.
*Si c'est faux :* un point de traduction de plus, avec ses cas tordus. Les trois cas réels
sont testés sur table.

**N — les drapeaux natifs ne traversaient pas le routage.** `jg install --cask firefox`
échouait sur « `--cask` inconnu de brew » : `Router` traitait chaque argument comme un nom de
paquet. Il partitionne désormais sur le `-` initial.
*Si c'est faux :* un paquet dont le nom commence par `-` serait irrésoluble. Aucun n'existe.

## Deux bugs critiques trouvés par la revue finale, invisibles aux revues par tâche

**C1 — `jg search` refusait de chercher.** Les trois tables déclaraient `search` avec
`Pool: PoolCatalogue`, donc la requête était résolue comme un nom de paquet exact. La commande
listait des voisins proches pour expliquer qu'elle ne trouvait rien. Corrigé en `PoolAucun` :
la requête part à tous les gestionnaires capables.

**C2 — la sortie de scoop était capturée puis jetée.** `list`, `search` et `source` chez scoop
sont des liaisons `Native` sans `Parse`, et un verbe normalisé capture au lieu de relayer :
les octets partaient à la poubelle, `reussites++`, code 0. Sous Windows, `jg list` aurait omis
toutes les applications scoop, en silence. Deux corrections : un garde-fou générique dans
`Executer` qui rend le silence impossible pour *toute* liaison de cette forme, et
`internal/scoop/parse.go`.

Ces deux-là sont de la même famille que P : des promesses de la spec qu'aucune tâche du plan
n'a tenues. Un plan détaillé ne protège pas de ça ; seule une relecture de la branche entière
les a vus.

## La réserve qui reste ouverte

**S — les parsers `search` et `source` de scoop ne sont pas fiables.** Ils ont été écrits sans
scoop et sans jeu d'essai réel, contre un format de sortie qui s'avère obsolète : sections
`'main' bucket:` là où scoop émet aujourd'hui un tableau. Sur une vraie machine Windows, ils
rendront zéro ligne **sans planter** — exactement le symptôme que C2 devait supprimer, et le
garde-fou ne l'attrape pas, puisqu'un parser qui ne reconnaît rien satisfait son contrat.

Décision : ne pas tenter une troisième version à l'aveugle. Le travail est marqué
« NON VÉRIFIÉ » dans le code et dans ses jeux d'essai. **C'est le premier point à traiter lors
de la passe Windows.**

## Décisions de portée

**W, R — la vérification Windows est reportée.** `winget` et `scoop` n'existent pas sur la
machine de développement. La tâche 1 n'a vérifié que la colonne brew ; les tables winget et
scoop portent un avertissement en en-tête. La moitié zsh de la tâche 14 a en revanche été
faite ici — le report initial de la tâche entière était trop large, et sans l'alias zsh la
façade était inutilisable au clavier sur le Mac. Seuls `jigger.psm1` et `tests/smoke.ps1`
restent.

**E — la branche part de `docs/conception-facade`**, pas de `main`, pour que la spec et le plan
soient présents pendant l'implémentation.

**T — une seule issue GitLab pour la branche**, pas trente-cinq. La branche est un livrable ;
trente-cinq issues `type::interne` sur des types et des tables noieraient la seule ligne qui
intéresse un lecteur de notes de version.

## Corrections de tests qui ne prouvaient rien

**A** — le jeu d'essai de `TestRoutagePoolInstalles` n'installait pas le paquet que le test
prétendait router. **L** — `TestLectureEchoueSiPersonneNeRepond` employait `outdated`, dont la
liaison scoop est `Direct` et ne peut donc pas échouer par le point d'injection ; le verbe est
passé à `list`. **J** — `voisins` ignorait son paramètre `pool`, masqué par une variable
locale : `jg uninstall <faute>` suggérait un paquet non installé.

## Décisions mineures

**B** — `{arg}` sans argument produit un argv malformé ; accepté, `Router` garantit des
arguments non vides pour les verbes concernés. **C** — un test d'alignement fragile, laissé
tel quel. **D** — certains tests dépendent de `managers.Available()`, donc de la machine.
**F** — les briefs sont extraits par un script maison, le script du skill ne lisant que
« Task N ». **G** — les tâches 3 et 4 ont été regroupées. **I** — pas de sentinelle
`ErrVerbeInconnu` : aucun appelant n'en a besoin. **K** — pas d'alias `errorsAs`.
**M** — `facade.Normalise`, pas `facadeNormalise`. **O** — étiquettes JSON en minuscules sur
`pm.Package`. **Q** — le surcoût du popup en façade est diagnostiqué (un tri complet là où une
fusion à trois voies suffirait) mais non optimisé : 3,9 ms sur un budget de ~8 ms, sur un jeu
d'essai volontairement défavorable.

## Ce qui reste à faire

1. **La passe Windows** — vérifier les tables winget et scoop contre les vraies CLI, refaire
   les parsers `search` et `source` de scoop contre de vraies sorties, poser `Set-Alias jg` et
   étendre `JIGGER_COMMANDS` dans `jigger.psm1`, ajouter le cas correspondant à `smoke.ps1`.
2. Les mineurs différés, listés dans la MR.
