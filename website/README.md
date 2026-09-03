# Le site de jigger

Un site statique, bilingue, sans build : trois pages servies telles quelles, avec quatre
schémas SVG écrits à la main, vingt-huit médias — quatorze enregistrements et leurs
affiches, macOS, Omarchy et Windows — et un vérificateur à huit contrôles.
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
   `--reseau`, les liens externes **des trois pages** sont interrogés et doivent répondre
   en 2xx ou 3xx — le guide d'installation n'est cité que par `utiliser.html` et
   l'ADR-0005 que par `ssh.html`. Attrape une ancre mal orthographiée et un lien externe
   mort.
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
   `data-os-block="macos"`, `="windows"` et `="omarchy"` doit être le même. Sans
   JavaScript les trois systèmes s'affichent : un bloc macOS sans son pendant Omarchy
   n'est pas une section masquée, c'est un trou — même quand un système n'a pas encore
   de capture, il a un bloc « demo-absent » qui le dit.

## Traduire

L'anglais est écrit **en clair dans le HTML** ; le français vit dans le dictionnaire
`FR` de `app.js`, entre les marqueurs `/* --- FR --- */` et `/* --- /FR --- */`. Une clé
française absente laisse l'anglais s'afficher — jamais de clé brute à l'écran, comme
dans le binaire.

Une entrée par ligne, de la forme `'cle.sous': 'texte',` : `verifier.sh` lit ce bloc avec
`grep`, pas avec un analyseur JavaScript.

Le `<title>` de chaque page porte lui aussi un `data-i18n` (`title.home`, `title.use`,
`title.ssh`) : l'anglais reste écrit dans la balise, le français vient du dictionnaire, et
chaque page garde son propre titre. Une table de titres dans `app.js` n'en connaissait
qu'un et l'appliquait aux trois.

Le dictionnaire porte des commentaires qui délimitent des blocs, mais pas un par page :
`/* --- les titres de page --- */` couvre les trois clés `title.*`,
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

## Les schémas

Ils sont quatre, écrits à la main — ni image, ni script. Trois sur l'accueil : le cadre
qui se resserre (`#popup`), la façade (`#facade`) et le grand schéma des trois canaux
(`#fonctionnement`). Le quatrième est sur `ssh.html` : le fichier lu, les hôtes proposés,
les motifs écartés.

- Ils sont **stylés par classes** (`.dia-*` dans `styles.css`), jamais par attribut
  `style` **ni par attribut de présentation** (`fill`, `stroke`, `font-size`…) : la CSP du
  vhost pose `style-src 'self'`, qui interdit le style en ligne, et une couleur posée en
  attribut vivrait hors des jetons de `:root` — ce que le contrôle 5 refuse. `text-anchor`
  est la seule exception admise : il ne porte ni couleur ni style.
- Leurs libellés se traduisent **comme le reste de la page**, un `data-i18n` par élément
  `<text>` — y compris le `<title>` du SVG, qui sert de description accessible et que
  `role="img"` + `aria-labelledby` désignent. `verifier.sh` contrôle donc ces clés-là
  comme les autres.
- Les noms de gestionnaires et les commandes ne se traduisent pas, comme partout ailleurs.

Les coordonnées du schéma de fonctionnement suivent une grille : trois bandes (shell,
jigger, gestionnaires) et trois canaux, l'aller sur x = 136/486/836 et le retour sur
404/754/1104. Déplacer une bande demande de bouger les extrémités des flèches en regard.

### Les regarder, dans les deux langues

Un schéma propre en anglais peut déborder en français : mêmes coordonnées, texte plus
long. Il n'y a pas d'autre moyen que de le rendre et de le regarder — **dans les deux
langues**. Extraire le `<svg>` dans un fichier autonome, y coller les jetons de `:root` et
les règles `.dia-*` dans une balise `<style>`, remplacer chaque libellé par la valeur du
dictionnaire `FR` pour la version française, puis :

