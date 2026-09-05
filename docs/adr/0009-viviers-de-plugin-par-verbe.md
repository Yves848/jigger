# ADR-0009 — Un plugin fournit ses candidats par verbe, en cache ou à la frappe

5 septembre 2026 — **acceptée**

## Contexte

La 0.16.0 a ouvert jigger à des gestionnaires tiers. Le premier plugin livré, `git`, a été
**retiré ce jour** : il ne rendait pas `git` plus commode, il lui substituait un autre
vocabulaire. `git ⇥` proposait `install`, `list`, `outdated`, `search`, `uninstall`,
`upgrade` — dont aucun n'est une commande git — et la ligne complétée se lisait comme du git
tout en exécutant `jigger-git run install …`.

Ce n'était pas une implémentation bâclée : **c'est la seule forme que le descripteur sait
exprimer.** Ce qu'un plugin peut déclarer aujourd'hui, comparé à un gestionnaire natif :

| Capacité | Natif (`brew`) | Plugin |
|---|---|---|
| Sous-commandes | oui | oui |
| Options par sous-commande | oui — `internal/brew/manager.go:58` | **non** — `Options(_)` rend `nil`, `internal/plugin/discovery.go:315` |
| Candidats par verbe | oui, code Go libre | **non** — un `Pool` parmi `catalogue`, `installees`, `aucun` |
| Source des candidats | libre | deux commandes de réchauffement, deux caches pour toute la machine |

Un plugin ne peut donc décrire qu'**un gestionnaire de paquets de plus**. Or la raison d'être
d'un plugin est autre : rendre commode une commande **existante**, en montrant dans le popup
les sous-commandes, options et arguments qu'elle accepte vraiment. Un helper `git` a besoin
des *branches* derrière `checkout`, des *fichiers modifiés* derrière `add`, des *distants*
derrière `push`, et de `-m` derrière `commit`. Rien de tout cela n'entre dans deux viviers
globaux sans options.

L'[ADR-0008](0008-execution-des-plugins.md) a laissé la question ouverte, en la renvoyant à
la **règle d'or** du [plan d'injection](../plans/2026-09-04-plugins-injection.md) §2 :

> Rien de lent dans le chemin du rendu (`jigger render` tourne à chaque frappe). Un plugin
> ne peut pas être un sous-processus appelé par `render`.

Trois mesures rouvrent le dossier, parce qu'elles contredisent la **prémisse** de la règle,
non son intention :

- **`render` est déjà un sous-processus par frappe.** Le greffon zsh lance
  `$JIGGER_BIN render --line "$LBUFFER" …` à chaque touche
  (`shell/jigger.plugin.zsh:253`). Un fork par frappe n'est pas une nouveauté, c'est le
  modèle en place.
- **Une frappe coûte 11 ms** aujourd'hui (`jigger render` sur une ligne `brew`, cache de
  8 593 formules).
- **Une réponse git contextuelle coûte 7 ms** (`git branch --format=…`).

Le cache existe parce qu'un catalogue Homebrew fait 8 593 entrées et se paie par un appel
lent à `brew` — pas parce que le fork serait interdit. Les branches d'un dépôt font trois
lignes et changent toutes les minutes : les réchauffer serait à la fois inutile et faux.

## Options pesées

| Option | Ce qu'elle coûte |
|---|---|
| **A. Viviers nommés, tous réchauffés** — on garde le modèle et on autorise plus de deux caches | La plus tentante : elle ne change presque rien, et respecte la règle d'or à la lettre. Elle est pourtant **fausse par construction** pour ce besoin — une branche créée il y a une minute serait absente jusqu'au prochain `warm`, et `git add ⇥` listerait les fichiers d'un autre répertoire que le courant. Elle répond à la lettre de la contrainte en manquant le besoin. |
| **B. Tout à la demande** — plus de cache, le plugin répond toujours à la frappe | Simple à décrire, ruineuse à l'usage : un catalogue de 8 593 entrées serait reconstruit à chaque touche. C'est précisément ce que le cache existe pour éviter. |
| **C. Le vivier déclare son régime** *(retenue)* | Deux mécanismes à documenter et à maintenir au lieu d'un. En échange, chaque vivier est servi comme il doit l'être : en cache quand il est gros et lent, à la frappe quand il est petit et contextuel. |
| **D. Les helpers restent des gestionnaires natifs, écrits en Go dans jigger** | Ferme l'extensibilité tierce pour l'usage même auquel les plugins servent, et impose une release de jigger par helper. Contredit ce que la 0.16.0 a promis. |

## Décision

**Le descripteur déclare, pour chaque verbe, un vivier nommé et des options ; chaque vivier
déclare son régime — `cache`, réchauffé par `warm` comme aujourd'hui, ou `direct`, demandé
au plugin au moment de la frappe, dans le répertoire courant et sous un délai ferme au-delà
duquel jigger n'affiche rien plutôt que d'attendre.**

## Conséquences

**Ce que ça coûte.** Une ligne qui concerne un plugin à vivier `direct` paie **un second fork
par frappe** — 7 ms mesurés pour une réponse git, sur une frappe qui en coûte 11. Le budget
double sur ces lignes-là. C'est acceptable parce que c'est borné et mesuré ; ça cesserait de
l'être si un plugin répondait en centaines de millisecondes, et c'est pourquoi le délai
n'est pas optionnel.

**La règle d'or est révisée, pas ignorée.** « Rien de lent dans le chemin du rendu » reste
vrai et devient la seule règle. « Un plugin ne peut pas être un sous-processus appelé par
`render` » tombe, remplacée par une borne chiffrée. Un futur lecteur du plan §2 doit lire
cet ADR avec lui.

**Un plugin peut désormais faire attendre le prompt.** C'est le vrai danger, et il n'existait
pas avant : un plugin lent, bloqué sur le réseau ou sur un verrou, tiendrait la frappe. Le
délai est donc obligatoire, et son expiration doit **ne rien afficher** — la doctrine de
l'[ADR-0006](0006-silence-sur-catalogue-vide.md), qui préfère le silence à un cadre inutile.

**Installer un plugin devient un acte de confiance plus lourd.** Son binaire ne s'exécute
plus seulement quand l'utilisateur lance un verbe : il tourne **à chaque frappe** d'une ligne
qui commence par son mot, dans le répertoire courant. La contrainte 4 du plan — « un plugin
tiers ne doit pas pouvoir exécuter de commande arbitraire » — gagne en poids : un plugin est
maintenant plus proche d'un hook de shell que d'une commande qu'on a demandée.
`docs/plugins.md` doit le dire à l'endroit où l'on explique comment en installer un.

**Ce qui reste ouvert, et ce qui remplacerait cet ADR.** Le protocole d'échange d'un vivier
`direct` — un argv et du JSON sur la sortie standard, vraisemblablement — n'est pas tranché
ici ; il découle de cette décision et se règle à l'implémentation. En revanche, deux
constats obligeraient à remplacer cet ADR plutôt qu'à l'amender : un plugin qui aurait besoin
d'un dialogue en plusieurs tours avec jigger, et la mesure, sur de vraies machines, que le
délai est régulièrement atteint — ce qui voudrait dire que le fork par frappe ne tient pas
ses promesses hors du banc d'essai.
