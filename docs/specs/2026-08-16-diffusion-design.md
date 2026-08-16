# La diffusion de jigger — conception

16 août 2026 — état : validé. Trois parties, dont deux réalisées le jour même.

## Objet

Faire connaître jigger à ceux qui en ont l'usage — les gens qui vivent dans un terminal —
sans dilapider le seul premier passage dont un projet dispose.

C'est l'entrée **A-7** du tableau des améliorations. Elle a grossi en cours de conception :
elle contient une chaîne de release, un travail d'empaquetage et une annonce. J'ai proposé
de la découper en trois chantiers ; Yves a tranché pour **une seule spec**, structurée en
trois parties. Ce document suit cette décision.

## Point de départ

Ce qui existait avant cette journée : un binaire en v0.9.0, une documentation anglaise avec
sa traduction française, un tap Homebrew, et un dépôt GitLab public que personne ne
connaissait.

Ce qui manquait, et que la conception a rendu visible : **rien n'était prêt à encaisser un
afflux**. Installer jigger sous Windows exigeait Go, un clone et un script. Aucun binaire
précompilé. Aucun endroit où un utilisateur de GitHub — c'est-à-dire presque tout le monde
— pouvait signaler un bogue. Aucun `CONTRIBUTING.md`. Annoncer dans cet état aurait envoyé
des curieux vers une porte fermée.

## Les six décisions

1. **D'abord trouvable, puis annoncé.** Une annonce ne se rejoue pas : un « Show HN » qui
   tombe à plat ne se recommence pas avec le même projet. Le projet doit encaisser le
   premier passage avant de le provoquer.
2. **Miroir GitHub, issues ouvertes, PR fermées.** GitLab reste la seule source de vérité ;
   le miroir existe pour que les gens trouvent le projet et signalent les problèmes là où
   ils sont déjà.
3. **Binaires précompilés et bucket scoop**, pour que Windows n'exige plus ni Go ni clone.
   C'est là que se perdent la plupart des curieux.
4. **Un exécuteur GitLab** plutôt qu'un script de release local : la provenance est
   vérifiable et les publications ne dépendent plus d'une machine.
5. **Canaux : Hacker News (Show HN) et Reddit** — r/commandline, r/zsh, r/PowerShell.
   Lobsters écarté faute d'invitation ; à reprendre si l'occasion se présente.
6. **Les textes sont rédigés ici, publiés par Yves.** Sa signature, son compte, sa
   réputation.

## §1 — La chaîne de release *(réalisée)*

Un tag déclenche quatre étapes : vérifier (`go vet`, `go test`), construire les quatre
cibles, publier au registre de paquets, créer la release avec les notes du CHANGELOG.

Trois choix méritent d'être retenus, parce qu'ils se défendent encore dans six mois :

- **L'image est épinglée sur la série qu'exige `go.mod`**, et non sur « la dernière ». Une
  release d'aujourd'hui doit pouvoir se rejouer.
- **L'exécuteur n'est pas privilégié.** Compiler du Go n'en a aucun besoin, et le mode
  privilégié donnerait à n'importe quel job les clés du conteneur hôte.
- **Un garde-fou de taille** refuse d'empaqueter ce qui n'est pas un binaire plausible.
  Le mode de panne à craindre n'est pas l'échec bruyant : c'est la release complète,
  d'apparence irréprochable, dont les archives ne contiennent que la licence et le README.

**Ce que le premier passage a trouvé** — et qui justifie à lui seul d'avoir monté un
exécuteur plutôt qu'un script local : trois tests passaient sur le Mac de développement et
échouaient partout ailleurs, parce qu'ils interrogeaient la machine au lieu de la façade.
Sans gestionnaire installé, jigger ne propose aucun verbe : le comportement était juste,
les tests mentaient.

**Conséquence sur le rituel de publication** : la CI crée la release à partir de la section
du CHANGELOG, qui doit donc être écrite et commitée **avant** le tag. Si elle manque, le
pipeline s'arrête plutôt que de publier une release muette.

## §2 — Les portes d'entrée *(réalisées)*

**Le bucket scoop** (`yves/scoop-jigger`) pointe sur l'archive Windows de la release. Son
manifeste n'est jamais écrit à la main : un script lit le condensat dans le `SHA256SUMS`
publié, valide le JSON, et — le contrôle qui compte — vérifie que l'archive répond **sans
authentification**, parce que c'est ainsi que scoop la téléchargera chez quelqu'un d'autre.

**Le miroir GitHub** (`Yves848/jigger`) est alimenté par miroir poussé depuis GitLab.
Issues activées, wiki et projets désactivés.

**Et une correction à la décision 2, qu'il faut assumer :** *GitHub ne permet pas de
désactiver les pull requests.* On peut couper les issues d'une case à cocher, pas les PR.
Il n'existe aucun réglage. On s'en approche par un gabarit de PR et une action qui referme
en expliquant où aller — les deux vivant dans le dépôt GitLab, donc rien à maintenir des
deux côtés. Une PR sur un miroir ne peut de toute façon pas être fusionnée : elle serait
écrasée à la synchronisation suivante. Mieux vaut le dire tout de suite que la laisser
mourir sans réponse.

**Les fichiers d'accueil** — `CONTRIBUTING.md` dit où va quoi et ce que le code attend ;
`SECURITY.md` donne une adresse et borne ce qu'une faille peut signifier, en décrivant ce
que jigger fait réellement sur une machine.

## §3 — L'annonce *(à faire)*

### Ce qu'on annonce

Pas « un greffon de complétion de plus ». Le message tient en une phrase : **une syntaxe
pour trois gestionnaires, et une fenêtre qui suit la frappe** — sur macOS, Windows et
Linux, dans zsh et PowerShell.

