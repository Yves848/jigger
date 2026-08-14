# Changelog

Toutes les évolutions notables de `jigger` sont consignées ici.

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le versionnage
[SemVer](https://semver.org/lang/fr/). Les versions antérieures à `v0.1.6` sont antérieures
à ce journal ; leur détail est dans l'historique git.

## [v0.7.0] — 2026-08-15

Le portage Windows des deux versions précédentes avait corrigé, chemin faisant, des
défauts qui n'avaient rien de propre à winget : ils étaient tout autant dans le greffon
zsh, où personne ne les avait vus. Cette version les y corrige — le côté Homebrew revient
au niveau du côté PowerShell.

### Corrigé

- **Le popup zsh ne s'affichait pour ainsi dire jamais.** Il n'était dessiné que s'il
  tenait sous la ligne de commande — or dans un terminal en usage, le prompt occupe la
  dernière ligne de l'écran, et il n'y a rien en dessous. Le greffon fait désormais la
  place en poussant l'écran, comme n'importe quel sélecteur en incrustation (`fzf
  --height`), et remet le curseur sur la ligne de commande, qui a monté d'autant. C'est
  le pendant du `New-JiggerRoom` de la 0.6.0 ; zle n'a rien à en savoir, il se déplace en
  relatif à partir de là où il a laissé le curseur.
- **⇥ ouvrait le sélecteur plein écran par surprise, sous zsh aussi.** Quand le popup
  n'avait rien à proposer — aucun candidat, ou pas la place de s'afficher —, ⇥ tombait
  sur `jigger pick`, qui dessinait par-dessus le prompt et attendait une touche. ⇥ rend
  maintenant la main à la complétion du shell ; le sélecteur plein écran reste ce qu'on
  obtient avec `JIGGER_LIVE=0`, c'est-à-dire quand on l'a demandé.
- **Une frappe pouvait attendre après brew.** Passé 24 h, le catalogue était reconstruit
  dans le chemin du rendu : `brew formulae` puis `brew casks`, soit une bonne seconde,
  la première fois qu'on tapait `brew i…` de la journée. Le catalogue se lit désormais
  dans le cache et rien d'autre ; un cache périmé est utilisé tel quel et déclenche un
  réchauffement détaché, qui le refera pour la frappe suivante — la règle que winget
  suivait déjà. À la toute première utilisation, le cadre le dit (« catalogue Homebrew en
  préparation… ») plutôt que d'annoncer « aucun candidat ».
- `brew list --versions` ne tourne plus, lui non plus, dans le chemin d'une frappe : ce
  recours au préfixe Homebrew inattendu est passé dans `jigger warm`.

### Ajouté

- **Le greffon zsh vérifie la version du binaire** au `source`, et signale un binaire
  absent du `PATH`. Greffon et binaire vont par paire : le greffon passe à `jigger render`
  des options qu'un binaire plus ancien ne connaît pas — celui-ci sort alors en erreur, et
  le popup ne s'affiche jamais, sans un mot. C'est le pendant du contrôle que le module
  PowerShell fait depuis la 0.6.0.
- **Le greffon zsh réchauffe le catalogue** : au chargement (`jigger warm` détaché, comme
  à l'import du module PowerShell), et après un `brew update`, `tap` ou `untap` — les
  seules sous-commandes qui changent la liste des noms connus. La détection des commandes
  mutantes ne vit plus dans le bloc oh-my-posh : le catalogue sert à la complétion, pas au
  prompt, et n'a donc pas à dépendre de `JIGGER_PROMPT`.
- Le harnais zpty **sait simuler le défilement** — il répondait jusqu'ici invariablement
  « ligne 3 » à l'interrogation de position, ce qui rendait le premier bogue ci-dessus
  strictement invisible. Deux cas de plus : le popup avec le prompt en bas de l'écran, et
  ⇥ sans candidat.

## [v0.6.0] — 2026-08-14

### Modifié

- **Les flèches pilotent le popup, mais seulement une fois qu'il a le clavier.** `↓` fait
  entrer dans la liste, `↑` en ressort dès le premier candidat ; tant que le popup n'a pas
  le focus, `↑` et `↓` restent l'historique du shell, popup ouvert ou non. Ouvrir une
  liste de candidats ne coûte donc pas l'accès à la commande précédente — ce qui était
  toute la raison de ne pas toucher aux flèches jusqu'ici.
- La ligne courante **montre** l'état du focus : soulignée en accent quand le popup a le
  clavier, au repos sur le fond des pastilles quand il ne l'a pas. Le pied change avec
  elle — `↓ parcourir` puis `↑↓ naviguer`. Sans ce signe, la règle serait invisible.
- `^N`/`^P` suivent exactement la même règle et restent disponibles.
- Les deux greffons rendent la touche à ce qu'elle faisait avant eux : si les flèches sont
  déjà tenues par un autre greffon (recherche par préfixe dans l'historique…), c'est lui
  qui reprend la main hors focus.

### Corrigé

- **L'écran scintillait quand PSReadLine affiche ses prédictions en liste.** `ListView`
  se dessine exactement là où le popup se dessine : les deux se disputaient les mêmes
  lignes à chaque frappe. jigger range la vue le temps du cadre — la prédiction repasse
  en `InlineView`, qui n'occupe aucune ligne à elle — et la rend ensuite.
- **Une ligne du cadre pouvait déborder, et le terminal la repliait.** Le nom d'un paquet
  était tronqué à une largeur fixe, sans tenir compte de la place que prenaient à droite
  la version et le point d'installé — or un identifiant winget hors catalogue suivi d'une
  version à quatre nombres (`ARP\Machine\X64\{226CEF88…  6.4.0.3079  ●`) dépasse largement.
  La ligne repliée faisait occuper au popup deux fois les lignes annoncées : le cadre se
  redessinait plus bas à chaque frappe, et l'écran se remplissait de cadres empilés. Le
  nom se taille désormais sur ce qui reste vraiment, et aucune ligne ne sort du cadre —
  un garde-fou la coupe si le calcul se trompe.
- **Le paquet fraîchement installé manquait à la complétion de `uninstall`.** Le
  rafraîchissement de la liste des installés était enfermé dans le bloc de prompt : sans
  `JIGGER_PROMPT=1`, il n'avait jamais lieu, et le cache restait faux jusqu'à sa
  péremption. Il ne dépend plus de lui.
- **Le popup ne s'affichait pour ainsi dire jamais sous PowerShell.** Il n'était dessiné
  que s'il tenait sous la ligne de commande — or dans un terminal en usage, le prompt
  occupe la dernière ligne de l'écran, et il n'y a rien en dessous. jigger fait désormais
  la place en poussant l'écran, comme n'importe quel sélecteur en incrustation, et
  recale l'ancre de PSReadLine du même nombre de lignes : sans cela, la ligne de commande
  se réaffichait plus bas, précédée d'autant de vide.
- **⇥ ouvrait le sélecteur plein écran par surprise.** Quand le popup n'avait rien à
  proposer — aucun candidat, ou pas la place de s'afficher —, ⇥ tombait sur `jigger pick`,
  qui dessinait par-dessus le prompt et attendait une touche. ⇥ rend maintenant la main à
  la complétion du shell ; le sélecteur plein écran reste ce qu'on obtient avec
  `JIGGER_LIVE=0`, c'est-à-dire quand on l'a demandé.
- **`winget` ou `scoop` seul annonçait « aucun candidat »** au lieu de proposer les
  sous-commandes : le mot en cours étant le nom de la commande, il était cherché parmi
  celles-ci — et aucune ne s'appelle « winget ».

### Ajouté

- `jigger render --focus=true|false` : le focus vit côté shell, comme l'index sélectionné,
  et revient à chaque rendu. `render` reste sans état.
- Le module PowerShell **vérifie la version du binaire** à l'import. Module et binaire vont
  par paire : un binaire plus ancien ne connaît pas les options que le module lui passe, il
  sort en erreur, et le popup ne s'affiche jamais — sans un mot. Il le dit désormais.
- **Un harnais de pseudo-terminal pour Windows** (`tests/conpty`) et la suite qui va avec
  (`tests/pty.ps1`, `make test-pty`). Il lance un pwsh dans un ConPTY, tape des touches,
  et **rend l'écran** tel qu'on le verrait — cadre compris. C'est le pendant de
  `tests/zpty.zsh`, et il existe pour la même raison : les trois bogues ci-dessus ne se
  voyaient qu'à l'écran, aucun n'aurait été trouvé autrement.

## [v0.5.0] — 2026-08-14

### Ajouté

- **Windows : winget et scoop**, avec la même ligne de commande et le même popup que
  Homebrew. C'est désormais le premier mot de la ligne qui désigne le gestionnaire —
  `brew`, `winget` ou `scoop` —, chacun apportant ses sous-commandes, ses options, son
  catalogue et ses corrections d'insertion. Un nouveau paquet `internal/pm` porte ce
  contrat ; `internal/brew`, `internal/winget` et `internal/scoop` le remplissent.
- **Module PowerShell** (`shell/jigger.psm1`), pendant du plugin zsh : popup vivant, ⇥,
  `^N`/`^P`/`^G`, et le bloc de prompt. PSReadLine n'offrant aucun crochet appelé à chaque
  frappe, jigger réenregistre les touches qui modifient la ligne derrière un relais qui
  rappelle la fonction PSReadLine d'origine avant de redessiner : aucun comportement
  d'édition n'est réécrit. `JIGGER_KEYS_EXTRA` couvre les touches non ASCII (« éèçàù »
  d'un clavier AZERTY).
- **`jigger warm`** reconstitue les catalogues lents hors du chemin d'un rendu.
  `--installed` refait les seules listes de paquets installés — ce qui change après une
  installation —, `--all` refait tout. `render` le lance détaché dès qu'il trouve un cache
  périmé : une frappe n'attend jamais après winget.
- **Corrections d'insertion propres à chaque gestionnaire** : scoop qualifie par son
  bucket un nom qui en occupe plusieurs (`main/flux`, là où scoop se contente d'un
  avertissement avant de choisir à ta place) ; winget protège par des guillemets un
  identifiant contenant des espaces.
- Segment oh-my-posh Windows (`shell/oh-my-posh/windows.segment.json`) et variables
  `JIGGER_WINGET_VERSION`, `JIGGER_WINGET_OUTDATED`, `JIGGER_SCOOP_OUTDATED`,
  `JIGGER_OUTDATED`.
- `tests/smoke.ps1` : la suite d'assertions du module PowerShell, et `make test-all` la
  lance à la place de la suite zsh sous Windows.

### Modifié

- Les candidats sont triés **sans tenir compte de la casse**. Un tri brut aurait rangé
  tous les identifiants winget capitalisés avant les autres — `Microsoft.PowerShell` loin
  devant `mailspring` — alors que le filtre, lui, ignore la casse : la liste aurait paru
  mélangée.
- `prompt.Status` nomme ses deux compteurs `Primary` et `Secondary` : ils portent les
  formulae et les casks sous Homebrew, winget et scoop sous Windows. Le format du fichier
  de cache, lui, ne change pas — les hooks des deux shells lisent la même ligne.

### Notes

- Le catalogue winget s'obtient en cherchant `.` : le point qui sépare l'éditeur du paquet
  dans tous les identifiants de la source officielle. winget n'ayant aucune sortie machine,
  ses tableaux sont découpés aux frontières de colonnes lues sur l'en-tête — la seule
  méthode qui survive à des en-têtes traduits et à des identifiants contenant des espaces.
  Les jeux d'essai sont des sorties réellement capturées, en français.
- scoop n'a besoin d'aucun cache : catalogue, paquets installés et mises à jour en attente
  se lisent tous sur le disque, en quelques millisecondes.

## [v0.4.3] — 2026-08-14

### Corrigé

- **Le bloc oh-my-posh restait figé après un `brew upgrade`.** Le cache n'était rafraîchi
  que par péremption de TTL : le compteur pouvait donc annoncer dix mises à jour en
  attente pendant une demi-heure après les avoir toutes installées. jigger repère
  désormais, en `preexec`, les commandes brew qui changent l'état (`install`, `upgrade`,
  `uninstall`, `update`, `tap`, `pin`…) et rafraîchit avant d'afficher le prompt suivant —
  la seule attente que le plugin s'autorise, et seulement là où le cache est *connu* faux.
  `JIGGER_PROMPT_SYNC=0` la rend détachée, au prix d'un compteur juste au prompt d'après.
- Le hook `precmd` de jigger se place de lui-même **en tête** de `precmd_functions` :
  chargé après oh-my-posh, il exportait ses compteurs une fois le prompt déjà calculé, et
  le bloc gardait un coup de retard quel que soit le reste.

### Ajouté

- `jigger prompt --refresh --wait` attend le verrou au lieu d'y renoncer. C'est ce
  qu'emprunte le rafraîchissement forcé : un rafraîchissement paresseux parti pendant
  l'upgrade tient le verrou tant que brew ne lui répond pas, et renoncer là aurait laissé
  le compteur faux dans le cas même qu'on corrige.

## [v0.4.2] — 2026-08-14

### Modifié

- Le bloc oh-my-posh utilise des **émojis** plutôt que des glyphes Nerd Font :
  `🍺 6.0.17  🧪 9  📦 1`. Ils ne dépendent d'aucune police particulière — le bloc
  s'affiche donc partout, là où un glyphe de la zone à usage privé se réduisait à un carré
  vide sans Nerd Font. La contrepartie : un émoji impose sa propre couleur, seuls les
  compteurs suivent celle du segment. Les glyphes Nerd Font restent documentés dans le
  README et dans le fragment, pour qui préfère du monochrome.

## [v0.4.1] — 2026-08-14

### Corrigé

- **L'icône Homebrew du bloc oh-my-posh ne s'est jamais affichée.** Le glyphe, écrit en
  clair dans le fichier, n'a pas survécu aux éditions successives du template : le bloc
  s'ouvrait sur un blanc depuis la v0.3.0. Tous les glyphes sont désormais écrits en
  **échappements JSON** (`\uf0fc`…), la seule forme qui traverse sans dommage les
  éditeurs, les copier-coller et les outils qui normalisent l'Unicode — c'est d'ailleurs
  ce que font les thèmes livrés par oh-my-posh.

### Modifié

- Chaque type de paquet a son **icône** plutôt qu'une lettre : une **fiole** pour les
  formulae, un **cube** pour les casks. ` 6.0.17   9   1` remplace
  ` 6.0.17 ⇡9F ⇡1C`. La flèche disparaît aussi : un compteur ne s'affichant jamais à
  zéro, sa seule présence dit déjà « à mettre à jour ».

## [v0.4.0] — 2026-08-14

### Ajouté

- Le bloc oh-my-posh compte désormais **formulae et casks séparément** :
  ` 6.0.17 ⇡7F ⇡2C`. Les badges `F`/`C` sont ceux du sélecteur. Deux nouvelles
  variables, `JIGGER_BREW_FORMULAE` et `JIGGER_BREW_CASKS`, exportées selon la même règle
  que les autres compteurs — **non définies quand elles valent zéro**, si bien que chaque
  moitié du bloc s'efface d'elle-même.

### Modifié

- Le segment livré dans `shell/oh-my-posh/brew.segment.json` affiche le détail F/C. Le
  total reste disponible dans `JIGGER_BREW_OUTDATED` : le template d'origine, à un seul
  chiffre, est rappelé dans le README et dans le fichier lui-même.

## [v0.3.0] — 2026-08-14

### Ajouté

- **Bloc oh-my-posh** : la version de brew et le nombre de mises à jour en attente
  s'affichent dans le prompt. `JIGGER_PROMPT=1` active un hook `precmd` qui exporte
  `JIGGER_BREW_VERSION` et `JIGGER_BREW_OUTDATED` ; le segment `text` à coller est livré
  dans `shell/oh-my-posh/brew.segment.json`. `JIGGER_BREW_OUTDATED` reste **non défini**
  quand tout est à jour, ce qui masque le compteur sans comparaison dans le template.
- `jigger prompt` : lit l'état de Homebrew en cache (`--refresh` l'interroge et le
  réécrit, `--path` donne le fichier).

### Détails d'implémentation

- `brew outdated` coûtant de une à cinq secondes, il ne tourne **jamais** dans le chemin
  du prompt : `jigger prompt --refresh` est lancé détaché quand le cache dépasse
  `JIGGER_PROMPT_TTL` (30 min par défaut), et le hook se contente de relire une ligne
  avec les builtins de zsh — **0,03 ms par prompt, aucun fork**.
- Écriture atomique du cache (fichier temporaire + `rename`) et verrou de
  rafraîchissement : dix terminaux ouverts ne déclenchent qu'un seul appel à brew. Un
  verrou de plus de 5 minutes est réputé abandonné.
- Si brew est injoignable, le cache précédent est **conservé** : un prompt n'affiche
  jamais d'erreur.
- `JIGGER_CACHE_DIR` permet de déplacer le cache (et sert aux tests).

## [v0.2.0] — 2026-08-13

### Ajouté

- **Popup vivant** : dès que la ligne courante est une commande `brew`, le sélecteur
  s'affiche sous le prompt et se filtre au fil de la frappe — plus besoin de presser ⇥
  pour le faire apparaître. `^N`/`^P` naviguent, `⇥` insère le candidat courant, `^G`
  ferme le popup pour la ligne en cours (`⇥` le rouvre). Les flèches `↑`/`↓` ne sont
  jamais détournées : l'historique zsh reste intact. (#2)
- Après `brew install`, où le mot est vide et le catalogue compte des milliers d'entrées,
  le cadre invite à taper une lettre au lieu d'égrener la liste ; sous 300 candidats —
  les paquets installés — la liste s'affiche directement. (#2)
- Réglages `JIGGER_LIVE` (0 = retour au mode ⇥ seul), `JIGGER_ROWS`, `JIGGER_MIN_COLUMNS`.
  Le popup se réduit ou s'efface si le terminal est court, étroit, ou muet à
  l'interrogation de position du curseur. (#2)
- Sous-commande `jigger render` : une frame du popup, sans état ni clavier, précédée d'une
  ligne de métadonnées (`count`, `sel`, `exec`, `left`). (#2)
- `tests/zpty.zsh` : suite de tests du widget dans un vrai pseudo-terminal
  (`make test-shell`), rejouable avec zsh-autosuggestions et zsh-syntax-highlighting
  chargés (`JIGGER_TEST_PLUGINS=1`). (#2)

### Modifié

- Les paquets installés sont lus directement dans `Cellar`/`Caskroom` (~1 ms) au lieu de
  `brew list --versions` (~300 ms) : un appel complet passe de ~300 ms à **~8 ms**. C'est
  ce qui rend tenable un rendu à chaque frappe, et ça accélère aussi le mode ⇥. (#2)

## [v0.1.6] — 2026-08-13

### Modifié

- ⇥ sur une ligne dont la complétion ne laisse qu'un seul candidat insère directement ce
  candidat, sans afficher le sélecteur — comme le fait la complétion zsh sur une
  correspondance unique. Le popup ne s'ouvre plus que lorsqu'il y a un choix à faire.
  (#1)

[v0.2.0]: https://gitlab.yg-devworks.com/yves/jigger/-/releases/v0.2.0
[v0.1.6]: https://gitlab.yg-devworks.com/yves/jigger/-/releases/v0.1.6
