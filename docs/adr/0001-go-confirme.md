# ADR-0001 — Go confirmé comme stack de jigger

15 août 2026 — **acceptée**

## Contexte

[Description.md](../Description.md) pose la technologie comme « à trancher dès le début »
et propose trois candidats : C# (+ Avalonia), Go (+ Bubble Tea / Lip Gloss), Rust.

Or jigger est en **v0.7.0** : ~5 000 lignes de Go, les trois gestionnaires de la phase 1
implémentés, des greffons zsh et PowerShell livrés, un bloc oh-my-posh, et deux harnais de
test en pseudo-terminal (`tests/zpty.zsh` sous zsh, `tests/conpty` sous ConPTY). La
question est donc déjà tranchée par les faits ; cet ADR l'acte par écrit pour qu'elle
cesse d'être rouverte.

## Contraintes qui gouvernent le choix

1. **La latence de démarrage d'un processus.** `jigger render` est lancé **à chaque
   frappe** ; le budget est de ~8 ms de travail, ~30 ms de bout en bout sous Windows.
   C'est la contrainte dure du produit.
2. **La cible est macOS *et* Windows.** Windows est le terrain le plus coûteux : ConPTY,
   PSReadLine, aucune sortie machine de winget.
3. **jigger est une TUI, pas une GUI.** C'est le compagnon en ligne de commande de l'app
   Cocktails, pas une seconde interface graphique.
4. **Un binaire autonome**, installable sans runtime préalable.

## Décision

**jigger reste en Go**, avec Bubble Tea et Lip Gloss pour l'affichage.

## Justification

- **C# est disqualifié par la contrainte 1.** Le coût de démarrage d'un processus .NET est
  incompatible avec un lancement par frappe. Avalonia, par ailleurs, répond à la
  contrainte 3 par la négative : jigger n'a pas d'interface graphique à dessiner.
- **Rust ferait techniquement l'affaire** mais n'achète rien face à Go sur ces quatre
  contraintes, au prix d'une réécriture intégrale — y compris des deux harnais PTY, qui
  sont la partie la plus coûteuse et la moins reproductible du dépôt.
- **Go les satisfait toutes**, et le coût de sortie est nul puisque c'est l'existant.

## Conséquences

- La modularité par gestionnaire passe par des **interfaces et un registre**
  (`pm.Manager`, `internal/managers`), pas par du chargement dynamique.
- Le « lazyload » demandé par [Description.md](../Description.md) doit se lire comme
  **paresse sur le travail, pas sur le code** : Go lie tout statiquement, et son paquet
  `plugin` est indisponible sous Windows — soit la moitié de la cible. jigger applique
  déjà cette paresse : `managers.All()` instancie des structs vides, `Available()` filtre,
  `Load()` ne lit que des caches.
- L'extensibilité par des tiers **sans recompilation**, si elle devient un objectif,
  demandera un protocole de sous-processus (des binaires `jigger-<pm>` dialoguant en JSON,
  à la manière des sous-commandes git). C'est une décision distincte, à instruire dans son
  propre ADR le jour où le besoin est réel.
- Les trois stacks de [Description.md](../Description.md) cessent d'être un choix ouvert.
  Ce document est la réponse à cette section.
