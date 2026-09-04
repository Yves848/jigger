# Empaqueter jigger

Ce dossier contient les recettes d'empaquetage pour les distributions où jigger
n'est **pas encore** publié. Rien ici n'est automatique : ce sont des fichiers à
soumettre à la main, chacun dans le dépôt de sa distribution.

Homebrew et scoop ne sont pas ici : ils ont leurs propres dépôts
(`yves/homebrew-cocktails` et `yves/scoop-jigger`) et sont déjà en service.

## AUR — `aur/`

Deux recettes, exclusives l'une de l'autre (`conflicts`) :

| Dossier | Ce qu'elle fait |
|---|---|
| `jigger` | construit depuis les sources, au tag ; demande Go |
| `jigger-bin` | pose le binaire précompilé ; ne demande rien |

**Les deux installent le greffon zsh**, et c'est le point à ne pas rater :
l'archive de publication ne contient que le binaire, la licence et le README.
Un paquet qui se contenterait de l'archive livrerait un jigger qui ne complète
rien — le greffon est ce qui dessine le cadre. `jigger-bin` va donc le chercher
séparément, au même tag.

Le greffon est posé dans `/usr/share/zsh/plugins/jigger/`, la convention Arch ;
le message d'après-installation donne la ligne `source` à ajouter.

### Soumettre

```sh
git clone ssh://aur@aur.archlinux.org/jigger.git aur-jigger
cp packaging/aur/jigger/{PKGBUILD,jigger.install} aur-jigger/
cd aur-jigger
makepkg --printsrcinfo > .SRCINFO   # exigé : l'AUR lit .SRCINFO, pas le PKGBUILD
makepkg -si                          # construire et installer, pour vérifier
git add PKGBUILD .SRCINFO jigger.install && git commit && git push
```

`.SRCINFO` n'est pas versionné ici : il se régénère et dépend de la version du
`makepkg` qui l'écrit. Le produire au moment de la soumission, sur une machine
Arch, est la seule façon qu'il soit juste.

## winget — `winget/`

Trois manifestes au schéma 1.6.0 : version, installeur, locale. L'archive n'est
pas un installeur mais un exécutable autonome, d'où
`InstallerType: zip` + `NestedInstallerType: portable` — winget pose lui-même
l'alias sur le `PATH`.

Comme sur Arch, le binaire seul ne complète rien : le module PowerShell vient du
dépôt. La description du manifeste le dit plutôt que de le taire.

### Soumettre

```sh
winget validate --manifest packaging/winget
winget install --manifest packaging/winget   # essai local avant toute soumission
```

Puis une pull request sur `microsoft/winget-pkgs`, dans
`manifests/y/YvesGodart/jigger/0.15.0/`. `wingetcreate submit` fait la même
chose sans quitter la ligne de commande.

## À reprendre à chaque version

Les sommes de contrôle et le numéro de version sont écrits en clair dans les
recettes ; ils sont **relevés sur la publication**, pas devinés :

```sh
curl -s https://gitlab.yg-devworks.com/api/v4/projects/25/packages/generic/jigger/<version>/SHA256SUMS
```

`jigger` (sources) est le seul à ne pas en porter : il suit un tag git, et
`SKIP` y est justifié — pas une facilité.
