# r/PowerShell

**Rang 3.** Public qui jugera sur un point précis : ce que le module fait à PSReadLine.
Le texte doit donc l'aborder de front, y compris ses deux réserves.

---

**Titre**

```
A winget/scoop picker that draws under the prompt, and what it does to PSReadLine
```

**Corps**

```markdown
I wrote a PowerShell module around a small Go binary. Type a `winget` or `scoop` command
and a frame appears under the prompt, narrowing down with every letter:

```
❯ winget install Git.
╭──────────────────────────────────────────────────────╮
│❯ winget install                         jigger 0.10.0│
│  ◆  Git.Git                                          │
│  ◆  GitHub.GitHubDesktop                             │
│                                                      │
│   ⇥  insert   ↓  browse   ^G  close                  │
╰──────────────────────────────────────────────────────╯
```

Since this is the part that decides whether you'd run it, here is what it does to your
line editor.

**It relays printable keys through PSReadLine handlers** rather than reading the console
itself, so your key bindings, your history and your editing keep working. `↑` and `↓`
stay yours until you press `↓` to enter the list; `↑` on the first candidate hands the
keyboard back.

**Two caveats, both real:**

- **Vi mode disables the live popup.** `⇥` still opens the picker. Relaying printable
  characters would break navigation in command mode, and a half-working Vi mode is worse
  than none.
- **`PredictionViewStyle = ListView` draws where the frame draws.** jigger switches
  prediction back to `InlineView` while the frame is up, and restores your setting the
  moment it clears. Otherwise the two fight over the same lines on every keystroke.

**Elevation and prompts pass straight through.** jigger runs the real `winget` and
`scoop`, relaying their output untouched — progress bars, licence prompts, UAC. It parses
catalogs, never results.

Install, since v0.10.0, needs neither Go nor a clone:

    scoop bucket add jigger https://gitlab.yg-devworks.com/yves/scoop-jigger.git
    scoop install jigger

then `Import-Module` the plugin from the repository in your `$PROFILE` — the guide has
the exact lines.

It also adds `jg`, one vocabulary across winget, scoop and Homebrew: `jg install fd`
reaches whichever knows the name. It **never chooses between two managers for you**; if
both know it, a picker opens.

Limits: PowerShell 7 recommended, PSReadLine required; no winget package for jigger
itself yet; scoop's `Format-Table` output is parsed by column position, which I had to
rewrite once already when a colour escape moved a header.

Apache-2.0. https://jigger.yg-devworks.com/ · https://gitlab.yg-devworks.com/yves/jigger
(self-hosted GitLab; mirrored on GitHub as well)

What I'd like to know: **is the ListView handling the right call?** Tucking someone's
prediction view away and restoring it is a liberty I wasn't entirely comfortable taking,
and if there's a way to coexist that I've missed, this is the subreddit that knows it.
```

---

## Notes pour la relecture

- Le cadre `winget install Git.` doit être **rejoué sur la machine Windows** avant de
  poster : celui-ci est composé d'après le format réel, mais les captures de ce dépôt ont
  toutes été prises sous macOS avec brew. Ne pas publier une capture qu'on n'a pas vue.
- Les deux réserves (mode Vi, `ListView`) sont dans le README depuis longtemps : les dire
  ici évite qu'un lecteur les découvre et le prenne pour un défaut caché.
- La mention du réécrivage de l'analyse `Format-Table` est volontaire : elle montre que le
  cas a été rencontré pour de vrai, ce que ce public reconnaîtra.
