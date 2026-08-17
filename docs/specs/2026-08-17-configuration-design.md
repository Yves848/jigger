# Un écran de configuration — conception

17 août 2026 — état : validé, implémentation directe.

## Objet

Donner à jigger un écran de réglage, et la configuration persistante qu'il suppose.

C'est l'entrée **A-14** du tableau. **A-15** en dépend pour sa durée de grâce.

## Ce que la mesure a établi

Le relevé, avant toute conception :

| Lu par | Réglages |
|---|---|
| **Le greffon seul** | `ROWS`, `KEY`, `KEYS_EXTRA`, `COMMANDS`, `MIN_COLUMNS`, `PROMPT`, `PROMPT_TTL`, `BIN` |
| Les deux | `LIVE`, `LANG`, `CACHE_DIR` |
| Le binaire seul | `PAGER` |

Huit sur douze appartiennent au greffon, lu au chargement du shell. C'est ce chiffre qui
décide de la forme de l'écran, et non l'inverse.

Le fichier, son emplacement, son format et la préséance sont arrêtés par
[l'ADR-0003](../adr/0003-fichier-de-configuration.md).

## §1 — Le fichier, et qui gagne

`os.UserConfigDir()/jigger/config`, format `clé = valeur`, préséance **environnement >
fichier > défauts**.

Conséquence non négociable : **l'écran affiche la provenance de chaque valeur**. Un réglage
écrasé par l'environnement est signalé comme tel, sans quoi l'écran montrerait une valeur
que la machine n'applique pas.

Les greffons ne lisent pas le fichier : `eval "$(jigger config --export)"`, fondu dans
l'appel de version qu'ils font déjà.

## §2 — Ce que l'écran montre

Trois groupes, qui correspondent à trois **natures**, pas à un classement esthétique :

1. **Ce qui prend effet tout de suite** — `PAGER`, `LANG`, `CACHE_DIR`.
2. **Ce qui prend effet au prochain shell** — les huit réglages du greffon. Dit une fois,
   sur le groupe, plutôt qu'à chaque ligne.
3. **Ce que jigger observe sans le posséder** — `$SCOOP`, `$SCOOP_GLOBAL`,
   `$HOMEBREW_PREFIX`, les gestionnaires détectés, les catalogues en cache et leur âge.
   **En lecture seule** : les proposer à la modification serait mentir, ils appartiennent
   aux gestionnaires.

Le troisième groupe est celui qui rend l'écran utile tout de suite : répondre à « qu'est-ce
que jigger voit de mon installation ? » demande aujourd'hui de lire la documentation et
d'inspecter son environnement à la main.

## §3 — La déclaration par gestionnaire

Une seconde table à côté de `pm.Bindings`, dans l'esprit de l'ADR-0002 : chaque
gestionnaire déclare ses réglages — clé, libellé traduit, type, défaut. **L'écran dérive sa
mise en page de ces déclarations** : ajouter un réglage à scoop le fait apparaître sans
toucher au code de l'écran.

Ce qu'il y a à déclarer aujourd'hui est modeste et réel : la **durée de validité du
catalogue**, écrite en dur à 24 h chez brew comme chez winget. La déclarer la rend
réglable — 1 h pour qui `tap` souvent, une semaine sur une liaison lente.

Conséquence assumée : ces durées cessent d'être des constantes du chemin de la frappe. On
s'en tient aux TTL plutôt que d'inventer une table riche qui n'aurait rien à porter.

## §4 — Comment on saura que ça marche

1. **La préséance est une fonction pure** — `Resoudre(env, fichier, defaut) → (valeur,
   provenance)`. Testée exhaustivement, sans fichier ni shell, valeurs vides comprises :
   une variable d'environnement vide compte comme absente, une valeur vide dans le fichier
   est un choix délibéré.
2. **Aller-retour du fichier** : écrire, relire, retrouver les mêmes valeurs — commentaires
   et espaces compris.
3. **Ce que `config --export` émet doit survivre aux deux shells.** Éprouvé par
   **exécution** : des valeurs hostiles — espaces, apostrophes, guillemets, accents,
   `$(...)`, retours à la ligne — passées à un vrai `zsh -c` et à un vrai `pwsh -c`, et
   comparées à ce qui ressort. Jamais par relecture du code d'échappement : c'est
   exactement là que le projet s'est fait prendre, une apostrophe non échappée ayant
   tronqué des messages dans les deux langues sur un chemin qu'aucun test n'exerçait.
4. **La provenance affichée est juste** : un réglage écrasé par l'environnement est
   signalé.

## Portée

Le fichier, sa lecture, l'export vers les greffons, l'écran, et la déclaration des TTL par
gestionnaire.

## Non-buts

- **L'écran n'écrit jamais dans `~/.zshrc` ni dans `$PROFILE`.** Il écrit son fichier, et
  rien d'autre.
- **Aucun secret dans le fichier.** A-15 y mettra une durée, jamais un mot de passe.
- **Pas de rechargement à chaud des réglages du greffon.** Ils prennent effet au prochain
  shell, et l'écran le dit plutôt que de le laisser découvrir.
- **Pas de table de réglages riche par gestionnaire** — les TTL suffisent aujourd'hui ; le
  reste serait spéculatif.
- **Pas de migration** : il n'y a rien à migrer, le fichier n'existe pas encore.

## Risques

| Risque | Parade |
|---|---|
| L'écran promet un effet qu'il n'a pas | La provenance est affichée, et le groupe « au prochain shell » est nommé comme tel |
| L'échappement casse un shell | Éprouvé par exécution dans les deux shells, avec des valeurs hostiles |
| Trois implémentations de la préséance divergent | Il n'y en a qu'une, en Go (ADR-0003) |
| Un fichier corrompu casse l'ouverture du shell | `config --export` n'émet rien plutôt que du bruit ; le greffon garde ses défauts |
| Les TTL configurables ralentissent la frappe | Lues une fois au chargement du catalogue, pas à chaque frappe |

## Décisions liées

- [ADR-0003](../adr/0003-fichier-de-configuration.md) — le fichier et sa préséance.
- [ADR-0002](../adr/0002-facade-table-declarative.md) — l'esprit déclaratif dont hérite §3.
- `docs/ameliorations.md` — A-14 (ce document), A-15 qui en dépend.
