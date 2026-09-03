#!/usr/bin/env bash
# Contrôles de la page avant déploiement. `--reseau` ajoute la vérification des
# liens externes (sinon la passe est hors-ligne, donc déterministe).
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
RACINE="$(cd .. && pwd)"
RESEAU=0
[ "${1:-}" = "--reseau" ] && RESEAU=1

# Les trois pages du site. Tout contrôle qui lit « la page » les lit toutes.
PAGES=(index.html utiliser.html ssh.html)

for p in "${PAGES[@]}"; do
    [ -f "$p" ] || { printf 'ÉCHEC  — page absente : %s\n' "$p" >&2; exit 1; }
done

echecs=0
ok()    { printf 'ok     — %s\n' "$1"; }
echec() { printf 'ÉCHEC  — %s\n' "$1" >&2; echecs=$((echecs + 1)); }

# --- 1. Parité des langues ------------------------------------------------
# Les clés employées dans le HTML, et celles définies dans le dictionnaire FR.
cles_html="$(cat "${PAGES[@]}" | grep -o 'data-i18n="[^"]*"' | cut -d'"' -f2 | sort -u)"
cles_fr="$(sed -n '/--- FR ---/,/--- \/FR ---/p' app.js \
           | grep -oE "^[[:space:]]+'[^']+':" \
           | sed -E "s/^[[:space:]]+'([^']+)':/\1/" | sort -u)"

manquantes="$(comm -23 <(printf '%s\n' "$cles_html") <(printf '%s\n' "$cles_fr"))"
orphelines="$(comm -13 <(printf '%s\n' "$cles_html") <(printf '%s\n' "$cles_fr"))"

if [ -n "$manquantes" ]; then
    echec "clés sans traduction française : $(echo "$manquantes" | tr '\n' ' ')"
else
    ok "toutes les clés du HTML sont traduites ($(printf '%s\n' "$cles_html" | wc -l | tr -d ' '))"
fi

if [ -n "$orphelines" ]; then
    echec "clés françaises inutilisées : $(echo "$orphelines" | tr '\n' ' ')"
else
    ok "aucune entrée orpheline dans le dictionnaire"
fi

# --- 2. Liens -------------------------------------------------------------
# Internes : chaque ancre doit désigner un id existant.
for ancre in $(grep -o 'href="#[^"]*"' index.html | cut -d'"' -f2 | cut -c2- | sort -u); do
    [ "$ancre" = "top" ] && continue
    if grep -q "id=\"$ancre\"" index.html; then
        ok "ancre #$ancre"
    else
        echec "ancre #$ancre ne désigne aucun id"
    fi
done

# Externes : seulement sur demande, car ça dépend du réseau.
if [ "$RESEAU" = 1 ]; then
    for url in $(grep -o 'href="https\?://[^"]*"' index.html | cut -d'"' -f2 | sort -u); do
        code="$(curl -sIL --max-time 10 -o /dev/null -w '%{http_code}' "$url" || echo 000)"
        case "$code" in
            2*|3*) ok "$url → $code" ;;
            *)     echec "$url → $code" ;;
        esac
    done
fi

# --- 3. Commandes d'installation ------------------------------------------
# Chaque commande affichée doit exister mot pour mot dans le guide : si le guide
# change et pas la page, la page ment.
guide="$RACINE/docs/getting-started.md"
commandes="$(grep -o '<code data-verifier="install">[^<]*</code>' index.html \
             | sed 's/<[^>]*>//g')"

# Here-string, pas un tube : un `while` en bout de tube tourne dans un sous-shell,
# où l'incrément de $echecs serait perdu et où le premier échec masquerait les suivants.
while IFS= read -r commande; do
    [ -z "$commande" ] && continue
    if grep -Fq "$commande" "$guide"; then
        ok "commande présente dans le guide : $commande"
    else
        echec "commande absente du guide : $commande"
    fi
done <<< "$commandes"

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

if [ "$echecs" -gt 0 ]; then
    printf '\n%d contrôle(s) en échec.\n' "$echecs" >&2
    exit 1
fi
printf '\nTout est vert.\n'
