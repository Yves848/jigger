# Le site de jigger — conception

16 août 2026 — état : validé, prêt pour le plan d'implémentation

## Objet

Donner à jigger **sa page**, à `jigger.yg-devworks.com` : une page unique, bilingue, qui
dise ce que l'outil est devenu et donne les trois lignes pour l'installer.

C'est l'entrée **A-6** du tableau des améliorations. **A-7**, la stratégie de diffusion,
vient après et s'appuie dessus : elle a besoin d'une URL qui répond, d'une image d'aperçu et
d'un miroir public.

## Point de départ

jigger est en v0.9.0 : trois gestionnaires, deux shells, une façade à douze verbes, une
documentation anglaise avec sa traduction française, et une fiche de projet à jour.

Ce qui existe déjà et qu'on ne réinvente pas :

- **`cocktails-website`** — une page statique bilingue (111 clés `data-i18n`, dictionnaire
  dans `app.js`, sélecteur mémorisé dans `localStorage`), déployée par `deploy-proxmox.sh`
  sur le LXC nginx `192.168.50.11`, publiée en HTTPS par le Caddy `192.168.50.10`. Elle
  répond en 0,2 s et sert `cocktails.yg-devworks.com`.
- **Une section `#jigger`** dans cette page, qui présente jigger comme le compagnon en ligne
  de commande de Cocktails. Elle parle de Tab et d'Homebrew ; elle ignore winget, scoop, la
  façade et le bilinguisme. **Elle est périmée** — sa révision est une entrée à part.
- **`jigger.yg-devworks.com`** est déjà au DNS et pointe sur l'infrastructure maison. HTTPS
  pas encore servi : Caddy n'a pas de route.

## Décisions structurantes

Quatre choix arrêtés, dont tout le reste découle.

1. **Hébergement maison, pour démarrer** — la recette Cocktails, qui existe et qui marche.
   D'où une contrainte permanente : **la page ne dépend de rien côté serveur**. Aucun code
   dynamique, aucune dépendance à nginx ni à Caddy dans les fichiers servis. Le jour où le
   lien doit encaisser un pic de trafic, le déménagement vers un hébergeur externe est une
   copie de dossier, pas une réécriture.
2. **Un site à lui, avec des liens croisés vers Cocktails.** Une annonce a besoin d'une URL
   qui parle de jigger, pas d'une ancre au milieu de la page d'une application graphique —
   et jigger tourne sur trois systèmes, là où Cocktails est macOS. Les liens croisés
   préservent ce que la suite a de vrai.
3. **Le site vit dans le dépôt jigger**, sous `website/` — et non dans un dépôt séparé
   comme Cocktails. La page et la documentation changent alors dans le même commit, passent
   sous la même relecture, et le tag de release les couvre ensemble. C'est la parade au
   défaut qui a coûté le plus cher à ce projet : deux endroits qui disent la même chose sans
   être vérifiés ensemble.
4. **Les captures sont de vraies sorties**, rejouées, jamais dessinées — la règle déjà tenue
   dans le README et le guide.

## §1 — Ce qu'on écrit

```
website/
  index.html              la page
  styles.css              dérivée de celle de Cocktails : la famille doit se voir
  app.js                  dictionnaire fr/en, sélecteur de langue
  public/                 favicon, og.png, captures
  deploy/nginx-jigger.conf
  deploy-proxmox.sh       décalque du script Cocktails, domaine jigger.yg-devworks.com
  verifier.sh             les contrôles du §4
  README.md               prévisualiser, vérifier, déployer
```

Le déploiement reprend la chaîne existante : archive des fichiers statiques, `scp` vers le
LXC nginx, dépôt dans `/var/www/jigger/releases/<horodatage>`, bascule du lien `current`,
puis ajout de la route HTTPS au Caddy. Prévisualisation locale par `python3 -m http.server`,
comme pour Cocktails.

## §2 — Ce que la page raconte

Le message a changé : jigger n'est plus « la complétion Homebrew », c'est **une syntaxe pour
trois gestionnaires, avec un popup qui suit la frappe**, sur deux shells.

1. **Accroche** — le cadre du popup en capture réelle, et la ligne d'installation dessous.
2. **Le popup vivant** — ce qu'il fait sans qu'on demande rien, les touches, et le point qui
   rassure ceux qui ont déjà été échaudés par un greffon de complétion : `↑` et `↓` restent
   l'historique du shell tant que le popup n'a pas le clavier.
