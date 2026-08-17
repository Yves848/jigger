# Le site de jigger

Une page statique, bilingue, sans build : trois fichiers servis tels quels.
La conception est dans [`docs/specs/2026-08-16-site-jigger-design.md`](../docs/specs/2026-08-16-site-jigger-design.md).

## Prévisualiser

```sh
cd website && python3 -m http.server 8080
```

Puis <http://localhost:8080/>. Ouvrir le fichier directement (`file://`) marche aussi,
mais les chemins absolus (`/styles.css`) ne résolvent pas : passez par le serveur.

Les fichiers vivent à plat, aux **mêmes chemins qu'en ligne** : `/styles.css`, `/app.js`,
`/jigger-icon.svg`, `/og.png`. Ce que vous voyez en local est ce qui sera servi.

## Vérifier

```sh
./verifier.sh            # parité des langues, ancres, commandes d'installation
./verifier.sh --reseau   # en plus : les liens externes répondent
```

Le déploiement lance `verifier.sh` de lui-même et s'arrête si un contrôle échoue.

## Traduire

L'anglais est écrit **en clair dans `index.html`** ; le français vit dans le dictionnaire
`FR` de `app.js`, entre les marqueurs `/* --- FR --- */`. Une clé française absente laisse
l'anglais s'afficher — jamais de clé brute à l'écran, comme dans le binaire.

Une entrée par ligne, de la forme `'cle.sous': 'texte',` : `verifier.sh` lit ce bloc avec
`grep`, pas avec un analyseur JavaScript.

Le vocabulaire de la ligne de commande ne se traduit pas — les noms des douze verbes, les
drapeaux, les noms de gestionnaires restent tels quels dans les deux langues.

## Les captures

Toutes les sorties de terminal affichées sont **de vraies sorties**. Chaque bloc porte en
commentaire HTML la commande qui l'a produite ; à rejouer quand l'affichage de jigger
change :

```sh
jigger render --line "brew install fire" --cols 60
jigger render --line "jg " --cols 60
STARSHIP_CONFIG=shell/starship/brew.toml starship prompt
```

## Refaire l'image d'aperçu

`og.html` est le gabarit ; il n'est pas déployé. Chrome le rend à la dimension exacte :

```sh
cd website && "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars \
  --screenshot=og.png --window-size=1200,630 "file://$PWD/og.html"

sips -g pixelWidth -g pixelHeight og.png     # doit dire 1200 × 630
```

## Déployer

```sh
./deploy-proxmox.sh
```

Vérifie la page, archive les fichiers, les dépose dans `/var/www/jigger/releases/<horodatage>`
sur le LXC nginx, bascule le lien `current`, installe le vhost, puis ajoute la route HTTPS
au Caddy. Les deux configurations sont validées avant rechargement et restaurées en cas
d'échec. Demande les clés SSH du Proxmox maison.
