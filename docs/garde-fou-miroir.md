# Le garde-fou du miroir GitHub

Le dépôt de référence est [GitLab](https://gitlab.yg-devworks.com/yves/jigger) ; GitHub en
est un miroir poussé, pour que le code reste lisible là où le monde le cherche.

Ce miroir est tombé le 16 août 2026 et **personne ne l'a su pendant un jour**. Le jeton qui
l'authentifiait avait été révoqué ; GitLab a réessayé, échoué, réessayé, sans que rien ne
remonte. Deux versions sont sorties entre-temps, `v0.11.0` et `v0.12.0`, que GitHub n'a
jamais vues. Le défaut a été découvert par hasard, en allant vérifier tout autre chose.

`tools/miroir` existe pour que cela ne se reproduise pas silencieusement.

## Ce qu'il surveille, et pourquoi ce n'est pas le jeton

Le garde-fou compare **la tête de `main` et les tags** des deux dépôts. Rien d'autre : les
branches de travail vont et viennent, et le miroir n'a pas à en rendre compte.

Surveiller l'état du miroir côté GitLab (`update_status`, `last_error`) aurait été plus
direct — et n'aurait rattrapé que la panne d'hier. Le jeton d'aujourd'hui expirera. La
protection de secrets de GitHub refusera un autre push, comme elle l'a fait le 17 août.
Quelqu'un désactivera l'entrée de miroir en faisant le ménage. L'**écart entre les deux
dépôts**, lui, dit la panne quelle qu'en soit la cause, y compris celles qu'on n'a pas
prévues.

Les deux dépôts étant publics, ce constat ne demande **aucune authentification**.

## Les trois usages

```sh
go run ./tools/miroir              # le verdict, code de sortie 1 s'il y a un écart
go run ./tools/miroir -issue       # ouvre ou referme l'issue GitLab (GARDE_FOU_TOKEN)
go run ./tools/miroir -notifier    # une bannière macOS en cas d'écart
```

Les codes de sortie : `0` tout va bien, `1` il y a un écart, `2` le garde-fou lui-même n'a
pas pu faire son travail (API injoignable, jeton manquant). Un `2` ne dit **rien** de l'état
du miroir, et ne doit pas se lire comme un feu vert.

## L'installation, une fois pour toutes

### 1. Le jeton

Un **jeton d'accès au projet** GitLab (Settings → Access tokens), portée `api`, rôle
*Reporter* suffit pour lire, *Developer* pour écrire les issues. Il ne sert qu'à l'issue.

Le poser en variable de CI (Settings → CI/CD → Variables) :

| Clé | Valeur | Options |
|---|---|---|
| `GARDE_FOU_TOKEN` | le jeton | **Masked**, **Protected** décoché |

*Protected* doit rester décoché : une planification ne tourne pas nécessairement sur une
branche protégée, et la variable serait alors absente sans que rien ne l'explique.

### 2. La planification

Settings → CI/CD → Pipeline schedules → New schedule. Une fois par jour suffit — la panne
qu'on cherche dure des jours, pas des minutes.

| Champ | Valeur |
|---|---|
| Interval Pattern | `0 8 * * *` (tous les jours à 8 h) |
| Target branch | `main` |
| Activated | oui |

Seul le job `miroir` s'y déclenche : tous les autres sont conditionnés à `$CI_COMMIT_TAG`.

### 3. La bannière locale, en option

Pour être prévenu pendant qu'on travaille, sans attendre l'issue. Un `launchd` qui lance le
garde-fou toutes les heures — `~/Library/LaunchAgents/com.yg.jigger.miroir.plist` :

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>            <string>com.yg.jigger.miroir</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/sh</string>
    <string>-c</string>
    <string>cd ~/git/jigger &amp;&amp; go run ./tools/miroir -notifier</string>
  </array>
  <key>StartInterval</key>    <integer>3600</integer>
  <key>RunAtLoad</key>        <true/>
</dict>
</plist>
```

```sh
launchctl load ~/Library/LaunchAgents/com.yg.jigger.miroir.plist
```

Sans `-issue`, il ne touche à rien et n'a besoin d'aucun jeton.

## Ce que dit l'issue

Elle porte le libellé `garde-fou::miroir` — c'est à lui que le garde-fou reconnaît son
issue d'un passage à l'autre, plutôt qu'à un titre qu'une relecture pourrait réécrire. Elle
liste les références qui divergent, avec leur sha de chaque côté, et rappelle les trois
causes à regarder dans l'ordre.

Elle **se referme d'elle-même** au premier passage où les deux dépôts se sont rejoints. Une
panne qui revient ouvre une issue neuve plutôt que de réanimer l'ancienne : ses sha doivent
être ceux du jour, pas ceux d'il y a trois semaines.

## Remettre le miroir en route

Le garde-fou constate, il ne répare pas. Quand il crie :

1. **Settings → Repository → Mirroring repositories.** Un jeton expiré ou révoqué s'y voit
   tout de suite. Le remplacer là, dans le champ *Password* — jamais dans l'URL, et jamais
   collé dans une conversation qui finit dans `docs/historique/` (cf. A-23).
2. **La protection de secrets de GitHub.** Elle refuse un push entier si un commit porte ce
   qu'elle prend pour un secret, et le miroir n'en rapporte qu'un échec sans détail. Le
   message complet n'apparaît qu'en poussant à la main.
3. **Un push local.** Un tag que seul GitHub porte ne vient pas d'une panne mais de
   quelqu'un qui a poussé directement — c'est ce qui s'est passé le 17 août.

En dépannage, depuis un poste authentifié auprès de GitHub :

```sh
git remote add github https://github.com/Yves848/jigger.git   # une fois
git push github main --follow-tags
```
