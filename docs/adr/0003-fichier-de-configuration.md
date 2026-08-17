# ADR-0003 — Un fichier de configuration, dicté aux greffons par le binaire

17 août 2026 — **acceptée**

## Contexte

jigger n'a jamais eu de configuration : tout se règle par variables d'environnement —
douze aujourd'hui, de `JIGGER_LIVE` à `JIGGER_PAGER`. C'était tenable tant que jigger était
un binaire qu'un greffon appelait ; ça ne l'est plus dès qu'on veut un écran de réglage
(A-14), et encore moins quand un réglage devra survivre à la fermeture du shell — la durée
de grâce d'une élévation, par exemple (A-15).

Un relevé, plutôt qu'une impression, montre où est la difficulté :

| Lu par | Réglages |
|---|---|
| **Le greffon seul** | `ROWS`, `KEY`, `KEYS_EXTRA`, `COMMANDS`, `MIN_COLUMNS`, `PROMPT`, `PROMPT_TTL`, `BIN` |
| Les deux | `LIVE`, `LANG`, `CACHE_DIR` |
| Le binaire seul | `PAGER` |

**Huit réglages sur douze sont lus par le greffon**, au chargement du shell, avant que le
binaire ne tourne. Un écran lancé depuis le binaire ne peut donc rien leur appliquer par
lui-même : la question n'est pas « comment dessiner un écran » mais « comment un réglage
franchit la frontière entre le binaire et le shell ».

## Options considérées

**A — Pas de fichier : l'écran imprime les lignes à coller.** Aucun format, aucune
préséance, aucune promesse non tenue. Mais ce n'est pas une configuration : c'est une aide
à la rédaction d'un `.zshrc`, et rien ne survit à l'écran.

**B — Un fichier que chacun analyse.** Format `clé = valeur`, lu par le binaire en Go, par
le greffon en zsh, par le module en PowerShell. Trente lignes chacun. Mais **trois
implémentations de la préséance**, qui divergeront — c'est précisément le défaut que ce
projet a corrigé trois fois cette semaine, la dernière étant une clé de cache qui ne portait
pas tout ce dont elle dépendait.

**C — Un fichier, mais le binaire le dicte.** Le greffon fait
`eval "$(jigger config --export)"`, le module PowerShell son équivalent. Le fichier n'est
lu, analysé et arbitré qu'en Go.

## Décision

**C.** Le binaire est la seule autorité sur la configuration.

- **Emplacement** : `os.UserConfigDir()/jigger/config` — `~/Library/Application Support/`
  sur macOS, `%APPDATA%` sous Windows, `~/.config/` ailleurs. Symétrique du cache, qui
  emploie déjà `os.UserCacheDir()`.
- **Format** : `clé = valeur`, commentaires au `#`. Sous-ensemble de TOML : le jour où des
  sections deviendraient nécessaires, on y passe sans casser les fichiers existants.
- **Préséance** : **environnement > fichier > défauts**.
- **Lecture par les greffons** : `jigger config --export`, fondu dans l'appel de
  vérification de version que les deux greffons font déjà.

## Justification

**Une seule implémentation de la préséance.** C'est l'argument décisif. Trois analyseurs,
c'est trois occasions de diverger sur une valeur vide, une casse, un espace autour du signe
égal — et une divergence de configuration se manifeste par « ça marche chez moi », le pire
symptôme à diagnostiquer.

**Le coût est mesuré, pas supposé** : `jigger --version` prend **2,5 ms**, dont 1,4 ms de
fork nu, sur un shell interactif qui met 130 à 320 ms à s'ouvrir. Un appel de plus est
indolore, et il se fond dans celui qui existe déjà.

**Aucune dépendance ajoutée.** Le projet n'en a aucune pour les formats de données ; TOML
en coûterait une, dans un binaire qui se veut autonome, pour un fichier d'une douzaine de
lignes.

**L'environnement garde le dernier mot.** Quiconque a posé `JIGGER_ROWS=12` dans son
`.zshrc` continue de l'emporter, et un `JIGGER_LANG=fr` de passage reste possible. Un
fichier qui écraserait l'environnement retirerait une capacité à des gens qui s'en servent
déjà.

## Conséquences

- **L'écran doit afficher la provenance de chaque valeur** — défaut, fichier, ou
  environnement. Sans cela, il montrerait une valeur qu'on vient de choisir pendant que la
  machine en applique une autre. C'est la contrepartie directe de la préséance retenue, et
  elle n'est pas négociable.
- **Les greffons dépendent du binaire au chargement**, un peu plus qu'avant. Le
  garde-fou de version existe déjà et couvre ce risque : un binaire absent ou trop ancien
  désactive le greffon en le disant.
- **Ce que `config --export` émet devient un contrat**, et il traverse deux langages de
  shell. Le projet s'est déjà fait prendre par une apostrophe non échappée qui tronquait des
  messages dans les deux langues. Ce point s'éprouve par exécution — des valeurs hostiles
  passées à un vrai `zsh -c` et à un vrai `pwsh -c` —, jamais par relecture.
- **Les durées de cache cessent d'être des constantes.** Déclarées par chaque gestionnaire,
  elles sont lues au lieu d'être compilées : c'est une modification du chemin de la frappe,
  pas seulement de l'affichage.
- **L'écran n'écrit jamais dans `~/.zshrc` ni dans `$PROFILE`.** Il écrit son fichier, et
  rien d'autre.
- La conception qui découle de cet ADR est dans
  [la spec du 17 août 2026](../specs/2026-08-17-configuration-design.md).
