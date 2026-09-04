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
  `GITHUB_RELEASE_TOKEN` en variable masquée du projet GitLab.

**Le miroir est public.** Tout ce qui est fusionné dans `main` est visible d'Internet dès
la synchronisation suivante — irréversiblement, un dépôt supprimé pouvant déjà avoir été
cloné ou indexé.

## Conséquences

- Les skills de cycle de vie d'ai-migration-kit (`create-issue`, `implement-issue`,
  `merge-pr`, `auto-dev`) sont **GitHub-only** et ne fonctionnent pas ici — passer par
  `glab`, ou par le skill `gitlab-changelog`, qui est le chemin habituel de ce dépôt.
- Le dépôt suit une consigne de consignation : chaque commit donne une issue GitLab
  (labels `type::*` et version cible), agrégée ensuite en notes de release et en
  `CHANGELOG.md`. Voir le skill `gitlab-changelog`.
- Si le MCP GitLab ne répond plus, le skill `reauth-mcp-gitlab` couvre la reconnexion —
  ne pas se rabattre sur l'API REST sans avoir essayé.
