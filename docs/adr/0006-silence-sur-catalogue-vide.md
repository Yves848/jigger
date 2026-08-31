# ADR-0006 — Un fournisseur se tait sur un catalogue vide, pas sur un fichier absent

31 août 2026 — **acceptée**

## Contexte

L'[ADR-0005](0005-completion-sans-facade.md) a ouvert la complétion à des fournisseurs qui
ne sont pas des gestionnaires de paquets. Le premier d'entre eux, `ssh`, n'a pas de verbes :
il propose son catalogue dès le premier mot, ou rien.

« Ou rien » demandait une règle. La conception du 30 août (§4) l'a écrite ainsi :

> `Available()` rend vrai si `~/.ssh/config` existe. Sur une machine sans configuration SSH,
> le fournisseur se tait plutôt que de proposer une liste vide.

L'intention est juste — sans cette règle, un cadre « aucune correspondance » se redessine à
**chaque frappe** de toute ligne `ssh`, `scp` ou `sftp`, et rien ne permet de l'éteindre.
Mais le critère retenu, lui, mesure la mauvaise chose.

La revue finale a nommé le contre-exemple : un `~/.ssh/config` qui ne contient qu'un bloc
`Host *`. C'est exactement ce que la documentation d'Apple fait écrire sur macOS —

```
Host *
  ServerAliveInterval 60
  AddKeysToAgent yes
```

— et ce fichier **existe**, donc `Available()` rend vrai. Mais `*` est un motif, que le
parseur écarte à juste titre : le catalogue est vide. Le cadre vide revenait donc à chaque
frappe, sur la configuration par défaut de la plateforme même que le sélecteur vise.

Le défaut n'est pas dans le code, qui suivait la spec fidèlement. Il est dans la spec.

## Décision

> Un fournisseur **sans verbes** se tait quand il n'a **aucun candidat à proposer**, sans
> considération pour l'existence d'un fichier de configuration.

```go
if sansVerbes && len(res.Items) == 0 {
    res.Silencieux = true
}
```

`Available()` n'intervient plus dans cette décision. Elle garde tout son rôle ailleurs —
`managers.Available()`, la façade, l'écran de configuration —, où la question posée est
bien « ce gestionnaire est-il installé sur cette machine ».

## Conséquences

- Le cas `Host *` seul se tait, comme l'absence de fichier. C'est le gain visé.
- `ssh zzzz` — une frappe qui ne correspond à aucun hôte — se tait aussi. C'est un
  changement de comportement assumé : un fournisseur sans verbes n'a rien d'autre à offrir
  qu'un catalogue, donc un catalogue filtré à vide n'a rien à montrer. Un gestionnaire à
  verbes, lui, a toujours ses sous-commandes à proposer.
- **`sansVerbes` porte désormais seul la protection de brew, winget et scoop.** Le retirer
  du prédicat ferait disparaître leur cadre « aucune correspondance » sur une machine où le
  gestionnaire n'est pas installé — le popup s'évanouirait sous les doigts au lieu de dire
  qu'il ne trouve rien. `TestGestionnaireAVerbesIndisponibleNeSeTaitPas` garde ce cas, et la
  mutation le fait tomber.
- La §4 de la conception du 30 août est amendée en conséquence : c'est la seule affirmation
  de ce document que l'implémentation a contredite.

## Ce qui a été écarté

**Garder `Available()` dans le prédicat et traiter `Host *` comme un cas particulier.**
Cela aurait demandé au fournisseur de distinguer « fichier absent » de « fichier présent
mais sans bloc nommé », soit deux chemins pour un seul comportement observable. Le critère
« rien à proposer » les couvre tous les deux, et couvre aussi ceux qu'on n'a pas prévus —
un fichier illisible, un `Include` cassé, une configuration entièrement faite de motifs.