3. **`jg`, une syntaxe pour trois** — les douze verbes, leur traduction vers chaque
   gestionnaire, et la règle qui compte : jamais de choix automatique entre deux
   gestionnaires qui connaissent le même nom.
4. **Ce qu'il garantit** — rien de lent dans le chemin d'une frappe ; la sortie des
   gestionnaires relayée telle quelle, invites, barres de progression et élévation
   comprises ; `--json` pour les machines.
5. **Installer** — `brew tap` sur macOS, `install-windows.ps1` sous Windows, `go install`
   partout.
6. **Le bloc de prompt** — oh-my-posh et starship, court, avec une capture.
7. **Cocktails** — les liens croisés.

**Identité visuelle.** jigger n'en a aucune, et on n'en invente pas : l'image d'aperçu sera
**le cadre du popup lui-même**, et le favicon reprendra ses angles arrondis. C'est ce que
l'outil montre de plus reconnaissable, et ça évite un logo décoratif à un programme qui n'en
a pas besoin.

## §3 — Le bilinguisme

Le mécanisme de Cocktails, repris tel quel : clés `data-i18n` dans le HTML, dictionnaire
dans `app.js`, sélecteur, choix mémorisé dans `localStorage`.

**Une seule différence, assumée :** Cocktails retombe sur le français quand le navigateur ne
demande pas l'anglais ; jigger retombera sur l'**anglais**, comme son binaire depuis la
v0.9.0. Cohérence avec le produit plutôt qu'avec le site voisin.

## §4 — Comment on saura que la page ne ment pas

`website/verifier.sh`, à lancer avant chaque déploiement. Trois contrôles, et chacun répond
à un défaut réellement rencontré dans ce projet :

- **Parité des langues** — chaque clé `data-i18n` du HTML existe dans les deux
  dictionnaires, et aucune entrée de dictionnaire n'est orpheline. C'est exactement le défaut
  poursuivi dans le binaire — un catalogue et ses appelants qui divergent — et il coûte dix
  lignes à empêcher ici.
- **Les commandes d'installation** — celles de la page sont **mot pour mot** celles du
  guide. Un `grep` croisé : si le guide change et pas la page, le script le dit. C'est le
  défaut qui a envoyé les utilisateurs Windows vers `make install`.
- **Les liens** — internes et externes, comme le contrôle déjà écrit pour les guides.

Les captures portent en commentaire la commande qui les produit, pour être rejouables.

## Portée

Une page, son déploiement, ses contrôles. Le sous-domaine est déjà au DNS ; la route Caddy
fait partie du travail.

## Non-buts

- **Pas de générateur statique.** Astro ou Hugo installeraient une chaîne Node dans un
  projet qui n'en a aucune, pour une page et un guide déjà lisibles en Markdown.
- **Pas de documentation republiée en HTML.** Le guide reste du Markdown, lu sur GitLab et
  bientôt sur GitHub ; la page y renvoie.
- **Pas de miroir GitHub ici** — c'est un prérequis d'A-7, pas du site.
- **Pas de révision de la section `#jigger` de Cocktails** — autre dépôt, autre MR.
- **Pas d'analytique, pas de formulaire, pas de cookie.** Rien qui demande un service tiers
  ni une bannière.

## Risques

| Risque | Parade |
|---|---|
| La page dérive de la documentation | Elle vit dans le même dépôt, sous la même relecture, et `verifier.sh` compare les commandes d'installation |
| Un pic de trafic tape sur la connexion domestique | Accepté « pour démarrer » ; la contrainte de portabilité du §1 fait du déménagement une copie |
| Les deux langues divergent | Contrôle de parité des clés, avant déploiement |
| Les captures vieillissent | Elles portent leur commande ; à rejouer à chaque release, comme celles du guide |
| Le style diverge de Cocktails | La feuille dérive de la sienne ; les deux pages resteront comparables à l'œil |

## Décisions liées

- [Internationalisation](2026-08-16-i18n-design.md) — l'anglais comme langue de publication,
  dont la page hérite.
- `docs/ameliorations.md` — A-6 (ce document), A-7 (la diffusion, qui suit).
