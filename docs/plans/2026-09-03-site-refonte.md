# Refonte du site jigger — plan d'implémentation

> **Pour les agents :** SOUS-SKILL REQUIS — `superpowers:subagent-driven-development`
> (recommandé) ou `superpowers:executing-plans` pour dérouler ce plan tâche par tâche.
> Les étapes emploient des cases à cocher (`- [ ]`).

**But :** faire passer le site de jigger d'une page unique à trois pages illustrées, à la
palette de ses propres captures.

**Architecture :** trois fichiers HTML servis tels quels, partageant `styles.css` et
`app.js`. Aucun build, aucune dépendance. `app.js` porte trois responsabilités et rien de
plus : le bilinguisme (déjà là), le sélecteur de système, et l'activation paresseuse des
vidéos. Le contrôle de non-régression est `verifier.sh` — c'est la suite de tests du site,
et chaque tâche l'étend **avant** d'écrire ce qu'il contrôle.

**Pile :** HTML, CSS, JavaScript sans framework ; `bash` + `grep` pour les contrôles ;
nginx pour servir.

**Spec :** [`docs/specs/2026-09-03-site-jigger-refonte-design.md`](../specs/2026-09-03-site-jigger-refonte-design.md)

## Contraintes globales

Elles valent pour **toutes** les tâches, sans être répétées dans chacune.

- **Aucun build, aucune dépendance, aucun framework.** Les fichiers déployés sont servis
  tels quels.
- **Aucun style en ligne, aucun script en ligne.** La CSP du vhost pose `style-src 'self'`
  et `default-src 'self'` : un attribut `style=` ou un `<script>` sans `src` est ignoré par
  le navigateur, silencieusement. Tout passe par des classes et par `app.js`.
- **Aucune nouvelle capture.** Les neuf enregistrements de `docs/media/out/` suffisent.
- **L'anglais est écrit en clair dans le HTML**, le français vit dans le dictionnaire `FR`
  d'`app.js`, une entrée par ligne, de la forme exacte `'cle.sous': 'texte',` — `verifier.sh`
  lit ce bloc avec `grep`, pas avec un analyseur JavaScript.
- **Le vocabulaire de la ligne de commande ne se traduit pas** : noms des douze verbes,
  drapeaux, noms de gestionnaires restent tels quels dans les deux langues.
- **Chaque tâche finit sur `./website/verifier.sh` au vert** et un commit.
- **Langue des commits, des commentaires et de la documentation : le français.**

### Les jetons de couleur, valeur pour valeur

Catppuccin Mocha. Ces valeurs sont exactement celles du thème des tapes VHS
(`docs/media/generer-tapes.sh`) : elles ne s'inventent pas.

| Jeton | Valeur | Rôle |
|---|---|---|
| `--base` | `#1e1e2e` | fond de page — **identique au fond des vidéos** |
| `--mantle` | `#181825` | fond des barres (en-tête, pied) |
| `--crust` | `#11111b` | fond des blocs de code |
| `--surface` | `#313244` | cartes |
| `--surface-2` | `#45475a` | bordures |
| `--text` | `#cdd6f4` | texte |
| `--subtext` | `#a6adc8` | texte sourdine |
| `--overlay` | `#6c7086` | texte très sourdine |
| `--mauve` | `#cba6f7` | accent principal |
| `--blue` | `#89b4fa` | accent secondaire — le `❯` de l'invite |
| `--green` | `#a6e3a1` | ce qui réussit, les badges ◆ |
| `--peach` | `#fab387` | ce qui attend, les avertissements |
| `--red` | `#f38ba8` | ce qui échoue |

### Les neuf enregistrements, nommés

`<plateforme>-<scénario>` ; chacun existe en `.mp4`, `.png` et `.gif` dans
`docs/media/out/`. **Le site embarque le `.mp4` et le `.png`, jamais le `.gif`.**

| Fichier | Ce qu'il montre |
|---|---|
| `macos-01-gestionnaire-natif` | `brew install fire` — le cadre suit la frappe |
| `macos-02-jg` | `jg install fd` — la façade |
| `macos-03-ssh` | `ssh ` — le sélecteur SSH |
| `windows-01-gestionnaire-natif` | `winget install fire` |
| `windows-02-jg` | `jg install node` — scoop **et** winget dans la même liste |
| `windows-03-ssh` | `ssh ` |
| `windows-04-regex` | `^R`, puis l'alternance entre parenthèses |
| `windows-05-installation` | `jg install hexy` ⇥ ⏎, scoop installe |
| `windows-06-upgrade` | `jg upgrade hyperf` ⇥ ⏎, 1.16.1 → 1.20.0 |

**macOS n'a pas de `04`, `05` ni `06`** : là où le sélecteur est sur macOS, ces trois blocs
montrent le texte et la commande, et disent que la capture viendra.

---

## Structure des fichiers

| Fichier | Responsabilité |
|---|---|
| `website/index.html` | l'accueil : présentation |
| `website/utiliser.html` | télécharger, installer, utiliser |
| `website/ssh.html` | le sélecteur SSH |
| `website/styles.css` | tout le style des trois pages |
| `website/app.js` | bilinguisme · sélecteur de système · activation des vidéos |
| `website/media/` | 18 fichiers, copiés de `docs/media/out/` |
| `website/verifier.sh` | la suite de contrôles, lancée par le déploiement |
| `website/deploy-proxmox.sh` | archive et publie |
| `website/deploy/nginx-jigger.conf` | vhost et CSP |
| `website/og.html`, `website/og.png` | l'aperçu social et son gabarit |
| `website/README.md` | prévisualiser, vérifier, traduire, déployer |

**Une précision sur les préfixes de clés.** La spec (§5) demande des clés préfixées par
page. Les clés de l'accueil existent déjà et **identifient déjà l'accueil** (`hero.`,
`popup.`, `facade.`, `dia.`, `guar.`, `prompt.`, `coc.`, `install.`) : les renommer serait
du bruit sans gain. On applique donc la règle aux nouveautés — `use.` pour `utiliser.html`,
`ssh.` pour `ssh.html`, `nav.` et `foot.` pour ce qui est partagé — et le dictionnaire est
découpé en blocs commentés dans l'ordre des pages.

---

### Task 1 : Le socle multi-pages

**Fichiers :**
- Modifier : `website/verifier.sh` (contrôles 1 à 3 : boucler sur trois pages)
- Créer : `website/utiliser.html`, `website/ssh.html` (squelettes : en-tête, pied, un titre)
- Modifier : `website/index.html` (navigation)
- Modifier : `website/app.js` (clés `nav.*` des nouvelles pages)

