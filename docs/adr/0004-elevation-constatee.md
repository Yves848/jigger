# ADR-0004 — L'élévation se constate, elle ne s'intercepte pas

17 août 2026 — **acceptée**

## Contexte

L'entrée **A-15** demande de « repérer le moment où l'utilisateur doit saisir un mot de
passe `sudo` ou s'élever en administrateur, et le lui demander dans une fenêtre Bubble
Tea ». Prise au mot, cette formulation impose d'**intercepter une invite** — donc de lire
le flux du sous-processus pendant qu'il tourne.

Or c'est précisément ce que jigger a choisi de ne pas faire. Pour tout verbe non normalisé
— `install`, `uninstall`, `upgrade`, `doctor`, `cleanup`… —,
`internal/facade/executer.go` donne au sous-processus les trois descripteurs du terminal
(`relais = true`). C'est ce qui fait fonctionner, **sans une ligne de code de TTY**, les
invites de winget, ses barres de progression, ses accords de licence et l'élévation UAC
elle-même. La spec de la façade en fait un choix explicite (§4).

Intercepter reviendrait à renoncer à cette propriété, et à allouer un pseudo-terminal à
l'exécution — chose que le dépôt sait faire dans ses harnais de test (`tests/zpty.zsh`,
`tests/conpty`), jamais en production. L'entrée A-15 le disait elle-même : « à rouvrir en
connaissance de cause, probablement par un ADR ». C'est ce document.

## Ce que la mesure a établi

Trois relevés, faits avant toute conception. Ce sont eux qui tranchent, pas l'intention de
départ.

**1. winget nomme la cause dans son code de sortie.** La table officielle des codes de
retour (`microsoft/winget-cli`, `doc/.../returnCodes.md`) en publie quatre qui parlent de
droits, et **deux disent l'inverse du troisième** :

| Code | Nom | Sens |
|---|---|---|
| `0x8A150019` | `COMMAND_REQUIRES_ADMIN` | la commande exige l'administrateur |
| `0x8A150056` | `INSTALLER_PROHIBITS_ELEVATION` | l'installeur **refuse** un contexte élevé |
| `0x8A15007D` | `ADMIN_CONTEXT_ACTION_PROHIBITED` | action interdite en contexte élevé sur un paquet installé pour l'utilisateur |
| `0x8A15C111` | `CONFIG_UNIT_IMPORT_MODULE_ADMIN` | un module de `winget configure` exige l'administrateur |

Un « code non nul → propose d'élever » serait donc **nuisible** dans deux cas sur quatre :
il pousserait à refaire, élevé, exactement ce qui vient d'échouer *pour cause d'élévation*.

**2. Go ne rend pas ces codes sous la forme publiée.** Mesuré sur cette machine avec le
même chemin que `facade.lancerReel` :

```
ExitCode() = 2316632089  (0x8A150019)
  == 0x8A150019 ?    true
  == -1978335207 ?   false
```

La table de Microsoft publie la forme **signée** (`-1978335207`) ; `exec.ExitError.ExitCode()`
rend sous Windows le **DWORD non signé** (`2316632089`). Recopier la colonne du tableau
dans le code aurait produit une comparaison qui n'est jamais vraie — une panne muette, la
pire espèce.

**3. `sudo` existe sous Windows 11, et il est désactivé.** `C:\WINDOWS\system32\sudo.exe`
est présent (build 26200), mais :

```
Sudo est désactivé sur cet ordinateur. Pour l'activer, accédez à Developer Settings
```

L'entrée A-15 affirmait que « sous Windows, ce n'est pas une invite console ». C'est devenu
à moitié faux : `sudo` peut élever en ligne. Mais il ne peut pas être **supposé** — ni
présent, ni activé.

## Décision

**jigger n'intercepte rien. Il laisse la commande tourner relayée, exactement comme
aujourd'hui, et lit son code de sortie après coup.**

Quand ce code désigne un défaut de droits, jigger le dit et **propose** de rejouer la
commande élevée. Il n'élève jamais de lui-même, et il se tait quand le code dit l'inverse.

Trois corollaires, qui font partie de la décision :

- **Le diagnostic appartient au gestionnaire.** C'est winget qui sait lire les codes de
  winget, comme c'est lui qui déclare ses verbes. Un gestionnaire qui n'en sait rien ne dit
  rien — même modèle de capacités que `cleanup` ou `doctor`.
- **Les contre-cas sont traités, pas ignorés.** Un code qui *interdit* l'élévation donne un
  message distinct, et aucune proposition.
- **La moitié Unix n'est pas traitée ici.** Aucun gestionnaire Unix ne publie de code de
  sortie équivalent, et l'entrée A-15 constate elle-même que le besoin y est marginal (brew
  refuse de tourner en root). Elle reste ouverte, et devra être instruite par ses propres
  moyens — `sudo -v` anticipé plutôt que détection — le jour où le besoin est démontré.

## Justification

- **La propriété de la spec §4 est intacte.** Aucune capture, aucun pseudo-terminal, aucun
  code de TTY : le relais reste le relais, et tout ce qui marchait continue de marcher.
- **Constater après coup ne coûte rien à l'utilisateur.** Sur `COMMAND_REQUIRES_ADMIN`,
  winget refuse **avant** d'agir : rien n'est à moitié fait, et rejouer est sans risque.
  C'est ce qui rend l'après-coup acceptable ici, et ce qui devra être revérifié pour tout
  autre code qu'on ajouterait à la table.
- **Le signal est un contrat, pas une formulation.** Un code de sortie documenté ne change
  pas de langue et ne se reformule pas d'une version à l'autre — contrairement au texte de
  l'erreur, qui suit la locale de l'utilisateur. C'est la même raison qui fait préférer,
  dans A-21, une table de vérifications connues à une extraction de la prose de `checkup`.
- **Proposer coûte une frappe ; élever tout seul coûte la confiance.** jigger exécute déjà
  des commandes mutantes à la demande ; en élever une sans un oui explicite serait d'un
  autre ordre.

## Conséquences

- Un contrat optionnel de plus dans `internal/pm`, sur le modèle de `pm.Bindings` :
  un gestionnaire *peut* savoir lire ses codes, il n'y est pas tenu.
- `facade.Executer` doit rendre plus que `(rows, code)`. Il gagne un `ExecuterAvec` qui
  rend un résultat complet, `Executer` restant l'appel court — c'est déjà la forme que le
  dépôt emploie pour `complete.Complete` / `CompleteAvec`.
- La proposition s'affiche sur le **terminal**, pas sur la sortie standard, et se dégrade
  en une ligne imprimée quand il n'y a pas de terminal : c'est le comportement déjà écrit
  pour la désambiguïsation (`main.trancher`), et il n'y a pas de raison d'en avoir deux.
- La table des codes est du **fait vérifiable** : elle se teste sans winget, sans Windows
  et sans élévation, puisqu'elle ne fait que traduire un entier.
- Le jour où un second gestionnaire publiera des codes de droits, il implémentera le même
  contrat sans que la façade ni l'interface changent.
