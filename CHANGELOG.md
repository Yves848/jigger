# Changelog

Toutes les évolutions notables de `jigger` sont consignées ici.

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le versionnage
[SemVer](https://semver.org/lang/fr/). Les versions antérieures à `v0.1.6` sont antérieures
à ce journal ; leur détail est dans l'historique git.

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
