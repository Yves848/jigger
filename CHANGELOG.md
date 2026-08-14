# Changelog

Toutes les évolutions notables de `jigger` sont consignées ici.

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le versionnage
[SemVer](https://semver.org/lang/fr/). Les versions antérieures à `v0.1.6` sont antérieures
à ce journal ; leur détail est dans l'historique git.

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
