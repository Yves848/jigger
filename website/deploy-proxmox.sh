#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
WEB_HOST="root@192.168.50.11"
PROXY_HOST="root@192.168.50.10"
DOMAIN="jigger.yg-devworks.com"
RELEASE="$(date -u +%Y%m%d%H%M%S)"
WORK_DIR="$(mktemp -d)"
# Les sockets Unix sont limités à 104 octets sur macOS, et TMPDIR y vit sous un
# long /var/folders : on garde les sockets de multiplexage SSH sous /tmp.
CONTROL_DIR="$(mktemp -d /tmp/jigger-ssh.XXXXXX)"
ARCHIVE="$WORK_DIR/jigger-site.tar.gz"

chmod 700 "$CONTROL_DIR"
trap 'rm -rf "$WORK_DIR" "$CONTROL_DIR"' EXIT

SSH_OPTIONS=(
  -o StrictHostKeyChecking=accept-new
  -o ControlMaster=auto
  -o ControlPersist=60
  -o "ControlPath=$CONTROL_DIR/%C"
)

echo "Contrôles avant publication…"
"$SCRIPT_DIR/verifier.sh"

# --- Empreintes de contenu -------------------------------------------------
# `styles.css` et `app.js` gardent leur nom d'un déploiement à l'autre. Un
# navigateur qui les a déjà vus peut donc servir l'ANCIEN fichier sous le
# nouveau nom, sans rien demander au serveur et sans que personne s'en aperçoive
# — c'est ce qui a rendu les schémas SVG en noir après la refonte du site, chez
# un visiteur dont le cache datait d'avant. Le vhost revalide désormais, mais
# une consigne ne rattrape jamais une entrée de cache déjà posée : seule l'URL
# le peut. On publie donc les pages avec une empreinte du contenu dans l'URL,
# `/styles.css?v=1a2b3c4d`. Le fichier change, l'empreinte change, l'URL change,
# et le cache ne peut plus se tromper de version.
#
# L'empreinte est posée sur la COPIE publiée, jamais dans le dépôt : les pages
# y restent lisibles telles quelles et le site s'ouvre en local sans rien
# construire, ce qui est tout l'intérêt d'un site sans build.
SITE_DIR="$WORK_DIR/site"
mkdir -p "$SITE_DIR"

# shasum sur macOS, sha256sum sur Linux : le script tourne des deux côtés.
empreinte() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -c1-8
  else
    shasum -a 256 "$1" | cut -c1-8
  fi
}

V_CSS="$(empreinte "$SCRIPT_DIR/styles.css")"
V_JS="$(empreinte "$SCRIPT_DIR/app.js")"
echo "Empreintes : styles.css?v=$V_CSS · app.js?v=$V_JS"

# Pas de `sed -i` : il demande un argument sur BSD et pas sur GNU. On écrit la
# copie, le fichier du dépôt n'est jamais touché.
for page in index.html parcours.html utiliser.html ssh.html 404.html; do
  sed -e "s|href=\"/styles\.css\"|href=\"/styles.css?v=$V_CSS\"|" \
      -e "s|src=\"/app\.js\"|src=\"/app.js?v=$V_JS\"|" \
      "$SCRIPT_DIR/$page" > "$SITE_DIR/$page"
  # Une page qui ne cite plus la feuille attendue sortirait sans style, et
  # personne ne le verrait avant la mise en ligne.
  grep -q "styles.css?v=$V_CSS" "$SITE_DIR/$page" \
    || { echo "empreinte non posée dans $page — le lien a-t-il changé de forme ?" >&2; exit 1; }
done

cp "$SCRIPT_DIR/styles.css" "$SCRIPT_DIR/app.js" \
   "$SCRIPT_DIR/jigger-icon.svg" "$SCRIPT_DIR/og.png" \
   "$SCRIPT_DIR/robots.txt" "$SCRIPT_DIR/sitemap.xml" "$SITE_DIR/"
