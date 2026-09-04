# ADR-0008 — Un plugin emprunte le chemin d'exécution natif

4 septembre 2026 — **acceptée**

## Contexte

Le [plan d'injection de plugins](../plans/2026-09-04-plugins-injection.md) ouvre jigger à
des gestionnaires tiers livrés en binaire séparé. Il tranche la découverte (un dossier, un
`config.json`) et le réchauffement (un sous-processus qui rend du JSON), mais il laisse une
question ouverte : **comment un verbe s'exécute-t-il chez un plugin ?**

Sa réponse, au §3.3, est une enveloppe JSON :

```bash
jigger-mespa run install foo bar --json   →  {"stdout":"...\n","code":0}
```

Le plugin capture la sortie de son gestionnaire, l'emballe, et jigger la déballe. C'est
cohérent avec le reste du protocole — tout le reste est du JSON — et c'est ce que la
première implémentation a écrit (`internal/facade/executer.go`, branche
`if plugin.IsPlugin(...)`).

Trois choses la contredisent.

**La première est que jigger ne capture pas les verbes d'écriture, et que c'est délibéré.**
`ExecuterAvec` sépare deux régimes : les verbes normalisés — `list`, `outdated`, `search`,
`source` — voient leur sortie capturée puis refondue en tableau ; tout le reste tourne
relayé, `Stdin`/`Stdout`/`Stderr` hérités du terminal. Le commentaire qui porte cette table
dit pourquoi :

```go
// verbesNormalises : ceux dont la sortie est tabulaire, donc capturée et refondue. Tout
// le reste est relayé — et c'est ce qui fait que les invites, les barres de progression et
// l'élévation UAC fonctionnent sans une ligne de code de TTY.
```

Une enveloppe JSON annule exactement cela. `git clone` sur un dépôt privé demande une phrase
de passe ; `brew install` pose des questions ; winget déclenche UAC. Sous enveloppe, la
question part dans un tampon que personne ne lit et le processus attend une réponse qui ne
viendra jamais.

**La deuxième est que la première implémentation lançait le mauvais programme.** Elle
appelait `plugin.Run(cible.Mgr.Cmd(), …)` — or `Cmd()` est le **mot de la ligne**, pas le
binaire. Pour un plugin nommé `git`, jigger aurait lancé le vrai `git` avec `run install …`
en arguments. Le descripteur distingue bien les deux (`name` et `cmd`), l'exécution les
confondait.

**La troisième est que le partage lecture/écriture y était déduit du `pool`.** Le premier
jet posait un parseur dès que le pool n'était pas `aucun` :

```go
if pool != pm.PoolAucun {
    b.Parse = parsePluginOutput
}
```

Mais le pool dit *où trouver les candidats*, pas *si la sortie est un tableau*. `install`
puise dans le catalogue et reste une écriture : il héritait donc d'un parseur qui aurait
essayé de lire la sortie de `git clone` comme du JSON.

## Options pesées

| Option | Ce qu'elle aurait coûté |
|---|---|
| **L'enveloppe JSON du plan**, `{"stdout":…,"code":…}`. C'était l'option tentante, et pas seulement parce qu'elle était déjà écrite : elle est uniforme — un seul format sur tout le protocole — et elle donne à jigger la main sur la sortie du plugin, donc la possibilité de la reformater. | Elle interdit toute interaction : invite d'authentification, barre de progression, élévation. Pour un plugin `git`, qui clone et tire, c'est la fonction principale qui tombe. Il aurait fallu réintroduire un canal de relais à côté de l'enveloppe — c'est-à-dire réécrire, en plus mal, le chemin natif qui existe déjà. |
| **Une enveloppe pour la lecture, un pseudo-terminal pour l'écriture.** Elle garde l'uniformité *et* l'interaction. | Elle met jigger dans le métier du PTY sur trois plateformes, dont ConPTY sous Windows, pour un bénéfice nul : le chemin natif obtient déjà le même résultat en héritant simplement des descripteurs. Un coût de portabilité majeur pour retrouver l'existant. |
| **Un `pm.Manager` par plugin, compilé.** Pas de sous-processus du tout. | C'est ce que l'[ADR-0001](0001-go-confirme.md) a écarté en actant Go : un binaire statique, donc pas d'extension sans recompilation. C'était précisément le problème que les plugins existent pour résoudre. |
| **Substituer le binaire dans le chemin natif.** | Retenue. |

## Décision

> **Un plugin s'exécute par le chemin natif, avec le seul binaire substitué.** La façade
> lance `plugin.Binaire(mgr)` au lieu de `mgr.Cmd()`, et rien d'autre ne change : même
> relais de terminal, même lecture du code de sortie, même rejeu sur défaut de droits.
>
> Trois règles en découlent, portées par le descripteur :
>
> 1. **`native` est l'argv complet** passé au binaire du plugin — jigger ne préfixe ni
>    `run`, ni le verbe, ni rien.
> 2. **Le partage lecture/écriture se décide sur le verbe**, par `pm.Normalise`, et non sur
>    le `pool`. La table qui le porte descend de `facade` dans `pm`, pour qu'un
>    gestionnaire puisse en dériver ses liaisons sans importer la façade.
> 3. **`name` et `cmd` sont deux choses distinctes** : le mot de la ligne et le programme à
>    lancer. Un gestionnaire natif gagne tout conflit de `name` avec un plugin, qui est
>    alors écarté avec un message sur la sortie d'erreur.

## Conséquences

**Ce que ça donne.** Un plugin hérite gratuitement de tout ce que la façade sait déjà faire :
le relais TTY, donc les invites et les barres de progression ; la lecture du code de sortie
et le `Rejeu` de l'[ADR-0004](0004-elevation-constatee.md), donc la proposition de rejouer
en élevé quand un code de sortie parle de privilèges ; l'arrêt en chaîne sur échec d'une
écriture. La branche spécifique aux plugins dans `ExecuterAvec` passe d'une quarantaine de
lignes à quatre.

**Ce que ça coûte.** Le protocole n'est plus uniforme : la lecture est du JSON, l'écriture
est du texte brut relayé. Un auteur de plugin doit donc écrire deux sortes de sous-commandes,
et savoir laquelle est laquelle — d'où la table des verbes normalisés dans
[la documentation des plugins](../plugins.md).

Surtout, **jigger ne voit plus rien de ce qu'écrit un verbe d'écriture.** Il ne peut pas
reformater la sortie d'un `install`, ni la traduire, ni en extraire quoi que ce soit : elle
va au terminal sans passer par lui. C'est le prix du relais, et c'est déjà celui que payent
les gestionnaires natifs.

Enfin, la substitution suppose que **le binaire du plugin accepte d'être lancé comme un
gestionnaire ordinaire** — un argv, un code de sortie, pas de dialogue sur stdin. Un plugin
qui aurait besoin d'un échange en plusieurs tours avec jigger n'entre pas dans ce modèle ;
il faudrait alors un ADR qui remplace celui-ci.

**Ce qui reste ouvert.** La complétion en temps réel par le plugin (phase P4 du plan) n'est
pas tranchée ici et ne doit pas l'être à la légère : elle mettrait un sous-processus dans le
chemin de `render`, contre la règle d'or du §2 du plan.
