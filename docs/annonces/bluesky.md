# Les messages du compte Bluesky

`@jigger.yg-devworks.com`. Même principe que [`mastodon.md`](mastodon.md) : les textes
s'écrivent ici avant d'être publiés.

Le profil — nom, description, identifiant par le domaine — vit dans
[`profils.md`](profils.md).

---

## 1. Le premier message

> **Publié le 2026-09-08** —
> [`bsky.app/profile/jigger.yg-devworks.com/post/3muy7pvqtgx2t`](https://bsky.app/profile/jigger.yg-devworks.com/post/3muy7pvqtgx2t)

Le pendant de celui de Mastodon, **réécrit et non tronqué**. Bluesky plafonne à
**300 caractères** contre 500 : le message de Mastodon y dépassait de 183, et couper à la
serpe aurait donné une phrase amputée plutôt qu'un texte plus court.

Ce qui a été retiré est ce que Mastodon porte déjà et qui n'est pas le cœur : la
non-capture des flèches, et la mention du binaire Go. Ce qui reste répond aux trois seules
questions d'un premier message — qu'est-ce que c'est, comment ça se comporte, où
regarder.

### Le texte (260 caractères sur 300, marge 40)

```text
jigger is a live package picker for your shell.

Type a package-manager command and the candidates appear under the prompt, narrowing with every letter. Tab inserts, Enter runs.

One vocabulary across Homebrew, winget, scoop and pacman.

jigger.yg-devworks.com
```

**Une marge de 40 caractères, et c'est délibéré.** Deux formulations plus riches tenaient
en 299 et 300 caractères ; les garder aurait interdit la moindre retouche, et Bluesky
compte en **graphèmes**, une unité qui ne coïncide pas toujours avec le nombre de
caractères qu'un éditeur affiche.

### Les facettes

Bluesky ne détecte **rien** tout seul quand on publie par l'API : une URL laissée dans le
texte y reste du texte mort. Le lien doit être déclaré dans un tableau `facets`, dont les
bornes se comptent en **octets** et non en caractères.

| Élément | Bornes (octets) |
|---|---|
| `jigger.yg-devworks.com` → `https://jigger.yg-devworks.com` | 238 – 260 |

**Pas de mots-dièse**, contrairement à Mastodon. Ils y sont le mécanisme de découverte
faute d'algorithme ; Bluesky en a un, et 300 caractères se dépensent mieux ailleurs.

### Le média

`docs/media/out/macos-01-gestionnaire-natif.png` — l'image fixe, et non l'animation.

Bluesky n'anime pas les GIF téléversés comme images : il les fige. L'animation passerait
par son canal **vidéo**, avec un traitement asynchrone et des quotas quotidiens — trois
pièces de plus pour un premier message qui n'a droit qu'à un essai. L'image fixe montre le
cadre ouvert avec ses candidats, ce qui est l'essentiel.

Description alternative (444 caractères), sans la mention du défilement que l'image
fixe ne montre pas :

```text
A terminal. At the prompt, "brew install fire" is typed one letter at a time. A bordered frame opens under the line, headed "brew install" on the left and "jigger 0.22.1" on the right, listing the matching packages: firealpaca, firebase-admin, firebase-cli, firebird-emu, firecamp, firefly, firefly-iota-desktop. An amber diamond marks a formula, a violet square a cask. A footer row reads: Tab insert, Enter execute, Down browse, Ctrl-G close.
```