cp -R "$SCRIPT_DIR/media" "$SITE_DIR/media"

# og.html est le gabarit qui a produit og.png : il n'a rien à faire en ligne.
tar -czf "$ARCHIVE" -C "$SITE_DIR" \
  index.html parcours.html utiliser.html ssh.html 404.html \
  robots.txt sitemap.xml styles.css app.js jigger-icon.svg og.png media

echo "Publication des fichiers sur ${WEB_HOST}…"
scp "${SSH_OPTIONS[@]}" \
  "$ARCHIVE" \
  "$SCRIPT_DIR/deploy/nginx-jigger.conf" \
  "${WEB_HOST}:/tmp/"

ssh "${SSH_OPTIONS[@]}" "${WEB_HOST}" bash -s -- "$RELEASE" <<'REMOTE_WEB'
set -euo pipefail
release="$1"
release_dir="/var/www/jigger/releases/$release"

install -d -m 0755 "$release_dir"
tar -xzf /tmp/jigger-site.tar.gz -C "$release_dir"
chown -R root:root "$release_dir"
find "$release_dir" -type d -exec chmod 0755 {} +
find "$release_dir" -type f -exec chmod 0644 {} +

install -d -m 0755 /var/www/jigger
ln -sfn "$release_dir" /var/www/jigger/current

nginx_config=/etc/nginx/sites-available/jigger
nginx_backup="${nginx_config}.before-deploy"
had_nginx_config=false
if [ -e "$nginx_config" ]; then
    cp -a "$nginx_config" "$nginx_backup"
    had_nginx_config=true
fi

install -m 0644 /tmp/nginx-jigger.conf "$nginx_config"
ln -sfn "$nginx_config" /etc/nginx/sites-enabled/jigger

if ! nginx -t; then
    if [ "$had_nginx_config" = true ]; then
        cp -a "$nginx_backup" "$nginx_config"
    else
        unlink /etc/nginx/sites-enabled/jigger 2>/dev/null || true
        rm -f "$nginx_config"
    fi
    echo "Configuration Nginx invalide : restauration effectuée" >&2
    exit 1
fi

systemctl reload nginx
rm -f /tmp/jigger-site.tar.gz /tmp/nginx-jigger.conf
REMOTE_WEB

echo "Ajout de la route HTTPS sur ${PROXY_HOST}…"
scp "${SSH_OPTIONS[@]}" \
  "$SCRIPT_DIR/deploy/caddy-jigger.conf" \
  "${PROXY_HOST}:/tmp/"

ssh "${SSH_OPTIONS[@]}" "${PROXY_HOST}" bash -s -- "${DOMAIN}" <<'REMOTE_PROXY'
set -euo pipefail
domain="$1"
caddyfile=/etc/caddy/Caddyfile
backup="${caddyfile}.before-jigger.$(date -u +%Y%m%d%H%M%S)"
cp -a "$caddyfile" "$backup"

if ! grep -Fq "$domain {" "$caddyfile"; then
    printf '\n' >> "$caddyfile"
    cat /tmp/caddy-jigger.conf >> "$caddyfile"
fi

if ! caddy validate --config "$caddyfile"; then
    cp -a "$backup" "$caddyfile"
    echo "Configuration Caddy invalide : restauration de $backup" >&2
    exit 1
fi

if ! systemctl reload caddy; then
    cp -a "$backup" "$caddyfile"
    systemctl reload caddy
    echo "Échec du rechargement de Caddy : restauration de $backup" >&2
    exit 1
fi
rm -f /tmp/caddy-jigger.conf
REMOTE_PROXY

echo "Vérification de https://${DOMAIN}/…"
curl --fail --silent --show-error --location \
  --retry 12 --retry-delay 5 --retry-all-errors \
  --output /dev/null \
  "https://${DOMAIN}/"

echo "jigger est publié sur https://${DOMAIN}/"
