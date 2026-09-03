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

Trois scénarios communs aux trois plateformes — c'est le catalogue qui change, jamais
le geste — et trois autres, propres à Windows, que la voie zsh ne sait pas jouer.

| Scénario | macOS | Omarchy | Windows |
|---|---|---|---|
| `01-gestionnaire-natif` | `brew install fire` | `yay -S visual-studio` | `winget install fire` |
| `02-jg` | `jg install fd` | `jg install fd` | `jg install node` |
| `03-ssh` | `ssh ` | `ssh ` | `ssh ` |

**Pourquoi `node` et non `fd` sous Windows.** Le geste doit rester le même, et c'est ce
qui a tranché. `jg install fd` n'y trouve qu'un seul paquet — le `fd` de scoop, winget
n'en connaissant aucun : le popup n'aurait qu'une ligne, et les deux `↓` du scénario n'y
déplaceraient rien. `jg install node` en trouve quatre, **répartis entre scoop et
winget** — la colonne de droite le dit —, ce qui est précisément la démonstration
attendue de `jg`. Le catalogue change d'une plateforme à l'autre ; la ligne tapée s'ajuste
pour que le geste, lui, ne change pas.

### Les trois scénarios propres à Windows

| Scénario | Ce qu'il montre |
|---|---|
| `04-regex` | la même ligne filtrée en texte simple, puis en expression régulière après `^R` |
| `05-installation` | la complétion, puis l'exécution : `jg install hexy` ⇥ ⏎, et scoop installe |
| `06-upgrade` | le même geste, autre verbe : `jg upgrade hyperf` ⇥ ⏎, et scoop passe de 1.16.1 à 1.20.0 |

Ils ne sont pas dans la voie zsh, et ce n'est pas un oubli : VHS pilote une frappe, pas
une session. `04-regex` demande de taper, presser `^R`, puis retaper — deux temps de
frappe séparés par une touche, qu'un tape exprimerait mal ; `05` et `06` **exécutent
vraiment** une commande dont la durée n'est pas connue d'avance. Les produire sur macOS
demanderait d'installer et de mettre à jour de vrais paquets sur la machine de qui les
produit ; sous Windows, la machine est une VM dédiée, et le script y range derrière lui.

**Ce que ces deux scénarios font à la machine, en toutes lettres.** `05` installe puis
désinstalle `hexyl` ; `06` installe `hyperfine` en 1.16.1, le laisse se mettre à jour en
1.20.0, puis le retire. Aucun paquet que la machine avait déjà n'est touché, et l'état
de départ est posé par le script lui-même — sans quoi un second passage filmerait « déjà
installé » au lieu d'une installation. C'est le prix d'une capture qui prouve quelque
chose : une exécution jouée ne prouverait rien.

Chacun rend trois fichiers dans `docs/media/out/` :

| Fichier | Sert à |
|---|---|
| `<plateforme>-<scénario>.gif` | l'enregistrement inséré dans la documentation |
| `<plateforme>-<scénario>.mp4` | le même, pour le site |
| `<plateforme>-<scénario>.png` | une image fixe, prise au moment où le popup est ouvert et au repos |

L'image fixe n'est pas capturée séparément : elle est **extraite de l'enregistrement**, à
un instant calculé à partir des `Sleep` du script et de la vitesse de frappe. Une seule
passe, et l'image fixe ne peut donc pas montrer autre chose que la vidéo. Cela vaut sur
les trois plateformes : la mécanique diffère sous Windows, le principe non.

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
par `GetBufferState` de PSReadLine, sans jamais interroger le terminal. C'est ce qui rend
la capture Windows possible sans empiler quoi que ce soit sous le shell — mais elle n'a
pas VHS pour autant, et la section qui lui est consacrée dit avec quoi il est remplacé.

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
| Barre d'onglets | éteinte (Windows) | le mode « focus » de Windows Terminal ; sinon elle mange une ligne et nomme le profil |
| Langue | `JIGGER_LANG=en` | l'anglais des documents de référence ; `fr` pour les versions françaises |

**Le nombre de lignes n'est pas dans cette table, et c'est voulu.** Personne ne le
déclare : VHS le déduit de la hauteur demandée (530 px) et de la hauteur de cellule que
`xterm.js` donne à la police. `capturer.ps1` fait la même déduction, mais il doit d'abord
**mesurer** la cellule — elle dépend de la police, du DPI et de la version de Windows
Terminal, et rien ne permet de la supposer. Il ouvre donc un terminal, mesure, ferme, et
rouvre au bon format.

