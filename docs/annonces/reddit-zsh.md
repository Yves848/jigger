# r/zsh

**Rang 2.** Public pointu sur un seul point : ce que le greffon fait à leur `zle`.
Le texte doit donc parler mécanique, pas fonctionnalités.

---

**Titre**

```
A Homebrew picker that hooks into zle without stealing your arrow keys
```

**Corps**

```markdown
I wrote a zsh plugin around a small Go binary. It shows a picker under the prompt as you
type a `brew` command:

```
❯ brew install fire
╭──────────────────────────────────────────────────────╮
│❯ brew install                           jigger 0.10.0│
│  ▣  firealpaca                                       │
│  ◆  firebase-cli                                     │
│  ▣  firebird-emu                                     │
│                                                      │
│   ⇥  insert   ↓  browse   ^G  close                  │
╰──────────────────────────────────────────────────────╯
```

Since this subreddit will ask, here is what it actually does to your line editor.

**It doesn't own your arrow keys.** `↑` and `↓` stay bound to whatever they were bound to
— history, `history-substring-search`, anything — until you press `↓` to *enter* the
list. `↑` on the first candidate hands the keyboard straight back. The frame shows which
mode you're in: the current line is underlined only while the popup holds the keyboard.

And when it gives a key back, it gives it back to **whatever had it before**, not to
zsh's default. If another plugin already holds `↑`, that plugin keeps it.

**It redraws below the prompt**, and pushes the screen up when the prompt sits on the
last line — the way `fzf --height` does. Without that, the frame would almost never have
room to show.

**It bails out rather than corrupting your line** when the terminal is too narrow, or
when the terminal doesn't answer a cursor-position query.

**Catalogs are cached on disk**, so a keystroke never waits on `brew`. Nothing slow sits
in the typing path — that was the design constraint everything else bent around.

Install is a tap and one `source` line:

    brew tap yves/cocktails https://gitlab.yg-devworks.com/yves/homebrew-cocktails.git
    brew install jigger
    # then in ~/.zshrc:
    source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"

It also adds `jg`, one vocabulary that reaches brew, winget or scoop depending on which
knows the package — useful if you switch machines, irrelevant if you don't.

Limits, up front: zsh and PowerShell only, no fish or bash; `brew` on macOS and Linux,
`winget`/`scoop` on Windows, nothing else; and it needs PSReadLine-equivalent behaviour
that plain `sh` doesn't have, so it's genuinely a zsh plugin rather than a portable one.

Apache-2.0. https://jigger.yg-devworks.com/ · https://gitlab.yg-devworks.com/yves/jigger
(self-hosted GitLab; mirrored on GitHub as well)

What I'd like to hear: **does the arrow-key handover behave in your setup?** The test
harness replays a real session in a `zpty` with `zsh-autosuggestions` and
`zsh-syntax-highlighting` loaded, which is what I had to hand. Plugin stacks in this
subreddit go deeper than mine, and the handover is the part most likely to have a case I
haven't seen — `history-substring-search` binding `↑` being the obvious one.
```

---

## Notes pour la relecture

- Ce texte assume que le lecteur connaît `zle` : ne pas l'expliquer, ce serait condescendant.
- Les deux greffons nommés sont ceux que `tests/zpty.zsh` charge réellement avec
  `JIGGER_TEST_PLUGINS=1` — vérifié dans le banc d'essai, pas supposé. Ne pas y ajouter de
  nom sans l'avoir essayé : c'est le public qui vérifiera.
- La dernière ligne sur `sh` est là pour couper court à « pourquoi pas un script portable ».
