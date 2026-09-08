# État de la chaîne de distribution

Répond d'un coup d'œil à la seule question que personne ne pouvait trancher sans ouvrir six
onglets : **ce qui est publié est-il cohérent ?**

- **Adresse** : `https://jigger.yg-devworks.com/etat/`, derrière le même mot de passe que
  `/stats/`.
- **Contenu** : la version servie par chaque canal — release GitLab, formule Homebrew,
  bucket scoop, miroir GitHub — comparée à la dernière publiée ; plus l'état du dernier
  pipeline et la date du dernier envoi au miroir.
- **Cadence** : toutes les quinze minutes. La chaîne de distribution ne bouge pas plus vite.

## Pourquoi cette page existe

Le besoin n'est pas théorique. La formule Homebrew a déjà traîné **six versions** en
arrière sans que rien ne le signale, et trois correctifs successifs du miroir GitHub ont
échoué avant qu'on en comprenne la cause (#163). Ces écarts ne cassent rien : un
utilisateur installe simplement une version d'il y a trois semaines, et personne ne
l'apprend.

**Elle s'est justifiée à sa première exécution** en révélant que le bucket scoop servait
`0.21.0` face à une release `0.22.1`.

## Ce qui a façonné le script

**Bibliothèque standard uniquement.** L'hôte web n'a ni `curl` ni `jq` — relevé avant
d'écrire, pas découvert après. Il a python3, et c'est tout ce sur quoi on peut compter.

**Aucun jeton.** Le dépôt, le tap, le bucket et le miroir sont tous publics. La page ne
porte donc aucun secret, ce qui la rend sûre à régénérer en boucle et supprime la question
de l'expiration.

**Aucune source n'est bloquante.** `lire()` ne lève jamais : une source muette s'affiche
« indisponible » plutôt que de faire échouer toute la page. Un tableau de bord qui
disparaît quand une de ses sources tousse ne sert à rien.

**Écriture atomique.** Chaque fichier est écrit sous un nom temporaire puis renommé : un
lecteur ne tombe jamais sur une page à moitié écrite, et une collecte interrompue laisse la
précédente en place.

## La différence avec `/stats/`, qui vaut d'être connue

`/stats/` doit **relâcher la CSP** : le rapport GoAccess embarque son JavaScript et son CSS
en ligne, et n'offre pas le choix.

`/etat/` garde la **CSP stricte du site**. Le générateur écrit sa feuille de style dans un
fichier voisin (`etat.css`) au lieu de l'incorporer — quelques lignes de plus, et pas une
exception de sécurité à maintenir.

## Installation

`website/deploy-proxmox.sh` s'en charge : envoi, installation, `daemon-reload`, armement du
timer et première génération immédiate. Le seul geste manuel est le fichier de mots de
passe, partagé avec `/stats/` — voir [`../stats/README.md`](../stats/README.md).
