# Les profils des comptes du projet

Les textes des comptes `jigger` sur les réseaux où l'on diffuse. Ils sont **écrits ici**,
comme les annonces, pour la même raison : un texte de profil se relit, se corrige et se
réutilise d'une plateforme à l'autre. Recopié à la main dans quatre formulaires, il diverge
en trois mois sans que personne s'en aperçoive.

**Différence avec les annonces du même dossier :** celles-ci sont publiées par Yves, en son
nom. Ceux-là appartiennent à un **compte projet** (décision du 2026-09-07, jigger#187) : ils
parlent au nom de jigger, jamais à la première personne d'Yves.

## Ce qui vaut pour tous

**Le lien du code pointe GitHub, pas le GitLab de référence.** C'est la même décision que
pour les annonces, et elle a été payée : le 17 août, un lien vers `gitlab.yg-devworks.com`
a fait rejeter un post sans motif. Un domaine personnel sans réputation tombe sous les
filtres. Le miroir GitHub existe pour ça.

Le site, lui, se cite sans réserve : c'est le domaine du projet, et c'est celui que la
vérification rend **vert** sur le profil.

**Avatar :** [`website/jigger-avatar.png`](../../website/jigger-avatar.png), 512 × 512,
coins transparents. Rendu depuis `website/jigger-icon.svg`, qui reste la source :

```sh
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars --force-device-scale-factor=8 \
  --default-background-color=00000000 \
  --screenshot=website/jigger-avatar.png --window-size=64,64 \
  "file://$PWD/website/jigger-icon.svg"
```

Le drapeau `--default-background-color=00000000` n'est pas décoratif : sans lui, Chrome
peint sa page en blanc derrière le SVG, et les coins arrondis deviennent un halo blanc sur
tout thème sombre.

**Pas de bannière.** `website/og.png` fait 1200 × 630 ; les bannières attendent du 3:1
(Mastodon 1500 × 500). Le recadrage couperait la moitié de la hauteur. Une bannière ratée
se remarque plus qu'une bannière absente — à faire à part, ou pas du tout.

## Mastodon — `@jigger@mastodon.social`

Instance retenue : `mastodon.social`. Non par affinité — `fosstodon.org` conviendrait mieux
au sujet — mais parce qu'elle est en **inscription ouverte**, là où fosstodon et hachyderm
demandent une candidature avec délai.

**Nom affiché**

```
jigger
```

**Bio** (500 caractères au maximum)

```
A live package picker in your shell, and one syntax across Homebrew, winget, scoop and
pacman. Type a package command and the candidates follow as you type; Tab inserts, Enter
runs. A small Go binary wired into zsh and PowerShell.

Release notes land here. Apache-2.0.
```

**La ligne vide avant « Release notes » compte.** Sans elle, Mastodon affiche
`PowerShell.Release notes land here.` — deux phrases collées, que l'œil lit comme une
coquille. Constaté sur le profil réel le 2026-09-07 : le saut s'était perdu à la copie. Le
champ accepte les retours à la ligne, il faut simplement qu'ils survivent au presse-papiers.

La dernière ligne n'est pas du remplissage. Un compte projet qui n'annonce que des versions
**se lit comme un robot**, et les lecteurs comme les plateformes le pénalisent. Dire ce que
le compte est évite qu'on le devine mal.

**Les quatre lignes de métadonnées.** Mastodon n'a pas de champ « site web » distinct : ce
sont ces lignes qui portent les liens, et c'est l'une d'elles qui se vérifie.

| Libellé | Contenu |
|---|---|
| `Site` | `https://jigger.yg-devworks.com` |
| `Code` | `https://github.com/Yves848/jigger` |
| `Install` | `brew install yves/cocktails/jigger` |
| `License` | `Apache-2.0` |

**La ligne `Site` est celle qui compte pour la vérification.** L'instance la suit, cherche
sur le site un `rel="me"` qui revient au profil, et affiche alors le domaine en vert. Ce
lien est déjà posé dans `website/index.html` (jigger#188) — il ne reste qu'à déployer le
site.

## Bluesky — `@jigger.yg-devworks.com`

**L'identifiant passe par le domaine**, et c'est mieux que n'importe quel `.bsky.social` :
`jigger.bsky.social` est déjà pris, tandis qu'un identifiant fondé sur un domaine qu'on
contrôle ne peut être squatté par personne.

L'ordre est contraint :

1. créer le compte avec un identifiant `.bsky.social` temporaire ;
2. relever son **DID** (`did:plc:…`) dans les paramètres ;
3. poser un enregistrement `TXT` sur `_atproto.jigger.yg-devworks.com`, de valeur
   `did=did:plc:…` ;
4. basculer l'identifiant sur le domaine.

**Nom affiché**

```
jigger
```

**Description** (256 caractères au maximum — plus courte que sur Mastodon)

```
A live package picker in your shell, and one syntax across Homebrew, winget, scoop and
pacman. A small Go binary wired into zsh and PowerShell.

Release notes land here. Apache-2.0 — github.com/Yves848/jigger
```

Bluesky n'a pas de lignes de métadonnées : le lien du code va donc dans la description,
et le site est porté par l'identifiant lui-même.
