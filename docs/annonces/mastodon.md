# Les messages du compte Mastodon

`@jigger@mastodon.social`. Les textes s'écrivent **ici** avant d'être publiés, comme les
annonces voisines et pour la même raison : un message écrit dans le formulaire n'est relu
par personne, et ne se retrouve plus le jour où l'on veut savoir ce qui a déjà été dit.

Le profil lui-même — nom, bio, métadonnées — vit dans [`profils.md`](profils.md).

---

## 1. Le premier message

> **Publié le 2026-09-07**, épinglé —
> [`mastodon.social/@jigger/117231575658736593`](https://mastodon.social/@jigger/117231575658736593)

**Ce qu'il fait :** dire ce qu'est jigger, une fois, à un compte qui n'a encore rien
publié. Ce n'est pas une annonce de version — le compte n'a pas d'audience à qui annoncer
quoi que ce soit. C'est la carte de visite que liront ceux qui arriveront par la suite.

**À épingler** une fois publié. Un compte de projet dont le premier message explique le
projet gagne à ce qu'il reste en tête, quoi qu'il publie ensuite.

### Le texte (483 caractères sur 500)

```text
jigger is a live package picker for your shell.

Type a package-manager command and the candidates appear under the prompt, narrowing with every letter. Tab inserts, Enter runs. The arrow keys stay your shell history until the frame asks for them.

One vocabulary across Homebrew, winget, scoop and pacman: jg install fd reaches whichever of them knows fd.

A small Go binary, wired into zsh and PowerShell. Apache-2.0.

https://jigger.yg-devworks.com

#cli #zsh #golang #commandline
```

**Pas de mise en forme.** Mastodon ne rend ni le gras ni les rétroquotes : `jg install fd`
s'afficherait avec ses rétroquotes visibles. D'où le texte nu.

**Quatre mots-dièse, pas davantage.** Mastodon n'a aucun algorithme de recommandation :
les mots-dièse *sont* le mécanisme de découverte, et ne pas en mettre revient à ne parler
qu'à ses abonnés — c'est-à-dire, ici, à personne. Mais au-delà de quatre ou cinq, la
lecture bascule du côté du prospectus.

### Le média

`docs/media/out/macos-01-gestionnaire-natif.gif` — le sélecteur sur une commande native,
qui est le cœur du propos. 1000 × 530, 0,2 Mo, très en deçà des limites de l'instance.

**La description alternative n'est pas facultative** sur Mastodon : c'est une convention
forte, et son absence se remarque. Elle décrit ce que l'enregistrement montre vraiment,
glyphes compris (540 caractères sur 1500) :

```text
Terminal recording. At the prompt, "brew install fire" is typed one letter at a time. A bordered frame opens under the line, headed "brew install" on the left and "jigger 0.22.1" on the right, listing the matching packages: firealpaca, firebase-admin, firebase-cli, firebird-emu, firecamp, firefly, firefly-iota-desktop. An amber diamond marks a formula, a violet square a cask. The selection moves down twice, then Tab inserts the chosen name into the command line. A footer row reads: Tab insert, Enter execute, Down browse, Ctrl-G close.
```

### Ce que le message n'affirme pas

Rien que `show-hn.md` n'affirme déjà, et rien qui ne se vérifie à l'écran. Pas de
superlatif, pas de comparaison avec un autre outil, pas de promesse de suite. Le public de
Mastodon punit le prospectus plus sûrement qu'il ne récompense l'enthousiasme.
