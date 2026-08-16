# Améliorations et modifications à venir

Le tableau des évolutions proposées, tenu à la façon d'un Kanban : on y prend ce qu'on
veut, quand on veut, sans suivre l'ordre d'arrivée.

## Comment il se tient

- **Un numéro** à la création — `A-1`, `A-2`… —, attribué une fois pour toutes et **jamais
  réutilisé**, même après réalisation ou abandon. C'est lui qu'on cite en conversation et
  dans les messages de commit.
- **Une priorité**, `à déterminer` par défaut. Elle ne se décide pas au moment de la
  proposition : on note d'abord, on arbitre ensuite.
- **Un état**, matérialisé par la section où vit l'entrée : `À faire`, `En cours`, `Fait`.
  Une entrée réalisée **garde sa ligne** et descend dans `Fait` avec son commit — le
  tableau raconte alors aussi ce qui a été construit.
- **Une provenance**, quand elle éclaire : une réserve publiée, un constat de relecture, une
  idée en passant. Dans six mois, elle dira si l'entrée pèse encore.

Le préfixe `A-` évite toute confusion avec les numéros d'issues GitLab (`#39`), qui
consignent ce qui a **déjà** changé et alimentent les notes de release. Ce document-ci
porte l'inverse : ce qui n'est pas encore fait.

---

## À faire

### A-1 — L'alias `jg` dans le greffon PowerShell

**Priorité :** à déterminer · **Provenance :** réserve publiée avec la v0.8.0, confirmée en
relecture de l'i18n

Seul le greffon zsh arme la façade. Sous PowerShell, `JIGGER_COMMANDS` vaut `winget,scoop`
et il n'y a pas d'alias : la façade s'y atteint en écrivant `jigger <verbe>` en toutes
lettres, et le popup ne la suit pas. Le point délicat est connu : l'alias et la liste des
commandes qui arment le popup sont **deux mécanismes distincts**, et il faut les deux — le
widget lit ce que l'utilisateur a tapé, pas l'expansion de l'alias.

### A-2 — Prouver `cleanup *` et `bucket rm` contre scoop

**Priorité :** à déterminer · **Provenance :** réserve publiée avec la v0.8.0

Les captures du 16 août ont vérifié la table des verbes scoop sur `scoop help` et
`scoop update --help` — mais deux liaisons restent d'usage courant et non prouvées : le `*`
de `cleanup *` et le nom du sous-verbe de suppression de bucket. `scoop cleanup --help` et
`scoop bucket --help` trancheraient, et le script de capture peut les prendre.

### A-3 — Exercer le garde PowerShell 5.1

**Priorité :** à déterminer · **Provenance :** relecture de la tâche 7 de l'i18n

`$PSVersionTable.PSVersion.Major -lt 6 -or $IsWindows` n'a jamais tourné sur un vrai
Windows PowerShell 5.1 — seulement sur pwsh 7. Le même garde existe à deux endroits du
module ; le risque est mutualisé, mais il n'est pas nul.

### A-4 — Les descriptions de touches PSReadLine ne sont pas traduites

**Priorité :** à déterminer · **Provenance :** relecture finale de l'i18n (constat parqué)

Les trois `-Description` de `Register-JiggerKey` sont passées de « français en dur » à
« anglais en dur » plutôt que par `Get-JiggerText`, et le commentaire du code affirme le
contraire. Visible seulement par `Get-PSReadLineKeyHandler` ou `Ctrl+Alt+?`.

### A-5 — Le banc de rendu ne tient qu'à cette machine

**Priorité :** à déterminer · **Provenance :** relecture finale de l'i18n

`tests/golden/render-fr.txt` embarque le numéro de version et les vrais noms du cache
Homebrew local. Il est sorti de `make test-all` pour cette raison, et reste lançable à la
main. Le rendre portable demanderait de neutraliser la bannière **et** de figer un catalogue
d'essai.

### A-6 — Le site de présentation

**Priorité :** à déterminer · **Provenance :** demande du 15 août

Le contenu est enfin stable et en anglais. Reste la question tranchée à moitié : le modèle
**(A)** — GitHub en façade publique, GitLab en source de vérité — a été retenu, mais rien
n'est fait. Précédent utile : `cocktails-website`, une page statique déployée sur le
Proxmox maison, ce qui rouvre la question de l'hébergement pour un lien destiné à circuler.

### A-7 — La stratégie de diffusion

**Priorité :** à déterminer · **Provenance :** demande du 15 août

Utilisateurs, contributeurs et vitrine, dans cet ordre. Dépend de A-6 pour la page
d'atterrissage, et du miroir public pour la mécanique de contribution.

### A-8 — La fiche du projet GitLab

**Priorité :** à déterminer · **Provenance :** plan d'i18n, étape volontairement non
déléguée

La description annonce encore « Assistance Homebrew pour le terminal », d'avant winget,
scoop et la façade ; les topics sont vides. C'est la première ligne que voit un visiteur.

### A-9 — Publier la v0.9.0

