# Le site de jigger — refonte 2026-09

*Conception. Rouvre, sur les points qu'elle traite, la spec du
[16 août 2026](2026-08-16-site-jigger-design.md), qui reste la trace de l'état antérieur.*

## Objet

Passer d'une page unique à un **site**, et le rendre montrable : trois pages, une identité
visuelle qui accueille les captures au lieu de les subir, et des illustrations partout où
un paragraphe ne suffit pas.

La demande, telle qu'elle a été formulée : une section générale de présentation, une
partie Utilisateur avec téléchargement, installation et utilisation, une section SSH à
part parce qu'elle l'est, beaucoup d'illustrations — et une première phase limitée à macOS
et Windows.

## Point de départ

Ce qui existe : `website/index.html`, une page, 389 lignes, bilingue, sans build, servie
telle quelle derrière une CSP qui interdit jusqu'au style en ligne. Elle a un hero, un
schéma SVG écrit à la main, des sorties de terminal réelles, et un déploiement qui marche.

Ce qui manque, et qui motive la refonte :

- **Aucune image, aucune animation.** Le popup, la façade et le sélecteur SSH sont décrits,
  jamais montrés. Or neuf enregistrements existent désormais dans `docs/media/out/` — trois
  sur macOS, six sur Windows — et rien ne les emploie.
- **Une seule page pour tout.** Télécharger, installer, comprendre et se dépanner tiennent
  dans le même défilement, sans hiérarchie.
- **Une identité qui ne va pas avec ses captures.** Le fond bleu-nuit `#07090e` n'a aucun
  rapport avec le Catppuccin Mocha des enregistrements : posée là, chaque vidéo ferait
  rectangle rapporté.

## Décisions structurantes

Cinq choix arrêtés, dont tout le reste découle.

1. **Trois pages, toujours sans build.** `index.html` (présentation), `utiliser.html`
   (télécharger, installer, utiliser), `ssh.html` (le sélecteur SSH). Chaque page est un
   fichier servi tel quel, partageant `styles.css` et `app.js`. La contrainte de la spec
   précédente tient : **le site ne dépend de rien côté serveur**, et déménager reste une
   copie de dossier.

2. **La page adopte la palette de ses captures : Catppuccin Mocha.** C'est le point où
   cette spec revient sur la précédente, qui voulait une identité *dérivée de Cocktails*,
   « la famille doit se voir ». La parenté visuelle valait tant que la page n'était que du
   texte ; elle coûte trop cher dès qu'on y pose neuf vidéos dont le fond est `#1e1e2e`.
   Une carte qui a le même fond que la vidéo qu'elle encadre ne l'encadre plus : elle la
   prolonge. Le lien croisé vers Cocktails, lui, demeure — la parenté se dit alors en
   toutes lettres plutôt que par la couleur.

3. **Le sélecteur de système est une préférence de site, pas un gadget de page.** Les
   commandes diffèrent d'un système à l'autre ; une page qui les empile toutes oblige le
   lecteur à trier. Deux boutons — *macOS*, *Windows* — échangent d'un coup les commandes
   **et** les vidéos. Il vit dans l'en-tête, à côté du sélecteur de langue, et vaut donc
   pour les trois pages ; `utiliser.html` le redouble en tête de page, collant, parce que
   c'est là qu'il décide le plus. C'est ce qui double la richesse perçue sans une capture
   de plus.

4. **Les médias sont copiés dans `website/media/`, et la copie est vérifiée.** Pas de lien
   symbolique : un `git checkout` sous Windows le casse. La duplication est acceptée, mais
   `verifier.sh` compare octet pour octet avec `docs/media/out/` — une capture refaite et
   non recopiée fait échouer le déploiement, elle ne passe pas en silence.

5. **Phase 1 : macOS et Windows.** Linux reste **annoncé** partout où il est vrai — le hero,
   le tableau des gestionnaires — parce que le taire ferait mentir le site. Mais la partie
   Utilisateur ne détaille que macOS et Windows, et un encart renvoie à la documentation
   pour Arch et Omarchy en attendant leurs captures.

## §1 — Ce qu'on écrit

```
website/
  index.html              l'accueil : présentation
  utiliser.html           télécharger · installer · utiliser
  ssh.html                le sélecteur SSH
  styles.css              refondu, Catppuccin Mocha
  app.js                  bilinguisme · navigation · sélecteur de système · vidéos
  media/                  18 fichiers : <plateforme>-<scénario>.mp4 et .png
  jigger-icon.svg
  og.png                  l'image d'aperçu, à refaire
  og.html                 son gabarit, non déployé
  deploy/nginx-jigger.conf
  deploy-proxmox.sh
  verifier.sh
  README.md
```

**L'en-tête est recopié dans les trois pages**, il n'est pas injecté par `app.js` : la
navigation doit exister sans JavaScript, et un moteur d'indexation doit la voir.
`verifier.sh` contrôle que les trois copies sont identiques — sélecteurs de langue et de
système compris.