Deux points intéressent un public de terminal plus que tout le reste, et doivent apparaître
tôt :

- **Les flèches restent l'historique du shell** tant que la fenêtre n'a pas le clavier.
  Quiconque a déjà été échaudé par un greffon qui confisque `↑` comprendra immédiatement.
- **Jamais de choix automatique** entre deux gestionnaires qui connaissent le même nom.

### Où, dans quel ordre

L'ordre n'est pas décoratif : il va du public le plus indulgent au plus exposé, pour que
les premiers retours corrigent le tir avant le passage qui compte.

| Rang | Canal | Pourquoi là |
|---|---|---|
| 1 | **r/commandline** | Le public le plus exactement ciblé, et le plus tolérant envers un outil jeune |
| 2 | **r/zsh** puis **r/PowerShell** | Un message par sous-forum, centré sur *ce shell-là* — pas le même texte recopié |
| 3 | **Hacker News**, Show HN | Le plus fort potentiel et le plus exigeant. Une seule fois |

**Espacer.** Deux ou trois jours entre chaque, pour que les retours d'un canal nourrissent
le texte du suivant, et parce que poster le même jour partout se voit.

### Ce que chaque texte doit faire

- **Montrer avant d'expliquer** : le cadre du popup en premier, la prose après.
- **Dire ce que ça ne fait pas.** Pas de paquet winget, pas de fish ni de bash, façade en
  phase 1. Un projet qui annonce ses limites se fait moins étriller que celui qui les
  laisse découvrir.
- **Ne pas vendre.** Ce public repère la formule marketing à la première ligne.
- **Une seule question ouverte** en fin de message — ce sur quoi l'avis des lecteurs est
  réellement souhaité — plutôt qu'un appel générique aux retours.

Chaque texte est **rédigé en anglais**, comme le reste du projet.

### Le jour de l'annonce

Ce n'est pas un envoi, c'est une présence. Le facteur qui décide du sort d'un Show HN n'est
pas le texte : c'est la disponibilité de l'auteur dans les deux heures qui suivent.

- **Poster en début de journée ouvrable** côté Amérique du Nord ; ne pas poster un vendredi
  ni un week-end.
- **Rester disponible deux heures.** Répondre à tout, y compris aux critiques — surtout aux
  critiques.
- **Un bogue signalé pendant l'annonce se corrige le jour même** s'il est petit. C'est la
  meilleure réponse possible à un premier passage.

### Ce qu'on regarde ensuite

Pas les étoiles : les **issues ouvertes par des inconnus**, les installations par scoop et
brew, et ce que les gens comprennent de travers dans le README — chaque question répétée
deux fois est un défaut de documentation, pas un défaut de lecteur.

## Prérequis avant de poster

Une liste à vérifier, pas à supposer. Tout est fait sauf la dernière ligne.

- [x] Le site répond en HTTPS, avec une image d'aperçu pour les liens partagés.
- [x] L'installation ne demande plus de compiler : `brew install jigger`,
      `scoop install jigger`.
- [x] Les deux voies ont été essayées **sur une vraie machine**, pas seulement décrites.
- [x] Le miroir GitHub existe, ses issues sont ouvertes, ses PR sont gérées.
- [x] `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`, `CHANGELOG.md` en place.
- [x] La documentation ne contient plus d'affirmation périmée.
- [ ] Les textes des quatre messages, relus par Yves.

## Portée

L'annonce et ce qu'elle exige. Les textes sont écrits dans ce dépôt, sous
`docs/annonces/`, pour qu'ils soient relus comme du reste — et pour qu'on retrouve dans six
mois ce qui avait été dit.

## Non-buts

- **Aucune publication en mon nom.** Les comptes sont ceux d'Yves ; je rédige, il poste.
- **Pas de Product Hunt, pas de Twitter/X, pas de liste de diffusion.** Public sans rapport,
  ou coût d'entretien sans rapport avec le bénéfice.
- **Pas de paquet winget dans cette entrée.** La soumission à `winget-pkgs` est un chantier
  à part, avec son propre processus de revue.
- **Pas de mesure d'audience sur le site.** Décidé en A-6 et non rouvert ici : pas
  d'analytique, pas de cookie.
- **Pas de relance.** Un canal qui ne prend pas ne se retente pas avec le même projet.

## Risques

| Risque | Parade |
|---|---|
| Le premier passage tombe à plat | On l'a préparé : rien à compiler, deux voies d'installation essayées pour de vrai, une porte pour les bogues |
| Un afflux tape sur la connexion domestique | Le site est statique et sans dépendance serveur ; le déménager est une copie de dossier (A-6, §1) |
| Un compte Reddit neuf se fait filtrer | Le compte d'Yves a de l'historique — vérifié avant de retenir le canal |
| Une PR arrive sur le miroir | Gabarit et action la referment en expliquant, plutôt que de la laisser sans réponse |
| Un bogue découvert en pleine annonce | La chaîne de release permet de publier un correctif dans l'heure ; c'est même la meilleure réponse possible |
| Le miroir se désynchronise | Le jeton GitHub expire un jour : le noter, et vérifier le miroir à chaque release |

## Décisions liées

- [Le site de jigger](2026-08-16-site-jigger-design.md) — A-6, dont l'annonce dépend.
- [Internationalisation](2026-08-16-i18n-design.md) — l'anglais comme langue de publication.
- `docs/ameliorations.md` — A-7 (ce document), A-18 (la section `#jigger` périmée du site
  Cocktails, à reprendre avant que les liens croisés se contredisent).
