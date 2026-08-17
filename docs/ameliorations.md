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

### A-1 — L'alias `jg` dans le greffon PowerShell

**Priorité :** à déterminer · **Provenance :** réserve publiée avec la v0.8.0, confirmée en
relecture de l'i18n

Seul le greffon zsh arme la façade. Sous PowerShell, `JIGGER_COMMANDS` vaut `winget,scoop`
et il n'y a pas d'alias : la façade s'y atteint en écrivant `jigger <verbe>` en toutes
lettres, et le popup ne la suit pas. Le point délicat est connu : l'alias et la liste des
commandes qui arment le popup sont **deux mécanismes distincts**, et il faut les deux — le
widget lit ce que l'utilisateur a tapé, pas l'expansion de l'alias.

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

### A-14 — Un écran de configuration (TUI)

**Priorité :** à déterminer · **Provenance :** demande du 16 août

Un écran plein terminal pour régler jigger, dont la mise en page découlerait des paramètres
**déclarés par chaque gestionnaire enregistré** — comme la table de verbes fait découler les
capacités de la façade.

Trois obstacles, dont deux sont des décisions d'architecture avant d'être du code. Ils ne
condamnent rien, mais ils expliquent pourquoi cette entrée est plus lourde qu'elle n'en a
l'air.

- **Il n'existe aucun fichier de configuration.** Tout se règle par variables
  d'environnement — `JIGGER_LIVE`, `JIGGER_ROWS`, `JIGGER_KEY`, `JIGGER_PROMPT`,
  `JIGGER_LANG`… Un écran de configuration suppose de créer un fichier, donc de trancher un
  ordre de préséance (environnement > fichier > défauts, vraisemblablement) et un
  emplacement. C'est de niveau ADR, comme l'ont été le choix de Go et la table déclarative.
- **La moitié des réglages n'appartient pas au binaire.** `JIGGER_ROWS`, `JIGGER_KEY`,
  `JIGGER_LIVE` sont lus par le **greffon**, au chargement du shell, avant tout appel à
  jigger. Un écran lancé depuis le binaire ne peut donc pas les appliquer au shell en cours :
  il écrit un fichier que le greffon lira au prochain démarrage, ou il imprime les lignes à
  coller. À décider explicitement, sinon l'écran promettra un effet qu'il n'a pas.
- **Les gestionnaires ne déclarent aujourd'hui aucun paramètre.** Les réglages existants
  sont globaux à jigger ; côté gestionnaires, il n'y a que des variables qui ne lui
  appartiennent pas (`$SCOOP`, `$SCOOP_GLOBAL`) et des durées de cache écrites en dur. Pour
  qu'une mise en page « claire et logique » se déduise des gestionnaires, il faut d'abord
  que chacun **déclare** ses paramètres — une seconde table à côté de `pm.Bindings`, dans
  l'esprit de l'ADR-0002 : ce qui est déclaré est vérifiable, ce qui est codé en dur ne
  l'est pas.

La partie visible, elle, est la moins coûteuse : Bubble Tea est déjà là, et `internal/ui`
sait dessiner un cadre, naviguer et filtrer.

### A-15 — Détecter les demandes d'élévation et les servir dans une fenêtre

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Dépend de :** A-14
pour le réglage de durée

Repérer, tous gestionnaires confondus, le moment où l'utilisateur doit saisir un mot de
passe `sudo` (macOS, Linux) ou s'élever en administrateur (Windows), et le lui demander dans
une fenêtre Bubble Tea — avec mémorisation pendant une durée réglable, comme le fait
Cocktails.

Quatre choses à savoir avant d'écrire quoi que ce soit. Aucune n'interdit la fonction, mais
chacune change la manière de la faire.

- **Aujourd'hui, jigger s'efface exprès devant ces invites.** Pour tous les verbes non
  normalisés, `internal/facade/executer.go` donne au sous-processus les entrées et sorties
  du terminal (`relais = true`). C'est ce qui fait que les invites de winget, ses barres de
  progression et son élévation fonctionnent « sans une ligne de code de TTY » — la spec §4
  en fait un choix explicite. Intercepter suppose de **capturer** ce flux : on renonce alors
  à la propriété autour de laquelle la façade a été bâtie. À rouvrir en connaissance de
  cause, probablement par un ADR.
- **`sudo` n'écrit pas son invite là où on l'attend.** Elle part sur le terminal, pas sur la
  sortie standard, et le mot de passe se lit sur `/dev/tty`, pas sur l'entrée standard.
  Analyser stdout ne verra jamais rien : il faudrait allouer un pseudo-terminal au
  sous-processus. Le dépôt sait le faire dans ses harnais de test (`tests/zpty.zsh`,
  `tests/conpty`), jamais à l'exécution.
- **Sous Windows, ce n'est pas une invite console.** L'élévation passe par UAC, une fenêtre
  du système : rien à détecter dans une sortie, rien à saisir dans un cadre. Il faut
  relancer le processus élevé (`Start-Process -Verb RunAs`), ce qui ouvre une autre console.
  Le mot « détecter » ne s'applique donc qu'aux deux plateformes Unix ; Windows demande un
  autre mécanisme, à traiter comme tel.
- **Le besoin réel est étroit.** Homebrew refuse de tourner en root et ne demande sudo que
  marginalement ; scoop installe par utilisateur ; seul winget en a besoin, pour les
  installations à l'échelle de la machine. L'effort mérite d'être proportionné à ça.

**Une piste qui évite de détenir le secret.** Plutôt que capturer un mot de passe et le
garder en mémoire, `sudo -v` valide l'autorisation et prolonge l'horodatage de sudo, dont la
durée de grâce est déjà réglable par le système (`timestamp_timeout`). jigger demanderait le
mot de passe une fois, dans sa fenêtre, le passerait à `sudo -S -v` sans jamais le
conserver, et laisserait sudo faire la mémorisation. Le réglage de durée d'A-14 piloterait
alors le rafraîchissement, pas la rétention. C'est plus simple à écrire, et il n'y a pas de
secret en mémoire à protéger, à effacer, ni à ne pas écrire dans un journal.

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