**Les médias embarqués sont le `.mp4` et le `.png`**, jamais le `.gif` : la vidéo pèse trois
fois moins pour le même contenu, et le PNG sert d'affiche. Dix-huit fichiers, de l'ordre
d'un mégaoctet au total.

## §2 — Ce que chaque page raconte

### `index.html` — la présentation

| Bloc | Ce qu'il dit |
|---|---|
| hero | la phrase, la promesse, deux boutons : *Télécharger*, *Voir en action* — et la vidéo de complétion du système détecté |
| ce que ça change | trois cartes : la complétion qui suit la frappe, une syntaxe pour tous, rien qui s'interpose |
| trois canaux | le schéma SVG existant, restylé |
| une seule syntaxe | la façade : un verbe, quatre dialectes — schéma + vidéo `02-jg` |
| ça ne s'interpose pas | les garanties : la sortie relayée telle quelle, aucun choix automatique entre gestionnaires |
| le bloc de prompt | ce que `jigger prompt` ajoute, et comment s'en passer |
| et Cocktails | le lien croisé, dit en toutes lettres |
| appel | vers `utiliser.html` |

### `utiliser.html` — la partie Utilisateur

Le rappel du sélecteur de système est **collant** en haut de la page ; tout ce qui suit
lui obéit.

1. **Télécharger** — Homebrew ou scoop selon le système, puis `go install`, puis les
   sources. Les mêmes commandes que `docs/installation.md`, et `verifier.sh` s'en assure.
2. **Brancher le shell** — le greffon zsh ou le module PowerShell, avec la contrainte
   d'ordre vis-à-vis d'oh-my-posh et starship.
3. **Vérifier** — `jigger --version`, et le seul diagnostic qui compte : un binaire, un seul.
4. **Utiliser** — la complétion (`01`), la façade (`02`), la recherche par expression
   régulière (`04`), une installation menée à son terme (`05`), une mise à jour (`06`).
5. **Les touches** — le tableau `⇥ ⏎ ↓ ↑ ^N ^P ^G ^R`.
6. **Les réglages** — `JIGGER_COMMANDS`, `JIGGER_LANG`, `JIGGER_LIVE`, le cache.
7. **Encart Linux** — Arch et Omarchy fonctionnent, leurs captures viendront, voici la
   documentation.

**Les démonstrations qui n'existent que sur Windows.** Les scénarios `04`, `05` et `06`
n'ont pas d'équivalent macOS : les produire supposerait d'installer et de mettre à jour de
vrais paquets sur une machine de travail. Côté macOS, ces trois blocs montrent donc le
texte et la commande, et **le disent** — « la capture macOS viendra » — plutôt que de
laisser un trou ou de faire passer une image Windows pour une image macOS.

### `ssh.html` — le sélecteur SSH

Il est à part parce qu'il l'est : `ssh` n'est pas un gestionnaire de paquets, et c'est
justement ce qu'il prouve — le contrat de complétion ne leur est pas réservé
([ADR-0005](../adr/0005-completion-sans-facade.md)).

| Bloc | Ce qu'il dit |
|---|---|
| pourquoi à part | le contrat de complétion, et ce que `ssh` en démontre |
| en action | les vidéos `03`, macOS et Windows |
| ce qui est complété | `ssh`, `scp`, `sftp` — trois fournisseurs, pas un à trois noms |
| pas de verbe | le catalogue vient dès l'espace |
| `scp` insère un deux-points | et pourquoi ce n'est pas cosmétique |
| le fichier | le schéma SVG : `~/.ssh/config`, l'`Include` suivi, les motifs écartés |

## §3 — Le sélecteur de système

Il est **commun aux trois pages** : deux boutons dans l'en-tête, à côté du sélecteur de
langue, et un rappel collant en tête d'`utiliser.html`. Les deux commandent la même
préférence.

Ils posent `data-os="macos"` ou `data-os="windows"` sur `<html>` ; le CSS montre le bloc
voulu. Les deux versions sont **dans le DOM** : sans JavaScript, la page montre les
deux, chacune sous son titre — elle ne se vide pas, elle devient seulement plus longue.

- **Choix initial** : la plateforme du navigateur, à défaut macOS.
- **Persistance** : `localStorage`.
- **Partage** : le choix s'écrit dans l'URL (`?os=windows`), et une URL qui en porte un
  l'emporte sur la mémoire.
