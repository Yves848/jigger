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

Ce `warm` n'est pas qu'une affaire de caches : c'est lui qui **arme le popup** sur le mot
du plugin. Il dépose les mots découverts dans `plugin-commands`, au fond du cache, et les
greffons — zsh comme PowerShell — le lisent à leur chargement. Aucun défaut ne pourrait les
connaître : ils dépendent de ce qui est installé sur cette machine-ci. Ouvrez un nouveau
shell après le `warm`, et le mot du plugin répond.

Pour ne PAS armer un plugin, sans avoir à réécrire `JIGGER_COMMANDS` en entier :

```sh
JIGGER_PLUGIN_COMMANDS=0
```

Un plugin peut porter le nom d'une commande que vous tapez cent fois par jour et que vous
préférez laisser tranquille. Sur les lignes qui ne le concernent pas, jigger se tait de
lui-même : le mot n'est pas un de ses verbes, il n'a rien à en dire et n'ouvre aucun cadre.

## Le plugin `git`

Livré avec jigger, dans [`packaging/plugins/git/`](../../packaging/plugins/git). Il ne
remplace pas git : **il le rend commode**. `git ⇥` propose les vraies sous-commandes, et
chaque verbe propose ce que ce verbe attend.

```console
$ git ⇥
  add  branch  checkout  commit  diff  fetch  log  merge  pull  push
  rebase  remote  restore  stash  status  switch  tag

$ git checkout ⇥                         $ git tag ⇥
  feat/site-refonte      2 days ago          v0.18.0    18 minutes ago
  main                5 minutes ago          v0.17.1       5 hours ago
  vieille    [behind 3] 2 days ago          v0.17.0      11 hours ago

$ git push ⇥           $ git add ⇥
  github                 docs/historique/2026-09-05.md
  origin                 packaging/plugins/git/config.json

$ git commit -⇥
  -m  -a  --amend  --no-edit  --no-verify  --fixup  -S
```

Les candidats sont **calculés dans le répertoire courant, à la frappe** : ce sont vos
branches de ce dépôt-ci, pas celles d'un cache réchauffé ce matin.

### Il n'a pas de binaire

C'est un descripteur, et rien d'autre. Le binaire qu'il déclare est **git lui-même** :
jigger lance donc le vrai git pour exécuter un verbe, et le vrai git pour peupler un vivier
(`git branch --format=…`, `git remote`, `git ls-files --modified --others`). Un helper n'a
pas besoin d'un programme à lui quand la commande qu'il assiste sait déjà répondre.

### L'installer

```sh
cp -r packaging/plugins/git ~/.config/jigger/plugins/
jigger warm --all
```

Puis ouvrez un nouveau shell. Rien à compiler.

### Ce qu'il ne fait pas

Il ne connaît que les dix-sept verbes qu'il déclare. Sur tout le reste — `git bisect`,
`git worktree`, `git submodule` — jigger **se tait** : il n'a rien à en dire, et un cadre
vide vaudrait moins que le silence. Vos lignes git ordinaires ne sont donc jamais gênées.


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

**Des viviers par verbe, pour un helper de commande.** Les trois `pool` ci-dessus décrivent
un gestionnaire de paquets. Un helper — une commande existante qu'on rend commode — a besoin
d'autre chose : les *branches* derrière `checkout`, les *fichiers modifiés* derrière `add`,
les *distants* derrière `push`. Un verbe peut donc puiser dans un **vivier nommé**, déclaré
à part :

```jsonc
"verbs": {
  "checkout": {"native": ["checkout", "{args}"], "pool": "branches",
               "options": ["-b", "--detach"]}
},

"pools": {
  // « direct » : demandé à la frappe, dans le répertoire courant.
  "branches": {"regime": "direct", "args": ["viviers", "branches"]}
}
```

Deux régimes, et le choix n'est pas de commodité :

| `regime` | Quand | Ce que ça coûte |
|---|---|---|
| `cache` | vivier **gros et lent** à produire, stable d'une heure à l'autre | réchauffé par `jigger warm`, comme le catalogue |
| `direct` | vivier **petit et contextuel**, faux dès qu'il est mis en cache | un sous-processus **à chaque frappe**, borné à 200 ms |

Le binaire interrogé est **toujours celui du plugin** : le vivier ne déclare que des
arguments. Un descripteur ne doit pas pouvoir faire lancer n'importe quel programme à chaque
frappe.

Un vivier `direct` rend **une ligne par candidat** sur sa sortie standard :
`nom`, `nom<TAB>badge`, ou `nom<TAB>badge<TAB>contexte` — même convention que le cache des
installés. Le **contexte** est la colonne de droite du popup, et c'est ce qui sépare un
helper d'une simple complétion : votre shell sait déjà compléter un nom de branche, il ne
vous dit pas laquelle est en retard ni quand elle a bougé. Gardez-le **court** — une colonne
plus large que le cadre chasse le nom. S'il échoue ou dépasse le délai, il ne rend rien et **rien ne s'affiche** : dans le
chemin du rendu, une erreur par frappe serait pire que le silence.

**`options`** liste les drapeaux proposés derrière `-` pour ce verbe. Le descripteur en est
la seule source : jigger n'a aucun moyen de deviner ce qu'une commande tierce accepte.

Le raisonnement complet — et ce que la décision coûte — est dans
l'[ADR-0009](../adr/0009-viviers-de-plugin-par-verbe.md).

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
