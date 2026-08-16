# Hacker News — Show HN

**Rang 4, et dernier.** Le plus fort potentiel, le public le plus exigeant, et **une seule
occasion** : un Show HN qui tombe à plat ne se recommence pas avec le même projet.

À poster une fois les trois messages Reddit passés, et leurs retours intégrés.

---

**Titre** — 80 caractères maximum sur HN ; celui-ci en fait 79, mesurés.

```
Show HN: Jigger – one syntax for Homebrew, winget and scoop, with a live picker
```

**Corps** — HN ne rend ni le gras ni les tableaux ; texte brut, paragraphes séparés par
une ligne vide, code indenté de deux espaces.

```text
I use a Mac and a Windows machine daily, and I kept typing brew commands into
PowerShell. Jigger is what came out of that: a small Go binary wired into zsh and
PowerShell.

Two things. First, a picker that appears under the prompt as you type a package-manager
command and narrows with every letter:

  ❯ brew install fire
  ╭──────────────────────────────────────────────────────╮
  │❯ brew install                           jigger 0.10.0│
  │  ▣  firealpaca                                       │
  │  ▣  firebase-admin                                   │
  │  ◆  firebase-cli                                     │
  │  ▣  firebird-emu                                     │
  │  ▣  firecamp                                         │
  │                                                      │
  │   ⇥  insert   ↓  browse   ^G  close                  │
  ╰──────────────────────────────────────────────────────╯

Nothing to trigger, and the arrow keys are never taken over: until the frame holds the
keyboard, up and down are still your shell history. When it hands a key back, it hands
it back to whatever had it before, not to the shell default.

Second, one vocabulary across all three managers. "jg install fd" reaches brew, winget
or scoop — whichever knows fd. Twelve verbs, of which seven exist everywhere.

The design decision I'd defend hardest is a refusal. When two managers both know a name,
jigger does not choose. A picker opens and you decide, and no setting turns that off.
Two packages sharing a name are not necessarily the same software, and a silent
arbitration between a "git" on winget and a "git" on scoop would destroy the only thing
the facade has going for it. There is an escape hatch for scripts, --pm, and that is
deliberately explicit.

The other constraint everything bent around: nothing slow may sit in the typing path.
Catalogs are cached on disk and filtered per manager before being merged, because
concatenating three catalogs — 14,000 names for winget alone — and then scanning would
cost the keystroke budget.

What it does not do: only brew, winget and scoop, so on Linux it drives Homebrew and not
apt or pacman; zsh and PowerShell, no fish or bash; PSReadLine's Vi mode disables the
live popup, deliberately, because relaying printable characters would break command
mode. There is no winget package for jigger itself yet.

It runs the real package-manager commands and relays their output untouched — progress
bars, licence prompts, elevation. It parses catalogs, never results. Nothing is
downloaded by jigger, nothing phones home, no daemon.

Apache-2.0, single static binary. brew install jigger, or scoop install jigger.

https://jigger.yg-devworks.com/
https://github.com/Yves848/jigger

The part I'm least sure about is the refusal above. Never choosing means an ambiguous
name always costs an interaction, even in cases where a reasonable default would be
right nine times out of ten. I decided a silent wrong install is worse than a prompt.
I'd be curious whether people who maintain multi-manager setups agree.
```

---

## Notes pour la relecture

- **Le titre.** HN coupe à 80 caractères. Pas de point final, pas de superlatif ; « Show
  HN: » compte dans la limite.
- **Pas de gras, pas de listes à puces** : HN les rend en texte brut et le message paraît
  bâclé. D'où la forme en paragraphes.
- **La question finale porte sur une décision**, pas sur le projet. Ce public répond
  volontiers à « voici mon arbitrage, ai-je tort ? » et beaucoup moins à « des retours ? ».
- **Le premier commentaire doit venir d'Yves**, dans la foulée du post : ce que jigger n'est
  pas encore, et ce qui est en cours. C'est l'usage sur Show HN, et ça désamorce la moitié
  des reproches.
- Ne pas poster avant que les trois messages Reddit soient passés : leurs retours vont
  probablement modifier ce texte, et c'est précisément à ça qu'ils servent.
