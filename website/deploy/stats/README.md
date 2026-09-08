# Statistiques de visite (GoAccess)

Tableau de bord **côté serveur**, régénéré depuis les journaux nginx. Aucun script chez le
visiteur, aucun cookie, aucun tiers : rien de ce qui suit ne demande la moindre ligne de
JavaScript sur les pages du site.

- **Adresse** : `https://jigger.yg-devworks.com/stats/`, protégée par `auth_basic`.
- **Données** : visiteurs uniques, pages, **référents**, navigateurs, systèmes, répartition
  horaire et journalière. IP anonymisées, robots écartés à la génération.
- **Fenêtre** : toutes les rotations conservées par logrotate, soit une quinzaine de jours.

## Installation

`website/deploy-proxmox.sh` s'en charge à chaque déploiement : il envoie les trois fichiers
de ce répertoire, les installe, recharge systemd, arme le timer et déclenche une première
génération. Il n'y a rien à faire à la main **sauf une chose**, une seule fois :

```sh
printf 'yves:%s\n' "$(openssl passwd -apr1 'MOT_DE_PASSE')" > /etc/nginx/jigger-stats.htpasswd
chmod 0640 /etc/nginx/jigger-stats.htpasswd
chown root:www-data /etc/nginx/jigger-stats.htpasswd
```

Un mot de passe ne se versionne pas, et le script ne peut pas en inventer un : un mot de
passe que personne n'a choisi est un mot de passe que personne ne retrouve. Le déploiement
signale son absence à chaque passage, parce que **nginx valide sa configuration sans
vérifier que ce fichier existe**. La page demande alors une authentification (401) puis
refuse tout identifiant (403) — ce qui ressemble à un mot de passe faux, et envoie
chercher du mauvais côté.

`goaccess` doit être présent sur l'hôte web (`apt-get install -y goaccess`). Il l'est déjà,
le site cocktails s'en servant sur la même machine.

## Deux écarts avec la recette de cocktails, dont ce répertoire descend

**La fenêtre.** Cocktails ne lit que le journal courant et sa première rotation — deux
jours. C'est un choix raisonnable pour un site qui reçoit du monde ; pour jigger, deux
jours ne montreraient presque rien. On lit donc toutes les rotations : mesuré le
2026-09-08, **10 631 lignes** au total contre quelques centaines sur deux jours.

**Le tuyau.** D'où le `zcat -f` en amont, et il n'est pas cosmétique : **GoAccess 1.7 ne
sait pas ouvrir un journal compressé**. Il n'échoue même pas franchement — il rend un
rapport vide, ce qu'on lit comme « pas de visites ». Éprouvé sur l'hôte avant d'écrire le
script. `-f` laisse passer les fichiers non compressés sans y toucher, ce qui permet de
traiter le courant, la rotation `.1` et les `.gz` d'un seul geste.

## Le piège de la CSP

La route `/stats/` porte sa **propre** CSP, relâchée, dans `../nginx-jigger.conf`. Le
rapport GoAccess embarque tout son JavaScript et son CSS en ligne : sous la CSP stricte du
site, la page s'affiche **entièrement vide**, sans autre trace qu'un message dans la
console du navigateur.

Et les quatre autres en-têtes de sécurité y sont **répétés**. Ce n'est pas une redondance :
un `add_header` posé dans un `location` annule tous ceux du bloc serveur. Le même piège est
déjà documenté ailleurs dans ce fichier, où il avait fait repartir les ressources statiques
sans CSP ni `nosniff`.
