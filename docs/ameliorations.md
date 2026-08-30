# Améliorations et modifications à venir

Le tableau des évolutions proposées, tenu à la façon d'un Kanban : on y prend ce qu'on
veut, quand on veut, sans suivre l'ordre d'arrivée.

## Comment il se tient

- **Un numéro** à la création — `A-1`, `A-2`… —, attribué une fois pour toutes et **jamais
  réutilisé**, même après réalisation ou abandon. C'est lui qu'on cite en conversation et
  dans les messages de commit.
- **Une priorité**, `à déterminer` par défaut. Elle ne se décide pas au moment de la
  proposition : on note d'abord, on arbitre ensuite.
- **Un état**, matérialisé par la section où vit l'entrée : `À faire`, `En cours`, `Fait`.
  Une entrée réalisée **garde sa ligne** et descend dans `Fait` avec son commit — le
  tableau raconte alors aussi ce qui a été construit.
- **Une provenance**, quand elle éclaire : une réserve publiée, un constat de relecture, une
  idée en passant. Dans six mois, elle dira si l'entrée pèse encore.

Le préfixe `A-` évite toute confusion avec les numéros d'issues GitLab (`#39`), qui
consignent ce qui a **déjà** changé et alimentent les notes de release. Ce document-ci
porte l'inverse : ce qui n'est pas encore fait.

---

## À faire

### A-25 — Un sélecteur de serveurs SSH

**Priorité :** à déterminer · **Provenance :** demande du 30 août

Proposer les serveurs connus dès qu'on tape `ssh`, comme jigger propose déjà les formules
dès qu'on tape `brew` — même popup, mêmes touches, même greffon. `scp` et `sftp` suivent, à
ceci près que leur cible s'écrit `hôte:`.

