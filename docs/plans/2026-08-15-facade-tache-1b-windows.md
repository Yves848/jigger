# Tâche 1b — la passe Windows de la façade

> **Pour la session Claude lancée sur Windows :** ce document est ton brief. Tu n'as pas
> besoin de lire tout le plan de la phase 1 ; les trois renvois ci-dessous suffisent.

**But :** vérifier contre les vraies CLI ce qui n'a pu l'être sur macOS, refaire les deux
parsers scoop écrits à l'aveugle, et finir le greffon PowerShell.

**Contexte :** [la spec](../specs/2026-08-15-facade-multi-gestionnaires-design.md) ·
[les décisions d'exécution](../specs/2026-08-15-facade-decisions-execution.md) ·
[le plan de la phase 1](2026-08-15-facade-phase-1.md)

## Comment démarrer

```powershell
git fetch origin
git checkout feat/facade-phase-1
go build ./...          # les fichiers Windows existent déjà : tty_windows.go, proc_windows.go
go test ./...
```

Les trois gestionnaires doivent être présents pour ce travail : `winget`, `scoop`, et Go ≥ 1.24.

## Contraintes globales

- Pas de dépendance nouvelle ; `go.mod` et `go.sum` ne changent pas.
- Commentaires, messages et tests en **français**, comme tout le dépôt.
- TDD pour tout changement de comportement.
- **Le comportement de la v0.7.0 ne bouge pas.** `jigger render` est appelé à chaque frappe
  par le greffon ; `jigger prompt` à chaque prompt. Les tests préexistants doivent passer
  **sans modification**.
- `make test-all` doit passer à la fin — sous Windows, cela lance `go test`, `tests/smoke.ps1`
  et `tests/pty.ps1`.
- Un commit par point, message en français, style nominal (cf. `git log --oneline`).

---

## Point 1 — vérifier les tables winget et scoop

Elles ont été écrites de mémoire et n'ont jamais rencontré une vraie CLI. Les deux fichiers
portent un avertissement en en-tête, à **retirer** une fois la vérification faite.

Les trois valeurs les plus incertaines, dans l'ordre du risque :

```powershell
winget pin --help          # « pin add » / « pin remove » existent-ils sous cette forme ?
winget source --help       # « source list » / « source add » / « source remove »
scoop help                 # « checkup », « hold », « unhold », « cleanup »
scoop update --help        # « scoop update * » met-il bien à jour les applications ?
```

Vérifie **chaque ligne** des deux colonnes de la section « Table de correspondance » de la
spec, pas seulement ces quatre-là. Pour chaque écart : corriger la table dans
`internal/winget/verbs.go` ou `internal/scoop/verbs.go`, corriger la spec, et corriger le test
d'argv correspondant dans le `verbs_test.go` du paquet.

Si un verbe n'existe pas chez un gestionnaire, **retire-le de sa table** et mets `—` dans la
spec : le modèle de capacités s'en accommode par construction, et `jg <verbe>` dira
proprement qui sait le faire.

## Point 2 — refaire les parsers `search` et `source` de scoop

**C'est le point le plus important, et le seul qui soit un vrai défaut.**

`internal/scoop/parse.go` a été écrit sans scoop, contre un format de sortie obsolète : des
sections `'main' bucket:` là où scoop émet aujourd'hui un tableau. Sur ta machine, ils rendent
vraisemblablement **zéro ligne, sans planter** — c'est-à-dire que `jg search` et `jg source`
diront « rien à signaler » en sortant en 0. Un parser qui ne reconnaît rien satisfait son
contrat : le garde-fou d'`Executer` ne l'attrape pas.

```powershell
scoop search git   > internal\scoop\testdata\search.txt
scoop bucket list  > internal\scoop\testdata\bucket-list.txt
scoop list         > internal\scoop\testdata\list.txt
```

Réécris `parseSearch` et `parseSource` contre ces sorties réelles, et **vérifie `parseList`**
au passage — c'est le plus plausible des trois, mais il n'a pas plus été vérifié que les
autres. Remplace les jeux d'essai écrits à la main, et retire l'en-tête « NON VÉRIFIÉ » du
fichier une fois que chaque parser tourne sur une vraie capture.

Un test qui n'assère que `len(rows) > 0` ne prouve rien : vérifie que les champs atterrissent
dans les bonnes colonnes (`Name`, `Version`, `Source`), comme le font déjà les tests winget.

## Point 3 — l'alias `jg` dans le greffon PowerShell

La moitié zsh est faite (`shell/jigger.plugin.zsh`, commit `412522a`) — lis-la comme modèle,
mais ne la copie pas mécaniquement : les deux greffons ne fonctionnent pas de la même façon.

Dans `shell/jigger.psm1` :

- `Set-Alias -Name jg -Value jigger -Scope Global` ;
- étendre la valeur par défaut de `JIGGER_COMMANDS`, aujourd'hui `'winget,scoop'`, pour y
  ajouter `jigger` et `jg` (ligne 47 environ).

Attention à la même subtilité que côté zsh : **l'alias et la liste des commandes qui arment le
popup sont deux mécanismes distincts**, et il faut les deux. Le widget lit ce que l'utilisateur
a tapé, pas l'expansion de l'alias — `jg` doit donc être reconnu **en tant que tel**.

Ajoute un cas à `tests/smoke.ps1` (ce qui se teste sans console) et, si le popup vivant est
concerné, à `tests/pty.ps1` (le vrai pseudo-terminal). Ne régresse pas l'armement sur `winget`
et `scoop`.

Mets à jour le README : la section « ce qui n'est pas encore là » annonce que le greffon
PowerShell n'a pas l'alias. Une fois fait, cette réserve disparaît.

## Point 4 — l'essai de bout en bout qui n'a jamais pu avoir lieu

Sur cette machine, deux gestionnaires cohabitent pour la première fois. C'est le seul endroit
où le routage se prouve vraiment :

```powershell
jg install git          # connu de winget ET de scoop → le sélecteur doit s'ouvrir
jg install git --pm scoop   # ne doit PAS s'ouvrir
jg install git | cat        # hors TTY : message + rappel de --pm, code 2
jg outdated             # doit fusionner winget et scoop, avec la colonne PM
jg list --json
jg doctor               # scoop sait (checkup), winget non
jg install fd Git.Git   # deux noms, deux gestionnaires, en une ligne
```

Le sélecteur de désambiguïsation n'a **jamais été exercé en vrai** : il a été relu, pas lancé.
C'est ici qu'il se prouve. Son pied doit afficher `↵ choisir` et `^G annuler`.

## Point 5 — les mineurs différés

Listés dans la [MR !8](https://gitlab.yg-devworks.com/yves/jigger/-/merge_requests/8). Traite
ceux qui te paraissent mériter le détour, laisse les autres.

---

## Ce qu'il ne faut pas faire

- **Ne pas affaiblir un test pour le faire passer.** Si un test échoue sous Windows, c'est une
  information : soit la table était fausse, soit le test l'était. Trancher, pas contourner.
- **Ne pas toucher au chemin natif.** `brew install ⇥`, `winget install ⇥`, `scoop install ⇥`
  doivent se comporter exactement comme en v0.7.0. Compare avant/après si tu touches à `ui`
  ou à `complete`.
- **Ne pas inventer une sortie de CLI.** Tout jeu d'essai doit être une capture réelle. C'est
  précisément l'erreur que cette tâche répare.