**Priorité :** à déterminer

`main.go` est encore en `0.8.0` et le CHANGELOG a une section `[Unreleased]` fournie :
l'internationalisation, l'installation Windows, les analyseurs scoop. La chaîne est connue
— consignation, CHANGELOG, tag, release, formule du tap, `brew upgrade`.

### A-10 — Mettre en page et paginer les sorties

**Priorité :** à déterminer · **Provenance :** demande du 16 août

`jigger list` — et les trois autres verbes tabulaires, `outdated`, `search`, `source` —
rendent aujourd'hui un tableau aligné qui défile d'un bloc. Les rendre **navigables** :
pagination, couleur, filtre au fil de la frappe, et un filtre qui accepte une **expression
rationnelle** plutôt qu'une simple sous-chaîne.

Trois choses à ne pas perdre de vue le jour où on s'y met :

- **Rien ne doit casser hors terminal.** `jigger list | grep`, un script, une CI : la
  pagination ne s'arme que si la sortie est un terminal — comme le fait `git`. Et `--json`
  n'est jamais paginé, c'est un contrat machine.
- **Le sélecteur existe déjà.** `internal/ui/picker.go` sait afficher une liste en plein
  écran, la filtrer et la parcourir ; il sert déjà à `jigger pick` et à la désambiguïsation
  de la façade. Le pager devrait le réemployer plutôt qu'ouvrir un deuxième dispositif
  d'affichage à maintenir — quitte à lui ajouter un mode « lecture seule ».
- **Le filtre passe de la sous-chaîne à la regex.** Changement de comportement pour qui
  utilise déjà `jigger pick` : une frappe comme `c++` cesse d'être un texte pour devenir un
  motif fautif. À trancher — regex par défaut avec repli en sous-chaîne quand le motif ne
  compile pas, ou une touche qui bascule entre les deux.

### A-11 — Filtrer le popup en regex ou en texte brut, quel que soit le gestionnaire

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Voisine de :** A-10

Le filtre du popup accepterait **deux modes** : texte brut comme aujourd'hui, ou expression
rationnelle. Indépendant du gestionnaire — brew, winget, scoop se filtrent pareil.

Ce qu'il faudra trancher en premier, parce que « le popup » désigne deux choses :

- **Le popup vivant** (`jigger render`, appelé à chaque frappe par le greffon) filtre sur
  le mot en cours de saisie **dans la ligne de commande** — mot qui finira inséré dans
  cette ligne. Un motif comme `^fire` n'a rien à y faire une fois inséré : si la regex
  s'applique ici, il faut décider ce que `⇥` insère, et le motif ne doit pas survivre à
  l'insertion.
- **Le sélecteur plein écran** (`jigger pick`, `JIGGER_LIVE=0`) a un champ de filtre
  **séparé** de la ligne. La regex y va de soi : rien de ce qu'on tape n'est destiné à être
  inséré tel quel.

Le second est le terrain naturel ; le premier demande une décision d'interface avant une
ligne de code.

Deux points à ne pas oublier :

- **La bascule doit se voir.** Le pied du cadre dit déjà ce que font les touches ; il devra
  dire dans quel mode on filtre, sinon un motif qui ne trouve rien passera pour un
  catalogue vide.
- **Le budget de frappe.** Le popup vivant tient ~8 ms par rendu, sur des catalogues de
  16 000 entrées côté winget. Compiler un motif et balayer la liste à chaque touche entre
  dans ce budget, mais un motif fautif ne doit ni coûter ni faire échouer le rendu — un
  motif qui ne compile pas se traite comme du texte brut, ou n'est pas appliqué du tout.

À faire d'un seul tenant avec A-10 : c'est le même moteur de filtre, et deux
implémentations divergeraient.

### A-12 — Trier les colonnes du sélecteur plein écran

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Dépend de :** A-10

Pouvoir trier la liste sur la colonne de son choix, et inverser l'ordre.

**Il n'y a pas encore de colonnes à trier**, et c'est le premier constat à faire. Le
sélecteur affiche aujourd'hui une liste d'items (`pm.Item` : nom, badge, installé, version,
gestionnaire) dont seuls le badge, le nom et — quand plusieurs gestionnaires répondent — le
code PM sont rendus. Les vraies colonnes (`PACKAGE`, `CURRENT`, `AVAILABLE`, `SOURCE`, `PM`)
vivent dans les tableaux de la façade, qui ne passent pas par le sélecteur. Le tri prend
donc son sens **après** A-10, quand ces tableaux s'afficheront dans le sélecteur.

À trancher le moment venu :

- **Les touches.** Rien n'est libre par hasard dans ce cadre : ⇥, ↩, ↑↓, ^N/^P, ^G et esc
  sont pris, et les caractères imprimables vont au filtre. Le tri devra passer par une
  combinaison qui ne mange pas la frappe — et se lire dans le pied, comme le reste.