Les deux mesures ne tombent pas exactement au même endroit : le cadre fait 32,1 px par
ligne sur la capture macOS, 33,9 px sur celle de Windows — six pour cent d'écart, les deux
terminaux n'arrondissant pas les métriques de MesloLGL de la même façon. La grille compte
donc 15 lignes sur un tape et 14 sous Windows. C'est justement pourquoi la mesure vaut
mieux qu'un nombre écrit : à 14 lignes fixées d'avance, un autre DPI aurait donné une
autre image.

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
winget install Gyan.FFmpeg          # ffmpeg et ffprobe
# MesloLGL Nerd Font installée, jigger sur le PATH, Windows Terminal présent

pwsh -NoProfile -File docs\media\capturer.ps1                    # les six
pwsh -NoProfile -File docs\media\capturer.ps1 -Scenario 04-regex # ou un seul
pwsh -NoProfile -File docs\media\capturer.ps1 -Preparer          # le décor, sans filmer
```

La capture est **pilotée de bout en bout**, comme sur Unix : rien à taper, rien à
enregistrer à la main. Ce sont les instruments qui changent, pas la partition.

| Sur zsh | Sur PowerShell | Pourquoi |
|---|---|---|
| VHS + ttyd + `xterm.js` | Windows Terminal | ttyd n'a pas d'équivalent Windows utilisable |
| tmux | *rien* | le module lit le curseur par PSReadLine, jamais par le terminal |
| la frappe de VHS | `SendInput` en Unicode, cadencé sur une horloge absolue | une disposition de clavier ne doit pas décider du contenu de l'image |
| les touches de VHS | `SendKeys` (`{TAB}`, `{DOWN}`, `^r`…) | ce sont des touches virtuelles, pas des caractères |
| l'encodeur de VHS | `ffmpeg -f gdigrab` sur le rectangle client | |

**Où s'arrête l'enregistrement.** Les quatre premiers scénarios ont un minutage exact :
la somme de leurs étapes. Les deux qui exécutent une commande, non — la leur télécharge.
On leur laisse donc une minute, puis on coupe la queue morte : `freezedetect` de ffmpeg
dit à quels instants l'image cesse de bouger, et le dernier d'entre eux est la fin de la
commande. Le GIF rendu dure ce que la commande a duré, plus deux secondes.

Tout ce que le script change sur la machine — les réglages de Windows Terminal — est
sauvegardé avant et rendu après, y compris si la capture échoue (`finally`).

#### Les six pièges, et ce qu'ils coûtent

Ils ne se voient pas dans le résultat : chacun produit une image plausible et fausse.
Ils sont notés ici parce que les retrouver a pris plus de temps que d'écrire le script.

1. **Le profil dédié n'est pas retenu.** Ajouter un profil « jigger-capture » dans
   `settings.json` serait plus propre que de toucher au profil par défaut. Windows
   Terminal 1.23 relit bien le fichier à chaud — on l'a vérifié en changeant la taille de
   police du profil par défaut, qui prend effet — mais il ne connaît pas un profil apparu
   après son démarrage : `--profile jigger-capture` retombe **silencieusement** sur le
   profil par défaut, et la capture sort avec la police et la transparence de
   l'utilisateur. D'où le choix d'habiller le profil par défaut lui-même.

2. **`SetProcessDPIAware`, et avant tout le reste.** Sans lui, `GetClientRect` rend des
   unités logiques quand `gdigrab` filme des pixels physiques, et la vidéo ne montre
   qu'un coin de la fenêtre. La machine de référence est à 96 ppp : l'appel n'y change
   rien, et c'est précisément ce qui rend le piège dangereux — il ne se manifeste que
   sur un écran mis à l'échelle, c'est-à-dire chez quelqu'un d'autre. Il doit être fait
   **avant** que quoi que ce soit crée une fenêtre, `System.Windows.Forms` compris.

3. **`SendKeys` ne tient pas la cadence.** Chaque appel coûte plusieurs dizaines de
   millisecondes ; un `Start-Sleep 90ms` entre deux caractères dérive de moitié sur une
   ligne de vingt. La première passe a rendu une image fixe prise en pleine frappe :
   `winget install fi` au lieu de `winget install fire`. La frappe est donc cadencée sur
   une horloge **absolue**, chaque caractère visant `i × 90 ms`.

4. **ffmpeg ne s'arrête pas sur `q`.** Il ne lit le clavier que depuis une vraie console ;
   par un tuyau, le `q` est ignoré, et le tuer laisse un conteneur inachevé que `ffprobe`
   refuse ensuite de mesurer. Il s'arrête donc sur `-t`, dont la valeur est celle du
   scénario. Reste que le début de l'enregistrement, lui, ne se devine pas : entre le
   lancement du processus et la première image il s'écoule une seconde variable, et tout
   décalage là décale l'image fixe d'autant. Le script attend donc la première ligne
   d'état (`frame=`) avant de lancer son horloge.

5. **`SendKeys` ne sait pas taper `|` sur un clavier AZERTY.** Il traduit chaque
   caractère en touche selon la disposition courante ; `|` y est un `AltGr+6` qu'il ne
   sait pas former, et il l'avale **sans rien dire**. La première prise du scénario regex
   a filmé `fire(birdblade)` et un franc « no matches ». La frappe passe donc par
   `SendInput` en mode Unicode, qui envoie le caractère lui-même : la capture ne dépend
   plus de la disposition de la machine qui la produit.

6. **scoop ne compare pas au bucket une version demandée à la main.** `scoop install
   hyperfine@1.16.1` engendre un manifeste et range l'application sous
   `<auto-generated>` ; `scoop status` cesse alors de la comparer au bucket, et
   `jg upgrade hyperfine` répond « latest version ». La première prise du scénario de
   mise à jour ne montrait donc aucune mise à jour. La préparation rattache l'application
   à son bucket d'origine avant de filmer.

Deux détails plus petits, du même genre : `adjustIndistinguishableColors` doit être mis à
`never`, sans quoi Windows Terminal retouche les couleurs de jigger et la capture ne
montre plus la palette de la charte ; et le curseur doit être un bloc **fixe** — la
fixture `profile.ps1` émet `DECSCUSR 2` — sinon l'image fixe le montre ou non selon la
phase du clignotement.

## Modifier un scénario

Les tapes sont **générés**, pas écrits à la main :

```sh
./docs/media/generer-tapes.sh       # réécrit docs/media/tapes/*.tape
```

Le préambule — police, taille, dimensions, palette, vitesse — est écrit une seule fois
dans le générateur et recopié tel quel dans les six tapes. C'est ce qui empêche macOS et
Omarchy de diverger par une recopie manuelle. Les tapes produits sont pour autant
**autonomes** : aucune directive `Source`, rien d'autre que VHS à installer. On copie un
fichier sur la machine cible et on le lance — c'est en cela que le script *est*
l'instruction.

**Il n'y a pas de tape Windows**, et il ne peut pas y en avoir : VHS y passerait par ttyd
et par tmux, dont aucun n'existe. Les gestes de Windows y sont tenus par la table
`$Scenarios` de `capturer.ps1`, qui reprend les `Sleep` du tape zsh du même scénario,
milliseconde pour milliseconde — et ils ne sont pas les mêmes d'un scénario à l'autre :
`02-jg` n'a qu'une flèche, `03-ssh` n'attend que 2,5 s avant la première.

Changer les `Sleep` d'un tape déplace l'instant où l'image fixe est extraite : la table
`instant()` de `capturer.sh` et la table `$Scenarios` de `capturer.ps1` sont à revoir
ensemble, sinon l'image fixe montrera une liste en train de défiler. Côté Windows,
l'instant n'est pas écrit : il est **calculé** — 800 ms d'attente, la frappe, puis deux
secondes —, et il redonne bien les valeurs qu'annoncent les tapes (4,5 s, 4,0 s, 3,0 s).

## Ce que le protocole ne couvre pas

- **Le prompt** (`jigger prompt`) n'a pas de scénario : il tient en une ligne de texte,
  que les documents citent mieux qu'une image.
- **Le sélecteur plein écran** (`JIGGER_LIVE=0`, puis `⇥`) non plus. Il mériterait un
  septième scénario ; personne ne l'a écrit.
- **Le sélecteur de routage** — celui qui s'ouvre quand deux gestionnaires connaissent le
  même nom — n'en a pas davantage : le déclencher demande de lancer une vraie
  installation ambiguë, et les READMEs s'en tiennent pour l'instant à un exemple
  illustratif.
- **Les captures d'Omarchy** ne sont pas dans le dépôt à ce jour : le tape est prêt, la
  machine n'était pas joignable. Les produire demande de lancer `./docs/media/capturer.sh`
  sur elle, et rien d'autre. Les trois scénarios `04` à `06` n'y ont pas d'équivalent, et
  n'en auront pas tant qu'ils supposeront d'installer pour de vrai sur une machine de
  travail.

Les captures **macOS et Windows**, elles, sont dans le dépôt et ont été produites par les
scripts ci-dessus. Celles de Windows viennent d'une machine ARM64 sous Windows 11 26200,
Windows Terminal 1.23.12811, PowerShell 7.5.4 — une machine virtuelle Parallels, ce qui
ne change rien à l'image : le popup ne sait pas où il tourne.