Ce n'est pas un gestionnaire de paquets, et c'est tout l'intérêt de l'entrée :
[l'ADR-0005](adr/0005-completion-sans-facade.md) tranche que le contrat de complétion
`pm.Manager` n'est pas réservé aux gestionnaires, et qu'un fournisseur peut l'implémenter
sans jamais implémenter `pm.Bindings`. La façade reste un vocabulaire de gestion de
paquets ; la complétion, non.

Conception complète : [spec du 30 août](specs/2026-08-30-selecteur-ssh-design.md).

### A-2 — Prouver `cleanup *` et `bucket rm` contre scoop

**Priorité :** à déterminer · **Provenance :** réserve publiée avec la v0.8.0

Les captures du 16 août ont vérifié la table des verbes scoop sur `scoop help` et
`scoop update --help` — mais deux liaisons restent d'usage courant et non prouvées : le `*`
de `cleanup *` et le nom du sous-verbe de suppression de bucket. `scoop cleanup --help` et
`scoop bucket --help` trancheraient, et le script de capture peut les prendre.

### A-3 — Exercer le garde PowerShell 5.1

**Priorité :** à déterminer · **Provenance :** relecture de la tâche 7 de l'i18n

`$PSVersionTable.PSVersion.Major -lt 6 -or $IsWindows` n'a jamais tourné sur un vrai
Windows PowerShell 5.1 — seulement sur pwsh 7. Le même garde existe à deux endroits du
module ; le risque est mutualisé, mais il n'est pas nul.

### A-4 — Les descriptions de touches PSReadLine ne sont pas traduites

**Priorité :** à déterminer · **Provenance :** relecture finale de l'i18n (constat parqué)

Les trois `-Description` de `Register-JiggerKey` sont passées de « français en dur » à
« anglais en dur » plutôt que par `Get-JiggerText`, et le commentaire du code affirme le
contraire. Visible seulement par `Get-PSReadLineKeyHandler` ou `Ctrl+Alt+?`.

### A-5 — Le banc de rendu ne tient qu'à cette machine

**Priorité :** à déterminer · **Provenance :** relecture finale de l'i18n

`tests/golden/render-fr.txt` embarque le numéro de version et les vrais noms du cache
Homebrew local. Il est sorti de `make test-all` pour cette raison, et reste lançable à la
main. Le rendre portable demanderait de neutraliser la bannière **et** de figer un catalogue
d'essai.

### A-12 — Trier les colonnes du sélecteur plein écran

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Dépend de :** A-10

Pouvoir trier la liste sur la colonne de son choix, et inverser l'ordre.

**Il n'y a pas encore de colonnes à trier**, et c'est le premier constat à faire. Le
sélecteur affiche aujourd'hui une liste d'items (`pm.Item` : nom, badge, installé, version,
gestionnaire) dont seuls le badge, le nom et — quand plusieurs gestionnaires répondent — le
code PM sont rendus. Les vraies colonnes (`PACKAGE`, `CURRENT`, `AVAILABLE`, `SOURCE`, `PM`)
vivent dans les tableaux de la façade, qui ne passent pas par le sélecteur. Le tri prend
donc son sens **après** A-10, quand ces tableaux s'afficheront dans le sélecteur.

À trancher le moment venu :

- **Les touches.** Rien n'est libre par hasard dans ce cadre : ⇥, ↩, ↑↓, ^N/^P, ^G et esc
  sont pris, et les caractères imprimables vont au filtre. Le tri devra passer par une
  combinaison qui ne mange pas la frappe — et se lire dans le pied, comme le reste.
- **Trier une version n'est pas trier du texte.** `0.10.0` précède `0.9.0` dans l'ordre
  lexicographique, et c'est faux. Un tri sur `CURRENT` ou `AVAILABLE` demande une
  comparaison qui comprenne les numéros ; jigger n'en a pas encore.
- **Dans le popup d'insertion, l'ordre a des conséquences.** Le premier candidat est celui
  que `⇥` insère : trier n'y est pas un confort d'affichage mais un changement de ce que
  fait la touche. En lecture seule (A-10), la question ne se pose pas.

### A-13 — Colorer les versions obsolètes

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Voisine de :** A-10

Faire ressortir à l'œil ce qui est en retard : la version installée et celle qui l'attend,
distinguées par la couleur plutôt que laissées à la lecture de deux colonnes alignées.

Le point de départ, vérifié : **les tableaux de la façade ne portent aujourd'hui aucune
couleur.** `internal/facade/format.go` aligne des colonnes en texte nu. Le popup, lui, est
coloré via lipgloss, avec une palette déjà posée — c'est d'elle qu'il faudra partir pour ne
pas inventer un second jeu de teintes.

Trois points à traiter :

- **La couleur ne doit jamais partir dans un tuyau.** Même règle que la pagination (A-10) :
  elle ne s'arme que si la sortie est un terminal, et `--json` n'en porte jamais. Précédent
  utile dans le dépôt : jigger ne devine pas son profil couleur pour le popup, c'est le
  shell qui le lui passe (`--color auto|never|16|256|truecolor`) parce que la sortie est
  capturée. Les tableaux, eux, sortent directement — ils peuvent décider seuls, mais la
  décision doit être explicite.
- **`outdated` l'a gratuitement, `list` non.** `jg outdated` connaît déjà les deux versions,
  la colonne `AVAILABLE` en témoigne. `jg list` ne les a pas : marquer les obsolètes dans la
  liste demanderait la comparaison, qui coûte de une à cinq secondes chez brew et winget.
  Soit on ne colore que là où l'information est déjà là, soit on l'affiche depuis le cache
  du bloc de prompt — qui existe et peut mentir d'une demi-heure.
- **La couleur ne doit pas être la seule porteuse d'information.** Un terminal en 16
  couleurs, un daltonien, une capture collée dans un ticket : ce qui distingue une version
  obsolète doit rester lisible sans elle. Le projet a déjà tranché ainsi ailleurs — un
  compteur du prompt qui ne s'affiche jamais à zéro dit « à mettre à jour » par sa seule
  présence, sans flèche ni couleur.

### A-22 — L'élévation côté Unix

**Priorité :** à déterminer · **Provenance :** moitié restée ouverte d'**A-15**, écartée à
la conception (cf. [spec §7](specs/2026-08-17-elevation-design.md))

A-15 est livrée pour Windows, où le gestionnaire **nomme la cause dans son code de
sortie**. Rien de tel côté Unix : aucun gestionnaire n'y publie de code qui dise « il
fallait être root ». Le mécanisme d'A-15 ne s'y transpose donc pas, et cette entrée existe
pour ne pas laisser croire l'inverse.

Trois choses restent vraies, et elles étaient déjà dans A-15 :

- **Le besoin est étroit.** Homebrew refuse de tourner en root et ne demande `sudo` que
  marginalement ; scoop installe par utilisateur. L'effort mérite d'être proportionné.
- **On ne peut pas détecter une invite `sudo` sans pseudo-terminal.** Elle part sur le
  terminal, pas sur la sortie standard, et le mot de passe se lit sur `/dev/tty`. Analyser
  stdout ne verra jamais rien. C'est ce qui a fait choisir le constat plutôt que
  l'interception ([ADR-0004](adr/0004-elevation-constatee.md)) — et côté Unix, il n'y a
  rien à constater.
- **Donc : anticiper, pas détecter.** `sudo -v` valide l'autorisation et prolonge
  l'horodatage de sudo, dont la durée de grâce est déjà réglable par le système
  (`timestamp_timeout`). jigger demanderait le mot de passe une fois, dans sa fenêtre, le
  passerait à `sudo -S -v` sans jamais le conserver, et laisserait sudo mémoriser. Pas de
  secret en mémoire à protéger, à effacer, ni à tenir hors des journaux.

Reste la question qu'A-15 n'a pas eu à trancher, et qui est le vrai coût de cette
entrée-ci : **à quoi jigger reconnaît-il qu'une commande va exiger `sudo` ?** Anticiper
suppose de le savoir avant de lancer. Une élévation demandée pour rien est plus pénible que
l'invite qu'elle remplace.

### A-16 — Étudier l'intégration d'autres gestionnaires

**Priorité :** à déterminer · **Provenance :** demande du 16 août

apt, pacman, npm, et les suivants. **Une étude par gestionnaire**, dans `docs/analyse/`,
concluant sur la faisabilité et le coût — pas une implémentation.

Pour que les études soient comparables entre elles et calibrées sur ce qu'on sait déjà, la
même grille pour chacune :

| Question | Pourquoi elle décide du coût |
|---|---|
| **Le catalogue** — comment lister tous les noms connus, et en combien de temps ? | C'est le premier poste de dépense. `pacman -Slq` rend des noms nus, instantanément ; `apt-cache pkgnames` aussi. À l'opposé, winget n'a aucune sortie machine et coûte deux secondes par appel — d'où son cache de 24 h et son réchauffement détaché. |
| **Les installés** — sur le disque, ou par un sous-processus ? | brew et scoop se lisent dans une arborescence, en une milliseconde. Tout ce qui demande un processus doit sortir du chemin d'une frappe. |
| **Les obsolètes** | Coûteux partout ; jigger les calcule en tâche de fond et les sert depuis un cache. |
| **Une sortie machine existe-t-elle ?** | La leçon scoop : sans elle, on analyse des tableaux à colonnes, avec leurs largeurs variables, leurs couleurs ANSI et leurs en-têtes traduits. C'est là que le coût explose et que les défauts se cachent. |
| **La table des douze verbes** | La partie la moins chère, et la seule qu'on sache déjà faire vite. Un verbe absent est un verbe non déclaré : le modèle de capacités s'en accommode. |
| **Élévation, plateforme, portée** | apt demande root, pacman aussi ; ni l'un ni l'autre n'existe sous macOS ni Windows. jigger n'a encore aucun gestionnaire Linux, alors que son greffon zsh y tourne. |
| **Le verdict** | Faisable / faisable avec réserves / hors modèle — et une estimation en journées, calibrée sur les précédents : brew fut bon marché, scoop mitigé, winget cher. |

**npm mérite d'être traité à part, et l'étude devra le dire franchement.** Ce n'est pas un
gestionnaire de paquets système mais un gestionnaire de dépendances **de projet** : il n'y a
pas d'ensemble « installé sur la machine » qui ait un sens hors d'un répertoire, le
catalogue est un service en ligne plutôt qu'une liste locale, et `install` y signifie autre
chose. La question à trancher n'est pas « peut-on écrire la table ? » mais « le vocabulaire
de la façade décrit-il encore la réalité ? ». Si la réponse est non, le dire est un résultat
d'étude parfaitement valable — et probablement plus utile qu'une intégration bancale.

### A-17 — Un mode « à blanc » pour toutes les commandes mutantes

**Priorité :** à déterminer · **Provenance :** demande du 16 août

Pouvoir demander à jigger ce qu'il ferait, sans qu'il le fasse — sur `install`, `uninstall`,
`upgrade`, `pin`, `unpin`, `cleanup`, `source add` et `source rm`.

**Deux significations, très différentes, et c'est le premier arbitrage :**

- **Montrer l'argv que jigger lancerait.** Uniforme, disponible partout, et sans dépendance
  aux gestionnaires. C'est aussi ce qui répond au risque propre à la façade : `jg install
  fd` devient `brew install fd`, `scoop install fd` ou `winget install --id fd --exact`
  selon qui connaît le nom — une traduction que l'utilisateur ne peut pas prévoir, et
  qu'aucune autre commande ne lui montre. Peu coûteux, immédiatement utile.
- **Demander au gestionnaire ce qu'il ferait.** C'est la vraie question — « qu'est-ce qui va
  changer sur ma machine ? » — mais elle dépend d'eux. brew a `--dry-run` sur `install`,
  `upgrade` et `cleanup` ; la table des options le déclare déjà. scoop et winget n'ont pas
  d'équivalent général. Le modèle de capacités s'applique tel quel : un gestionnaire qui ne
  sait pas répondre le dit, comme il dit aujourd'hui qu'il ne sait pas faire `cleanup`.

Les deux se complètent : le premier montre la traduction, le second la conséquence. Le
premier peut être livré seul et servir tout de suite.

Trois points à ne pas manquer :

- **La résolution doit rester complète.** Un « à blanc » utile fait tout le travail sauf
  l'exécution : il résout le nom, choisit le gestionnaire — et ouvre le sélecteur si le nom
  est ambigu, sinon il ne montrera pas la commande qui aurait vraiment tourné.
- **`--yes` n'a rien à y faire.** Il accepte les accords de licence de winget ; à blanc, on
  n'accepte rien du tout.
- **Le nom du drapeau.** `--dry-run` est ce que brew et l'usage général emploient, et jigger
  relaie déjà les drapeaux natifs sans les interpréter : un `jg install --dry-run fd` part
  aujourd'hui tel quel vers brew. Introduire un `--dry-run` **à jigger** change ce
  comportement pour ce mot précis. À trancher : soit jigger l'intercepte et le traduit pour
  chaque gestionnaire, soit il en prend un autre pour lui.

### A-21 — Soigner ce que `doctor` diagnostique

**Priorité :** à déterminer · **Provenance :** demande du 17 août, posée comme un
*nice to have* · **S'appuie sur :** A-15, livrée depuis — les remèdes qui demandent une
élévation ont désormais un chemin (`internal/elevate`) · **Recoupe :** A-17 pour la
prévisualisation

`jg doctor` dit ce qui ne va pas ; il ne sait pas le réparer. Or scoop, lui, **livre déjà
le remède avec le diagnostic** — c'est ce qui rend l'idée réaliste :

```
WARN  '7-Zip' is not installed! It's required for unpacking most programs.
      Please Run 'scoop install 7zip'.
WARN  LongPaths support is not enabled. You can enable it by running:
      sudo Set-ItemProperty 'HKLM:\SYSTEM\...\FileSystem' -Name 'LongPathsEnabled' -Value 1
```

Cinq points à connaître avant d'écrire quoi que ce soit. Aucun n'interdit la fonction,
chacun change la manière de la faire.

- **Aujourd'hui, jigger ne voit pas cette sortie.** `doctor` n'est pas dans
  `verbesNormalises` (`internal/facade/executer.go`) : le sous-processus hérite du
  terminal, et jigger ne lit rien. Soigner suppose de **capturer** — donc une troisième
  catégorie de verbe, capturé *et* réémis tel quel, à côté du relais et du tableau refondu.
  C'est la première décision, et elle touche au choix autour duquel la façade est bâtie
  (spec §4).
- **Deux classes de remèdes, très inégales.** Installer un paquet manquant (`7zip`,
  `innounp`, `dark`), jigger sait déjà le faire — c'est son métier, et cela couvre 3 des 5
  avertissements ci-dessus. Changer un réglage système (LongPaths, mode développeur)
  demande une élévation : depuis A-15, `internal/elevate` sait la servir, mais rien ne dit
  qu'il faille traiter les deux classes dans le même chantier.
- **La capacité n'est pas la même d'un gestionnaire à l'autre.** `scoop checkup` nomme la
  commande à lancer, en clair. `brew doctor` rend de la prose anglaise, souvent sans
  commande exécutable. Le modèle de capacités s'applique tel quel : celui qui ne sait pas
  soigner le dit, comme winget dit qu'il ne sait pas `doctor`.
- **Analyser la sortie d'un autre outil n'est pas un contrat.** La formulation de scoop peut
  changer à toute version, et rien ne garantit qu'elle reste anglaise. Une table de
  vérifications connues, reconnues à un marqueur stable, vaut mieux qu'une extraction
  générique de « la commande entre quotes » — et **ce qui n'est pas reconnu reste affiché
  tel quel**, jamais deviné.
- **Rien ne doit se lancer tout seul.** `doctor` est en lecture aujourd'hui ; soigner mute
  la machine. Le geste juste est de montrer ce qui serait lancé et de demander — ce que
  prépare A-17. Et le mot « doctor » devrait alors entrer dans `$script:Mutants`
  (`shell/jigger.psm1`) et son équivalent zsh, sans quoi la liste des installés resterait
  fausse après une réparation.

### A-23 — Filtrer les secrets à la capture des prompts

**Priorité :** haute · **Provenance :** incident du 17 août ([#76](https://gitlab.yg-devworks.com/yves/jigger/-/issues/76))

Le hook `UserPromptSubmit` recopie **chaque** prompt dans `docs/historique/`, et rien n'y
filtre ce qui ressemble à un secret. Le 16 août, un jeton GitHub collé dans la conversation
pour configurer le miroir s'est retrouvé en clair dans un dépôt public ; il a fallu le
révoquer. Ce qui a été perdu ce jour-là n'était pas le jeton — c'était le temps de le
découvrir, un jour plus tard, par hasard.

Deux traitements possibles, non exclusifs :

- **Un filtre à la capture** sur les motifs connus — `ghp_`, `github_pat_`, `glpat-`,
  `sk-`, `AKIA`, les clés privées PEM. Le prompt est écrit avec la valeur remplacée par une
  note. Simple, mais il ne connaîtra jamais que ce qu'on lui a appris.
- **Un journal hors du dépôt publié** — la capture brute vit ailleurs, seule la synthèse
  entre dans `docs/`. Plus sûr par construction, au prix du va-et-vient entre deux
  emplacements quand on écrit l'entrée du jour.

Le hook vit dans `~/.claude/hooks/historiser-prompt.py`, donc **hors de ce dépôt** : la
correction ne s'y verra pas, mais c'est ce dépôt qui en porte les conséquences.

À noter pour qui reprend : GitHub, lui, a refusé le push — sa protection de secrets inspecte
tous les commits poussés, pas seulement le sommet. GitLab n'a rien dit. Le dépôt public le
plus exposé était donc celui qui n'avertissait pas.

---

## En cours

### A-7 — La stratégie de diffusion

**Priorité :** en cours · **Provenance :** demande du 15 août

Décisions arrêtées le 16 août, en conception :

- **D'abord trouvable, puis annoncé.** Une annonce ne se rejoue pas : le projet doit
  encaisser le premier passage avant de le provoquer.
- **Miroir GitHub** alimenté depuis GitLab, **issues ouvertes, PR fermées**. GitHub ne
  permet pas de désactiver les PR : on s'en approche par un gabarit et une action qui
  referme poliment en renvoyant vers GitLab.
- **Binaires précompilés et bucket scoop**, pour que Windows n'exige plus Go ni clone.
- **Un exécuteur GitLab** plutôt qu'un script local : la provenance est vérifiable et les
  releases ne dépendent plus d'une machine.
- **Canaux :** Hacker News (Show HN) et Reddit (r/commandline, r/zsh, r/PowerShell).
  Lobsters écarté faute d'invitation. Les textes sont rédigés ici, **publiés par Yves**.
- **Une seule spec** pour l'ensemble, en trois parties — choix assumé contre l'avis de
  découper en trois chantiers.

Fait à ce jour :

- **L'exécuteur** `cibuilder-go` (LXC 117) et la **chaîne de release** (`.gitlab-ci.yml`),
  éprouvée par un tag d'essai puis par la publication réelle de la v0.10.0.
- **Le bucket scoop** (`yves/scoop-jigger`), dont le manifeste se génère depuis le
  `SHA256SUMS` publié. Installation vérifiée sur une vraie machine Windows.
- **Les fichiers d'accueil** : `CONTRIBUTING.md`, `SECURITY.md`, et le gabarit plus
  l'action qui referment les PR sur le miroir — GitHub ne permettant pas de les désactiver.
- **Le miroir GitHub** `Yves848/jigger` : public, issues ouvertes, alimenté par miroir
  poussé depuis GitLab, topics posés. Les deux dépôts pointent sur le même commit.

- **La conception** ([`docs/specs/2026-08-16-diffusion-design.md`](specs/2026-08-16-diffusion-design.md)),
  écrite après coup pour ne consigner que ce qui a tenu.
- **Les quatre textes d'annonce** ([`docs/annonces/`](annonces/)), un par canal, avec
  l'ordre et l'espacement.

Reste, et ce n'est plus du ressort de l'assistant : la relecture des textes par Yves, le
rejeu du cadre `winget` sur le Dell XPS, puis la publication — r/commandline d'abord, Show
HN en dernier, à deux ou trois jours d'intervalle.


---

## Fait

### A-24 — Un garde-fou qui surveille le miroir GitHub

**Fait le :** 2026-08-17 · **Provenance :** panne muette du miroir, découverte le 17 août ·
**Document :** [garde-fou du miroir](garde-fou-miroir.md) · **Fichiers :** `tools/miroir/`,
`.gitlab-ci.yml`

Le miroir GitHub est tombé le 16 août et personne ne l'a su. Deux versions ont été publiées
entre-temps, que GitHub n'a jamais vues ; le défaut a été découvert par hasard, en allant
vérifier tout autre chose. Une synchronisation qui échoue ne fait pas de bruit — c'est
précisément ce qui la rend coûteuse.

**Le garde-fou surveille le symptôme, pas la cause.** Il compare la tête de `main` et les
tags des deux dépôts, tous deux publics, donc sans aucune authentification. Surveiller le
jeton n'aurait rattrapé que la panne d'hier : celui d'aujourd'hui expirera, la protection de
secrets de GitHub refusera un autre push, quelqu'un désactivera l'entrée de miroir. L'écart
entre les deux dépôts, lui, dit la panne quelle qu'en soit l'origine.

Il tourne comme **pipeline planifié** — indépendant des machines de l'utilisateur, qui sont
justement éteintes quand une panne s'installe. En cas d'écart, il ouvre une issue portant le
libellé `garde-fou::miroir`, et la referme d'elle-même au premier passage où les deux dépôts
se sont rejoints. Une panne qui revient rouvre une issue neuve plutôt que de réanimer
l'ancienne : ses sha doivent être ceux du jour.

Un mode `-notifier` affiche une bannière macOS, pour qui veut le lancer aussi en local.

**Un piège rencontré en l'écrivant :** GitLab crée une issue par `POST` sur la collection
mais la ferme par `PUT` sur l'issue. Poster sur une issue existante rend un `404` qui se lit
« issue introuvable » alors que c'est la méthode qui est fausse — le cycle complet
(ouverture, non-duplication, fermeture) a été exercé pour de bon contre le projet, sans quoi
le défaut serait passé.

### A-15 — Constater l'élévation, et proposer de rejouer (Windows)

**Fait le :** 2026-08-17 · **Provenance :** demande du 16 août · **Documents :**
[ADR-0004](adr/0004-elevation-constatee.md),
[spec](specs/2026-08-17-elevation-design.md) · **Fichiers :** `internal/pm/droits.go`,
`internal/winget/droits.go`, `internal/elevate/`, `internal/facade/executer.go`, `main.go`

L'entrée demandait de **détecter** une invite d'élévation et de la servir dans une fenêtre.
La conception l'a retournée : jigger **ne détecte rien**. Il laisse la commande tourner
relayée — invites, barres de progression et UAC compris — et lit son code de sortie après
coup. C'est l'ADR-0004, et c'est ce qui préserve la propriété autour de laquelle la façade
a été bâtie : aucune capture, aucun pseudo-terminal, aucune ligne de code de TTY.

**Trois mesures ont décidé de tout**, et aucune n'était devinable :

- **winget nomme la cause dans son code de sortie** — mais quatre codes parlent de droits,
  et **deux disent l'inverse des autres** (`INSTALLER_PROHIBITS_ELEVATION`,
  `ADMIN_CONTEXT_ACTION_PROHIBITED`). Un « code non nul → propose d'élever » aurait été
  nuisible deux fois sur quatre : il aurait poussé à refaire, élevé, ce qui venait
  d'échouer *pour cause d'élévation*. D'où trois valeurs à `pm.Droits`, pas deux.
- **Go ne rend pas ces codes sous la forme publiée.** Microsoft imprime la forme signée
  (`-1978335207`) ; `exec.ExitError.ExitCode()` rend sous Windows le DWORD non signé
  (`2316632089`). Recopier la colonne du tableau officiel aurait donné une comparaison
  jamais vraie — et une panne parfaitement muette. Un test existe pour ce seul piège.
- **`sudo` existe sous Windows 11, et il est désactivé.** `C:\WINDOWS\system32\sudo.exe`
  est là (build 26200) mais éteint tant qu'on ne l'allume pas dans les paramètres
  développeur. Il ne peut donc pas être supposé : jigger lit le registre, et se rabat sur
  `ShellExecuteEx` + verbe `runas` — en **attendant** le processus, pour que le verdict
  revienne là où la commande a été tapée.

**Ce que jigger fait, et ne fait pas.** Il dit la cause, propose, et n'élève jamais sans un
oui explicite — la ligne ouverte par défaut dans le cadre est *annuler*. Sans terminal
(un tube, un script), aucune question : la ligne exacte s'imprime et le code d'origine est
rendu. Sur les deux codes qui interdisent l'élévation, aucune proposition — un message qui
dit l'inverse.

Deux choses que l'entrée attendait et qui n'ont **pas** lieu d'être : le réglage de durée
de grâce hérité d'A-14 — jigger ne détient aucun secret et ne mémorise aucune autorisation,
c'est UAC ou `sudo` qui décide de ce qu'il redemande, et un réglage qui ne piloterait rien
serait un mensonge dans l'écran de configuration —, et l'élévation anticipée, écartée parce
qu'il faudrait deviner juste et qu'on élèverait parfois pour rien.

La moitié Unix est **A-22**, et elle ne se traitera pas de cette façon.

### A-20 — `⏎` complète la dernière partie, puis exécute

**Fait le :** 2026-08-17 · **Provenance :** demande d'Yves à l'usage — « `winget li ⏎` :
il faut utiliser la ligne sélectionnée et l'insérer comme si on avait fait un `⇥` », puis,
sur la première version : « là, il faut faire 2× Entrée » · **Fichiers :**
`shell/jigger.plugin.zsh`, `shell/jigger.psm1`, `main.go`,
`tests/{smoke.ps1,pty.ps1,zpty.zsh}`

Une frappe de moins, et la même règle partout : **`⏎` pose le candidat désigné dans la
ligne, puis l'exécute — dans la même frappe.** `winget li ⏎` lance `winget list`.

Elle vaut à **tous les niveaux de l'arbre** — verbe, sous-verbe, option, nom de paquet —
parce qu'elle ne connaît rien de ces niveaux : elle ne compare que la ligne tapée à la
ligne complétée, que le binaire rend déjà (`left=`) à chaque frappe. Rien à ajouter au
protocole, rien de nouveau à décider par contexte.

Trois points qu'il fallait trancher :

- **Presser `⏎`, c'est dire « pars ».** La première version s'arrêtait après la
  complétion, laissant un second `⏎` exécuter : c'était une frappe économisée d'un côté et
  reperdue de l'autre. jigger ne juge pas à la place de l'utilisateur si la ligne complétée
  est correcte — elle part, et le gestionnaire dira ce qu'il en pense.
- **Le focus n'entre pas dans la règle.** `⇥` insère la ligne désignée même quand le popup
  n'a pas le clavier (c'est ce que dit sa teinte au repos) ; `⏎` fait de même, sans quoi le
  cas de la demande — `winget li ⏎`, sans être entré dans la liste — ne serait pas servi.
- **`^G` est l'échappatoire**, et elle existait déjà : elle ferme le popup pour la ligne en
  cours, donc `⏎` y exécute exactement ce qui est tapé.

Le pied du cadre passe à quatre pastilles — `⇥ insérer`, `↩ exécuter`, `↓ parcourir`,
`^G fermer` : deux gestes distincts, deux libellés, et le même vocabulaire que le sélecteur
plein écran. Les quatre tiennent dans la largeur du cadre ; les captures de la
documentation et du site, qui étaient à 54 colonnes, ont été reprises à la largeur réelle.

Côté zsh, `⏎` est repris comme les flèches et `^R` (A-19) : `^M` et `^J` passent par
`_jigger_accept`, qui pose le candidat puis rend la touche au widget qui la tenait avant
nous — l'`accept-line` de zsh, ou celui qu'un autre greffon a enveloppé.

### A-1 — L'alias `jg` dans le greffon PowerShell

**Fait le :** 2026-08-17 · **Provenance :** réserve publiée avec la v0.8.0, confirmée en
relecture de l'i18n · **Fichiers :** `shell/jigger.psm1`, `tests/smoke.ps1`, `tests/pty.ps1`

`jg` existe des deux côtés. Sous PowerShell, `Set-Alias` le pose et le module l'exporte —
`Remove-Module jigger` le reprend, ce qui est la porte de sortie de qui tenait déjà à son
propre `jg`.

**Les deux mécanismes, comme annoncé.** L'alias fait s'exécuter la ligne ; la liste des
commandes surveillées fait apparaître le popup. Le relais lit le tampon de PSReadLine, où
aucun alias n'a été développé : c'est **`jg` tel qu'il est tapé** qui devait être reconnu,
et `jigger` séparément.

Trois décisions que l'entrée n'avait pas tranchées :

- **La façade arme le popup *toujours*, au lieu d'entrer dans le défaut de
  `JIGGER_COMMANDS`.** Étendre le défaut à `winget,scoop,jigger,jg` n'aurait rien changé
  pour quiconque a recopié `winget,scoop` dans son profil — la documentation l'a montré
  pendant trois versions —, et la façade serait restée éteinte chez lui sans qu'il puisse
  deviner le rapport. Le greffon zsh code déjà `jigger` et `jg` en dur ; les deux côtés
  répondent maintenant à la même frappe.
- **L'alias désigne `$script:Exe`, pas le mot « jigger ».** Avec `JIGGER_BIN` posé sur le
  binaire d'un dépôt, `jg` et le popup parlaient sinon de deux exécutables différents.
- **Il est posé après les vérifications d'installation, mais avant celle de PSReadLine** —
  et exporté sur place. Un alias vers un binaire absent ou trop ancien ne sert personne
  (c'est aussi l'ordre du greffon zsh), mais l'absence de PSReadLine n'éteint que le
  popup, pas la syntaxe unique — or le module rend alors la main sans jamais atteindre son
  `Export-ModuleMember` final.

**Un défaut latent corrigé au passage** : `$script:Commands` recevait le résultat d'un
pipeline, donc une chaîne nue quand `JIGGER_COMMANDS` ne portait qu'un nom. Le `+=` qui
ajoute la façade l'aurait concaténée — `wingetjigger` — au lieu d'allonger la liste.

**Une seconde réparation, trouvée en lançant la suite Go sur cette machine** : la résolution
de langue a une source que `t.Setenv` ne vide pas — la culture de l'utilisateur Windows,
`GetUserDefaultLocaleName`. « Toutes les sources sont muettes » était donc un état
inatteignable sous Windows, et les deux tests qui en dépendaient s'en accommodaient
chacun à sa façon : `TestLangueInconnueRetombeSurAnglais` assérait en fait la langue de la
machine d'essai (il échouait sur un Windows français, et sur lui seul),
`TestLocaleExotiqueDonneAnglais` se dispensait de tourner là où cette culture existe —
c'est-à-dire sur la seule plateforme où elle décide. `culture` est désormais une variable de
paquet, comme `catalogue` l'est déjà, et les tests la posent : les sept cas de la locale
exotique valent sur les trois plateformes, et un test neuf couvre la source elle-même —
dernier recours, jamais avant `LANG`, et une culture inconnue qui retombe sur l'anglais.
Éprouvé mordant par mutation (retirer la consultation fait échouer le test neuf).

**Éprouvé** sur une vraie machine Windows, winget et scoop installés : `tests/smoke.ps1`
(81 assertions, dont l'alias, sa cible et quatorze lignes reconnues ou non — `jgit` et
`jgx` compris, un préfixe n'étant pas la façade) et `tests/pty.ps1` dans un vrai
pseudo-terminal (26 assertions) — le cadre apparaît sur `jg ins` sous l'en-tête
« ❯ jigger », `jigger ins` l'arme aussi, `jgit ins` non, et `⇥` écrit `jg install`.

### A-6 — Le site de présentation

**Fait le :** 2026-08-16 · **En ligne :** <https://jigger.yg-devworks.com/> ·
**Commits :** `de53ec5` → `43fa4ce` (branche `feat/site`)

Conception : [`docs/specs/2026-08-16-site-jigger-design.md`](specs/2026-08-16-site-jigger-design.md) ·
plan : [`docs/plans/2026-08-16-site.md`](plans/2026-08-16-site.md)

Une page unique et bilingue, source dans le dépôt sous `website/`, servie par le Proxmox
maison — nginx pour les fichiers, Caddy pour l'HTTPS, releases horodatées et lien
`current`. Anglais en clair dans le HTML, français en dictionnaire : le repli se fait sur
l'anglais, comme dans le binaire depuis la v0.9.0.

`website/verifier.sh` garde la page contre trois dérives, et le déploiement refuse de
partir s'il échoue : une clé sans traduction, une entrée de dictionnaire orpheline, une
ancre morte, ou une commande d'installation qui ne correspond plus mot pour mot au guide.
Les quatre contrôles ont été prouvés mordants avant d'être adoptés.

Le miroir GitHub du modèle **(A)** n'en faisait pas partie : il est passé à A-7.

### A-18 — La section `#jigger` du site Cocktails

**Fait le :** 2026-08-16 · **Dépôt :** `yves/cocktails-website` · **MR :** !1 · **En ligne**

Elle décrivait un outil qui n'existait plus : complétion de `brew` seul déclenchée par Tab,
« aucun alias » alors que `jg` en est un, « ne requiert que brew », « rien de binaire à
faire confiance » — et des touches d'aide (`↩ exécuter`, `esc annuler`) qui n'ont jamais
été celles de jigger.

Elle dit désormais les trois gestionnaires, le cadre qui suit la frappe, la syntaxe `jg`,
et les flèches qui restent l'historique du shell. Liens croisés vers le site de jigger
posés des deux côtés. Les deux langues reprises ensemble, parité des clés contrôlée,
vérification faite dans un navigateur puis sur la page en ligne.

**Leçon d'organisation :** `~/git/cocktails/website/` est un **dépôt imbriqué**
(`cocktails-website`), ignoré par son parent — et l'`origin` du dépôt `cocktails` pousse
vers GitLab **et** GitHub. Un premier commit est parti au mauvais endroit avant que je m'en
aperçoive ; la branche vide a été supprimée des deux remotes.

### A-10 — Mettre en page et paginer les sorties

**Fait le :** 2026-08-16 · **Conception :** [`docs/specs/2026-08-16-pagination-design.md`](specs/2026-08-16-pagination-design.md)

Les quatre verbes tabulaires s'affichent dans une vue navigable quand la sortie est un
terminal **et** que le contenu dépasse l'écran : filtre au fil de la frappe, bascule
texte/regex sur `^R`, sélection multiple sur `⇥`, `↵` qui imprime les lignes retenues.

Trois engagements tenus, et vérifiés :

- **La table brute ne bouge pas d'un octet** hors terminal — comparé binaire contre
  binaire sur `list`, `outdated`, `source` et `--json`.
- **Un seul cœur de navigation** (`internal/ui/liste.go`) sous le sélecteur et sous la
  vue : les 366 lignes de tests du sélecteur passent **sans une modification**, ce qui est
  le seul juge honnête de la refactorisation.
- **Une seule source pour les colonnes** (`facade.Colonnes`), avec un test croisé qui
  interdit à la table brute et à la vue de diverger.

Éprouvé de bout en bout dans un vrai pseudo-terminal : filtrage, `^R`, motif invalide qui
ne vide pas la liste, `⇥` qui coche, `↵` qui imprime, `^G` qui n'imprime rien.

A-11 (regex dans le popup), A-12 (tri des colonnes) et A-13 (versions obsolètes) partent
maintenant d'un terrain préparé.

### A-14 — Un écran de configuration (TUI)

**Fait le :** 2026-08-17 · **ADR :** [0003](adr/0003-fichier-de-configuration.md) ·
**Conception :** [`docs/specs/2026-08-17-configuration-design.md`](specs/2026-08-17-configuration-design.md)

`jigger config` ouvre un écran à trois groupes — ce qui prend effet tout de suite, ce qui
attend le prochain shell, et ce que jigger voit sans le posséder. Chaque ligne affiche **sa
provenance**, parce que l'environnement garde le dernier mot sur le fichier.

Les trois obstacles que l'entrée annonçait ont été levés, chacun par une décision :

- **Il n'existait aucun fichier.** Il y en a un désormais, `clé = valeur`, sans dépendance,
  à l'emplacement que le système prévoit. Préséance **environnement > fichier > défauts**.
- **La moitié des réglages n'appartient pas au binaire** — huit sur douze, mesurés. Les
  greffons ne lisent pas le fichier : ils demandent au binaire de le leur dicter
  (`config --export`), ce qui laisse **une seule implémentation de la préséance**.
- **Les gestionnaires ne déclaraient rien.** brew et winget déclarent leur durée de
  validité de catalogue, jusque-là écrite en dur ; l'écran les affiche sans rien savoir
  d'eux, et la durée est désormais **lue**, pas compilée.

**Un défaut trouvé en s'éprouvant soi-même** : `key = ^ ` — Ctrl-Espace, une valeur
documentée — se relisait `^`, l'espace mangé par le nettoyage. Et mon propre test
consacrait le bogue. Corrigé, puis verrouillé par un test qui fait traverser des valeurs à
espaces significatifs.

**Ce qui a été éprouvé par exécution, jamais par relecture** : vingt-deux valeurs hostiles —
apostrophes, `$(...)`, backticks, accents, antislashs — passées par un vrai `zsh -c` et un
vrai `pwsh -c`, et comparées à ce qui en ressort.

### A-11 — Filtrer le popup en regex ou en texte brut, quel que soit le gestionnaire

**Fait le :** 2026-08-16 · **Provenance :** demande du 16 août

`^R` bascule le filtre entre texte brut et expression rationnelle, **sur les trois
surfaces** : popup vivant, sélecteur plein écran, vue paginée. Le titre du cadre affiche
`[regex]` tant que c'est actif.

**Rien n'est confisqué au shell.** Hors ligne surveillée, `^R` retourne à ce qu'il faisait —
la recherche inverse dans l'historique. C'est le principe posé par A-19, et le mécanisme
existait déjà pour les flèches. Vérifié : presser `^R` sur `echo bonjour` ne bascule rien.

**Le filtre ne touche que les noms de paquets.** Les verbes, les sous-commandes et les
options gardent leur préfixe : sur douze verbes, une expression rationnelle
n'apprendrait rien et surprendrait.

**Un motif fautif ne retient rien**, ici — l'inverse du sélecteur plein écran, et
délibérément : dans le popup, le motif *est* le mot de la ligne de commande, et déverser
16 000 entrées parce qu'une parenthèse manque serait une avalanche, pas une aide.

**Deux défauts trouvés en chemin, dans les deux greffons :** la clé de cache du cadre ne
portait pas le mode, si bien qu'une bascule ne redessinait rien — la ligne n'ayant pas
changé. Et la condition d'activation, calquée sur « il y a des candidats », empêchait de
basculer précisément quand la liste était vide, c'est-à-dire quand on en a le plus besoin.

**Éprouvé** en pseudo-terminal zsh : bascule, motif `^fire.*x$` qui ne trouve que
`firefox`, délégation hors popup, et les deux suites de greffons au complet. Le rendu par
défaut du popup est **identique à l'octet près**. Côté PowerShell, le module est écrit et
son banc passe, mais la bascule elle-même **reste à essayer sur une vraie machine
Windows**.

### A-19 — Les raccourcis des listes à sélection multiple

**Fait le :** 2026-08-16 · **Provenance :** demande du 16 août, après A-10

Le jeu de touches est arrêté, et un principe le porte : **jigger ne prend une touche que
tant qu'il a le clavier**, et la rend à ce qu'elle faisait sinon. Ce n'est pas neuf — le
greffon le fait déjà pour les flèches (`_jigger_up` délègue au widget qu'il a relevé). Le
reconnaître comme règle générale a dissous le conflit qui bloquait A-11.

| Action | Touche | Justification |
|---|---|---|
| (Dé)sélectionner une ligne | `⇥` | Rien à insérer dans une vue de lecture, donc aucune concurrence |
| Tout (dé)sélectionner | `^A` | Le geste universel. Repris au champ de saisie, qui garde `Début` |
| Pages | `PgPréc` `PgSuiv` | `^B` et `^F` **rendus** au curseur du champ |
| Regex | `^R` | Sur les deux surfaces, avec délégation au shell hors focus |

**« Tout cocher » porte sur le filtre**, pas sur le catalogue : on filtre, puis on prend ce
qui reste. Et c'est une bascule « tout ou rien » — une sélection partielle se **complète**
au lieu de s'inverser, ce qui serait imprévisible.

**Un défaut d'A-10 corrigé au passage** : `^B`/`^F` avaient été liés au défilement par
page, alors qu'ils appartiennent au champ de saisie. Le champ avait perdu son déplacement
de curseur sans que personne s'en aperçoive.

Ce qu'A-19 a tranché pour A-11 : la bascule regex du popup vivant sera **`^R`**, avec la
même délégation que les flèches — recherche inverse de zsh tant que jigger n'a pas le
focus. La même touche sur les deux surfaces, sans rien confisquer.

### A-8 — La fiche du projet GitLab

**Fait le :** 2026-08-16 · **Sans commit** — la fiche vit côté serveur, pas dans le dépôt

La description datait d'avant winget, scoop et la façade, et les topics étaient vides.

```
avant   Assistance Homebrew pour le terminal (Go + Bubble Tea) — compagnon CLI de Cocktails
après   One syntax for Homebrew, winget and scoop, with a live popup that follows your
        typing. A small Go binary wired into zsh and PowerShell — CLI companion to Cocktails.
```

Topics posés : `cli`, `tui`, `package-manager`, `homebrew`, `winget`, `scoop`, `zsh`,
`powershell`, `go`, `bubbletea`, `terminal`.

En anglais, comme le reste de ce qui est publié depuis la v0.9.0 — c'est la première ligne
que lit un visiteur, et la seule que voit celui qui ne clique pas.

### A-9 — Publier la v0.9.0

**Fait le :** 2026-08-16 · **Commit :** `dfa3292` · **Tag :** `v0.9.0` ·
[release](https://gitlab.yg-devworks.com/yves/jigger/-/releases/v0.9.0)

Tout ce qui avait été fusionné restait invisible pour qui installe par le tap — la
réparation des analyseurs scoop comprise. C'est publié : `brew upgrade jigger` mène de la
0.8.0 à la 0.9.0.

Deux choses au-delà de la chaîne habituelle :

- **Les deux greffons exigent désormais la 0.9.0** (contre 0.8.0 côté zsh, 0.6.0 côté
  PowerShell). Un binaire 0.8.0 ne parle que français, alors que ces greffons disent leurs
  messages dans les deux langues : les apparier produirait deux langues dans la même
  fenêtre. Le refus a été vérifié avec un faux binaire 0.8.0 dans le `PATH`.
- **Le tap livre enfin le segment starship** : le répertoire `shell/starship` n'existait pas
  dans l'archive v0.8.0, ce qui était noté à la publication précédente et attendait ce tag.

Les cadres de la documentation ont été régénérés dans les deux langues, et la référence du
banc de rendu recapturée — elle porte le numéro de version, ce qui est précisément l'objet
d'A-5.