- **Trier une version n'est pas trier du texte.** `0.10.0` précède `0.9.0` dans l'ordre
  lexicographique, et c'est faux. Un tri sur `CURRENT` ou `AVAILABLE` demande une
  comparaison qui comprenne les numéros ; jigger n'en a pas encore.
- **Dans le popup d'insertion, l'ordre a des conséquences.** Le premier candidat est celui
  que `⇥` insère : trier n'y est pas un confort d'affichage mais un changement de ce que
  fait la touche. En lecture seule (A-10), la question ne se pose pas.

### A-13 — Colorer les versions obsolètes

**Priorité :** à déterminer · **Provenance :** demande du 16 août · **Voisine de :** A-10

Faire ressortir à l'œil ce qui est en retard : la version installée et celle qui l'attend,
distinguées par la couleur plutôt que laissées à la lecture de deux colonnes alignées.

Le point de départ, vérifié : **les tableaux de la façade ne portent aujourd'hui aucune
couleur.** `internal/facade/format.go` aligne des colonnes en texte nu. Le popup, lui, est
coloré via lipgloss, avec une palette déjà posée — c'est d'elle qu'il faudra partir pour ne
pas inventer un second jeu de teintes.

Trois points à traiter :

- **La couleur ne doit jamais partir dans un tuyau.** Même règle que la pagination (A-10) :
  elle ne s'arme que si la sortie est un terminal, et `--json` n'en porte jamais. Précédent
  utile dans le dépôt : jigger ne devine pas son profil couleur pour le popup, c'est le
  shell qui le lui passe (`--color auto|never|16|256|truecolor`) parce que la sortie est
  capturée. Les tableaux, eux, sortent directement — ils peuvent décider seuls, mais la
  décision doit être explicite.
- **`outdated` l'a gratuitement, `list` non.** `jg outdated` connaît déjà les deux versions,
  la colonne `AVAILABLE` en témoigne. `jg list` ne les a pas : marquer les obsolètes dans la
  liste demanderait la comparaison, qui coûte de une à cinq secondes chez brew et winget.
  Soit on ne colore que là où l'information est déjà là, soit on l'affiche depuis le cache
  du bloc de prompt — qui existe et peut mentir d'une demi-heure.
- **La couleur ne doit pas être la seule porteuse d'information.** Un terminal en 16
  couleurs, un daltonien, une capture collée dans un ticket : ce qui distingue une version
  obsolète doit rester lisible sans elle. Le projet a déjà tranché ainsi ailleurs — un
  compteur du prompt qui ne s'affiche jamais à zéro dit « à mettre à jour » par sa seule
  présence, sans flèche ni couleur.

### A-14 — Un écran de configuration (TUI)

**Priorité :** à déterminer · **Provenance :** demande du 16 août

Un écran plein terminal pour régler jigger, dont la mise en page découlerait des paramètres
**déclarés par chaque gestionnaire enregistré** — comme la table de verbes fait découler les
capacités de la façade.

Trois obstacles, dont deux sont des décisions d'architecture avant d'être du code. Ils ne
condamnent rien, mais ils expliquent pourquoi cette entrée est plus lourde qu'elle n'en a
l'air.

- **Il n'existe aucun fichier de configuration.** Tout se règle par variables
  d'environnement — `JIGGER_LIVE`, `JIGGER_ROWS`, `JIGGER_KEY`, `JIGGER_PROMPT`,
  `JIGGER_LANG`… Un écran de configuration suppose de créer un fichier, donc de trancher un
  ordre de préséance (environnement > fichier > défauts, vraisemblablement) et un
  emplacement. C'est de niveau ADR, comme l'ont été le choix de Go et la table déclarative.
- **La moitié des réglages n'appartient pas au binaire.** `JIGGER_ROWS`, `JIGGER_KEY`,
  `JIGGER_LIVE` sont lus par le **greffon**, au chargement du shell, avant tout appel à
  jigger. Un écran lancé depuis le binaire ne peut donc pas les appliquer au shell en cours :
  il écrit un fichier que le greffon lira au prochain démarrage, ou il imprime les lignes à
  coller. À décider explicitement, sinon l'écran promettra un effet qu'il n'a pas.
- **Les gestionnaires ne déclarent aujourd'hui aucun paramètre.** Les réglages existants
  sont globaux à jigger ; côté gestionnaires, il n'y a que des variables qui ne lui
  appartiennent pas (`$SCOOP`, `$SCOOP_GLOBAL`) et des durées de cache écrites en dur. Pour
  qu'une mise en page « claire et logique » se déduise des gestionnaires, il faut d'abord
  que chacun **déclare** ses paramètres — une seconde table à côté de `pm.Bindings`, dans
  l'esprit de l'ADR-0002 : ce qui est déclaré est vérifiable, ce qui est codé en dur ne
  l'est pas.

La partie visible, elle, est la moins coûteuse : Bubble Tea est déjà là, et `internal/ui`
sait dessiner un cadre, naviguer et filtrer.

---

## En cours

*(vide)*

---

## Fait

*(vide)*
