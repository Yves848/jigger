# Capturer jigger

Comment sont produites les images et les enregistrements de la documentation, et
comment en produire d'autres — sur macOS, sur Omarchy et sur Windows — qui soient
**comparables entre eux**.

C'est le point de tout ce qui suit. Trois captures d'un même popup prises sur trois
machines différentes ne prouvent rien si la police, la palette, la largeur, le prompt et
les serveurs SSH changent d'une image à l'autre : le lecteur ne peut plus distinguer ce
qui tient à la plateforme de ce qui tient au photographe. Le protocole fige donc tout ce
qui n'est pas le sujet.

## Ce qui est produit

Trois scénarios, les mêmes partout — c'est le catalogue qui change, jamais le geste.

| Scénario | macOS | Omarchy | Windows |
|---|---|---|---|
| `01-gestionnaire-natif` | `brew install fire` | `yay -S visual-studio` | `winget install fire` |
| `02-jg` | `jg install fd` | `jg install fd` | `jg install fd` |
| `03-ssh` | `ssh ` | `ssh ` | `ssh ` |

Chacun rend trois fichiers dans `docs/media/out/` :

| Fichier | Sert à |
|---|---|
| `<plateforme>-<scénario>.gif` | l'enregistrement inséré dans la documentation |
| `<plateforme>-<scénario>.mp4` | le même, pour le site |
| `<plateforme>-<scénario>.png` | une image fixe, prise au moment où le popup est ouvert et au repos |

L'image fixe n'est pas capturée séparément : elle est **extraite de l'enregistrement**, à
un instant calculé à partir des `Sleep` du script et de la vitesse de frappe. Une seule
passe, et l'image fixe ne peut donc pas montrer autre chose que la vidéo.

## Pourquoi tmux

C'est la seule surprise du protocole, et elle mérite d'être comprise avant qu'on essaie
de l'enlever.

Le popup vivant demande au terminal où se trouve le curseur — un DSR, `ESC[6n` — et
n'attend la réponse que **150 ms** ; au bout de deux échecs, `_jigger_row` éteint le popup
pour la session en posant `JIGGER_LIVE=0` (`shell/jigger.plugin.zsh`). C'est un bon
comportement : un terminal qui ne répond pas coûterait sinon le délai d'attente à chaque
frappe.

Or un enregistreur s'interpose. VHS passe par `ttyd` et un `xterm.js` dans un Chrome sans
écran : l'aller-retour navigateur → ttyd → pseudo-terminal ne tient pas dans les 150 ms.
Une capture faite naïvement montre donc un terminal **sans popup**, et laisse croire que
jigger ne fait rien.

tmux résout cela parce qu'il est un vrai émulateur de terminal, local au shell capturé :
c'est lui qui répond au DSR, depuis son propre état, immédiatement. On empile donc
`VHS → tmux → zsh`, et le popup s'affiche.

Windows n'a besoin de rien de tout cela : le module PowerShell lit la position du curseur
par `GetBufferState` de PSReadLine, sans jamais interroger le terminal.

## Le décor figé

Ce que le protocole impose, et pourquoi.

| Réglage | Valeur | Pourquoi |
|---|---|---|
| Police | MesloLGL Nerd Font, 22 pt | les glyphes `◆ ▣ ● ⇥ ↩` du cadre doivent exister |
| Dimensions | 1000 × 530 px | le cadre fait 60 colonnes ; au-delà, l'image n'est que du vide |
| Palette | Catppuccin Mocha | la charte du homelab, écrite en JSON dans chaque tape |
| Vitesse de frappe | 90 ms/caractère | assez lent pour qu'on voie la liste se resserrer |
| Fréquence | 24 im/s | |
| Prompt | `❯ ` bleu, rien d'autre | un chemin daterait et localiserait la machine |
| Historique | coupé | `↑` rejouerait ce que la machine a tapé avant |
| Barre tmux | éteinte | elle affiche le nom d'hôte et la date |
| Langue | `JIGGER_LANG=en` | l'anglais des documents de référence ; `fr` pour les versions françaises |

La palette est écrite **en JSON dans chaque tape** plutôt que choisie parmi les 348 thèmes
intégrés de VHS : la liste des thèmes varie d'une version à l'autre, une palette écrite ne
varie pas.

