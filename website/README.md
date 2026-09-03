# Le site de jigger

Une page statique, bilingue, sans build : trois fichiers servis tels quels.
La conception est dans [`docs/specs/2026-09-03-site-jigger-refonte-design.md`](../docs/specs/2026-09-03-site-jigger-refonte-design.md).

## Prévisualiser

```sh
cd website && python3 -m http.server 8080
```

Puis <http://localhost:8080/>. Ouvrir le fichier directement (`file://`) marche aussi,
mais les chemins absolus (`/styles.css`) ne résolvent pas : passez par le serveur.

Les trois pages — `index.html`, `utiliser.html`, `ssh.html` — vivent à plat, aux
**mêmes chemins qu'en ligne** : `/styles.css`, `/app.js`, `/jigger-icon.svg`, `/og.png`,
`/media/…`. Ce que vous voyez en local est ce qui sera servi.

## Vérifier

```sh
./verifier.sh            # hors ligne, déterministe
./verifier.sh --reseau   # en plus : les liens externes répondent
```

Le déploiement lance `verifier.sh` de lui-même et s'arrête au premier échec. Huit
contrôles :

1. **Parité des langues** — chaque `data-i18n` des trois pages a une entrée dans le
   dictionnaire `FR` de `app.js`, et chaque entrée du dictionnaire est employée quelque
   part. Attrape une clé posée dans le HTML et jamais traduite, et une traduction
   devenue orpheline après suppression d'un bloc.
2. **Liens** — chaque ancre `href="#…"` désigne un `id` qui existe dans la page. Avec
   `--reseau`, les liens externes d'`index.html` sont interrogés et doivent répondre en
   2xx ou 3xx. Attrape une ancre mal orthographiée et un lien externe mort.
3. **Commandes d'installation** — chaque `<code data-verifier="install">` doit exister
   mot pour mot dans `docs/getting-started.md` ou `docs/installation.md`. Deux fichiers,
   pas un : le parcours pas à pas vit dans le premier, le bloc à copier-coller par
   plateforme et le `git clone` du greffon Windows vivent dans le second. Attrape une
   commande affichée sur le site qui a divergé de la documentation.
4. **En-têtes identiques** — le bloc `<header class="site-header">` d'`utiliser.html`
   et de `ssh.html` doit être identique à celui d'`index.html`, à la classe `on` près.
   L'en-tête est recopié dans chaque page plutôt qu'injecté par `app.js`, pour que la
   navigation existe sans JavaScript ; ce contrôle est le prix de ce choix. Attrape une
   navigation modifiée sur une page et oubliée sur les deux autres.
5. **Couleurs codées en dur** — aucune couleur (`#rgb`, `#rrggbb`, `rgb()`, `rgba()`) en
   dehors du bloc `:root` de `styles.css`. Attrape une couleur posée à la main dans une
   règle, qui ne suivrait pas le jour où la palette change.
6. **Médias** — chaque `.mp4` et chaque `.png` de `website/media/` doit être identique,
   octet pour octet, à son original dans `docs/media/out/`. Voir la section suivante.
7. **Démonstrations** — chaque vidéo citée par `data-src="/media/….mp4"` doit avoir une
   affiche déclarée via `poster="…"`, et le fichier doit exister sur disque. Une vidéo
   sans affiche montre un rectangle noir tant qu'elle n'a pas chargé, et rien du tout
   sous `prefers-reduced-motion` : l'affiche est le contenu de repli, pas un ornement.
8. **Symétrie des blocs par système** — dans chaque page, le nombre de blocs
   `data-os-block="macos"` doit égaler celui des blocs `data-os-block="windows"`. Sans
   JavaScript les deux systèmes s'affichent : un bloc macOS sans son pendant Windows
   n'est pas une section masquée, c'est un trou.

## Traduire

L'anglais est écrit **en clair dans le HTML** ; le français vit dans le dictionnaire
`FR` de `app.js`, entre les marqueurs `/* --- FR --- */` et `/* --- /FR --- */`. Une clé
française absente laisse l'anglais s'afficher — jamais de clé brute à l'écran, comme
dans le binaire.

Une entrée par ligne, de la forme `'cle.sous': 'texte',` : `verifier.sh` lit ce bloc avec
`grep`, pas avec un analyseur JavaScript.

Le dictionnaire porte des commentaires qui délimitent des blocs, mais pas un par page :
`/* --- partagé : en-tête et pied --- */` couvre `nav.*` (les trois pages en dépendent),
`/* --- page « utiliser » --- */` couvre les clés `use.*`, `/* --- page « ssh » --- */`
couvre les clés `ssh.*` et `dia4.*`. Les clés de l'accueil (`hero.*`, `home.*`,
`popup.*`, `dia.*`, `dia2.*`, `dia3.*`, `facade.*`, `guar.*`, `prompt.*`, `coc.*`,
`cta.*`, `foot.*`) suivent le bloc SSH jusqu'à la fin du dictionnaire, sans en-tête
propre — repérez-les par le préfixe de la clé, pas par un commentaire.

Le vocabulaire de la ligne de commande ne se traduit pas — les noms des douze verbes, les
drapeaux, les noms de gestionnaires restent tels quels dans les deux langues.

## Les médias

`website/media/` est une **copie** de `docs/media/out/`, pas un lien. La duplication est
assumée : un lien symbolique ne survit pas à un `git checkout` sous Windows, et ce dépôt
doit rester clonable des deux côtés.

Le contrôle 6 de `verifier.sh` compare chaque fichier octet pour octet à son original.
Une capture refaite dans `docs/media/out/` et non recopiée dans `website/media/` fait
échouer ce contrôle, et donc le déploiement — elle ne part jamais en ligne en silence.
Après avoir régénéré une vidéo ou une affiche, recopiez le fichier :

```sh
cp docs/media/out/<fichier> website/media/<fichier>
```

## Refaire l'image d'aperçu

`og.html` est le gabarit ; il n'est pas déployé. Chrome le rend à la dimension exacte :

```sh
cd website && "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars \
  --screenshot=og.png --window-size=1200,630 "file://$PWD/og.html"

sips -g pixelWidth -g pixelHeight og.png     # doit dire 1200 × 630
```

L'image est commune aux trois pages ; seuls `og:url`, `og:title` et `og:description`
changent d'une page à l'autre, repris du `<title>` et de la `<meta name="description">`
de chacune.

## Déployer

```sh
./deploy-proxmox.sh
```

Vérifie la page, archive les fichiers, les dépose dans `/var/www/jigger/releases/<horodatage>`
sur le LXC nginx, bascule le lien `current`, installe le vhost, puis ajoute la route HTTPS
au Caddy. Les deux configurations sont validées avant rechargement et restaurées en cas
d'échec. Demande les clés SSH du Proxmox maison.
