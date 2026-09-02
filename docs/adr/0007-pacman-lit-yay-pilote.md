# ADR-0007 — pacman lit, yay pilote

2 septembre 2026 — **acceptée**

## Contexte

Le module pacman ([conception du 2 septembre](../specs/2026-09-02-pacman-design.md)) branche
**deux** mots de commande sur jigger : `pacman` et `yay`. Pour la complétion, c'est sans
conséquence — le popup s'ouvre sur celui que la ligne nomme, et chacun a son catalogue.

Pour la façade, ça l'est beaucoup moins : `pacman` et `yay` ne sont pas deux gestionnaires
qui coexistent, ce sont **deux portes sur la même base alpm**. `yay -Q` et `pacman -Q`
rendent les mêmes 1 055 lignes. C'est une situation que jigger n'avait jamais rencontrée :
winget et scoop, les deux seuls gestionnaires à cohabiter jusqu'ici, ont des catalogues
disjoints et des paquets installés disjoints.

Le moteur de routage est explicite sur ce qu'il fait de deux gestionnaires capables du même
verbe (`internal/facade/routage.go:74`) :

```go
// Pas de nom à résoudre : tous les gestionnaires capables agissent […]
if pool == pm.PoolAucun || len(noms) == 0 {
    cibles := make([]Cible, 0, len(capables))
    for _, m := range capables {
        cibles = append(cibles, Cible{Mgr: m, Args: args})
    }
```

Si `pacman` et `yay` déclaraient tous deux la table complète, sur une machine Omarchy — qui
a les deux — cela donnerait :

- `jg list` : **chaque paquet installé listé deux fois**, une fois avec la colonne PM
  `pacman`, une fois avec `yay`. Idem `jg outdated`, `jg search`.
- `jg install fd` : `fd` est connu des deux catalogues, donc `connaissent()` rend deux
  propriétaires, donc `Router` rend une `Ambiguite` — le popup « 2 gestionnaires » s'ouvre
  **à chaque installation**, pour proposer un choix qui n'en est pas un.

L'ambiguïté existe pour trancher entre deux logiciels différents portant le même nom
(le `git` de winget et celui de scoop). L'appliquer à deux chemins vers le même paquet
serait exactement le « choix silencieux qui rend une façade impossible à croire » que la
conception de la façade voulait éviter, retourné contre elle.

S'ajoute une contrainte de droits, indépendante mais convergente : **pacman exige root**
pour `-S`, `-R` et `-U`, et jigger n'élève rien ([ADR-0004](0004-elevation-constatee.md)).
**yay refuse au contraire de tourner en root** et appelle `sudo` lui-même, au bon moment,
pour la seule étape qui en a besoin.

## Options pesées

| Option | Ce qu'elle aurait coûté |
|---|---|
| **Les deux déclarent tout, l'échec est relayé.** C'était l'option tentante : la plus simple à écrire, et fidèle à l'esprit « la commande tourne relayée, jigger lit le code après coup ». | Elle ne résout ni le doublon de `jg list` ni l'ambiguïté de `jg install`, qui ne sont pas des échecs mais du bruit permanent. Et sur une machine sans yay, `jg install fd` lancerait `pacman -S fd`, qui échoue sur les droits — un verbe déclaré mais qui ne marche jamais, ce que le modèle de capacités existe précisément pour éviter. |
| **Ouvrir `pm.Binding` à un champ `Bin`**, pour qu'une liaison puisse lancer `sudo pacman` plutôt que `pacman`. | Techniquement le vrai correctif du problème de droits : `lancerReel` lance `cible.Mgr.Cmd()`, le binaire n'est aujourd'hui pas négociable. Mais cela met jigger dans le métier de l'élévation, contre l'[ADR-0004](0004-elevation-constatee.md), et pour un gain nul là où yay est installé — c'est-à-dire sur la quasi-totalité des machines Arch de bureau. Coût élevé, bénéfice marginal, décision structurante prise pour un cas particulier. |
| **Ne pas brancher pacman sur la façade du tout**, complétion seule, sur le modèle `ssh` de l'[ADR-0005](0005-completion-sans-facade.md). | Défendable, et honnête. Mais elle jette quatre verbes de lecture — `list`, `outdated`, `search`, `info` — que pacman rend parfaitement, sans droits, en moins de 200 ms. Sur une machine Arch sans yay (un serveur, un conteneur), `jg outdated` ne répondrait rien alors que `pacman -Qu` était là. |
| **pacman ne déclare que la lecture, et seulement en l'absence de yay.** | Retenue. |

## Décision

> **`yay`, quand il est installé, est le seul des deux à piloter.** `pacman` ne déclare de
> table de verbes que si `yay` est absent de la machine, et cette table ne contient que les
> verbes de **lecture** — `list`, `outdated`, `search`, `info` —, ceux qui n'exigent pas
> root.

```go
// Verbs — cf. ADR-0007. Deux tables déclarées en même temps feraient lister deux fois
// les mêmes paquets et rendraient « jg install fd » ambigu entre deux portes sur la
// même base alpm.
func (m Manager) Verbs() map[pm.Verb]pm.Binding {
    if m.cmd == "pacman" {
        if yayPresent() {
            return nil
        }
        return verbesLecture
    }
    return verbesYay
}
```

Une table vide n'est pas un cas particulier du moteur : `managers.Tables` parcourt
`b.Verbs()`, et une map nil s'y parcourt zéro fois. Le gestionnaire reste complété,
réchauffé et présent dans l'écran de configuration — il n'est simplement capable d'aucun
verbe, ce que jigger sait déjà dire de `ssh`.

## Conséquences

- **Sur une machine Omarchy** (les deux installés) : `jg install`, `jg outdated`, `jg list`
  passent par yay, qui couvre dépôts **et** AUR. Aucun doublon, aucune ambiguïté, et
  l'élévation est le problème de yay — qui le résout mieux que jigger ne le ferait.
- **Sur une machine Arch sans yay** : les quatre verbes de lecture répondent par pacman.
  `jg install` répond « aucun gestionnaire disponible ne sait faire ça » — le message exact
  que la façade rend déjà pour `jg doctor` sous Windows.
- **Le coût, assumé :** `jg install --pm pacman` n'existe pas, même en `sudo jg install`.
  Qui veut installer par pacman en présence de yay tape `pacman -S` — et jigger le complète,
  ce qui est le service principal du module. On échange une capacité rarement voulue contre
  la disparition d'une ambiguïté systématique.
- **`Verbs()` devient dépendant de l'environnement.** C'était jusqu'ici une constante par
  gestionnaire. Le test de bonne formation de la table (`TestTable…EstBienFormee`, présent
  chez brew, winget et scoop) doit donc couvrir les **deux** branches, pas celle de la
  machine qui fait tourner les tests. `yayPresent` est une variable de paquet pour cette
  raison, pas un appel direct à `exec.LookPath`.
- **Ce qui rouvrira la question :** un `pm.Binding.Bin` justifié par un autre besoin (apt,
  dnf — qui exigent root et n'ont pas d'équivalent de yay), ou l'arrivée d'un second
  assistant AUR installé en même temps que yay, qui ramènerait l'ambiguïté entre `yay` et
  `paru`. Ce jour-là, l'arbitrage « un seul pilote » devra se dire autrement qu'en
  nommant yay.
