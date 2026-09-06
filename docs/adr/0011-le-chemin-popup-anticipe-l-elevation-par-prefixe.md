# ADR-0011 — Le chemin popup anticipe l'élévation, et le fait par un préfixe

6 septembre 2026 — **acceptée**

## Contexte

L'[ADR-0004](0004-elevation-constatee.md) a réglé l'élévation **du chemin façade** :
laisser la commande tourner relayée, lire son code de sortie après coup, proposer de la
rejouer. Le mécanisme est écrit, éprouvé, et il ne s'applique pas ici.

Il ne s'applique pas parce que la complétion ne passe pas par la façade
([ADR-0005](0005-completion-sans-facade.md)) : jigger n'exécute rien sur ce chemin, il rend
une ligne, et c'est zsh qui la lance (`_jigger_accept`, `shell/jigger.plugin.zsh`). Il n'y
a donc **aucun code de sortie à constater**, et rien à rejouer — la commande n'a jamais
appartenu à jigger. Ce n'est pas une lacune du mécanisme d'A-15, c'est un endroit où il n'a
jamais été question qu'il agisse.

Le symptôme est celui de l'[issue #167](https://gitlab.yg-devworks.com/yves/jigger/-/issues/167) :
sur Arch, `pacman -S fd` complété par le popup puis lancé par ⏎ rend
`error: you cannot perform this operation unless you are root`. Le popup a fait son travail
— la ligne est juste, le paquet est le bon — et elle part quand même vouée à l'échec.

**Ce que la mesure a établi.** Les vingt opérations pacman de la table ont été passées sur
un conteneur Arch, en utilisateur non root, en relevant lesquelles répondent
`you cannot perform this operation unless you are root` :

- exigent root : `-S` `-Syu` `-Sy` `-Sc` `-Sw` `-Su` `-R` `-U` `-D` `-Fy`
- s'en passent : `-Ss` `-Si` `-Sl` `-Sg` `-Sp` `-Sup` `-Qs` `-Qu` `-Fs` `-V`

Deux enseignements en sortent, et aucun n'était déductible. D'abord `-Sc` et `-Sw` exigent
root alors que `_jigger_pacman_mutant` les range en lecture : les deux tables répondent à
des questions différentes et ne peuvent pas être fusionnées. Ensuite `-Sup` ne l'exige
**pas**, alors qu'il porte le `u` d'un `-Su` qui, lui, l'exige : il n'imprime que des URI.
Un prédicat écrit au jugé l'aurait élevé.

**Ce que le dépôt avait déjà prévu — et qui ne tient pas.**
`internal/elevate/elevate_other.go` annonce en toutes lettres que « la moitié Unix d'A-15
devra s'instruire autrement, en *anticipant* (`sudo -v` avant de lancer) plutôt qu'en
constatant ». L'intuition d'anticiper était juste. La forme proposée, non — c'est ce que
cet ADR tranche.

## Options pesées

| Option | Ce qu'elle apporte | Ce qu'elle coûte | Verdict |
|---|---|---|---|
| **Poser `sudo ` en tête de la ligne, à l'exécution** | la commande part élevée, du premier coup ; l'invite reste celle de sudo, sur le terminal ; jigger ne détient aucun secret | jigger réécrit une ligne que l'utilisateur a composée — la seule fois où il le fait sans qu'on lui ait demandé de compléter | **retenue** |
| **`sudo -v` avant de lancer**, comme annoncé dans `elevate_other.go` | ne touche pas à la ligne de l'utilisateur, ce qui est très tentant | **elle ne marche pas.** `sudo -v` rafraîchit le cache d'identification ; la ligne qui part ensuite reste `pacman -S fd`, sans sudo. Élever l'*horodatage* n'élève pas la *commande*. Pour qu'elle serve, il faudrait préfixer derrière — c'est-à-dire l'option retenue, avec une invite de mot de passe en plus | écartée : incohérente prise seule |
| **Demander le mot de passe dans le cadre** (A-22, [spec §7](../specs/2026-08-17-elevation-design.md)) | le dessein d'origine d'A-15 ; ne réécrit pas la ligne | jigger devient un composant sensible : un mot de passe en mémoire, l'écho du terminal à museler, rien à laisser fuir dans un journal. Et le compte n'y est pas : il faudrait *encore* préfixer la ligne ensuite, ou l'exécuter soi-même — ce qui contredirait l'ADR-0005 | écartée : coût de sécurité réel pour un gain qui reste à démontrer |
| **Ne rien faire** | zéro ligne de code | le popup complète soigneusement une ligne dont il sait qu'elle échouera, et laisse l'utilisateur la rappeler pour retaper `sudo ` devant | écartée : c'est le défaut signalé |

