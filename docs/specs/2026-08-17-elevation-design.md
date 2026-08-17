# Constater l'élévation, et proposer de rejouer — conception

17 août 2026 — état : validé, implémentation directe.

## Objet

C'est l'entrée **A-15**, moitié Windows. La règle et ses raisons sont arrêtées par
[l'ADR-0004](../adr/0004-elevation-constatee.md) : jigger n'intercepte rien, il lit le code
de sortie après coup, et **propose** — il n'élève jamais de lui-même.

Cette spec dit comment, et où.

## §1 — Le contrat : le gestionnaire lit ses propres codes

Un contrat optionnel de plus dans `internal/pm`, sur le modèle de `pm.Bindings` : un
gestionnaire *peut* savoir traduire ses codes de sortie, il n'y est pas tenu. Celui qui ne
sait pas ne dit rien, et rien ne se propose.

```go
type Droits int

const (
    DroitsRien       Droits = iota // le code ne parle pas de droits
    DroitsRequis                   // il faut relancer élevé
    DroitsInterdits                // il faut relancer SANS élévation
)

// Elevateur : le gestionnaire qui sait ce que ses codes de sortie disent des droits.
type Elevateur interface {
    Droits(code int) Droits
}
```

Trois valeurs et non deux : `DroitsInterdits` existe parce que deux des quatre codes de
winget disent l'inverse du troisième (ADR-0004). Les confondre avec `DroitsRien` ferait
taire jigger là où il a justement quelque chose d'utile à dire.

## §2 — La table de winget

`internal/winget/elevation.go`, et rien d'autre : quatre constantes, une fonction pure.

| Code Go (non signé) | Publié par Microsoft | Nom | Verdict |
|---|---|---|---|
| `0x8A150019` | `-1978335207` | `COMMAND_REQUIRES_ADMIN` | `DroitsRequis` |
| `0x8A15C111` | `-1978285807` | `CONFIG_UNIT_IMPORT_MODULE_ADMIN` | `DroitsRequis` |
| `0x8A150056` | `-1978335146` | `INSTALLER_PROHIBITS_ELEVATION` | `DroitsInterdits` |
| `0x8A15007D` | `-1978335107` | `ADMIN_CONTEXT_ACTION_PROHIBITED` | `DroitsInterdits` |

**La colonne de gauche est celle qui va dans le code.** Microsoft publie la forme signée ;
`exec.ExitError.ExitCode()` rend sous Windows le DWORD non signé. La mesure est reportée
dans l'ADR ; le code porte le commentaire, sans quoi la prochaine relecture « corrigera »
la constante en recopiant le tableau officiel, et la comparaison ne sera plus jamais vraie.

Fonction pure d'un entier vers un verdict : elle se teste sans winget, sans Windows et sans
élévation.

## §3 — Ce que la façade rend

`Executer` rend aujourd'hui `([]pm.Package, int)`, et un seul appelant s'en sert. Plutôt
que d'allonger sa signature — dix sites d'appel dans les tests —, on ajoute la forme
longue, comme `complete.Complete` / `CompleteAvec` le font déjà :

```go
type Rejeu struct {
    Cmd    string     // le gestionnaire à relancer
    Argv   []string   // ses arguments, tels qu'ils ont été passés
    Droits pm.Droits  // DroitsRequis ou DroitsInterdits
}

type Resultat struct {
    Rows []pm.Package
    Code int
    Rejeu *Rejeu // non nil quand un code a parlé de droits
}

func ExecuterAvec(v pm.Verb, cibles []Cible, o Opts) Resultat
func Executer(v pm.Verb, cibles []Cible, o Opts) ([]pm.Package, int) // inchangée
```

Deux règles :

- **Seuls les verbes d'écriture** sont concernés. Une lecture qui échoue enchaîne sur le
  gestionnaire suivant (c'est déjà la règle) ; il n'y a rien à rejouer.
- **L'argv rendu est celui qui a réellement tourné**, `accords()` compris. Rejouer autre
  chose que ce qui a échoué serait un piège.

La façade ne demande rien et n'élève rien : elle constate et rend. C'est `main` qui a le
terminal, donc c'est `main` qui propose.

## §4 — La proposition

Même forme que la désambiguïsation (`main.trancher`), et pour la même raison : il n'y a pas
lieu d'avoir deux façons de poser une question.

- **Avec un terminal** — un cadre Bubble Tea, ouvert sur `/dev/tty` (ou son équivalent
  Windows) et non sur la sortie standard, deux entrées : *relancer en administrateur* /
  *annuler*. Pied `↵ choisir · ^G annuler`. **Annuler est le défaut** : la ligne
  sélectionnée à l'ouverture est *annuler*, et `^G` comme `Échap` valent non.
- **Sans terminal** — `jg install … | tee`, un script, une tâche planifiée : aucune
  question. jigger imprime sur la sortie d'erreur la cause et la ligne exacte à relancer,
  puis rend le code d'origine. Un pipeline ne doit jamais se bloquer sur une invite.
- **Sur `DroitsInterdits`** — aucune proposition, jamais. Un message qui dit l'inverse :
  cette commande veut être relancée **sans** élévation.

Le code de sortie de jigger reste celui du gestionnaire dans tous les cas où l'utilisateur
n'a pas rejoué.

## §5 — Le rejeu

Deux chemins, et jigger dit lequel il va prendre **avant** que l'utilisateur réponde.

1. **`sudo` activé** — `sudo <cmd> <argv…>`, lancé par le chemin ordinaire de la façade,
   relais compris. C'est le meilleur cas : selon le mode configuré, l'élévation peut rester
   dans la console courante.

   *Activé* se lit dans le registre : `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Sudo`,
   valeur `Enabled`. La clé est **absente** quand la fonction n'a jamais été activée (mesuré
   sur cette machine) ; absente ou nulle valent désactivé. Le *mode* n'est pas lu : jigger
   n'a pas à promettre où la sortie va s'afficher.

2. **Sinon** — `ShellExecuteEx`, verbe `runas`, sur le gestionnaire lui-même. C'est ce que
   fait `Start-Process -Verb RunAs`. Windows ouvre une console élevée séparée : un processus
   élevé ne peut pas s'attacher à la console d'un processus qui ne l'est pas, c'est une
   frontière du système, pas un défaut d'implémentation. jigger **attend** le processus
   (`SEE_MASK_NOCLOSEPROCESS`) et rend son code : la fenêtre se referme, mais le verdict
   revient là où la commande a été tapée.

   Les arguments passent par `syscall.EscapeArg` — `ShellExecuteEx` prend une ligne unique,
   et un identifiant winget à espaces la couperait en deux.

**Hors Windows**, le rejeu n'existe pas : le paquet expose la même fonction, qui répond
« pas ici ». Aucun `//go:build` dans la façade, un seul dans le paquet dédié.

## §6 — Ce que ça change ailleurs

- **Les greffons.** Rien. Le rejeu est un sous-processus de jigger, pas une ligne de shell :
  `Test-JiggerMutating` voit déjà `winget install …` dans la ligne tapée, et le cache des
  installés est invalidé sur cette base — que la commande ait été rejouée ou non.
- **`--yes`.** Inchangé : `accords()` s'applique à l'argv rejoué comme au premier, puisque
  c'est le même.
- **Les traductions.** Quatre chaînes de plus au catalogue, donc quatre au test de parité.

## §7 — Ce qui n'est pas fait, et pourquoi

- **La moitié Unix d'A-15.** Aucun gestionnaire Unix ne publie de code de sortie
  équivalent ; l'entrée constate elle-même que le besoin y est marginal. Elle reste ouverte,
  et devra être instruite autrement — `sudo -v` anticipé plutôt que détection.
- **La durée de grâce réglable**, que l'entrée A-15 attendait d'A-14. Elle n'a pas d'objet
  ici : jigger ne détient aucun secret et ne mémorise aucune autorisation ; c'est UAC ou
  `sudo` qui décide de ce qu'il redemande. Un réglage qui ne piloterait rien serait un
  mensonge dans l'écran de configuration.
- **L'élévation anticipée** (deviner qu'une commande va l'exiger, et élever d'emblée). Écarté
  à la conception : il faudrait deviner juste, et élever parfois pour rien.
