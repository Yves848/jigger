#!/usr/bin/env python3
"""État de la chaîne de distribution de jigger.

Répond d'un coup d'œil à la seule question que personne ne peut trancher aujourd'hui sans
ouvrir six onglets : **ce qui est publié est-il cohérent ?**

Le besoin n'est pas théorique. La formule Homebrew a déjà traîné SIX versions en arrière
sans que rien ne le signale, et trois correctifs successifs du miroir GitHub ont échoué
avant qu'on comprenne pourquoi (jigger#163). Ces écarts ne se voient pas : rien ne casse,
un utilisateur installe simplement une version d'il y a trois semaines.

Deux contraintes ont façonné ce fichier, toutes deux relevées sur l'hôte web :

  * **Bibliothèque standard uniquement.** La machine n'a ni `curl` ni `jq`. Elle a
    python3, et c'est tout ce sur quoi on peut compter.
  * **Aucun jeton.** Le dépôt, le tap, le bucket et le miroir sont tous publics : la page
    n'a donc aucun secret à porter, et c'est ce qui la rend sûre à régénérer en boucle.

La page produite est statique. Rien n'est calculé chez le visiteur — pas de script, pas de
requête, pas de CSP à relâcher, contrairement au rapport GoAccess voisin.
"""

import html
import json
import os
import urllib.error
import urllib.request
from datetime import datetime, timezone

SORTIE = os.environ.get("JIGGER_ETAT_DIR", "/var/www/jigger-etat")
GITLAB = "https://gitlab.yg-devworks.com/api/v4"
PROJET = 25
DELAI = 15


def lire(url, brut=False):
    """Rend le corps de l'URL, ou None. Ne lève jamais : une source absente est une
    information à afficher, pas une raison de ne rien afficher du tout."""
    try:
        r = urllib.request.Request(url, headers={"User-Agent": "jigger-etat"})
        corps = urllib.request.urlopen(r, timeout=DELAI).read().decode("utf-8", "replace")
        return corps if brut else json.loads(corps)
    except (urllib.error.URLError, OSError, ValueError, TimeoutError):
        return None


def version_de_la_formule(texte):
    """La version vit dans l'URL de l'archive, jamais dans un champ à elle."""
    if not texte:
        return None
    for ligne in texte.splitlines():
        if ligne.strip().startswith("url "):
            for morceau in ligne.replace('"', " ").split("/"):
                if morceau.startswith("v") and morceau[1:2].isdigit():
                    return morceau[1:]
    return None


def collecte():
    releases = lire(f"{GITLAB}/projects/{PROJET}/releases?per_page=1") or []
    derniere = releases[0] if releases else None
    attendue = (derniere or {}).get("tag_name", "").lstrip("v") or None

    formule = version_de_la_formule(
        lire("https://gitlab.yg-devworks.com/yves/homebrew-cocktails/-/raw/main/Formula/jigger.rb", brut=True))

    scoop_json = lire("https://gitlab.yg-devworks.com/yves/scoop-jigger/-/raw/main/bucket/jigger.json", brut=True)
    try:
        scoop = json.loads(scoop_json).get("version") if scoop_json else None
    except ValueError:
        scoop = None

    miroir = lire("https://api.github.com/repos/Yves848/jigger")
    m_rel = lire("https://api.github.com/repos/Yves848/jigger/releases/latest")
    pipelines = lire(f"{GITLAB}/projects/{PROJET}/pipelines?per_page=1") or []

    return {
        "attendue": attendue,
        "release": derniere,
        "canaux": [
            ("Release GitLab", attendue, attendue, (derniere or {}).get("_links", {}).get("self")),
            ("Formule Homebrew", formule, attendue,
             "https://gitlab.yg-devworks.com/yves/homebrew-cocktails/-/blob/main/Formula/jigger.rb"),
            ("Bucket scoop", scoop, attendue,
             "https://gitlab.yg-devworks.com/yves/scoop-jigger/-/blob/main/bucket/jigger.json"),
            ("Miroir GitHub", (m_rel or {}).get("tag_name", "").lstrip("v") or None, attendue,
             "https://github.com/Yves848/jigger/releases"),
        ],
        "pipeline": pipelines[0] if pipelines else None,
        "miroir_pousse": (miroir or {}).get("pushed_at"),
    }


