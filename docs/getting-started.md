# Getting started with jigger

*Read this in [French](fr/getting-started.md).*

From install to first completion, in about ten minutes. This guide is meant to be
read start to finish; the [README](../README.md) covers the same ground in more
detail, and explains _why_ things are built this way.

`jigger` wires a package picker into your shell: the moment you type a package-manager
command, a frame appears under the prompt and tracks your keystrokes.

```
❯ brew install fire
╭──────────────────────────────────────────────────────────╮
│❯ brew install                               jigger 0.10.0│
│  ▣  firealpaca                                           │
│  ▣  firebase-admin                                       │
│  ◆  firebase-cli                                         │
│  ▣  firebird-emu                                         │
│  ▣  firecamp                                             │
│                                                          │
│   ⇥  insert   ↩  execute   ↓  browse   ^G  close         │
╰──────────────────────────────────────────────────────────╯
```

And, across all three managers, **one syntax**: `jg install fd` reaches whichever one
knows `fd`, without you having to know which (§ 6).

| Platform | Shell | Completed commands |
|---|---|---|
| macOS, Linux | zsh | [Homebrew](https://brew.sh) |
| Windows | PowerShell 7 | [winget](https://learn.microsoft.com/windows/package-manager/), [scoop](https://scoop.sh) |
| both | both | `ssh`, `scp`, `sftp` — the servers of your `~/.ssh/config` |

That last row isn't a package manager, and jigger never runs it: typing `ssh ` offers the
hosts declared in `~/.ssh/config`, each with its `HostName` alongside, and ⇥ inserts the
one you're after. `scp` inserts `host:`, colon attached. On a machine with no
`~/.ssh/config`, nothing shows up at all; `JIGGER_COMMANDS` (§ 7) decides which commands
get intercepted.

## 1. Prerequisites

- **The manager itself** — and nothing else. jigger depends on no service, makes no
  network calls, and works only with what `brew`, `winget`, or `scoop` already has on
  disk.
- **zsh** (ships with macOS) or **PowerShell 7** with PSReadLine (ships with Windows).
- **Go ≥ 1.26**, only to compile — the Homebrew package handles that on its own.

## 2. Install the binary

### macOS and Linux — via Homebrew (recommended)

The tap is hosted on the project's GitLab, hence the explicit URL:

```sh
brew tap yves/cocktails https://gitlab.yg-devworks.com/yves/homebrew-cocktails.git
brew install jigger
```

The formula builds the binary on your machine (`go` is pulled in as a build
dependency), installs the zsh plugin under `share/`, and along the way sets up
`brew-jigger` — which makes `brew jigger …` usable like any other brew command.

To upgrade later: `brew upgrade jigger`.

### Windows — via scoop (recommended)

Since v0.10.0, releases carry prebuilt binaries, and a [scoop](https://scoop.sh) bucket
points at them. Nothing to compile, no Go to install:

```powershell
scoop bucket add jigger https://gitlab.yg-devworks.com/yves/scoop-jigger.git
scoop install jigger
```

`scoop bucket add` takes **two** arguments — the local name you choose, then the
repository. Passing only a name makes scoop look it up in its own directory of known
buckets, and it answers `unknown bucket`.

To upgrade later: `scoop update jigger`.

The bucket installs the **binary** only. The PowerShell plugin — the part that makes the
popup appear as you type — comes from the repository; § 3 wires it in.

### Any platform — via Go

```sh
go install gitlab.yg-devworks.com/yves/jigger@latest
```

The binary lands in `$GOBIN` (`~/go/bin` by default, or `%USERPROFILE%\go\bin` on
Windows). Check that this directory is on your `PATH`.

### From source

```sh
git clone https://gitlab.yg-devworks.com/yves/jigger.git
cd jigger
make install            # → ~/.local/bin/jigger  (PREFIX=… to change it)
```

On **Windows**, use the script instead — `make install` calls `install(1)`, a POSIX tool
Windows doesn't have, and `make` isn't shipped there either:

```powershell
pwsh -NoProfile -File install-windows.ps1
```

It builds, then puts `jigger` within reach: a **scoop shim** when scoop is around — the
shim points at the binary *in the repository*, so a plain `go build` is enough to update
what `jigger` runs, which is what you want while developing — or a **copy** into
`%USERPROFILE%\bin`, added to your user `PATH`, when it isn't. `-Methode`, `-Prefixe`,
`-Profil` and `-Simuler` let you steer or preview it.

This is the route for **developing** jigger, or for running a version that isn't
released yet. To simply use it, the scoop bucket above is less work. There is still no
winget package. Either way, the PowerShell plugin comes from the cloned repository.

> **Only one binary on the `PATH`.** If you installed through more than one route,
> `which -a jigger` (or `Get-Command jigger -All`) will tell you. An old binary
> shadowing a newer install in the `PATH` is the most painful failure to
> diagnose — hence the check in § 4.

## 3. Wire the plugin into the shell

### zsh

```sh
# in ~/.zshrc
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

From source, replace the path with `/path/to/jigger/shell/jigger.plugin.zsh`.
Then reload: `exec zsh`.

The order of `source` calls in `~/.zshrc` doesn't matter — the plugin places itself
wherever it needs to in zsh's hooks.

### PowerShell

```powershell
# in $PROFILE   (notepad $PROFILE to open it)
Import-Module C:\path\to\jigger\shell\jigger.psm1
```

Then reload: `. $PROFILE`, or open a new tab.

One ordering constraint, and this one's real: if you use oh-my-posh or starship,
import jigger **after** it (see § 8).

And one thing to know about keys: PSReadLine keeps only the **last** handler bound to a
chord. A profile that binds `^R` after the import — an fzf history search, typically —
takes the key back, and the regex toggle becomes unreachable. Offer it the key first;
it says whether it wants it:

```powershell
Set-PSReadLineKeyHandler -Chord Ctrl+r -ScriptBlock {
    # $true on a winget/scoop line: the popup switched to regex. $false everywhere
    # else — nothing was touched, and the key is yours.
    if (Invoke-JiggerRegex) { return }
    Invoke-FzfHistory
}
```

The module can't do this by itself: PSReadLine hands back a script-block binding's
description, never the block, so jigger has no way to call yours. Under zsh, where
`bindkey` names the widget, the plugin picks `^R` up before taking it and gives it back
on its own — nothing to do there.

The same goes for the keys that take over the screen and hand the line back — a file
explorer on `^U`, a drive picker on `^D`. Their handler replaces ours too, and the frame
is left behind, stale or orphaned. `Update-JiggerPopup` puts it back in agreement with
the line:

```powershell
Set-PSReadLineKeyHandler -Chord Ctrl+u -ScriptBlock {
    yazi
    [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
    Update-JiggerPopup
}
```

Call it with no precaution whatsoever: off a package-manager line it erases the frame and
returns, and it never throws — a keystroke is not allowed to paint the screen red.
Handlers that end on `AcceptLine()` want it too, right before that call: otherwise the
command's output cuts the frame in two.

## 4. Check that it works

```sh
jigger --version        # → jigger 0.10.0, or newer
```

Open a fresh shell and type `brew ins` (or `winget ins`) **without pressing Enter**.
The frame should appear under the prompt and narrow down with every letter.

Nothing shows up? The plugin says so when it refuses to load: a message, at shell
startup, reports that the binary is missing from the `PATH` — or that it's too old
for this plugin. The two go together: a binary that's behind doesn't understand the
options the plugin passes it, and the popup would never appear, without a word. If no
message shows up either, go to § 9.

**On the very first use**, the frame may say "building the catalog…": jigger never
holds up a keystroke waiting on the package manager, so it builds its catalog in the
background instead. A few seconds later, it's there — and it stays there (24h cache,
renewed on its own).

## 5. Use it

Just type a command. The popup lives on its own:

```
brew install fire         packages named "fire…", updated with every letter
brew uninstall ␣          installed packages only
brew list --              list's own options
winget install Git.       same idea, on Windows
scoop uninstall 7z
```

| Key | Effect |
|---|---|
| `⇥` | inserts the current candidate |
| `⏎` | completes the last part **and** runs the line, in a single keystroke |
| `↓` | enters the list, then moves down one candidate |
| `↑` | moves up; on the first candidate, hands the keyboard back to the shell |
| `^N` / `^P` | the same, for those who prefer them to arrow keys |
| `^G` | closes the popup for the current line (`⇥` reopens it) |
| `^R` | switches the filter between plain text and regex. The frame's title shows `[regex]` while it is on, and the key goes back to the shell's reverse history search whenever the popup isn't up. Under PowerShell, a profile that binds `^R` after the import takes it back — see § 3 |

Three things worth knowing, most of what makes this comfortable:

- **`⏎` completes, then runs — in the same keystroke.** `winget li ⏎` runs `winget list`:
  that's `⇥` you no longer have to type, and it holds at every level — verb, sub-verb,
  option, package name. Pressing `⏎` means "go", and the line goes: completed if a
  candidate was designated, as typed otherwise. `^G` closes the popup for the current line
  if you want to run exactly what you typed.
- **The arrow keys remain your history** as long as the popup doesn't hold the
  keyboard — open or not. The frame shows which: the current line underlined and the
  footer reading `↑↓ navigate` when it has focus, at rest and `↓ browse` when it
  doesn't.
- **jigger corrects what it inserts** whenever the command would otherwise be wrong:
  `--cask` added in front of a Homebrew cask, the qualified name `main/flux` for a
  scoop package present in several buckets, quotes around a winget identifier that
  contains spaces.

The badges in front of the names distinguish the two kinds of packages: ◆ for the
ordinary case (formula, catalog package on winget, `main` bucket), ▣ for the other
one (cask, application outside the catalog, third-party bucket).

## 6. One syntax: `jg`

Everything above speaks the language of each manager. `jg` speaks a single one for
all three:

```sh
jg install fd            # brew, winget, or scoop — whichever knows "fd"
jg outdated              # what's due for an upgrade, everywhere
jg search ripgrep
jg info fd
```

`jg` is an alias for `jigger`, set up by both plugins — the zsh one and the
PowerShell module; you can type either one.
**The facade only adds, it never replaces**: `brew install fd` keeps working
exactly as before, popup included.

### The twelve verbs

`jg ⇥` reminds you of them, and the popup offers them the same way it offers
packages:

```
❯ jg
╭──────────────────────────────────────────────────────────╮
│❯ jigger                                     jigger 0.10.0│
│  •  cleanup                                              │
│  •  doctor                                               │
│  •  info                                                 │
│  •  install                                              │
│                                                          │
│   ⇥  insert   ↩  execute   ↓  browse   ^G  close         │
╰──────────────────────────────────────────────────────────╯
```

`install`, `uninstall`, `upgrade`, `list`, `outdated`, `search`, `info` — the seven
that all three managers know how to do. Then `source` (brew's `tap`, scoop's
`bucket`), `pin`, `unpin`, `cleanup`, and `doctor`, which don't exist everywhere.
Asking winget for a verb it doesn't have — `cleanup`, `doctor` — fails cleanly,
naming who would know how to do it.

`source` comes in three forms: `jg source` lists, `jg source add <repo>` adds,
`jg source rm <repo>` removes.

### Long listings: the paged view

`list`, `outdated`, `search` and `source` can return hundreds of rows. When the output
is a terminal **and** the rows don't fit on screen, jigger shows them in a navigable
view instead of scrolling past:

| Key | Effect |
|---|---|
| type | filters as you go |
| `^R` | switches between plain-text and regex matching — the current mode is always shown |
| `⇥` | selects the row (`Space` can't: the filter field has the keyboard) |
| `^A` | selects **everything the filter leaves** — or clears it, if all of it is already selected |
| `↵` | confirms — prints the selected rows, or the current one if none are selected |
| `^G`, `esc` | leaves without printing anything |
| `↑` `↓`, `PgUp` `PgDn` | move |

**Nothing changes when the output isn't a terminal.** `jg list | grep fd` prints the same
plain table it always has, byte for byte, and `--json` is never paged — it's a machine
contract.

To pick rows *into* a pipe, ask for it explicitly:

```sh
jg install $(jg search fd --select)
```

`--select` draws the view on the terminal and sends only the chosen names — one per line
— down the pipe. `JIGGER_PAGER=0` disables the automatic view entirely.

### How the manager gets chosen

jigger looks the name up in the catalog of each manager present:

- **only one knows it** → it wins, without asking anything;
- **several know it** → the picker opens and you decide;
- **none knows it** → an error, with the closest neighbors.

There's **never an automatic choice** between two managers, and no setting
introduces one: two packages sharing the same name aren't necessarily the same
software.

`--pm <manager>` is the escape hatch — for settling an ambiguity outside a terminal
(script, CI, pipe), reaching a package too recent for the cached catalog, or
targeting a verb with no name:

```sh
jg install git --pm scoop
jg doctor --pm brew
```

### Tables, and `--json`

The four verbs that render a list — `list`, `outdated`, `search`, `source` — print
an aligned table, and the same content as JSON with `--json`:

```
$ jg list
PACKAGE                   CURRENT
alembic                   1.8.12
aom                       3.14.1
assimp                    6.0.5
```

`jg outdated` adds an `AVAILABLE` column, the version waiting for you — and answers
"nothing to report" once everything is up to date.

A `PM` column gets added when **several** managers have answered — no point showing
it when it would read the same everywhere.

Everything else (`install`, `info`…) **relays the manager's output as is**: prompts,
progress bars, and UAC elevation work as if you'd typed the native command yourself,
precisely because jigger doesn't get in the way. Under winget, `--yes` accepts the
license agreements; it's never implicit.

### What isn't there yet

- **The winget and scoop translations haven't been checked against the real
  CLIs** — development happened on a Mac. Only the brew column has actually run for
  real. The full table, with this warning, is in the
  [README](../README.md#one-syntax).

## 7. Configure

Settings are **environment variables**, to be set **before** the `source` or the
`Import-Module` — the plugin reads its keys and sets its hooks at load time.

```sh
# ~/.zshrc, before the source
JIGGER_ROWS=12
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

```powershell
# $PROFILE, before the Import-Module
$env:JIGGER_ROWS = '12'
Import-Module C:\path\to\jigger\shell\jigger.psm1
```

| Variable | Default | Role |
|---|---|---|
| `JIGGER_LIVE` | `1` | live popup. `0` = ⇥ opens the full-screen picker, and nothing shows up unasked |
| — | — | in that full-screen picker, `^R` switches the filter between plain text and regex; the mode shows on the filter line |
| `JIGGER_ROWS` | `8` | candidates shown — lower it on a short terminal |
| `JIGGER_KEY` | `^I` (Tab) | insertion key. `'^ '` for Ctrl-Space; under PowerShell, a PSReadLine name (`Ctrl+Spacebar`) |
| `JIGGER_MIN_COLUMNS` | `30` | below this width, the frame stops making sense: nothing shows up |
| `JIGGER_CACHE_DIR` | `~/Library/Caches/jigger`, `%LOCALAPPDATA%\jigger` | cache location |
| `JIGGER_BIN` | `jigger` | which binary the plugin calls. Handy while developing: Homebrew's `bin` usually comes before `~/.local/bin`, so a freshly built jigger would otherwise never be the one that runs |
| `JIGGER_PAGER` | `1` | `0` disables the paged view: listing verbs always print the plain table |
| `JIGGER_LANG` | your locale's language | messages: `en` or `fr`. Read before `LC_ALL`, `LC_MESSAGES` and `LANG` — and this is how you get French back in an English-speaking shell. Anything jigger can't translate falls back to English |
| `JIGGER_COMMANDS` | zsh: `brew pacman yay ssh scp sftp` · PowerShell: `winget,scoop,ssh,scp,sftp` | commands that trigger the popup, separated by spaces or commas. `jigger` and `jg` are **always** added to whatever you set — they're jigger's own commands, and turning them off would be a bug. `ssh`, `scp` and `sftp` live in the default instead, not among the always-on ones: they're third-party commands, and this setting exists precisely so you can choose whether they get intercepted. The two defaults differ because the machines do — `brew`, `pacman` and `yay` on one side, `winget` and `scoop` on the other. The zsh list does not depend on the distribution: a `pacman` typed on macOS is still completed, at worst against an empty catalogue |

`JIGGER_COMMANDS` is also how you turn the **SSH picker** off:
`JIGGER_COMMANDS='brew pacman yay'` under zsh, `$env:JIGGER_COMMANDS = 'winget,scoop'` under
PowerShell. You rarely need to: on a machine with no `~/.ssh/config`, the provider says
nothing at all and no frame is drawn.

One setting exists only under PowerShell, for lack of a useful zsh equivalent:

| Variable | Default | Role |
|---|---|---|
| `JIGGER_KEYS_EXTRA` | `éèêàçùâîôûëïüö°²µ§£€` | keys relayed in addition to printable ASCII |

`JIGGER_KEYS_EXTRA` deserves a note: PSReadLine offers no hook called on every
keystroke, so jigger re-registers, one by one, the keys that modify the line. On an
AZERTY keyboard, the unshifted digit row produces "éèçàù" — hence this default
value, and the setting for layouts it doesn't cover.

### Settings that stick

Environment variables vanish with the shell. Since v0.12.0, `jigger config` opens a screen
that writes them down:

```sh
jigger config
```

Three groups, and the grouping is the point:

- **What takes effect immediately** — the binary reads it on every call.
- **What takes effect in your next shell** — eight of the twelve settings are read by the
  plugin when your shell starts. The screen says so on the group rather than pretending
  otherwise.
- **What jigger sees but doesn't own** — `$SCOOP`, `$HOMEBREW_PREFIX`, the managers it
  found. Read only: they belong to the managers, and offering to change them would be a lie.

Every line shows **where its value comes from** — default, file, or environment. That
matters because **the environment still wins**: a `JIGGER_ROWS=12` in your `~/.zshrc`
overrides the file, and the screen tells you instead of showing a value the machine ignores.

| Key | Effect |
|---|---|
| `↑` `↓` | move |
| `↵` | edit the selected setting |
| `r` | reset it to its default (it leaves the file) |
| `q`, `esc` | save and quit |

`jigger config --path` prints where the file lives; it is plain `key = value`, meant to be
edited by hand too. `jigger config --list` prints the same table without the screen, which
is what a script wants.

**The screen never touches `~/.zshrc` or `$PROFILE`.** It writes its own file, and nothing
else.

## 8. The prompt block (optional)

jigger can also show, in your prompt, the **manager's version** and **pending
upgrades**:

```
 yves@MacBook  ~/git/jigger   main  🍺 6.0.17  🔬 7  📦 2 ❯      ← macOS
 PS D:\jigger  💻 1.29.280  📦 48  🥄 1 ❯                        ← Windows
```

Nothing slow sits on the prompt's path: the count runs detached and drops its result
into a one-line file, which the hook reads back using only shell primitives. Each
counter disappears once it hits zero.

**Enable the hook** — before the plugin loads:

```sh
JIGGER_PROMPT=1                                    # ~/.zshrc
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
```

```powershell
$env:JIGGER_PROMPT = '1'                           # $PROFILE, AFTER oh-my-posh/starship
Import-Module C:\path\to\jigger\shell\jigger.psm1
```

**Add the segment** — a file ready to paste, per prompt and per platform.

*oh-my-posh*: work on a copy — the themes it ships with get overwritten on every
update:

```sh
mkdir -p ~/.config/oh-my-posh
cp "$(brew --prefix oh-my-posh)/themes/catppuccin_mocha.omp.json" \
   ~/.config/oh-my-posh/my-theme.omp.json
```

Paste the content of [`shell/oh-my-posh/brew.segment.json`](../shell/oh-my-posh/brew.segment.json)
— or of [`windows.segment.json`](../shell/oh-my-posh/windows.segment.json) — into
the `segments` array of the block you want, then point your profile at your copy:

```sh
eval "$(oh-my-posh init zsh --config ~/.config/oh-my-posh/my-theme.omp.json)"
```

*starship*: nothing to copy beforehand, there's only one config file —
append [`shell/starship/brew.toml`](../shell/starship/brew.toml), or
[`windows.toml`](../shell/starship/windows.toml):

```sh
cat /path/to/jigger/shell/starship/brew.toml >> ~/.config/starship.toml
```

These are `env_var` modules, which starship's default format already shows: there's
nothing else to do.

The block only appears on the **second prompt**: nothing shows up until the first
count is done. The related settings (`JIGGER_PROMPT_TTL`, `JIGGER_PROMPT_SYNC`) and
the exposed variables are described in the
[README](../README.md#prompt-block); they work just as well for a homemade prompt.

## 9. When it doesn't work

| Symptom | Likely cause |
|---|---|
| "binary not found in PATH" at shell startup | the install directory isn't on the `PATH` — or the shell wasn't reloaded |
| "binary … is at X, but this plugin requires Y" | two competing installs. `which -a jigger` (`Get-Command jigger -All`); `brew upgrade jigger`, `make install`, or `install-windows.ps1` on Windows |
| no frame, no message | terminal too narrow (`JIGGER_MIN_COLUMNS`), or a terminal that doesn't answer the cursor-position query — jigger then abstains rather than draw blind |
| frame missing under PowerShell in **Vi mode** | the live popup is disabled there on purpose: relaying printable characters would break command mode. ⇥ still works |
| display fighting with PSReadLine prediction | jigger sets `PredictionViewStyle = ListView` for as long as the frame shows and restores it afterward; if it's stuck on `InlineView`, a fresh shell sorts it out |
| "building the catalog…" that never ends | run `jigger warm --all` by hand to see what the manager says |
| the prompt's counter is wrong | it only sees what goes through this shell; an upgrade run elsewhere is picked up once the TTL expires |
| `jg`: "unknown verb" | it isn't one of the twelve — `jg ⇥` lists them. The native command, though, is always spelled out in full: `brew tap`, not `jg tap` |
| `jg`: "unknown to brew" on a package that exists | the cached catalog is older than the package. `jg … --pm brew <name>` bypasses it, `jigger warm --all` refreshes the cache |
| `jg`: "manager unavailable for this verb" | the requested `--pm` isn't installed, or doesn't support this verb; the message names which ones do |
| `^R` doesn't switch to regex under PowerShell | your profile binds `^R` after the import and takes the key. `Get-PSReadLineKeyHandler -Bound` shows who holds it: `jigger:regex` means us. Have that handler call `Invoke-JiggerRegex` first (§ 3) |
| a frame left behind after some other key | same cause: a handler bound after the import took a key we relay. End it with `Update-JiggerPopup` (§ 3) |

To isolate a conflict with another line-editing plugin, `JIGGER_LIVE=0` turns off
everything tied to keystrokes: only ⇥ remains, and it opens the full-screen picker.

**Uninstalling**: remove the line from `~/.zshrc` (or `$PROFILE`), then
`brew uninstall jigger` — or delete the binary. The cache goes with
`rm -rf "$(dirname "$(jigger prompt --path)")"`.

## 10. Going further

The plugin is only a client: the subcommands work on their own, and that's the best
way to understand what's happening.

```sh
jigger complete "brew install fire" # candidates, one per line
jigger complete "jg "               # … and the facade's verbs
jigger render --line "brew ins" --cols 80   # one popup frame, metadata included
jigger pick "brew uninstall 7z"     # the full-screen picker
jigger demo                         # static, colored preview
jigger prompt                       # cached state, as read by the hook
jigger warm --all                   # rebuilds the catalogs (slow)
```

The facade's verbs are called the same way without the plugin — `jg` being only an
alias, `jigger outdated --json` works anywhere, including in a script or a CI where
no interactive shell is loaded.

- [README](../README.md) — what jigger does, and why each choice was made this way.
- [CHANGELOG](../CHANGELOG.md) — what changed from one version to the next.
- `docs/` — architecture decisions (ADRs), designs in progress, and the project's
  journal.
