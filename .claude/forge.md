# Forge

- **Status:** décidée
- **Server:** gitlab.yg-devworks.com (GitLab auto-hébergé, perso)
- **Project:** `yves/jigger` — id 25, branche par défaut `main`
- **CLI:** `glab` — poser `GITLAB_HOST=gitlab.yg-devworks.com` pour toute commande lancée
  hors de ce dépôt (`glab repo create` vise gitlab.com par défaut, quel que soit l'hôte où
  l'on est connecté)
- **Review unit:** Merge Request (**MR**)
- **Auth check:** `glab auth status --hostname gitlab.yg-devworks.com` — le `--hostname`
  est indispensable : sans lui, une entrée `gitlab.com` résiduelle sans jeton fait sortir
  la commande en erreur alors que cet hôte-ci est parfaitement authentifié
- **Detected from:** `git remote get-url origin` →
  `https://gitlab.yg-devworks.com/yves/jigger.git`

## Ne pas se laisser tromper par `.github/`

Ce dépôt porte **à la fois** `.gitlab-ci.yml` et `.github/workflows/`, ce qui ressemble à
un signal contradictoire. Ce n'en est pas un : `origin` tranche, et GitLab est le dépôt de
référence.

`github.com/Yves848/jigger` en est un **miroir poussé**, pour que le code reste lisible là
où le monde le cherche. Conséquences pratiques :

- **Ne jamais pousser sur GitHub.** Un commit poussé là serait écrasé à la synchronisation
  suivante.
- Le seul workflow GitHub (`.github/workflows/pr-fermeture.yml`) referme les pull requests
  ouvertes sur le miroir en expliquant où aller — GitHub ne permet pas de les désactiver.
  Il vit dans le dépôt GitLab et voyage avec le miroir.
- Le miroir est tombé une fois sans que personne ne le sache pendant un jour, deux versions
  étant sorties entre-temps. `tools/miroir` et
  [`docs/garde-fou-miroir.md`](../docs/garde-fou-miroir.md) existent pour ça.
- Les binaires de release sont publiés **sur les deux** : la CI GitLab pousse aussi les
  archives vers la release GitHub (`tools/publier-github.sh`, job `github:`), qui demande
  `GITHUB_RELEASE_TOKEN` en variable masquée — et **non protégée** — du projet GitLab.

**Le miroir est public.** Tout ce qui est fusionné dans `main` est visible d'Internet dès
la synchronisation suivante — irréversiblement, un dépôt supprimé pouvant déjà avoir été
cloné ou indexé.

## Publier une release — cinq pièges vérifiés

**Ne pas créer la release à la main.** Le job `release:` du stage `publier` s'en charge au
tag : il teste si elle existe, la crée sinon, tire ses notes du `CHANGELOG.md` et la nomme
`jigger <version>` (sans le `v`). La créer soi-même avant ne casse rien — le job bascule
sur sa branche « release existante » et écrase titre et description — mais c'est un geste
inutile, et le titre posé à la main est perdu. Le skill `gitlab-changelog` prescrit
l'étape 4 « créer la release » de façon générique ; **ici, elle est automatisée**.

**`GITHUB_RELEASE_TOKEN` doit exister *et être valide* avant de taguer.** Le job
`github:` publie les archives sur la release du miroir, et sans jeton il échoue —
bruyamment, ce qui est voulu : « sans jeton on ne peut rien publier : on s'arrête plutôt
que de faire semblant ». Le rattrapage est prévu, `tools/publier-github.sh <tag> [dist/]`
servant la CI et la main ; sans dossier, il va chercher les archives dans le registre
générique de GitLab.

Elle **existe depuis le 4 septembre 2026** et porte un **PAT dédié à portée fine**
(`github_pat_…`) nommé `jigger`. Sa configuration exacte importe, et deux réglages y sont
faciles à manquer : *Repository access* doit être sur **`Only select repositories`** →
`Yves848/jigger`, **pas** sur `Public repositories` qui est le défaut et ne donne qu'une
lecture ; et la permission **`Contents: Read and write`** doit être ajoutée dans le bloc
*Repository* — celui qui n'apparaît qu'une fois un dépôt sélectionné. `Metadata: Read-only`
s'ajoute alors tout seul, c'est normal. Rien d'autre : ni *Actions*, ni *Workflows*, ni
aucune permission de compte. GitHub ne renvoie **aucun en-tête `github-authentication-token-expiration`**
dessus : il est sans date de fin, et ne lâchera donc pas la CI de lui-même. Le refaire, le
cas échéant, passe par <https://github.com/settings/personal-access-tokens/new> ; l'API ne
sait pas fabriquer un PAT, seule l'interface web le fait.

Un premier essai y avait posé le jeton OAuth du `gh` local (`gho_…`). Il a été remplacé le
jour même : ce jeton-là **tourne** dès que `gh` se réauthentifie sur la machine, ce qui
rendrait la variable silencieusement invalide, découverte au tag suivant. Ne pas y revenir
par commodité.

Vérifier avant le tag — et **seule une écriture réelle prouve quelque chose**. Deux
contrôles paraissent suffisants et ne le sont pas :

- la **présence** de la variable : une valeur tronquée de 13 caractères y a séjourné, en
  répondant `401` à GitHub ;
- un **`200` sur `/user`**, ou même la lecture réussie de la release : un PAT à portée
  fine *sans aucun droit* passe ces deux tests, parce qu'un tel jeton peut toujours lire
  les dépôts publics — et le miroir en est un. Le bloc `permissions` que renvoie
  `GET /repos/…` ne vaut rien non plus : il décrit **le rôle du compte** sur le dépôt, pas
  ce que le jeton a le droit de faire, et affiche donc `push: true` pour un jeton en
  lecture seule.

```bash
# glab sort en 401 sur cet endpoint : c'est le jeton du trousseau qu'il faut.
TOK=$(printf "protocol=https\nhost=gitlab.yg-devworks.com\n\n" | git credential fill | sed -n 's/^password=//p')
GH=$(curl -s -H "PRIVATE-TOKEN: $TOK" \
     https://gitlab.yg-devworks.com/api/v4/projects/25/variables/GITHUB_RELEASE_TOKEN \
     | python3 -c "import sys,json;print(json.load(sys.stdin)['value'])")

# Sonde d'écriture idempotente : on repose sur la dernière release le nom qu'elle porte déjà.
read -r ID NOM < <(curl -s -H "Authorization: Bearer $GH" \
     https://api.github.com/repos/Yves848/jigger/releases/latest \
     | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['id'],d['name'])")
curl -s -o /dev/null -w '%{http_code}\n' --request PATCH \
     --header "Authorization: Bearer $GH" --header "Accept: application/vnd.github+json" \
     --data "{\"name\":\"$NOM\"}" \
     "https://api.github.com/repos/Yves848/jigger/releases/$ID"
```

Attendu : `200`. Un `403` — message « Resource not accessible by personal access token »,
en-tête `x-accepted-github-permissions: contents=write` — veut dire que le jeton lit mais
n'écrit pas, et que le prochain tag repartira avec une pipeline rouge.

**La variable doit être masquée mais NON protégée.** C'est contre-intuitif — un secret,
on le protège — et c'est faux ici : le projet n'a **aucun tag protégé**
(`GET projects/25/protected_tags` rend une liste vide) et le job `github:` ne tourne que
sur tag (`rules: if: $CI_COMMIT_TAG`). Une variable protégée ne lui serait donc *jamais*
exposée, et l'échec serait exactement celui d'un jeton absent — de quoi accuser le jeton
pendant un moment.

**Le miroir est plus lent que la CI.** Le job `github:` part dans la seconde qui suit le
tag ; la réplication GitLab → GitHub, elle, est **asynchrone** et s'exécute quand elle
s'exécute. GitHub refuse alors en `422` de créer une release sur un tag qu'il ne connaît
pas encore — message trompeur, qui ressemble à un problème de droits alors que c'est une
course. La `v0.17.0` l'a gagnée, la `v0.17.1` l'a perdue six heures plus tard : la chaîne
n'était pas fiable, elle avait eu de la chance.

`tools/publier-github.sh` **attend le tag** avant de publier — une minute au plus, puis
échec explicite (#147).

**Attendre ne suffisait pas** (#154). Le miroir **ne repart pas tout seul** quand une
poussée arrive juste après son passage : mesuré sur la v0.18.0, le tag est arrivé *deux
secondes* après une synchro réussie, et le miroir n'avait toujours pas rejoué vingt-cinq
minutes plus tard, dans un état `finished` et sans erreur.

Le script **réveille donc le miroir** avant de l'attendre, et il cherche son jeton dans cet
ordre — aucun n'est requis :

`GITLAB_API_TOKEN` est **posée**, masquée et non protégée, et porte un **jeton d'accès de
projet** — `jigger-ci-miroir`, portée `api`, rôle Maintainer, lié à ce seul projet et
révocable seul. Pas le PAT personnel : réveiller un miroir ne justifie pas un jeton qui porte
les droits sur tous les dépôts.

**⚠️ Il expire le 31 décembre 2026.** Ce jour-là, le réveil cessera sans bruit et la chaîne
redeviendra dépendante de la chance. Le renouveler se fait dans *Settings → Access Tokens* du
projet, ou par `POST /projects/25/access_tokens`.

`CI_JOB_TOKEN` **ne convient pas**, et c'est mesuré, pas supposé : essayé en vraie grandeur
sur la v0.19.0, l'API a répondu **401**. Un jeton de job n'a aucun droit sur les réglages du
projet, dont relèvent les miroirs. Le script ne le réessaie plus.

**La fenêtre est de cinq minutes, et `204` ne veut pas dire que le miroir a tourné.** C'est
le piège qui a fait échouer trois correctifs successifs : GitLab accepte la demande de
synchronisation et l'ignore si elle tombe dans les cinq minutes suivant le passage précédent.
Mesuré deux fois le 5 septembre — un réveil accepté à 4 min 53 s n'a jamais été exécuté, la
même demande à 7 min 01 s est passée aussitôt, et une insistance a fait démarrer le miroir à
5 min 00 pile.

Le script observe donc `last_update_started_at`, la seule valeur qui dise que le miroir a
**réellement démarré**, et insiste jusqu'à ce qu'elle avance — jusqu'à huit minutes, en
sortant dès que c'est le cas. Il commence par vérifier si le tag est déjà là : une fois sur
deux le miroir a répliqué de lui-même, et le chemin rapide coûte alors moins d'une seconde.

Sans la variable, le script se contente d'attendre, et la chaîne dépend de l'ordonnancement
du miroir. Sur cinq releases, **trois l'ont perdue** — v0.17.1, v0.18.0 et v0.20.0 — et deux
l'ont gagnée.

Sans jeton qui marche, le script se contente d'attendre — ni mieux ni pire qu'avant. Le
journal du job dit lequel a servi : `→ miroir réveillé (JOB-TOKEN)`, ou
`· JOB-TOKEN refusé par l'API du miroir : HTTP 401`.

Le rattrapage à la main reste celui-ci :

```bash
TOK=$(printf "protocol=https\nhost=gitlab.yg-devworks.com\n\n" | git credential fill | sed -n 's/^password=//p')
curl -X POST -H "PRIVATE-TOKEN: $TOK" \
     https://gitlab.yg-devworks.com/api/v4/projects/25/remote_mirrors/2/sync
```

puis relancer le job `github:`. Ce rattrapage a fonctionné deux fois, sur la v0.17.1 et sur
la v0.18.0.

L'état du miroir se lit sur `GET projects/25/remote_mirrors` : `last_successful_update_at`
dit tout, et un `update_status: finished` sans erreur ne signifie pas que la dernière poussée
est passée, seulement que la dernière *exécution* s'est bien terminée. Comparer cet
horodatage à celui du tag (`git log -1 --format=%cI <tag>`) tranche en une commande.

**Le tag n'est pas le seul geste.** `main.go` porte `var version`, et un test
(`TestLesBannieresSuiventLaVersion`) exige que les bannières « jigger X.Y.Z » de six
fichiers le suivent — les deux READMEs, les deux guides, les deux pages du site. Il nomme
chaque fichier en retard, donc le laisser parler plutôt que les chercher. `docs/installation.md`
et sa version française annoncent aussi un numéro concret sans être gardés.

## Conséquences

- Les skills de cycle de vie d'ai-migration-kit (`create-issue`, `implement-issue`,
  `merge-pr`, `auto-dev`) sont **GitHub-only** et ne fonctionnent pas ici — passer par
  `glab`, ou par le skill `gitlab-changelog`, qui est le chemin habituel de ce dépôt.
- Le dépôt suit une consigne de consignation : chaque commit donne une issue GitLab
  (labels `type::*` et version cible), agrégée ensuite en notes de release et en
  `CHANGELOG.md`. Voir le skill `gitlab-changelog`.
- Si le MCP GitLab ne répond plus, le skill `reauth-mcp-gitlab` couvre la reconnexion —
  ne pas se rabattre sur l'API REST sans avoir essayé.
- **`main` est protégée avec `allow_force_push: false`.** Un `push --force-with-lease`,
  même une minute après le commit fautif, est refusé par le hook `pre-receive` du serveur.
  Un message de commit raté se rectifie donc par un commit de plus, pas par un `--amend` :
  inutile de proposer la réécriture, elle ne passera pas.
