# Les plugins

*Lire en [anglais](../plugins.md).*

Un plugin apprend à jigger un gestionnaire de paquets qu'il ne connaît pas — sans le
recompiler. On écrit un binaire, on pose un `config.json` à côté, et `brew`, `winget` et le
vôtre se comportent pareil dans le popup et derrière `jg`.

Le design est [le plan d'injection de plugins](../plans/2026-09-04-plugins-injection.md) ;
la façon dont un plugin est **exécuté** est
[l'ADR-0008](../adr/0008-execution-des-plugins.md).

## En installer un

Un plugin est un dossier qui porte un descripteur et, le plus souvent, son binaire :

```
~/.config/jigger/plugins/<nom>/
├── config.json      le descripteur
└── jigger-<nom>     le binaire (facultatif — le $PATH marche aussi)
```

Trois dossiers sont parcourus, dans cet ordre — le premier descripteur trouvé pour un nom
donné gagne, si bien que ce que vous posez dans votre configuration supplante toujours un
plugin installé pour toute la machine :

| Dossier | À quoi il sert |
|---|---|
| `~/.config/jigger/plugins/` | le vôtre (respecte `$XDG_CONFIG_HOME`) |
| `/usr/local/lib/jigger-plugins/` | système, en lecture seule |
| `<cache jigger>/plugins/` | installé par un tiers |

Puis on remplit les caches une fois :

```sh
jigger warm --all
```

Un plugin dont le binaire est introuvable est ignoré en silence, et repris dès qu'il
apparaît. Un plugin qui prend le nom d'un gestionnaire natif est refusé, avec une ligne sur
la sortie d'erreur : `brew`, c'est `brew`.

## Le plugin `git`

Il est livré avec jigger, dans [`packaging/plugins/git/`](../../packaging/plugins/git). Il
voit **vos clones locaux comme des paquets** : installer, c'est cloner ; désinstaller,
c'est supprimer le clone ; mettre à jour, c'est tirer.

```sh
cp -r packaging/plugins/git ~/.config/jigger/plugins/
go build -o ~/.config/jigger/plugins/git/jigger-git ./cmd/jigger-git
jigger warm --all
```

```console
$ jg list --pm git
PACKAGE        CURRENT                SOURCE
config         feat/clavier-macarchy  https://gitlab.yg-devworks.com/yves/config.git
jigger         main                   https://gitlab.yg-devworks.com/yves/jigger.git
omarchy        fix/sddm-greeter…      https://github.com/Yves848/omarchy.git
```

La **version** d'un dépôt est sa branche courante — c'est ce qui distingue deux états du
même clone. Sa **source** est l'URL d'origine.

### Où il cherche

`$JIGGER_GIT_ROOTS` (une liste à la façon du `$PATH`) a le dernier mot. Sans elle : `~/git`,
`~/Projets`, `~/Code`, `~/dev`, `~/src`. Il descend de deux niveaux, ce qui attrape aussi
bien `~/git/projet` que `~/git/client/projet` — et il s'arrête à tout dossier portant un
`.git`, si bien que les sous-modules et les worktrees ne remontent pas comme des paquets à
part.

### Les verbes

| Commande | Ce qu'elle fait |
|---|---|
| `jg list --pm git` | les clones trouvés |
| `jg outdated --pm git [--fetch]` | les clones en retard sur leur amont |
| `jg search --pm git <motif>` | filtre le catalogue |
| `jg install --pm git <nom>` | le clone |
| `jg uninstall --pm git <nom> [--force]` | supprime le clone |
| `jg upgrade --pm git [<nom>…]` | `git pull --ff-only`, tous si vous n'en nommez aucun |

`outdated` lit la référence de suivi, qui ne bouge qu'au fetch : sans réseau, il rend ce
que la dernière synchronisation sait, et peut très bien répondre *rien* d'un dépôt qui a
pris dix commits entre-temps. `--fetch` va le demander. Ce n'est pas le défaut, et c'est
délibéré : `outdated` peut porter sur des dizaines de dépôts, et une lecture ne doit pas
partir en réseau sans qu'on l'ait demandé.

`upgrade` tire en `--ff-only` : une mise à jour ne doit pas fabriquer un commit de fusion
dans votre dos, ni vous laisser au milieu d'un conflit.

### La suppression est gardée

`uninstall` supprime un dossier pour de bon : il refuse tant que le clone porte du travail
que la suppression perdrait — modifications non validées, commits non poussés, ou pas de
distant du tout. `--force` lève la garde, mais il faut l'écrire.

```console
$ jg uninstall --pm git jigger
jigger-git : jigger : des commits ne sont pas poussés — relancez avec --force pour le supprimer quand même
```

### D'où viennent les URL de clonage

Rien n'est deviné. jigger ne fabriquera pas `https://github.com/<nom>.git` à partir d'un
mot pour cloner ce qui répondra. Un nom se résout dans cet ordre :

1. **une URL** que vous donnez — `jigger-git run install https://…` (toute forme que git
   sait cloner) ;
2. **`depots.json`**, la table que vous écrivez à la main, à côté du descripteur ;
3. **`connus.json`**, les origines que jigger retient des clones déjà vus.

C'est la troisième qui rend le modèle complet : sans elle, un dépôt supprimé par
`uninstall` ne pourrait plus jamais être recloné, alors que jigger venait d'en afficher
l'URL.

```jsonc
// ~/.config/jigger/plugins/git/depots.json
{
  "jigger":  "https://gitlab.yg-devworks.com/yves/jigger.git",
  "omarchy": "https://github.com/Yves848/omarchy.git"
}
```

`jg install` n'accepte que des noms du catalogue — c'est le garde-fou qui rattrape une
faute de frappe avant qu'elle ne clone quelque chose. Pour cloner une URL qui n'est dans
aucune des deux tables : soit vous l'inscrivez dans `depots.json`, soit vous appelez le
binaire directement — `jigger-git run install <url>`.

## En écrire un

Un plugin est un programme qui répond en JSON. Les verbes de lecture écrivent **un** seul
document sur la sortie standard et rien d'autre ; les verbes d'écriture relaient leur outil
tel quel, terminal compris, pour qu'une demande de mot de passe ou une barre de progression
arrive jusqu'à l'utilisateur.

```console
$ jigger-mien catalog
{"names":["foo","bar"],"badges":{"foo":"R","bar":"X"}}

$ jigger-mien list
[{"name":"foo","version":"1.2.3","kind":"R","source":"…"}]

$ jigger-mien run install foo        # relaie, le code de sortie dit ce qu'il en est
```

Un paquet porte `name`, `version`, `available`, `kind` et `source` ; seul `name` est
obligatoire. `kind` est un badge : `R` pour la classe ordinaire, `X` pour l'autre — le
popup les peint différemment.

### Le descripteur

```jsonc
{
  "name": "mien",              // le mot tapé dans le terminal
  "version": "1.0.0",
  "cmd": "jigger-mien",        // le binaire — PAS la même chose que name
  "platforms": ["linux", "darwin", "windows"],

  "verbs": {
    // « native » est l'argv COMPLET passé au binaire : jigger ne préfixe rien.
    "list":      {"native": ["list"],                       "pool": "aucun"},
    "search":    {"native": ["search", "{args}"],           "pool": "aucun"},
    "install":   {"native": ["run", "install", "{args}"],   "pool": "catalogue"},
    "uninstall": {"native": ["run", "uninstall", "{args}"], "pool": "installees"}
  },

  "warmup": {
    "catalog":   {"cmd": "jigger-mien", "args": ["catalog"]},
    "installed": {"cmd": "jigger-mien", "args": ["list"]}
  },

  "parse": {"package_fields": ["name", "version", "kind", "source"], "encoding": "utf-8"}
}
```

`name` et `cmd` ne sont **pas** la même chose. `name` est le mot de la ligne, `cmd` est le
programme à lancer. Les confondre dans un plugin `git` lancerait le vrai git.

**`pool`** dit d'où viennent les candidats d'un verbe, et jigger y confronte les
arguments :

| `pool` | Candidats | Effet sur les arguments |
|---|---|---|
| `catalogue` | tout ce qui est connu | doivent être des noms du catalogue |
| `installees` | les installés seulement | doivent être installés |
| `aucun` | aucun | passent tels quels |

Un terme de recherche n'est pas un nom de paquet : `search` prend `aucun`, sans quoi jigger
refuserait de chercher un mot qui n'est justement pas encore un nom connu.

**Les marqueurs** de `native` : `{args}` étale tous les arguments dans un seul appel — le
cas ordinaire ; `{arg}` fait **un appel par argument**, pour un gestionnaire qui n'installe
qu'un paquet à la fois. Les drapeaux tapés sur la ligne sont mis en tête des arguments, et
`{arg}` les enverrait donc dans un appel à eux seuls : préférez `{args}` sauf besoin
contraire.

**Quels verbes sont analysés** se décide sur le verbe, pas sur le pool : `list`, `outdated`,
`search` et `source` voient leur sortie capturée et lue comme du JSON, tout le reste est
relayé. `install` puise ses candidats dans le catalogue, mais c'est une écriture, et la
relayer est ce qui laisse passer une invite d'authentification.

### Les règles qu'un plugin respecte

- **Jamais dans le chemin du rendu.** `jigger render` tourne à chaque frappe. Un plugin est
  lancé par `jigger warm` et pour exécuter un verbe — jamais pour compléter une ligne. La
  complétion lit les caches que `warmup` a remplis.
- **Trente secondes, puis il est tué.** Un plugin qui ne rend pas la main ne doit pas
  laisser `jigger warm` planté derrière son verrou.
- **Échouer est permis, mentir non.** Un plugin qui sort en erreur laisse le cache
  précédent en place plutôt que de l'écraser par du vide, et ce qu'il a écrit sur la sortie
  d'erreur revient dans le message de jigger.
- **Un plugin ne réécrit pas la ligne.** `Insert` rend le nom tel quel : les corrections
  d'insertion sont des bugs de façade, pas un pouvoir donné à un tiers.
