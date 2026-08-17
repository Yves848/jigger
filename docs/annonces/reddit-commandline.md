# r/commandline

**Rang 1.** Le public le plus exactement ciblé, et le plus tolérant envers un outil jeune.

---

**Titre**

```
jigger — one syntax for Homebrew, winget and scoop, with a picker that follows your typing
```

**Corps**

```markdown
I kept typing `brew install` on my Mac and `scoop install` on my Windows box, then
getting them backwards. So I wrote a small Go binary that does two things.

First, a frame that appears under the prompt as you type, and narrows down with every
letter:

```
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
```

Nothing to trigger. `⇥` inserts, `↓` enters the list, `^G` dismisses it for that line.

**Your arrow keys are never taken over.** Until the popup holds the keyboard, `↑` and `↓`
are still your shell history — popup open or not. That was the one thing I refused to
compromise on, having been burned by completion plugins that quietly eat `↑`.

Second, one vocabulary across all three managers:

    jg install fd        # brew, winget or scoop — whichever knows "fd"
    jg outdated          # everything waiting for an upgrade
    jg search ripgrep

Twelve verbs: install, uninstall, upgrade, list, outdated, search, info, source, pin,
unpin, cleanup, doctor. **It never picks between two managers for you** — if both know
the name, a picker opens and you decide. Two packages sharing a name aren't necessarily
the same software.

What it does *not* do, so nobody wastes their evening:

- Only brew, winget and scoop. No apt, no pacman, no npm — on Linux it drives Homebrew.
- zsh and PowerShell only. No fish, no bash.
- No winget package for jigger itself yet (`brew install jigger` /
  `scoop install jigger` work).
- It runs your package manager's real commands and relays their output untouched. It
  parses catalogs, not results.

Apache-2.0, single static binary, no daemon, nothing phoning home.

Site: https://jigger.yg-devworks.com/
Code: https://gitlab.yg-devworks.com/yves/jigger (self-hosted GitLab — there's a GitHub
mirror too, if that's where you'd rather read it)

The question I'd actually like an answer to: **which manager would you want next?** I've
been going back and forth between apt, pacman and npm, and they're not the same amount of
work — npm's "installed" set is per-project, which breaks an assumption the other three
share.
```

---

## Notes pour la relecture

- Le cadre est une **vraie capture** (`jigger render --line "brew install fire" --cols 56`).
  Le rejouer si l'affichage change avant de poster.
- Reddit rend les blocs de code indentés de quatre espaces de façon plus fiable que les
  triples accents graves selon les clients ; le cadre en accents graves passe bien sur le
  site et l'application officielle, mais **vérifier l'aperçu** avant d'envoyer.
- La question finale est sincère : elle rejoint A-16 (étude d'intégration d'autres
  gestionnaires). Les réponses ont donc une valeur au-delà de l'annonce.