### Les serveurs SSH sont inventés

Une capture du sélecteur SSH montre le contenu d'un `~/.ssh/config` — donc
l'infrastructure de qui l'a produite. `docs/media/fixtures/home/.ssh/config` fournit six
serveurs fictifs, et `capturer.sh` pose `HOME` dessus le temps de la capture : jigger lit
`$HOME/.ssh/config` sans surcharge possible (`internal/ssh/manager.go`).

Le fixture n'est pas qu'un décor, il est aussi un test. Il contient un `Include`, dont
provient l'hôte `atelier` — s'il n'apparaît pas dans la capture, les `Include` ne sont plus
suivis. Et il contient deux motifs, `Host *.exemple.net` et `Host *`, qui ne doivent
**jamais** apparaître.

## Produire les captures

### macOS et Omarchy

```sh
brew install vhs ffmpeg tmux        # macOS
sudo pacman -S ffmpeg tmux && yay -S vhs   # Omarchy

./docs/media/capturer.sh            # tous les scénarios de cette machine
./docs/media/capturer.sh macos-03-ssh   # ou un seul
```

La plateforme est déduite d'`uname` : sur macOS le script ne produit que les `macos-*`,
sur Arch que les `omarchy-*`. Il ouvre un serveur tmux sur un **socket dédié**
(`-L jiggercap`) : les sessions tmux réelles de la machine ne sont jamais touchées.

### Windows

```powershell
pwsh -File docs\media\capturer.ps1 -Preparer -Scenario 01-gestionnaire-natif
# … enregistrer avec Win+Alt+R, taper la ligne annoncée, ↓ ↓ ⇥ …
pwsh -File docs\media\capturer.ps1 -Convertir "$env:USERPROFILE\Videos\Captures"
```

La frappe n'y est pas pilotée : ni `ttyd` ni `tmux` n'ont d'équivalent utilisable. Le
script fige le décor — profil PowerShell de capture, dimensions, police, palette, hôtes
SSH — annonce la ligne exacte à taper, et convertit ensuite l'enregistrement avec les
mêmes réglages ffmpeg que sur Unix. La cohérence vient du décor et des lignes, pas du
pilotage.

> **`capturer.ps1` n'a jamais été exécuté.** La documentation a été produite depuis un Mac,
> sans machine Windows joignable. Le script est écrit d'après `shell/jigger.psm1` et
> d'après `capturer.sh`, dont il reprend les constantes. Signaler les écarts plutôt que de
> les corriger en silence : c'est la seule partie du protocole qui n'a pas tourné.

## Modifier un scénario

Les tapes sont **générés**, pas écrits à la main :

```sh
./docs/media/generer-tapes.sh       # réécrit docs/media/tapes/*.tape
```

Le préambule — police, taille, dimensions, palette, vitesse — est écrit une seule fois
dans le générateur et recopié tel quel dans les neuf tapes. C'est ce qui empêche macOS et
Omarchy de diverger par une recopie manuelle. Les tapes produits sont pour autant
**autonomes** : aucune directive `Source`, rien d'autre que VHS à installer. On copie un
fichier sur la machine cible et on le lance — c'est en cela que le script *est*
l'instruction.

Changer les `Sleep` d'un tape déplace l'instant où l'image fixe est extraite : la table
`instant()` de `capturer.sh` et la table `$Scenarios` de `capturer.ps1` sont à revoir
ensemble, sinon l'image fixe montrera une liste en train de défiler.

## Ce que le protocole ne couvre pas

- **Le prompt** (`jigger prompt`) n'a pas de scénario : il tient en une ligne de texte,
  que les documents citent mieux qu'une image.
- **Le sélecteur plein écran** (`JIGGER_LIVE=0`, puis `⇥`) non plus. Il mériterait un
  quatrième scénario ; personne ne l'a écrit.
- **Les captures d'Omarchy et de Windows** ne sont pas dans le dépôt à ce jour : les
  tapes et les scripts sont prêts, les machines n'étaient pas joignables. Les produire
  demande de lancer les commandes ci-dessus sur chacune, et rien d'autre.