La deuxième ligne est celle qui compte. Elle était écrite dans le dépôt, elle avait l'air
d'une décision déjà prise, et elle ne résiste pas à l'examen : c'est précisément le genre de
chose qu'un registre existe pour retenir.

## Décision

**Sur le chemin popup, jigger pose `sudo ` en tête de la ligne au moment où elle part, et
seulement quand l'opération exige root. Il ne détient jamais de mot de passe : l'invite
reste celle de sudo, sur le terminal.**

Trois corollaires en découlent, et ils font partie de la décision :

1. **À l'exécution, jamais à l'insertion.** ⏎ porte le préfixe, ⇥ ne le porte pas. Ce n'est
   pas un choix d'ergonomie : ⏎ ne complète la ligne que s'il reste un candidat à poser
   (`_jigger_completable`), or un `pacman -Syu` frappé en entier n'en a aucun et doit
   pourtant être élevé. Poser le préfixe à l'insertion l'aurait manqué. Accessoirement, ⇥
   ne réécrit ainsi jamais une ligne qu'on est en train de composer.
2. **Le gestionnaire qui sait s'élever lui-même n'est pas préfixé.** `yay` et `paru`
   appellent sudo au bon moment et refusent de tourner en root : les élever casserait ce
   qui marche.
3. **Le préfixe est désarmable** (`JIGGER_SUDO=0`), parce qu'une règle `sudoers` sans mot de
   passe, un alias maison ou une session déjà root rendent le geste inutile ou nuisible.

## Conséquences

- **Ce que ça coûte.** jigger modifie une ligne écrite par l'utilisateur. On l'assume parce
  que la modification est *visible* (zsh redessine la ligne avant de l'exécuter, l'écho et
  l'historique portent le `sudo`), *réversible* (rien n'est exécuté par jigger) et
  *désarmable*. Retirer l'une des trois remettrait la décision en cause.
- **Ce que ça coûte, deuxièmement.** Une table de droits à tenir, distincte de celle des
  obsolètes. Deux tables sur le même sujet apparent divergeront un jour ; c'est le prix de
  ne pas faire payer à l'une la question de l'autre, et il se paie en tests.
- **Ce que ça ferme.** Les options **longues** (`--sync`, `--remove`) ne sont pas lues : les
  distinguer de leurs lectures (`--sync --search`) demanderait une vraie analyse d'options.
  Ferme aussi l'élévation d'une ligne où `pacman` n'est pas la première commande — sur
  `echo hi && pacman -S fd`, préfixer la *ligne* élèverait `echo`. Dans les deux cas la
  ligne part sans préfixe, c'est-à-dire comme avant : la décision dégrade proprement.
- **Ce qui la remettrait en cause.** Un gestionnaire du chemin popup qui saurait, lui, se
  rejouer élevé — c'est le cas sous Windows, où `internal/elevate` fonctionne et où le
  greffon PowerShell lance la ligne par `Invoke-Expression`. La question s'y pose autrement
  et n'est pas tranchée ici.

## Suites

- [#168](https://gitlab.yg-devworks.com/yves/jigger/-/issues/168) — le chemin façade sous
  Unix, où il n'y a pas de ligne à préfixer : la décision d'ici ne s'y transpose pas.
- [#169](https://gitlab.yg-devworks.com/yves/jigger/-/issues/169) — le chemin popup n'élève
  sur aucune autre plateforme. Le présent ADR ne traite que zsh et pacman.
