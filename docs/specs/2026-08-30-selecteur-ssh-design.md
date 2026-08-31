# Sélecteur de serveurs SSH — conception

30 août 2026 — état : validé, prêt pour le plan d'implémentation.

## Objet

C'est l'entrée **A-25**. La règle et ses raisons sont arrêtées par
[l'ADR-0005](../adr/0005-completion-sans-facade.md) : le contrat de complétion n'est pas
réservé aux gestionnaires de paquets, et un fournisseur peut implémenter `pm.Manager` sans
jamais implémenter `pm.Bindings`.

Cette spec dit comment, et où.

Ce qu'on veut, en une phrase : **dès qu'on tape `ssh`, le popup propose les serveurs
connus, et ⇥ insère celui qu'on vise** — exactement ce que `brew` obtient déjà pour ses
formules, avec la même popup et les mêmes touches.

## §1 — Ce que ce fournisseur n'est pas

Il n'exécute rien. `ssh` s'exécutera de lui-même quand l'utilisateur pressera ⏎, comme
aujourd'hui : jigger n'aura fait que compléter la ligne.

Il n'a donc **aucune table de verbes**, n'apparaît pas dans la façade, et `jg install` ne
le concerne pas. `Options()` rend le vide, `InstalledOnly()` rend `false`, `Warm()` ne fait
rien.

Cette pauvreté est le sujet, pas un manque : c'est elle qui permet à un fournisseur de
complétion de coûter deux méthodes.

## §2 — La position de l'opérande

`completeWith` traite le mot qui suit la commande comme une **sous-commande**, et ne passe
au catalogue qu'ensuite. C'est la grammaire de `brew install firefox`.

`ssh archlight` n'a pas de verbe. La règle posée par l'ADR-0005 s'applique :

> Un fournisseur qui ne déclare aucune sous-commande propose son catalogue dès le premier
> mot.

En pratique, dans `completeWith`, la branche `firstWord` teste
`len(m.Subcommands()) == 0` et bascule sur le même code que la branche « paquet ». Les
badges suivent : chaque hôte s'affiche avec son `HostName` en regard.

**Les trois gestionnaires existants déclarent tous des sous-commandes** : aucun ne change
de comportement, et leurs tests le prouvent sans qu'on ait à en écrire de nouveaux pour
eux.

## §3 — Trois commandes, trois fournisseurs

`Manager.Cmd()` rend un mot. Élargir l'interface pour que `ssh` en rende trois obligerait
`brew`, `winget` et `scoop` à répondre à une question qu'ils ne se posent pas.

On enregistre donc **trois fournisseurs** — `ssh`, `scp`, `sftp` — construits par la même
fonction et partageant un catalogue calculé une fois. `managers.All()` en rend six.

### Ce que `Insert()` rend, et pourquoi ce n'est pas la même chose

| Commande | Ligne visée | Insertion |
|---|---|---|
| `ssh` | `ssh archlight` | `archlight` |
| `scp` | `scp fichier archlight:/tmp/` | `archlight:` |
| `sftp` | `sftp archlight` | `archlight` |

`scp` attend `hôte:chemin`, deux-points collés. Insérer le nom nu produirait une commande
qui copie vers un **fichier local nommé `archlight`** — une erreur silencieuse, qui écrase
peut-être quelque chose. Le deux-points fait partie du candidat.

### La position de l'opérande pour `scp`

`scp` prend deux opérandes et la cible peut être l'une ou l'autre : `scp local hôte:dist`
comme `scp hôte:dist local`. On ne cherche pas à deviner laquelle : **les hôtes sont
proposés à toute position d'opérande**, et l'utilisateur choisit. Proposer un hôte là où il
n'en veut pas coûte une touche ⎋ ; ne pas le proposer là où il en veut coûte la
fonctionnalité.

## §4 — Le catalogue

`Load()` lit `~/.ssh/config` et rend un `pm.Catalog` :

| Champ | Contenu |
|---|---|
| `Names` | les motifs `Host` sans joker, triés |
| `Versions` | le `HostName` du bloc, quand il diffère du nom |
| `Badges` | vide — le glyphe `•` en découle, et il convient |
| `Installed`, `Qualified` | vides — ces notions n'ont pas de sens ici |

### Pourquoi `Versions` porte une adresse

`Badge` n'est pas du texte libre : `glyphe()` le traduit en `◆` (formula, winget, bucket
main), `▣` (cask, autre) ou `•` par défaut. Un hôte n'appartient à aucune des deux classes
de paquets — le `•` est donc exactement juste, sans qu'on ait rien à déclarer.

Le seul champ rendu en **texte libre à droite de la ligne** est `Versions` :

```go
if it.Version != "" {
    right = verStylePkg.Render(it.Version)
}
```

C'est donc lui qui porte le `HostName`, et le popup affiche `archlight   192.168.50.207`.