- **Vidéos** : celles du système inactif ne se chargent pas (`preload="none"`, `src` posé à
  l'activation).

## §4 — Les illustrations

Quatre schémas SVG écrits à la main, aux mêmes règles que celui qui existe déjà : stylés
**par classes** — la CSP pose `style-src 'self'`, qui interdit aussi le style en ligne —,
libellés traduits un `data-i18n` par `<text>`, et un `<title>` qui sert de description
accessible.

| Schéma | Ce qu'il montre | Où |
|---|---|---|
| trois canaux | shell → jigger → gestionnaires | accueil |
| le popup se resserre | la liste qui fond à chaque lettre — animé | accueil |
| la façade | un verbe, quatre dialectes | accueil |
| le fichier SSH | `~/.ssh/config`, l'`Include` suivi, les motifs écartés | ssh |

Et les neuf vidéos : boucle muette, `playsinline`, PNG en affiche, jamais de son, jamais
de contrôle qui déborde du cadre.

**`prefers-reduced-motion` coupe tout** : les schémas ne s'animent plus, les vidéos ne
démarrent plus — l'affiche reste, et elle suffit à comprendre —, les apparitions au
défilement deviennent immédiates. Ce n'est pas une dégradation : c'est la même page, arrêtée.

## §5 — Le bilinguisme

La mécanique ne change pas : **l'anglais est écrit en clair dans le HTML**, le français vit
dans le dictionnaire `FR` d'`app.js`. Une clé absente laisse l'anglais s'afficher — jamais
de clé brute à l'écran, comme dans le binaire.

Ce qui change : les clés sont **préfixées par page** (`home.`, `use.`, `ssh.`), et le
dictionnaire est découpé en trois blocs commentés dans le même ordre que les pages.
`verifier.sh` lit ces blocs avec `grep`, pas avec un analyseur JavaScript — la forme
`'cle.sous': 'texte',` sur une ligne reste donc obligatoire.

Le vocabulaire de la ligne de commande ne se traduit pas : les noms des douze verbes, les
drapeaux et les noms de gestionnaires restent tels quels dans les deux langues.

## §6 — Comment on saura que le site ne ment pas

`verifier.sh` s'étend, et reste ce qu'il est : du shell et `grep`, lancé par le déploiement,
qui l'arrête au premier échec.

| Contrôle | Ce qu'il attrape |
|---|---|
| parité des langues | un `data-i18n` sans entrée française, sur l'une des trois pages |
| en-têtes identiques | une navigation modifiée dans une page et pas dans les deux autres |
| ancres | un lien inter-pages vers une ancre qui n'existe pas |
| médias | un fichier de `website/media/` qui diffère de `docs/media/out/`, ou qui manque |
| commandes d'installation | une commande de la page qui ne figure plus dans `docs/installation.md` |
| liens externes (`--reseau`) | une URL morte |

## §7 — Déploiement

Inchangé dans son principe : archive des fichiers statiques, `scp` vers le LXC nginx, dépôt
dans `/var/www/jigger/releases/<horodatage>`, bascule du lien `current`, route HTTPS au
Caddy. Deux ajustements :

- l'archive embarque les **trois pages** et le dossier `media/` ;
- le vhost gagne **`media-src 'self'`** dans sa CSP, sans quoi les vidéos sont refusées.

## Portée

- Trois pages, leur style, leur navigation, leur bilinguisme.
- Quatre schémas SVG, dont un restylé.
- L'emploi des neuf enregistrements existants.
- `verifier.sh` étendu, `deploy-proxmox.sh` ajusté, CSP corrigée, `README.md` du site réécrit.
- L'image d'aperçu `og.png`, refaite aux nouvelles couleurs.

## Non-buts

- **Aucun build, aucun framework, aucune dépendance.** La règle qui tient depuis la
  première spec, et qui rend le déménagement trivial.
- **Aucune nouvelle capture.** Les neuf existantes suffisent à cette phase.
- **Pas de page Linux**, pas de thème clair, pas de moteur de recherche, pas de mesure
  d'audience, pas de formulaire.
- **Pas de quatrième page pour le prompt** : il reste une section de l'accueil.

## Risques

- **La duplication des médias dérive.** Parade : la comparaison octet pour octet dans
  `verifier.sh`, exécutée par le déploiement.
- **Le sélecteur de système cache de l'information.** Parade : les deux versions restent
  dans le DOM, le choix s'écrit dans l'URL, et sans JavaScript tout s'affiche.
- **La rupture avec l'identité de Cocktails déroute** qui connaît les deux sites. Parade :
  le lien croisé est conservé et la parenté se dit en toutes lettres.
- **Les trois démonstrations sans équivalent macOS déséquilibrent la page.** Parade : le
  dire, à l'endroit exact où le trou se voit.

## Décisions liées

- [Spec du site, 16 août 2026](2026-08-16-site-jigger-design.md) — l'état antérieur, et la
  décision d'identité que celle-ci rouvre.
- [ADR-0005](../adr/0005-completion-sans-facade.md) — le contrat de complétion n'est pas
  réservé aux gestionnaires de paquets ; c'est ce qui justifie une page SSH.
- [Le protocole de capture](../captures.md) — d'où viennent les neuf enregistrements, et à
  quelles conditions ils sont comparables.