**Interfaces :**
- Consomme : rien.
- Produit : la constante shell `PAGES=(index.html utiliser.html ssh.html)` dans
  `verifier.sh` ; le bloc `<header class="site-header">…</header>` **identique** dans les
  trois pages, à la classe `on` près ; les clés `nav.home`, `nav.use`, `nav.ssh`.

- [ ] **Étape 1 : écrire le contrôle qui échoue — trois pages, un seul en-tête**

Dans `website/verifier.sh`, juste après la ligne `[ "${1:-}" = "--reseau" ] && RESEAU=1`,
ajouter :

```bash
# Les trois pages du site. Tout contrôle qui lit « la page » les lit toutes.
PAGES=(index.html utiliser.html ssh.html)

for p in "${PAGES[@]}"; do
    [ -f "$p" ] || { printf 'ÉCHEC  — page absente : %s\n' "$p" >&2; exit 1; }
done
```

Puis, à la fin du fichier, **avant** le bloc `if [ "$echecs" -gt 0 ]`, le contrôle
d'identité des en-têtes. La classe `on` se déplace d'une page à l'autre : on la neutralise
avant de comparer.

```bash
# --- 4. En-têtes identiques -----------------------------------------------
# L'en-tête est recopié dans chaque page plutôt qu'injecté par app.js : la
# navigation doit exister sans JavaScript. Le prix, c'est cette vérification.
entete() { sed -n '/<header class="site-header">/,/<\/header>/p' "$1" | sed 's/ class="on"//'; }
reference="$(entete index.html)"
for p in "${PAGES[@]:1}"; do
    if [ "$(entete "$p")" = "$reference" ]; then
        ok "en-tête identique à celui d'index.html : $p"
    else
        echec "en-tête différent de celui d'index.html : $p"
    fi
done
```

- [ ] **Étape 2 : lancer le contrôle, vérifier qu'il échoue**

```sh
cd website && ./verifier.sh
```

Attendu : `ÉCHEC — page absente : utiliser.html`, code de sortie 1.

- [ ] **Étape 3 : ajouter les liens de navigation dans `index.html`**

Remplacer le bloc `<nav aria-label="Main navigation">…</nav>` de `website/index.html` par :

```html
      <nav aria-label="Main navigation">
        <a href="/" class="on" data-i18n="nav.home">Home</a>
        <a href="/utiliser.html" data-i18n="nav.use">Use it</a>
        <a href="/ssh.html" data-i18n="nav.ssh">SSH</a>
      </nav>
```

Et, dans le même en-tête, remplacer `<a class="brand" href="#top">jigger</a>` par
`<a class="brand" href="/">jigger</a>` : l'ancre `#top` n'a de sens que sur l'accueil, et
les trois en-têtes doivent être identiques.

- [ ] **Étape 4 : créer `website/utiliser.html`**

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>jigger — download, install, use</title>
  <meta name="description" content="Download jigger, wire it into zsh or PowerShell, and see what it does — on macOS and on Windows.">
  <meta name="theme-color" content="#07090e">
  <link rel="icon" href="/jigger-icon.svg">
  <link rel="stylesheet" href="/styles.css">
  <script src="/app.js" defer></script>
</head>
<body>
  <!-- EN-TÊTE : copie exacte de celui d'index.html, hormis la classe « on ». -->
  <header class="site-header">
    <div class="wrap header-inner">
      <a class="brand" href="/">jigger</a>
      <nav aria-label="Main navigation">
        <a href="/" data-i18n="nav.home">Home</a>
        <a href="/utiliser.html" class="on" data-i18n="nav.use">Use it</a>
        <a href="/ssh.html" data-i18n="nav.ssh">SSH</a>
      </nav>
      <div class="lang-toggle" role="group" aria-label="Language">
        <button type="button" data-lang="en" aria-pressed="true">EN</button>
        <button type="button" data-lang="fr" aria-pressed="false">FR</button>
      </div>
    </div>
  </header>

  <main id="top">
    <section class="page-head">
      <div class="wrap">
        <p class="eyebrow" data-i18n="use.eyebrow">Use it</p>
        <h1 data-i18n="use.h1">Download, wire, use.</h1>
      </div>
    </section>
  </main>

  <footer class="site-footer">
    <div class="wrap">
      <p data-i18n="foot.made">Free software, Apache 2.0. Built in the open.</p>
      <p class="foot-links">
        <a href="https://gitlab.yg-devworks.com/yves/jigger" data-i18n="foot.repo">Repository</a>
        <a href="https://gitlab.yg-devworks.com/yves/jigger/-/blob/main/docs/getting-started.md" data-i18n="foot.guide">Getting started</a>
        <a href="https://gitlab.yg-devworks.com/yves/jigger/-/blob/main/CHANGELOG.md" data-i18n="foot.changelog">Changelog</a>
      </p>
    </div>
  </footer>
</body>
</html>
```

- [ ] **Étape 5 : créer `website/ssh.html`**

Même fichier qu'à l'étape 4, avec ces quatre substitutions et rien d'autre :

- `<title>jigger — the SSH picker</title>`
- `<meta name="description" content="Type ssh and the popup offers the servers of your ~/.ssh/config. Same frame, same keys — only the catalogue changes.">`
- la classe `on` sur `<a href="/ssh.html">` au lieu de `<a href="/utiliser.html">`
- le bloc `<section class="page-head">` devient :

```html
    <section class="page-head">
      <div class="wrap">
        <p class="eyebrow" data-i18n="ssh.eyebrow">The SSH picker</p>
        <h1 data-i18n="ssh.h1">Not just package managers.</h1>
      </div>
    </section>
```

- [ ] **Étape 6 : traduire les clés nouvelles**

Dans `website/app.js`, à l'intérieur du bloc `/* --- FR --- */ … /* --- /FR --- */`,
remplacer les quatre lignes `'nav.*'` existantes par :

```javascript
    /* --- partagé : en-tête et pied --- */
    'nav.home': 'Accueil',
    'nav.use': 'L’utiliser',
    'nav.ssh': 'SSH',
    /* --- page « utiliser » --- */
    'use.eyebrow': 'L’utiliser',
    'use.h1': 'Télécharger, brancher, utiliser.',
    /* --- page « ssh » --- */
    'ssh.eyebrow': 'Le sélecteur SSH',
    'ssh.h1': 'Pas seulement les gestionnaires de paquets.',
```

Les anciennes clés `nav.popup`, `nav.facade`, `nav.how` et `nav.install` disparaissent avec
les liens qu'elles servaient : le contrôle 1 signale toute entrée orpheline.

- [ ] **Étape 7 : les trois classes que les nouvelles pages emploient**

`.page-head`, `.note` et `.encart` n'existent pas encore dans `styles.css` : une classe
absente ne casse rien de visible, elle laisse seulement le bloc sans style — c'est le
genre d'oubli qu'aucun contrôle n'attrape. Ajouter à la fin de `website/styles.css` :

```css
/* --- pages intérieures ---------------------------------------------------
   L'accueil a son hero ; les deux autres pages ont ce bandeau, plus sobre. */