def verdict(vue, attendue):
    if vue is None:
        return "indisponible", "La source n'a pas répondu"
    if attendue is None:
        return "indisponible", "Aucune version de référence"
    if vue == attendue:
        return "a-jour", "À jour"
    return "en-retard", f"En retard sur {attendue}"


CSS = """:root { color-scheme: dark; }
* { box-sizing: border-box; }
body { margin: 0; padding: 2.5rem 1.5rem; background: #11111b; color: #cdd6f4;
       font: 15px/1.6 ui-monospace, "SF Mono", Menlo, monospace; }
main { max-width: 54rem; margin: 0 auto; }
h1 { font-size: 1.3rem; margin: 0 0 .3rem; color: #94e2d5; }
p.sous { margin: 0 0 2rem; color: #7f849c; }
table { width: 100%; border-collapse: collapse; margin-bottom: 2rem; }
th, td { text-align: left; padding: .65rem .5rem; border-bottom: 1px solid #313244; }
th { color: #7f849c; font-weight: normal; text-transform: uppercase; font-size: .75rem;
     letter-spacing: .08em; }
a { color: #89b4fa; }
.a-jour { color: #a6e3a1; }
.en-retard { color: #f38ba8; font-weight: bold; }
.indisponible { color: #f9e2af; }
.version { color: #cdd6f4; }
footer { color: #585b70; font-size: .8rem; border-top: 1px solid #313244; padding-top: 1rem; }
"""


def rendre(d):
    e = html.escape
    lignes = []
    for nom, vue, attendue, lien in d["canaux"]:
        classe, texte = verdict(vue, attendue)
        cible = f'<a href="{e(lien)}">{e(nom)}</a>' if lien else e(nom)
        lignes.append(f"      <tr><td>{cible}</td>"
                      f'<td class="version">{e(vue or "—")}</td>'
                      f'<td class="{classe}">{e(texte)}</td></tr>')

    p = d["pipeline"] or {}
    etat_p = p.get("status", "inconnu")
    classe_p = "a-jour" if etat_p == "success" else ("en-retard" if etat_p == "failed" else "indisponible")
    rel = d["release"] or {}

    return f"""<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>jigger — état de la distribution</title>
<link rel="stylesheet" href="etat.css">
</head>
<body>
<main>
  <h1>État de la chaîne de distribution</h1>
  <p class="sous">Ce qui est publié est-il cohérent&nbsp;? Une ligne rouge signale un canal
  qui sert une version plus ancienne que la dernière publiée.</p>

  <table>
    <thead><tr><th>Canal</th><th>Version servie</th><th>Verdict</th></tr></thead>
    <tbody>
{chr(10).join(lignes)}
    </tbody>
  </table>

  <table>
    <thead><tr><th>Repère</th><th>Valeur</th></tr></thead>
    <tbody>
      <tr><td>Dernière release</td><td class="version">{e(rel.get("name") or "—")}
          &nbsp;<span class="sous">{e((rel.get("released_at") or "")[:16].replace("T", " "))}</span></td></tr>
      <tr><td>Dernier pipeline</td><td class="{classe_p}">{e(etat_p)}
          &nbsp;<span class="version">{e(p.get("ref") or "")}</span></td></tr>
      <tr><td>Dernier envoi au miroir</td>
          <td class="version">{e((d["miroir_pousse"] or "—")[:16].replace("T", " "))}</td></tr>
    </tbody>
  </table>

  <footer>Régénéré le {e(datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M"))} UTC.
  Données publiques, aucun jeton. Page statique&nbsp;: rien n'est calculé chez vous.</footer>
</main>
</body>
</html>
"""


def main():
    os.makedirs(SORTIE, exist_ok=True)
    d = collecte()
    # Écriture par fichier temporaire puis renommage : un lecteur ne tombe jamais sur une
    # page à moitié écrite, et une collecte interrompue laisse la précédente en place.
    for nom, contenu in (("etat.css", CSS), ("index.html", rendre(d))):
        tmp = os.path.join(SORTIE, nom + ".tmp")
        with open(tmp, "w", encoding="utf-8") as f:
            f.write(contenu)
        os.replace(tmp, os.path.join(SORTIE, nom))


if __name__ == "__main__":
    main()
