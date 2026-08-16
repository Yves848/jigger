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

---

## En cours

*(vide)*

---

## Fait

*(vide)*