**Le nom du champ ment, et c'est assumé.** L'alternative — renommer `Version` en `Detail`
dans `pm.Item` — toucherait les trois gestionnaires, l'UI et les tests de rendu pour un
gain de vocabulaire, bien au-delà du périmètre d'un sélecteur SSH. Le code qui remplit ce
champ porte un commentaire disant ce qu'il y met et pourquoi ; c'est le prix de ne pas
refactoriser un projet qui marche pour y greffer une fonctionnalité.

### Ce qu'on lit, et ce qu'on écarte

- Les directives **`Include`** sont suivies, en résolvant `~` et les chemins relatifs à
  `~/.ssh/`. C'est indispensable : une configuration moderne se répartit en fragments, et
  celle de l'auteur en génère un depuis un inventaire.
- Un bloc `Host` peut porter **plusieurs motifs** (`Host archlight aquarium 192.168.50.207`).
  Tous sont retenus comme noms — ce sont autant de façons valides de désigner la machine.
- Les motifs contenant `*`, `?` ou `!` sont **écartés** : `Host *` n'est pas un serveur.
- Un fichier illisible n'est pas une erreur : il rend un catalogue vide. Le popup dira
  « aucun candidat », ce qui est vrai.

### Pas de réchauffement

`Warm()` ne fait rien. Lire quelques fragments de configuration coûte une milliseconde ;
il n'y a ni sortie machine à analyser, ni service distant à interroger, ni cache de 24 h à
tenir. C'est le seul fournisseur de jigger dans ce cas, et c'est ce qui le rend bon marché.

`Available()` rend vrai si `~/.ssh/config` existe. Sur une machine sans configuration SSH,
le fournisseur se tait plutôt que de proposer une liste vide.

> **Amendé le 31 août — [ADR-0006](../adr/0006-silence-sur-catalogue-vide.md).** Le critère
> du silence n'est pas `Available()` mais le **catalogue vide**. Un `~/.ssh/config` qui ne
> contient qu'un bloc `Host *` — ce que la documentation d'Apple fait écrire sur macOS —
> existe bel et bien, mais `*` est un motif : aucun candidat n'en sort, et la règle écrite
> ci-dessus laissait alors un cadre vide se redessiner à chaque frappe. C'est la seule
> affirmation de cette conception que l'implémentation a contredite.

## §5 — Le greffon

Une ligne : `_jigger_commands=( brew jigger jg ssh scp sftp )`.

> **Corrigé à la revue finale.** Cette liste était posée en dur, alors que tous les
> autres réglages du greffon suivent l'idiome `: "${JIGGER_X:=…}"`. Sans surcharge
> possible, un utilisateur ne pouvait pas éteindre l'interception. `JIGGER_COMMANDS`
> suit désormais le même idiome sous zsh que sous PowerShell — une dizaine de lignes,
> non pas une seule.

Rien d'autre ne change côté zsh. La popup, les flèches, le rattrapage des frappes, la
fermeture sur ⏎ — tout est déjà écrit et ne connaît pas la nature de ce qu'il affiche.

**Côté PowerShell**, `jigger.psm1` déclare sa propre liste. `ssh` existe sous Windows
depuis OpenSSH intégré, et `~/.ssh/config` s'y lit pareil : le fournisseur fonctionne des
deux côtés sans code spécifique. La liste PowerShell est mise à jour de la même façon.

## §6 — Les tests

| Ce qui est couvert | Nature |
|---|---|
| Lecture d'un `~/.ssh/config` simple : noms, `HostName` en badge | table, sur fichier de test |
| Un bloc à plusieurs motifs rend plusieurs noms | table |
| `Host *`, `Host web-?`, `Host !prod` sont écartés | table |
| Une directive `Include` est suivie, `~` et relatif résolus | table, sur arborescence temporaire |
| Un `Include` circulaire ne boucle pas | garde |
| Fichier absent ou illisible → catalogue vide, pas d'erreur | table |
| `Insert()` rend `hôte:` pour `scp`, `hôte` pour `ssh` et `sftp` | table |
| `completeWith` propose le catalogue au premier mot quand `Subcommands()` est vide | unitaire, dans `internal/complete` |
| **Les trois gestionnaires existants gardent leur comportement** | les tests actuels de `complete`, inchangés |

Aucun test ne lit le `~/.ssh/config` réel de la machine : tous travaillent sur des fichiers
construits par le test. C'est ce qui les rend vrais ailleurs que chez l'auteur.

## §7 — Ce que cette spec ne fait pas

- **Pas d'options.** `-p`, `-i`, `-L` et les autres ne sont pas proposés. `Options()` rend
  le vide. On pourra y revenir ; rien dans ce design ne l'empêche.
- **Pas de `rsync`.** Sa grammaire lui est propre et ses options sont légion. Il mérite sa
  propre décision, pas d'être glissé ici.
- **Pas de `known_hosts`.** Le fichier est souvent haché (`HashKnownHosts yes`), et les
  noms y sont alors illisibles. Une machine qu'on joint sans l'avoir déclarée n'a pas de
  nom à proposer.
- **Aucune vérification que l'hôte répond.** Le popup se dessine pendant la frappe : y
  glisser une sonde réseau serait la meilleure façon de le rendre poussif.