```sh
qlmanage -t -s 1400 -o /tmp/dia schema.svg
```

`qlmanage` rend au carré ; pour la dimension exacte du `viewBox`, Chrome est plus fidèle :

```sh
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
  --disable-gpu --hide-scrollbars --window-size=1240,754 \
  --screenshot=/tmp/dia/schema.png "file:///chemin/absolu/schema.svg"
```

Puis **ouvrir l'image**. Une étiquette qui sort de sa boîte, ou qui passe sous une flèche,
se corrige par la géométrie — boîte élargie, étiquette déplacée, libellé coupé en deux
lignes avec une clé `data-i18n` par ligne. Jamais en raccourcissant le français : ce
serait appauvrir une langue pour épargner un ajustement de coordonnées.

## Les captures

Les sorties de terminal affichées dans les blocs `<pre class="frame">` sont **de vraies
sorties**, pas des maquettes. Chaque bloc porte en commentaire HTML la commande qui l'a
produite ; à rejouer quand l'affichage de jigger change :

```sh
jigger render --line "jg " --cols 60
STARSHIP_CONFIG=shell/starship/brew.toml starship prompt
```

Le résultat se recolle tel quel dans le bloc, et le commentaire reste au-dessus. Les
enregistrements vidéo, eux, ne se rejouent pas ici : ils viennent de `docs/media/out/`
(voir la section précédente).

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

### Les fichiers ne sont pas mis en cache pour longtemps, et c'est voulu

Aucun nom ne porte d'empreinte : `styles.css` s'appelle `styles.css` d'un déploiement à
l'autre, et une capture refaite garde le sien — c'est le protocole qui l'impose, une
image s'appelle d'après son scénario. Un cache à durée fixe sert donc l'**ancien** fichier
sous le nouveau nom. Le vhost a longtemps posé `expires 7d` : un visiteur déjà venu
pouvait recevoir le HTML du jour habillé par le CSS de la semaine précédente, sans qu'un
rechargement ordinaire n'y change rien — les schémas SVG, dont les classes ne sont pas
dans l'ancien CSS, tombaient alors sur les valeurs par défaut de SVG, c'est-à-dire un
`fill` noir sur fond sombre.

`deploy/nginx-jigger.conf` pose donc `expires -1` sur les images, les vidéos, le CSS et le
JS : le navigateur revalide à chaque visite, nginx répond `304` sur l'`ETag` tant que rien
n'a bougé. Quelques octets par visite, et un déploiement est visible tout de suite.

Mais une consigne d'en-tête ne rattrape **jamais** une entrée de cache déjà posée : un
navigateur qui détient l'ancien fichier avec l'ancienne consigne ne redemandera rien avant
son expiration. Seule l'URL peut le forcer. `deploy-proxmox.sh` estampille donc, à la
publication, une empreinte du contenu sur les deux fichiers dont dépend l'affichage :

```html
<link rel="stylesheet" href="/styles.css?v=c4534944">
<script src="/app.js?v=1f73781b" defer></script>
```

Huit caractères du SHA-256 du fichier. Le fichier change, l'empreinte change, l'URL change,
et aucun cache ne peut plus servir une version pour une autre. Le script échoue si
l'estampille ne s'est pas posée — une page qui ne cite plus sa feuille sortirait sans
style, et personne ne le verrait avant la mise en ligne.

L'estampille ne vit que dans la **copie publiée** : les pages du dépôt gardent
`href="/styles.css"`, s'ouvrent en local sans rien construire, et `verifier.sh` les lit
telles quelles. C'est ce qui permet d'avoir des URL versionnées sans build.

C'est `expires` et non `add_header Cache-Control` : dans nginx, un `add_header` posé dans
un `location` **annule** ceux du bloc serveur. L'ancienne version en posait un, et ces
fichiers repartaient sans CSP, sans `nosniff`, sans `X-Frame-Options` ni
`Referrer-Policy` — un effet de bord que rien ne signalait.