.page-head { padding: 54px 0 18px; }
.page-head h1 { font-size: clamp(30px, 4.5vw, 46px); margin: 6px 0 0; }

.note { color: var(--subtext); font-size: 15px; }

.encart {
  border: 1px solid var(--surface-2);
  border-left: 3px solid var(--peach);
  border-radius: var(--radius);
  background: var(--mantle);
  padding: 22px 24px;
}
.encart h2 { margin-top: 0; font-size: 21px; }
```

- [ ] **Étape 8 : relancer le contrôle, vérifier qu'il passe**

```sh
cd website && ./verifier.sh
```

Attendu : `Tout est vert.`, code de sortie 0. En particulier : `en-tête identique à celui
d'index.html : utiliser.html` et `: ssh.html`.

- [ ] **Étape 9 : commit**

```bash
git add website/verifier.sh website/index.html website/utiliser.html website/ssh.html website/app.js website/styles.css
git commit -m "Le site prend trois pages, et le vérificateur les lit toutes"
```

---

### Task 2 : La palette des captures

**Fichiers :**
- Modifier : `website/styles.css:6-22` (le bloc `:root`) et le reste du fichier
- Modifier : `website/index.html`, `website/utiliser.html`, `website/ssh.html` (`theme-color`)
- Modifier : `website/og.html` (le gabarit de l'aperçu)
- Modifier : `website/verifier.sh` (contrôle 5)

**Interfaces :**
- Consomme : les trois pages de la tâche 1.
- Produit : les treize jetons `--base`, `--mantle`, `--crust`, `--surface`, `--surface-2`,
  `--text`, `--subtext`, `--overlay`, `--mauve`, `--blue`, `--green`, `--peach`, `--red`,
  employés par toutes les tâches suivantes.

- [ ] **Étape 1 : écrire le contrôle qui échoue — aucune couleur en dur**

Dans `website/verifier.sh`, avant le bloc final :

```bash
# --- 5. Aucune couleur écrite en dur hors des jetons -----------------------
# Une couleur posée à la main dans une règle échappe à la palette : elle ne
# suivra pas le jour où la palette bouge, et c'est exactement ce qui a fait
# diverger la page de ses propres captures.
hors_jetons="$(sed '/^:root {/,/^}/d' styles.css \
               | grep -nE '#[0-9a-fA-F]{3,8}\b|rgba?\(' || true)"
if [ -n "$hors_jetons" ]; then
    echec "couleurs en dur hors de :root dans styles.css — $(printf '%s' "$hors_jetons" | head -3 | tr '\n' ' ')"
else
    ok "toutes les couleurs de styles.css viennent des jetons"
fi
```

- [ ] **Étape 2 : lancer, vérifier l'échec**

```sh
cd website && ./verifier.sh
```

Attendu : `ÉCHEC — couleurs en dur hors de :root`, la feuille actuelle en contenant
plusieurs (`rgba(7,9,14,.72)` dans `.site-header`, `#06080f` dans `.lang-toggle button.on`,
les deux `radial-gradient` de `body::before`).

- [ ] **Étape 3 : remplacer le bloc `:root` de `website/styles.css`**

```css
:root {
  /* Catppuccin Mocha — la palette des captures. Le fond de la page est le fond
     des vidéos : une carte qui encadre une vidéo la prolonge au lieu de la poser. */
  --base:      #1e1e2e;
  --mantle:    #181825;
  --crust:     #11111b;
  --surface:   #313244;
  --surface-2: #45475a;
  --text:      #cdd6f4;
  --subtext:   #a6adc8;
  --overlay:   #6c7086;
  --mauve:     #cba6f7;
  --blue:      #89b4fa;
  --green:     #a6e3a1;
  --peach:     #fab387;
  --red:       #f38ba8;

  --radius:    18px;
  --maxw:      1080px;
  --sans: system-ui, -apple-system, "SF Pro Text", "Segoe UI", Roboto, sans-serif;
  --mono: ui-monospace, "SF Mono", "JetBrains Mono", "Fira Code", Menlo, monospace;
}
```

- [ ] **Étape 4 : convertir toutes les règles**

Les anciens noms (`--bg`, `--panel`, `--card`, `--line`, `--line-2`, `--muted`, `--accent`,
`--accent-2`, `--amber`) disparaissent : chaque occurrence est remplacée dans le corps de
la feuille.

| Ancien | Nouveau |
|---|---|
| `var(--bg)` | `var(--base)` |
| `var(--panel)` | `var(--mantle)` |
| `var(--card)` | `var(--surface)` |
| `var(--line)` | `var(--surface-2)` |
| `var(--line-2)` | `var(--overlay)` |
| `var(--muted)` | `var(--subtext)` |
| `var(--accent)` | `var(--mauve)` |
| `var(--accent-2)` | `var(--blue)` |
| `var(--amber)` | `var(--peach)` |
| `background: rgba(7,9,14,.72);` (`.site-header`) | `background: var(--mantle);` |
| `color: #06080f;` (`.lang-toggle button.on`) | `color: var(--crust);` |
| `rgba(255,255,255,.08)` | `var(--surface-2)` |
| `rgba(255,255,255,.14)` | `var(--overlay)` |

Et le halo de `body::before` :

```css
body::before {
  content: "";
  position: fixed; inset: 0; z-index: -1;
  background:
    radial-gradient(1100px 700px at 78% -8%, color-mix(in srgb, var(--mauve) 14%, transparent), transparent 60%),
    radial-gradient(900px 620px at 8% 12%, color-mix(in srgb, var(--blue) 8%, transparent), transparent 55%);
}
```

- [ ] **Étape 5 : la couleur de barre système, sur les trois pages**

Dans chacune des trois pages, remplacer :

```html
  <meta name="theme-color" content="#07090e">
```

par :

```html
  <meta name="theme-color" content="#1e1e2e">
```

- [ ] **Étape 6 : le gabarit de l'aperçu**

Dans `website/og.html`, remplacer les couleurs par les mêmes valeurs. Le fichier n'est pas
déployé, mais il produit `og.png` : le laisser en bleu ferait mentir l'aperçu partagé.

- [ ] **Étape 7 : relancer, vérifier le vert, et regarder la page**

```sh
cd website && ./verifier.sh && python3 -m http.server 8080
```

Ouvrir `http://localhost:8080/` : le fond doit être `#1e1e2e`, exactement celui des PNG de
`docs/media/out/`. Ouvrir l'un d'eux dans un autre onglet pour comparer.

- [ ] **Étape 8 : commit**

```bash
git add website/styles.css website/og.html website/index.html website/utiliser.html website/ssh.html website/verifier.sh
git commit -m "Le site prend la palette de ses captures"
```

---

### Task 3 : Les médias, et le contrôle qui interdit qu'ils dérivent

**Fichiers :**
- Créer : `website/media/` (18 fichiers)
- Modifier : `website/verifier.sh` (contrôle 6)
- Modifier : `website/deploy-proxmox.sh` (la liste archivée)
- Modifier : `website/deploy/nginx-jigger.conf:15` (la CSP)

**Interfaces :**
- Consomme : rien.
- Produit : les chemins `/media/<plateforme>-<scénario>.mp4` et `.png`, employés par les
  tâches 4 et suivantes.

- [ ] **Étape 1 : écrire le contrôle qui échoue — la copie doit être exacte**

Dans `website/verifier.sh`, avant le bloc final :

```bash
# --- 6. Médias ------------------------------------------------------------
# website/media/ est une COPIE de docs/media/out/. La duplication est assumée
# — un lien symbolique ne survit pas à un git checkout sous Windows —, mais
# elle est vérifiée : une capture refaite et non recopiée échoue ici, elle ne
# part pas en ligne en silence.
source_medias="$RACINE/docs/media/out"
if ! compgen -G 'media/*.mp4' > /dev/null; then
    echec "aucun média dans website/media/"
else
    for f in media/*.mp4 media/*.png; do
        nom="$(basename "$f")"
        if [ ! -f "$source_medias/$nom" ]; then
            echec "média sans original dans docs/media/out : $nom"
        elif cmp -s "$f" "$source_medias/$nom"; then
            ok "média identique à son original : $nom"
        else
            echec "média différent de docs/media/out : $nom"
        fi
    done
fi
```

- [ ] **Étape 2 : lancer, vérifier l'échec**

```sh
cd website && ./verifier.sh
```

Attendu : `ÉCHEC — aucun média dans website/media/`.

- [ ] **Étape 3 : copier les dix-huit fichiers**

```sh
cd website && mkdir -p media
for s in macos-01-gestionnaire-natif macos-02-jg macos-03-ssh \
         windows-01-gestionnaire-natif windows-02-jg windows-03-ssh \
         windows-04-regex windows-05-installation windows-06-upgrade; do
  cp "../docs/media/out/$s.mp4" "../docs/media/out/$s.png" media/
done
ls media | wc -l    # doit dire 18
```

- [ ] **Étape 4 : relancer, vérifier le vert**

```sh
cd website && ./verifier.sh
```

Attendu : dix-huit lignes `ok — média identique à son original`.

- [ ] **Étape 5 : autoriser les vidéos dans la CSP**

Dans `website/deploy/nginx-jigger.conf`, ligne 15, insérer `media-src 'self';` après
`img-src 'self';` :

```
    add_header Content-Security-Policy "default-src 'self'; style-src 'self'; img-src 'self'; media-src 'self'; form-action 'self'; base-uri 'self'; frame-ancestors 'self'" always;
```

Sans `media-src 'self'`, le navigateur refuse les `<video>` — et il le fait sans rien
montrer sur la page : seule la console le dit.

- [ ] **Étape 6 : archiver les nouveaux fichiers au déploiement**

Dans `website/deploy-proxmox.sh`, remplacer le bloc `tar -czf "$ARCHIVE" …` par :

```bash
# og.html est le gabarit qui a produit og.png : il n'a rien à faire en ligne.
tar -czf "$ARCHIVE" -C "$SCRIPT_DIR" \
  index.html utiliser.html ssh.html styles.css app.js jigger-icon.svg og.png media
```

- [ ] **Étape 7 : commit**

```bash
git add website/media website/verifier.sh website/deploy-proxmox.sh website/deploy/nginx-jigger.conf
git commit -m "Les captures entrent dans le site, et le vérificateur interdit qu'elles dérivent"
```

---

### Task 4 : Le composant de démonstration

**Fichiers :**
- Modifier : `website/styles.css` (bloc `.demo`)
- Modifier : `website/app.js` (fonction `activerDemos`)
- Modifier : `website/index.html` (une démonstration, dans la section `#popup`)
- Modifier : `website/verifier.sh` (contrôle 7)

**Interfaces :**
- Consomme : les chemins `/media/*` de la tâche 3, les jetons de la tâche 2.
- Produit : le motif `<figure class="demo">` — repris tel quel par les tâches 6, 7 et 8 —
  et la fonction `activerDemos()` d'`app.js`, appelée à l'initialisation et à chaque
  changement de système.

- [ ] **Étape 1 : écrire le contrôle qui échoue — une vidéo a toujours son affiche**

Dans `website/verifier.sh`, avant le bloc final :

```bash
# --- 7. Démonstrations ----------------------------------------------------
# Une vidéo sans affiche montre un rectangle noir tant qu'elle n'est pas
# chargée — et, sous prefers-reduced-motion, elle ne montre jamais rien
# d'autre. L'affiche n'est donc pas un ornement : c'est le contenu de repli.
for p in "${PAGES[@]}"; do
    while IFS= read -r src; do
        [ -z "$src" ] && continue
        affiche="${src%.mp4}.png"
        if grep -Fq "poster=\"$affiche\"" "$p"; then
            ok "démonstration avec affiche : $(basename "$src") ($p)"
        else
            echec "démonstration sans affiche $affiche dans $p"
        fi
        [ -f ".$src" ] || echec "média absent : $src (cité par $p)"
    done <<< "$(grep -o 'data-src="/media/[^"]*\.mp4"' "$p" | cut -d'"' -f2)"
done
```

- [ ] **Étape 2 : lancer, vérifier que le contrôle passe à vide**

```sh
cd website && ./verifier.sh
```

Attendu : `Tout est vert.` — aucune démonstration n'existe encore, la boucle ne tourne pas.
C'est normal : le contrôle prend son sens à l'étape 5.

- [ ] **Étape 3 : le style du composant**

Ajouter à la fin de `website/styles.css` :

```css
/* --- démonstrations ------------------------------------------------------
   La carte a le fond des vidéos : elle les prolonge au lieu de les encadrer.
   Seule la bordure dit où finit l'image. */
.demo { margin: 28px 0; }
.demo-frame {
  border: 1px solid var(--surface-2);
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--base);
  line-height: 0;
}
.demo-frame video,
.demo-frame img { width: 100%; height: auto; display: block; }
.demo figcaption {
  margin-top: 10px;
  color: var(--subtext);
  font-size: 15px;
}
.demo-absent {
  border: 1px dashed var(--surface-2);
  border-radius: var(--radius);
  padding: 22px;
  color: var(--subtext);
  background: var(--mantle);
}
```

- [ ] **Étape 4 : l'activation paresseuse dans `app.js`**

Dans `website/app.js`, avant la ligne `setLang(initial);`, insérer :

```javascript
  /* --- démonstrations ---------------------------------------------------
     Les vidéos ne portent pas leur src : app.js le pose au moment où la
     démonstration devient visible. Deux raisons — ne pas télécharger les
     enregistrements du système que le lecteur n'a pas choisi, et respecter
     prefers-reduced-motion, où l'affiche suffit et rien ne démarre. */
  var immobile = false;
  try {
    immobile = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  } catch (e) {}

  function activerDemos() {
    if (immobile) return;
    Array.prototype.forEach.call(document.querySelectorAll('video[data-src]'), function (v) {
      /* offsetParent vaut null quand un ancêtre est en display:none — c'est
         ainsi qu'on sait qu'une démonstration appartient au système inactif. */
      if (v.offsetParent === null) return;
      if (!v.src) { v.src = v.getAttribute('data-src'); }
      var p = v.play();
      if (p && p.catch) { p.catch(function () {}); }
    });
  }
```

Puis, juste après `setLang(initial);`, ajouter :

```javascript
  activerDemos();
```

- [ ] **Étape 5 : poser la première démonstration**

Dans `website/index.html`, dans la section `<section id="popup">`, après le paragraphe
`data-i18n="popup.lede"`, insérer :

```html
        <figure class="demo">
          <div class="demo-frame">
            <video muted loop playsinline preload="none"
                   poster="/media/macos-01-gestionnaire-natif.png"
                   data-src="/media/macos-01-gestionnaire-natif.mp4"
                   aria-labelledby="demo-mac01"></video>
          </div>
          <figcaption id="demo-mac01" data-i18n="home.demo.mac01">brew install fire — the frame appears on its own and narrows with every letter. Captured on macOS.</figcaption>
        </figure>
```

Et dans le dictionnaire `FR` d'`app.js`, dans le bloc de l'accueil :

```javascript
    'home.demo.mac01': 'brew install fire — le cadre arrive tout seul et se resserre à chaque lettre. Pris sur macOS.',
```

- [ ] **Étape 6 : relancer, vérifier le vert, et regarder**

```sh
cd website && ./verifier.sh && python3 -m http.server 8080
```

Attendu : `ok — démonstration avec affiche : macos-01-gestionnaire-natif.mp4 (index.html)`.
Sur `http://localhost:8080/`, la vidéo tourne en boucle, sans son ni contrôle. Vérifier
ensuite avec le mouvement réduit activé (macOS : Réglages → Accessibilité → Moniteur →
Réduire les animations) : l'affiche reste, rien ne démarre.

- [ ] **Étape 7 : commit**

```bash
git add website/styles.css website/app.js website/index.html website/verifier.sh
git commit -m "Un composant de démonstration : vidéo muette, affiche de repli, chargement paresseux"
```

---

### Task 5 : Le sélecteur de système

**Fichiers :**
- Modifier : les trois pages (le bloc `os-toggle` dans l'en-tête)
- Modifier : `website/styles.css` (blocs `.os-toggle` et `[data-os-block]`)
- Modifier : `website/app.js` (fonction `setOs`)
- Modifier : `website/verifier.sh` (contrôle 8)

**Interfaces :**
- Consomme : `activerDemos()` de la tâche 4.
- Produit : l'attribut `data-os` sur `<html>` (`"macos"` ou `"windows"`) ; l'attribut
  `data-os-block="macos"` / `"windows"` posé sur tout élément propre à un système — c'est
  le seul mécanisme employé par les tâches 6, 7 et 8.

- [ ] **Étape 1 : écrire le contrôle qui échoue — jamais un système sans l'autre**

Dans `website/verifier.sh`, avant le bloc final :

```bash
# --- 8. Symétrie des blocs par système ------------------------------------
# Sans JavaScript, les deux systèmes s'affichent : un bloc macOS sans son
# pendant Windows n'est pas une section masquée, c'est un trou. Le compte
# doit donc être le même dans chaque page.
for p in "${PAGES[@]}"; do
    n_mac="$(grep -c 'data-os-block="macos"' "$p" || true)"
    n_win="$(grep -c 'data-os-block="windows"' "$p" || true)"
    if [ "$n_mac" = "$n_win" ]; then
        ok "blocs par système équilibrés dans $p ($n_mac de chaque)"
    else
        echec "$p : $n_mac bloc(s) macOS pour $n_win bloc(s) Windows"
    fi
done
```

- [ ] **Étape 2 : lancer, vérifier que le contrôle passe à vide (0 = 0)**

```sh
cd website && ./verifier.sh
```

Attendu : `ok — blocs par système équilibrés dans index.html (0 de chaque)`.

- [ ] **Étape 3 : le sélecteur dans l'en-tête des trois pages**

Dans **chacune** des trois pages, juste avant `<div class="lang-toggle" …>`, insérer :

```html
      <div class="os-toggle" role="group" aria-label="Operating system">
        <button type="button" data-os="macos" aria-pressed="true">macOS</button>
        <button type="button" data-os="windows" aria-pressed="false">Windows</button>
      </div>
```

Les trois en-têtes doivent rester identiques : le contrôle 4 le vérifie.

- [ ] **Étape 4 : le style**

Ajouter à `website/styles.css` :

```css
/* --- sélecteur de système ------------------------------------------------
   Sans JavaScript, aucun data-os n'est posé sur <html> et les deux blocs
   s'affichent, chacun sous son titre. La page ne se vide jamais. */
.os-toggle { display: flex; gap: 2px; }
.os-toggle button {
  background: none; border: 0; color: var(--subtext);
  font: inherit; font-size: 14px; padding: 5px 11px;
  border-radius: 999px; cursor: pointer;
}
.os-toggle button.on { background: var(--mauve); color: var(--crust); }

html[data-os="macos"]   [data-os-block="windows"] { display: none; }
html[data-os="windows"] [data-os-block="macos"]   { display: none; }
/* Le titre interne d'un bloc ne sert qu'au cas sans JavaScript : dès qu'un
   système est choisi, le sélecteur le dit déjà. */
html[data-os] .os-block-title { display: none; }

.os-sticky {
  position: sticky; top: 56px; z-index: 9;
  background: var(--mantle);
  border-bottom: 1px solid var(--surface-2);
  color: var(--subtext); font-size: 14px;
  padding: 8px 0;
}
.os-sticky .os-name { color: var(--text); font-weight: 600; }
```

- [ ] **Étape 5 : la logique dans `app.js`**

Dans `website/app.js`, après la fonction `activerDemos`, insérer :

```javascript
  /* --- sélecteur de système ---------------------------------------------
     Une préférence de site, pas de page : elle vaut pour les trois. L'ordre
     de décision — l'URL d'abord (un lien partagé doit imposer ce qu'il
     montre), puis le choix mémorisé, puis la plateforme du navigateur. */
  function setOs(os) {
    os = (os === 'windows') ? 'windows' : 'macos';
    docEl.setAttribute('data-os', os);
    Array.prototype.forEach.call(document.querySelectorAll('.os-toggle button'), function (b) {
      var on = b.getAttribute('data-os') === os;
      b.setAttribute('aria-pressed', on ? 'true' : 'false');
      b.classList.toggle('on', on);
    });
    try { localStorage.setItem('jigger-os', os); } catch (e) {}
    activerDemos();
  }

  Array.prototype.forEach.call(document.querySelectorAll('.os-toggle button'), function (b) {
    b.addEventListener('click', function () { setOs(b.getAttribute('data-os')); });
  });

  var osUrl = null;
  try { osUrl = new URLSearchParams(location.search).get('os'); } catch (e) {}
  var osMemo = null;
  try { osMemo = localStorage.getItem('jigger-os'); } catch (e) {}
  var osNav = /win/i.test(navigator.platform || navigator.userAgent || '') ? 'windows' : 'macos';
  setOs(osUrl || osMemo || osNav);
```

- [ ] **Étape 6 : relancer, vérifier le vert, et éprouver le repli**

```sh
cd website && ./verifier.sh && python3 -m http.server 8080
```

Trois vérifications à l'écran :

1. Cliquer *Windows* : le bouton s'allume, `<html>` porte `data-os="windows"` (inspecteur).
2. Recharger : le choix tient.
3. Désactiver JavaScript dans le navigateur et recharger : la page reste entière et
   affiche les deux systèmes — pas d'écran vide.

- [ ] **Étape 7 : commit**

```bash
git add website/index.html website/utiliser.html website/ssh.html website/styles.css website/app.js website/verifier.sh
git commit -m "Un sélecteur macOS/Windows commun aux trois pages"
```

---

### Task 6 : La page Utiliser

**Fichiers :**
- Modifier : `website/utiliser.html` (tout le contenu)
- Modifier : `website/app.js` (bloc FR `use.*`)
- Modifier : `website/verifier.sh` (contrôles 1, 2 et 3 : boucler sur les trois pages)

**Interfaces :**
- Consomme : `data-os-block` (tâche 5), `<figure class="demo">` (tâche 4).
- Produit : les ancres `#telecharger`, `#brancher`, `#verifier`, `#utiliser`, `#touches`,
  `#reglages`, `#linux` — citées par l'accueil (tâche 8).

- [ ] **Étape 1 : étendre les contrôles 1, 2 et 3 aux trois pages**

Contrôle 1 — la parité des langues lit désormais les trois pages. Remplacer la ligne
`cles_html=…` par :

```bash
cles_html="$(cat "${PAGES[@]}" | grep -o 'data-i18n="[^"]*"' | cut -d'"' -f2 | sort -u)"
```

Contrôle 2 — les ancres, page par page. Remplacer la boucle existante par :

```bash
for p in "${PAGES[@]}"; do
    for ancre in $(grep -o 'href="#[^"]*"' "$p" | cut -d'"' -f2 | cut -c2- | sort -u); do
        [ "$ancre" = "top" ] && continue
        if grep -q "id=\"$ancre\"" "$p"; then
            ok "ancre #$ancre ($p)"
        else
            echec "ancre #$ancre ne désigne aucun id dans $p"
        fi
    done
done
```

Contrôle 3 — les commandes d'installation, page par page :

```bash
guide="$RACINE/docs/getting-started.md"
for p in "${PAGES[@]}"; do
    commandes="$(grep -o '<code data-verifier="install">[^<]*</code>' "$p" | sed 's/<[^>]*>//g')"
    while IFS= read -r commande; do
        [ -z "$commande" ] && continue
        if grep -Fq "$commande" "$guide"; then
            ok "commande présente dans le guide : $commande"
        else
            echec "commande absente du guide ($p) : $commande"
        fi
    done <<< "$commandes"
done
```

- [ ] **Étape 2 : lancer, vérifier le vert (rien n'a encore changé côté contenu)**

```sh
cd website && ./verifier.sh
```

- [ ] **Étape 3 : écrire le contenu de `utiliser.html`**

Remplacer le `<main id="top">…</main>` par le rappel collant et sept sections. **Le motif se
répète** : chaque partie propre à un système vit dans un `<div data-os-block="…">` et porte
un `<h3 class="os-block-title">` qui ne se voit que sans JavaScript.

Le rappel collant, juste après `<section class="page-head">` :

```html
    <div class="os-sticky">
      <div class="wrap">
        <span data-i18n="use.showing">Showing commands for</span>
        <span class="os-name" data-os-block="macos">macOS</span>
        <span class="os-name" data-os-block="windows">Windows</span>
      </div>
    </div>
```

Le motif complet, ici § 1 — Télécharger. **Les six autres sections le reprennent tel quel**,
en changeant l'ancre, le numéro, les titres et le contenu :

```html
    <section id="telecharger">
      <div class="wrap">
        <p class="eyebrow">1</p>
        <h2 data-i18n="use.dl.h2">Download it.</h2>

        <div data-os-block="macos">
          <h3 class="os-block-title">macOS</h3>
          <p data-i18n="use.dl.mac">Homebrew, from the project's tap. Nothing compiles.</p>
          <pre><code data-verifier="install">brew install yves848/jigger/jigger</code></pre>
        </div>

        <div data-os-block="windows">
          <h3 class="os-block-title">Windows</h3>
          <p data-i18n="use.dl.win">scoop, from the project's bucket. The release carries a prebuilt binary.</p>
          <pre><code data-verifier="install">scoop bucket add jigger https://gitlab.yg-devworks.com/yves/scoop-jigger.git</code></pre>
          <pre><code data-verifier="install">scoop install jigger</code></pre>
        </div>

        <p class="note" data-i18n="use.dl.go">Any platform, if you have Go: <span class="mono">go install gitlab.yg-devworks.com/yves/jigger@latest</span></p>
      </div>
    </section>
```

Le contenu des six sections restantes :

| Ancre | Titre EN | Bloc macOS | Bloc Windows | Illustrations |
|---|---|---|---|---|
| `#brancher` | *Wire it into your shell.* | une ligne `source` dans `~/.zshrc` ; l'ordre vis-à-vis de starship | un `Import-Module` dans `$PROFILE` ; l'ordre vis-à-vis d'oh-my-posh | — |
| `#verifier` | *Check it took.* | `jigger --version` ; `which -a jigger` — une seule ligne | `jigger --version` ; `Get-Command jigger -All` | — |
| `#utiliser` | *Now type a command.* | démos `macos-01`, `macos-02` ; trois blocs « capture à venir » | démos `windows-01`, `windows-02`, `windows-04`, `windows-05`, `windows-06` | cinq `<figure class="demo">` par système |
| `#touches` | *The keys.* | commun aux deux (hors `data-os-block`) | idem | tableau `⇥ ⏎ ↓ ↑ ^N ^P ^G ^R` |
| `#reglages` | *Settings.* | commun aux deux | idem | tableau `JIGGER_COMMANDS`, `JIGGER_LANG`, `JIGGER_LIVE`, `JIGGER_CACHE_DIR` |
| `#linux` | *And Linux?* | — | — | l'encart ci-dessous |

**Contrainte d'équilibre.** Le contrôle 8 exige autant de blocs macOS que de blocs Windows
par page. Les trois démonstrations qui n'existent que sur Windows ont donc chacune leur
pendant macOS, qui dit la vérité au lieu de laisser un trou :

```html
        <div data-os-block="macos">
          <h3 class="os-block-title">macOS</h3>
          <p class="demo-absent" data-i18n="use.regex.mac">The same <kbd>^R</kbd> toggle works on macOS — its capture isn't recorded yet. Switch to Windows above to watch it.</p>
        </div>
```

L'encart Linux, en fin de page :

```html
    <section id="linux">
      <div class="wrap">
        <aside class="encart">
          <h2 data-i18n="use.linux.h2">And Linux?</h2>
          <p data-i18n="use.linux.p">Arch and Omarchy already work — pacman and yay, same popup, same verbs. Their captures aren't recorded yet, so this page doesn't walk you through them. The documentation does.</p>
          <p><a href="https://gitlab.yg-devworks.com/yves/jigger/-/blob/main/docs/installation.md" data-i18n="use.linux.link">Read the install guide <span class="arw">→</span></a></p>
        </aside>
      </div>
    </section>
```

- [ ] **Étape 4 : traduire — une entrée par clé, dans le bloc `use.` d'`app.js`**

Une ligne par `data-i18n` posé à l'étape 3, de la forme `'use.xxx': '…',`. Le contrôle 1 dit
laquelle manque : le lancer, corriger, relancer, jusqu'au vert.

- [ ] **Étape 5 : relancer jusqu'au vert, puis regarder**

```sh
cd website && ./verifier.sh && python3 -m http.server 8080
```

Sur `http://localhost:8080/utiliser.html` : basculer macOS ↔ Windows et vérifier que les
commandes **et** les vidéos changent d'un coup, et qu'aucune section ne se vide.

- [ ] **Étape 6 : commit**

```bash
git add website/utiliser.html website/app.js website/verifier.sh
git commit -m "La page Utiliser : télécharger, brancher, utiliser — par système"
```

---

### Task 7 : La page SSH

**Fichiers :**
- Modifier : `website/ssh.html` (tout le contenu)
- Modifier : `website/app.js` (bloc FR `ssh.*`)

**Interfaces :**
- Consomme : `data-os-block` (tâche 5), `<figure class="demo">` (tâche 4).
- Produit : les ancres `#pourquoi`, `#en-action`, `#ce-qui-compte`, `#le-fichier`.

- [ ] **Étape 1 : écrire le contenu**

Cinq sections, dans cet ordre.

1. **`#pourquoi`** — un paragraphe : `ssh` n'est pas un gestionnaire de paquets, et c'est
   exactement ce qu'il démontre — le contrat de complétion ne leur est pas réservé. Lien
   vers l'ADR-0005 sur le dépôt.
2. **`#en-action`** — les deux démonstrations, une par système, motif de la tâche 4 :

```html
        <div data-os-block="macos">
          <h3 class="os-block-title">macOS</h3>
          <figure class="demo">
            <div class="demo-frame">
              <video muted loop playsinline preload="none"
                     poster="/media/macos-03-ssh.png"
                     data-src="/media/macos-03-ssh.mp4"
                     aria-labelledby="demo-mac03"></video>
            </div>
            <figcaption id="demo-mac03" data-i18n="ssh.demo.mac">Type <span class="mono">ssh</span> and a space: the servers of your <span class="mono">~/.ssh/config</span>, each with its address alongside.</figcaption>
          </figure>
        </div>
```

   Le bloc Windows est le même, avec `windows-03-ssh`, `demo-win03` et la clé
   `ssh.demo.win`.
3. **`#ce-qui-compte`** — trois points : `ssh`, `scp` et `sftp` sont **trois fournisseurs**
   et non un à trois noms ; aucun verbe, donc le catalogue vient dès l'espace ; `scp`
   insère un deux-points, et pourquoi ce n'est pas cosmétique — `scp rapport.pdf nas /tmp`
   copie vers un fichier **local** nommé « nas ».
4. **`#le-fichier`** — un `<div id="dia-ssh"></div>` vide, qui recevra le schéma à la tâche
   9, et un paragraphe décrivant ce que le fichier contient : un `Include`, six hôtes, et
   deux motifs que jigger n'affiche jamais.
5. **Un appel** vers `/utiliser.html`.

Les hôtes cités dans le texte sont ceux du fixture — `passerelle`, `nas`, `proxmox`,
`omarchy`, `windows`, `atelier` — et **jamais** d'hôtes réels : c'est la règle des captures,
elle vaut aussi pour la prose.

- [ ] **Étape 2 : traduire le bloc `ssh.` d'`app.js`**

Une ligne par clé posée à l'étape 1.

- [ ] **Étape 3 : relancer jusqu'au vert**

```sh
cd website && ./verifier.sh
```

Le contrôle 8 doit dire `blocs par système équilibrés dans ssh.html (1 de chaque)`.

- [ ] **Étape 4 : commit**

```bash
git add website/ssh.html website/app.js
git commit -m "La page SSH : ce que le sélecteur prouve, et ce qu'il complète"
```

---

### Task 8 : L'accueil refondu

**Fichiers :**
- Modifier : `website/index.html`
- Modifier : `website/app.js` (clés `home.*` nouvelles, clés `install.*` retirées)

**Interfaces :**
- Consomme : tout ce qui précède.
- Produit : rien que les tâches suivantes consomment.

- [ ] **Étape 1 : le hero**

Dans `<section class="hero">`, garder l'accroche et le sous-titre, remplacer les deux
boutons par *Download* → `/utiliser.html#telecharger` et *See it work* →
`/utiliser.html#utiliser`, et ajouter une démonstration par système
(`macos-01` / `windows-01`) dans deux `data-os-block`, au motif de la tâche 4.

- [ ] **Étape 2 : élaguer ce qui a déménagé**

La section `#install` de l'accueil devient un simple appel vers `utiliser.html` : les
commandes ne vivent plus qu'à un seul endroit, et le contrôle 3 ne les vérifie plus qu'une
fois. Supprimer du dictionnaire les clés `install.*` devenues orphelines — le contrôle 1
les nomme.

- [ ] **Étape 3 : le paragraphe SSH renvoie à sa page**

Dans la section `#popup`, la clé `popup.ssh` garde son texte et gagne un lien vers
`/ssh.html`. Mettre à jour l'entrée française correspondante.

- [ ] **Étape 4 : relancer jusqu'au vert, puis relire la page entière**

```sh
cd website && ./verifier.sh && python3 -m http.server 8080
```

Relire l'accueil dans les deux langues et dans les deux systèmes : quatre lectures.

- [ ] **Étape 5 : commit**

```bash
git add website/index.html website/app.js
git commit -m "L'accueil montre au lieu de décrire, et laisse l'installation à sa page"
```

---

### Task 9 : Les trois schémas

**Fichiers :**
- Modifier : `website/index.html` (deux SVG)
- Modifier : `website/ssh.html` (un SVG, dans `#le-fichier`)
- Modifier : `website/styles.css` (classes `.dia-*`)
- Modifier : `website/app.js` (clés des libellés)

**Interfaces :**
- Consomme : les jetons de la tâche 2.
- Produit : rien.

- [ ] **Étape 1 : restyler le schéma existant**

Les classes `.dia-*` de `styles.css` emploient encore les anciens noms de jetons — la tâche
2 les a renommées, ce qui suffit. Vérifier à l'écran que les trois canaux gardent leurs
couleurs distinctes : lecture en `var(--blue)`, exécution en `var(--peach)`, état en
`var(--green)`.

**Le modèle à suivre, et il existe déjà.** Le schéma des trois canaux vit dans
`website/index.html`, section `#fonctionnement` — un `<svg>` écrit à la main, avec ses
`<marker>` de flèches, un `<title id="dia-title">` traduit, et un `data-i18n` sur chaque
`<text>`. Les trois schémas de cette tâche en reprennent la structure exacte : mêmes
conventions de nommage de classes (`.dia-*`), même façon de traduire, même `<title>`
accessible. Le relire avant d'écrire quoi que ce soit épargne d'inventer une seconde
manière de faire la même chose.

- [ ] **Étape 2 : le schéma « le cadre se resserre »**

Trois états côte à côte — `fire`, `fireb`, `firebi` — chacun montrant la liste qui
raccourcit. L'animation passe par des classes, **jamais** par un attribut `style` :

```css
@keyframes dia-etape { 0%, 28% { opacity: 1 } 33%, 100% { opacity: .22 } }
.dia-etape-1 { animation: dia-etape 4.5s infinite; }
.dia-etape-2 { animation: dia-etape 4.5s infinite 1.5s; }
.dia-etape-3 { animation: dia-etape 4.5s infinite 3s; }
@media (prefers-reduced-motion: reduce) {
  .dia-etape-1, .dia-etape-2, .dia-etape-3 { animation: none; opacity: 1; }
}
```

Le SVG porte un `<title>` traduit qui décrit la même chose en une phrase, et un
`data-i18n` par `<text>`.

- [ ] **Étape 3 : le schéma « la façade »**

Un verbe au centre — `jg install fd` —, quatre gestionnaires autour, une flèche pleine vers
celui qui connaît le nom et trois flèches éteintes vers les autres. Mêmes règles : classes,
`data-i18n` par `<text>`, un `<title>` accessible.

- [ ] **Étape 4 : le schéma « le fichier SSH »**

Trois colonnes : à gauche `~/.ssh/config` et son `Include conf.d/*.conf` ; au centre les six
hôtes du fixture, `atelier` marqué comme venant de l'`Include` ; à droite `Host *.exemple.net`
et `Host *`, barrés, avec la mention qu'ils ne sont jamais proposés.

- [ ] **Étape 5 : relancer jusqu'au vert et vérifier le mouvement réduit**

Le contrôle 1 exige une traduction par `<text data-i18n>`. Vérifier ensuite, mouvement
réduit activé, que les trois schémas restent lisibles à l'arrêt.

- [ ] **Étape 6 : commit**

```bash
git add website/index.html website/ssh.html website/styles.css website/app.js
git commit -m "Trois schémas de plus : le cadre qui se resserre, la façade, le fichier SSH"
```

---

### Task 10 : L'aperçu, le README et la publication

**Fichiers :**
- Modifier : `website/og.png` (régénéré)
- Modifier : `website/README.md`
- Modifier : les trois pages (métadonnées `og:*` et `twitter:*`)

**Interfaces :**
- Consomme : tout ce qui précède.
- Produit : le site publiable.

- [ ] **Étape 1 : refaire l'aperçu**

```sh
cd website && "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars \
  --screenshot=og.png --window-size=1200,630 "file://$PWD/og.html"
sips -g pixelWidth -g pixelHeight og.png     # doit dire 1200 × 630
```

- [ ] **Étape 2 : les métadonnées sociales de chaque page**

Chaque page a sa propre `og:url` (`https://jigger.yg-devworks.com/`, `…/utiliser.html`,
`…/ssh.html`), son `og:title` et sa `og:description`, repris du `<title>` et de la
`<meta name="description">` posés aux tâches 1 et 2. L'image reste commune.

- [ ] **Étape 3 : réécrire `website/README.md`**

Cinq sections : prévisualiser (le serveur local, et le fait que les trois pages vivent à
plat), vérifier (les huit contrôles, un par ligne, avec ce que chacun attrape), traduire
(les blocs du dictionnaire, dans l'ordre des pages), **les médias** (d'où ils viennent, et
que le vérificateur interdit leur dérive), déployer. Remplacer le lien vers la spec du
16 août par celui de la spec du 3 septembre.

- [ ] **Étape 4 : la passe complète, réseau compris**

```sh
cd website && ./verifier.sh --reseau
```

- [ ] **Étape 5 : commit**

```bash
git add website/og.png website/README.md website/index.html website/utiliser.html website/ssh.html
git commit -m "L'aperçu, le README du site, et les métadonnées de chaque page"
```

- [ ] **Étape 6 : publier — sur demande explicite seulement**

```sh
cd website && ./deploy-proxmox.sh
```

Le déploiement touche un serveur en ligne : **ne pas le lancer sans que l'utilisateur le
demande**. Le vérificateur tourne de lui-même en tête du script et l'arrête au premier
échec.

---

## Ce que ce plan ne fait pas

Repris de la spec, pour qu'aucune tâche ne dérive :

- aucun build, aucun framework, aucune dépendance ;
- aucune nouvelle capture ;
- pas de page Linux, pas de thème clair, pas de recherche, pas de mesure d'audience ;
- pas de quatrième page pour le bloc de prompt : il reste une section de l'accueil.
