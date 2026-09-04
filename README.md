# jigger

*Read this in [French](README.fr.md).*

**Package-manager assistance in the terminal** — context-aware completion and an
interactive picker, right inside _your_ real shell.

![jigger completing a brew install line, the popup narrowing with every letter](docs/media/out/macos-01-gestionnaire-natif.gif)

*Captured on macOS — nothing is typed but the command itself. The [same popup on the three platforms](docs/getting-started.md#the-same-popup-on-all-three),
and [how the captures are produced](docs/captures.md).*

`jigger` is a small, self-contained Go binary (near-instant startup) wired into the
shell: the moment you type a package-manager command, a **popup**
([Bubble Tea] / [Lip Gloss]) appears under the prompt and tracks your keystrokes,
offering the right candidates for the context. ⇥ inserts the current candidate into the
line — you never have to ask for it.

| Platform | Shell | Completed commands |
|---|---|---|
| macOS, Linux | zsh (`shell/jigger.plugin.zsh`) | [Homebrew](https://brew.sh) |
| Arch Linux | zsh | [pacman](https://wiki.archlinux.org/title/Pacman), [yay](https://github.com/Jguer/yay) — repositories and the [AUR](https://aur.archlinux.org) |
| Windows | PowerShell (`shell/jigger.psm1`) | [winget](https://learn.microsoft.com/windows/package-manager/), [scoop](https://scoop.sh) |
| all | both | `ssh`, `scp`, `sftp` — the servers of your `~/.ssh/config` |

The **first word of the line** decides: `brew`, `winget`, `scoop`, `pacman`, `yay` — and
`ssh`, `scp` or `sftp`, whose candidates are servers rather than packages. Each one brings its own
subcommands, options, and catalog; everything else — the popup, the keys, the prompt
block — is shared.

jigger never *runs* `ssh`: it completes the line you will run yourself, and stays out of
the way when it has **no host to offer** — no `~/.ssh/config`, or nothing matching what
you typed: no popup. Which commands get intercepted is yours to choose, in both shells,
through `JIGGER_COMMANDS`
([settings](#settings)). See [ADR-0005](docs/adr/0005-completion-sans-facade.md): the
completion contract is not reserved for package managers.

Command-line companion to the **Cocktails** GUI app, but **fully independent**: it needs
nothing but the package manager itself.

## What it does

- **Context-aware completion**
  - first word → subcommands (`install`, `uninstall`, `search`…);
  - after `install`, `show`, `info`… → **all** known packages;
  - after `uninstall`, `upgrade`, `pin`… → only **installed** packages;
  - after `-` → the subcommand's **options** (`winget install --exact`,
    `brew list --versions`…).
- **An SSH server picker**: type `ssh `, `scp ` or `sftp ` and the popup offers the hosts
  declared in `~/.ssh/config` — `Include` directives followed, `Host *` and other patterns
  left out — each one showing its `HostName` alongside. A command with no verb puts its
  operand right after the command name, so the catalog comes straight away. `scp` inserts
  `host:`, colon attached, because `scp file host /tmp` would silently copy to a *local*
  file named `host`. On a machine with no `~/.ssh/config`, nothing shows up at all.
  [The SSH picker](docs/ssh.md) covers it in full.
- **Badges** and an **"installed" indicator** in the picker: ◆ for the ordinary case
  (formula, catalog package on winget, `main` bucket), ▣ for the other one (cask,
  application detected outside the catalog, third-party bucket).
- **Automatic corrections** — the kind that prevent a broken command:
  - brew: picking a "pure" cask behind `install`/`reinstall` inserts `--cask <name>`;
  - scoop: a name present in several buckets is inserted qualified, `main/flux`;
  - winget: an identifier containing spaces is inserted in quotes.
- **A popup that's alive**: the frame appears as soon as you type "`winget `" and narrows
  itself down with every keystroke, without a single key press. `↓` enters the list, `⇥`
  inserts, `⏎` completes and runs in one keystroke, `^G` closes.
- **Explicit focus**: the popup only takes the arrow keys once you've entered it. `↓`
  moves you in; `↑` moves you back out on the first candidate — and until it holds the
  keyboard, `↑`/`↓` remain the shell's history. The current line shows which is which:
  underlined when the popup holds the keyboard, at rest when it doesn't.
- **Prompt block** (optional): the manager's version and pending upgrades in the prompt,
  counted separately — never slowing it down. Segments ready to paste for
  **oh-my-posh** and **starship**.

## Installation

Turnkey, one copy-paste block per platform: **[Installation, end to end](docs/installation.md)**.

Start to finish, step by step: **[Getting started](docs/getting-started.md)** —
install, wire into the shell, configure, troubleshoot. What follows is the summary.

```powershell
# Windows — prebuilt, nothing to compile
scoop bucket add jigger https://gitlab.yg-devworks.com/yves/scoop-jigger.git
scoop install jigger
```

```sh
# anywhere else, or to build it yourself (Go ≥ 1.26)
go install gitlab.yg-devworks.com/yves/jigger@latest   # → $GOBIN/jigger
#   or:  git clone … && make install          (Windows: install-windows.ps1)
```

The `jigger` binary must be on the `PATH`.

### zsh (Homebrew)

Through the tap, which builds the binary, installs the plugin, and sets up
`brew jigger`:

```sh
brew tap yves/cocktails https://gitlab.yg-devworks.com/yves/homebrew-cocktails.git
brew install jigger
```

```sh
# in ~/.zshrc
source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"
#   from source:  source /path/to/jigger/shell/jigger.plugin.zsh
```

Reload your shell (`exec zsh`).

### PowerShell (winget, scoop)

The scoop bucket installs the **binary** only — the module comes from the repository:

```powershell
git clone https://gitlab.yg-devworks.com/yves/jigger.git $HOME\git\jigger
```

```powershell
# in $PROFILE  (notepad $PROFILE to open it)
Import-Module $HOME\git\jigger\shell\jigger.psm1
```

Reload your shell (`. $PROFILE`, or a new tab). PowerShell 7 is recommended;
PSReadLine is required (it ships with Windows).

Two caveats, worth knowing:

- PSReadLine's **Vi mode** disables the live popup (⇥ still works): relaying printable
  characters would break navigation in command mode;
- `PredictionViewStyle = ListView` draws in the same spot as the popup. jigger tucks the
  prediction view away for as long as the frame is showing — it switches back to
  `InlineView`, right after the cursor — and hands it back the moment the frame clears.
  Otherwise the two would fight over the same lines on every keystroke.

## Usage

Just type a command — the popup lives on its own:

```
winget ␣                 → the subcommands
winget install Git.      → the "Git.…" packages, updated with every letter
scoop uninstall ␣        → the installed applications
winget list --           → the options of list
brew install fire        → same thing, on macOS
```

| Key | Effect |
|---|---|
| `⇥` | inserts the current candidate (corrected if needed) |
| `⏎` | completes the last part **and** runs the line, in a single keystroke |
| `↓` | enters the list, then moves down one candidate |
| `↑` | moves up; on the first candidate, hands the keyboard back to the shell |
| `^N` / `^P` | the same, for those who prefer them to arrow keys |
| `^G` | closes the popup for the current line (`⇥` reopens it) |
| `^R` | switches the filter between plain text and regex; the frame's title shows `[regex]` while it is on |

`⏎` **completes, then runs — in the same keystroke**, and at every level of the tree:
verb, sub-verb, option, package name. `winget li ⏎` runs `winget list`; it's `⇥` you no
longer have to type. Pressing `⏎` means "go", and the line goes: completed if a candidate
was designated, as typed otherwise — jigger doesn't decide on your behalf whether it is
correct. `^G` closes the popup for the current line if you want to run exactly what you
typed.

As long as the popup doesn't hold the keyboard, `↑` and `↓` remain the **shell's
history** — whether the popup is open or not: opening a candidate list doesn't cost
access to the previous command. What they'll do is shown in the frame — footer
`↓ browse` and the current line at rest while it doesn't have focus, `↑↓ navigate` and
an underlined line once it does. And jigger always hands the key back to whatever it
was doing before: if another plugin already holds your arrow keys (prefix search in
history, for example), that one keeps control.

`^R` switches the **filter**, and only the filter: the line, the frame and the keys
don't move. Three things worth knowing — the pattern is not anchored, so `fire` in regex
mode also matches `arrayfire`; the case is ignored in both modes; and it applies to
**package names only**, verbs, subcommands and flags keeping prefix matching. A pattern
that doesn't compile matches nothing, and the frame says so rather than listing the whole
catalog. Outside the popup, `^R` remains your shell's reverse history search.

After `winget install`, the word is empty and the catalog holds thousands of entries:
the frame then invites you to type at least one letter rather than listing everything.

### Settings

The same in both shells — to be set **before** the `source` / `Import-Module`:

```sh
JIGGER_LIVE=0     # disables the live popup: ⇥ opens the full-screen picker
JIGGER_ROWS=12    # candidates shown (default 8; reduced if the terminal is short)
JIGGER_KEY='^ '   # insertion key (default Tab)
JIGGER_LANG=fr    # message language: en or fr
JIGGER_COMMANDS='brew pacman ssh'  # commands that trigger the popup (default
                             # 'brew ssh scp sftp'; jigger and jg are always added).
                             # This is how you turn the SSH picker off.
```

```powershell
$env:JIGGER_LIVE = '0'
$env:JIGGER_ROWS = '12'
$env:JIGGER_KEY  = 'Ctrl+Spacebar'   # PSReadLine key names
$env:JIGGER_LANG = 'fr'              # message language: en or fr
$env:JIGGER_COMMANDS = 'winget,scoop,ssh,scp,sftp'  # commands that trigger the popup
                                                     # (jigger and jg are always added)
$env:JIGGER_KEYS_EXTRA = 'éèçàù'           # keys to relay in addition to ASCII
```

jigger speaks **English and French**. Left alone, it takes the language from `LC_ALL`,
`LC_MESSAGES`, `LANG` — then, under Windows, from the system's own language — and falls
back to English for anything it can't translate. `JIGGER_LANG` overrides all of that,
and is how you get French back in an English-speaking shell. The binary and the plugin
read it the same way, so the popup and the plugin's own messages never disagree.

The popup clears itself if the terminal is too narrow — and, under zsh, if it doesn't
respond to the cursor-position query.

When the prompt sits on the terminal's last line — the ordinary case for a terminal in
active use —, **jigger pushes the screen up** to make room for the frame, the same way
`fzf --height` does. That holds for both shells: without it, the popup would almost
never get to show at all.

Each of the two plugins **checks the binary's version** on load. Plugin and binary go
together: an older binary doesn't know about the options the plugin passes it, exits
with an error, and the popup never shows — without a word. It says so now.

`JIGGER_KEYS_EXTRA` deserves a note: PSReadLine offers no hook called on every
keystroke. jigger therefore re-registers, one by one, the keys that modify the line —
the printable ASCII ones, plus those in this list. On an AZERTY keyboard, the digit
row, unshifted, produces "éèçàù" — hence the setting, and its default value.

## One syntax

Above the three native popups, `jg <verb> [package…]` — an alias for `jigger <verb>…`,
set up by both plugins — speaks one vocabulary to all three managers. `jg install fd`
does exactly what `brew install fd` would (or `scoop install fd`, or
`winget install --id fd --exact`): the facade just works out, for `fd`, which manager
knows it and how to ask that manager for it.

### Twelve verbs, three translations

**Universal** — all three managers can do these:

| `jg` verb | brew | winget | scoop |
|---|---|---|---|
| `install {pkgs}` | `install {pkgs}` | `install --id {pkg} --exact` | `install {pkgs}` |
| `uninstall {pkgs}` | `uninstall {pkgs}` | `uninstall --id {pkg} --exact` | `uninstall {pkgs}` |
| `upgrade [pkgs]` | `upgrade [pkgs]` | `upgrade --id {pkg}` | `update {pkgs}` / `update *` |
| `list` | `list --versions` | `list` | `list` |
| `outdated` | `outdated --json=v2` | `list --upgrade-available` | read from disk, no subprocess |
| `search {q}` | `search {q}` | `search {q}` | `search {q}` |
| `info {pkg}` | `info {pkg}` | `show --id {pkg}` | `info {pkg}` |

`{pkgs}` for brew and scoop means a single call with every name on it; `{pkg}` for winget
means one `--id` per call, one call per name. winget only accepts one identifier at a
time; jigger therefore invokes it once for every name that resolves to it, in sequence.

**Convergent** — the same concept, a different name for each (or for two out of three):

| `jg` verb | brew | winget | scoop |
|---|---|---|---|
| `source` | `tap` | `source list` | `bucket list` |
| `source add {arg}` | `tap {arg}` | `source add {arg}` | `bucket add {arg}` |
| `source rm {arg}` | `untap {arg}` | `source remove {arg}` | `bucket rm {arg}` |
| `pin {pkg}` | `pin {pkg}` | `pin add --id {pkg}` | `hold {pkg}` |
| `unpin {pkg}` | `unpin {pkg}` | `pin remove --id {pkg}` | `unhold {pkg}` |
| `cleanup` | `cleanup` | _(no such concept)_ | `cleanup *` |
| `doctor` | `doctor` | _(no such concept)_ | `checkup` |

`cleanup` and `doctor` don't exist for winget: asking for them with winget as the only
available manager fails cleanly, naming who would know how to do it and why it isn't
this one — that's the capability model speaking, not a silent error.

> **The winget and scoop columns: to be taken with caution.** They come from the spec
> and, as of today, have never been checked against a real install. Only the brew column
> has actually run for real (`brew <verb> --help`, one at a time).
> `internal/winget/verbs.go` and `internal/scoop/verbs.go` carry the same warning as a
> comment; a Windows pass will lift it — the captures are not that pass: they show the
> popup, not the verb tables.

### Routing: never an automatic choice

jigger looks up the requested name in the catalog of each available manager:

- **only one knows it** → it wins, without anyone having to say so;
- **several know it** → the picker opens, badges included — the same popup as
  completion, with a different title on its frame;
- **none knows it** → an error, with the closest neighbors when the catalog has any
  to offer.

There's no fourth case: no setting (there's no `JIGGER_PM_ORDER`) decides on the
user's behalf. Two packages sharing the same name aren't necessarily the same software,
and a silent arbitration between the two is precisely what would make a facade
impossible to trust.

Two errors captured for real (brew, the only manager present on this machine):

```
$ jg frobnicate
jigger: "frobnicate" — unknown verb. "jg ⇥" lists what jigger can do

$ jg info zzznonexistentpkgzzz
jigger: "zzznonexistentpkgzzz" — unknown to brew
        If the package is too recent for the catalog: jg … --pm brew zzznonexistentpkgzzz
```

`--pm <manager>` is the escape hatch — for settling an ambiguity outside a terminal
(pipe, script, CI), reaching a package too recent for the cached catalog, or targeting a
verb with no name (`jg doctor --pm scoop`). Captured for real, on a machine that only
has brew — so the failure here is a missing manager, not an ambiguity:

```
$ jg list --pm scoop
jigger: --pm scoop — manager unavailable for this verb. Available: brew
```

On Windows, with both winget and scoop present, a real ambiguity would open the picker
(illustrative example: the routing picker is the one frame Windows has no capture of yet
— [docs/captures.md](docs/captures.md) says what its six scenarios cover):

```
$ jg install git
┌─ git: 2 managers ────────────────┐
│ ◆ Git.Git            winget      │
│ ▣ git                scoop/main  │
└─ ↵ choose   ^G cancel ───────────┘
```

### `--json`, `--yes`

The four verbs that render a **table** — `list`, `outdated`, `search`, `source` —
accept `--json` for the same data, machine-readable. Everything else (`install`,
`uninstall`, `info`…) relays the manager's output **as is**: prompts, progress bars,
and UAC elevation work without a single extra line of code, precisely because jigger
doesn't get in the way.

Captured for real (`brew`, macOS):

```
$ jg outdated
PACKAGE  CURRENT  AVAILABLE
nushell  0.114.1  0.115.0

$ jg outdated --json
[
  {
    "name": "nushell",
    "version": "0.114.1",
    "available": "0.115.0",
    "kind": "F",
    "source": "",
    "pm": "brew"
  }
]
```

And an example of raw relay, on `info` (never normalized) — truncated here, but every
line below is what `brew info fd` actually prints:

```
$ jg info fd
==> fd: stable 10.4.2 (bottled), HEAD
Simple, fast and user-friendly alternative to find
https://github.com/sharkdp/fd
Conflicts with:
  fdclone (because both install `fd` binaries)
Not installed
…
```

`--yes` accepts winget's **license agreements**
(`--accept-package-agreements --accept-source-agreements`) on `install`/`uninstall`/
`upgrade`. It's **never implicit**: without it, winget's prompt shows up normally — the
output being relayed, nothing stops you from answering it by hand. For brew and scoop,
which have no such notion, `--yes` does nothing.

### When administrator privileges are needed

**Windows.** jigger still doesn't get in the way: the command runs normally, and it's its
**exit code** that says what happened. When winget refuses for lack of privileges, jigger
says so and offers to re-run it elevated — never on its own, and never without an explicit
yes (the line open by default is *cancel*).

```
$ jg install Some.Package
jigger (winget): this command requires administrator privileges.
╭──────────────────────────────────────────────────────────╮
│❯ Run as administrator?                      jigger 0.17.0│
│  •  cancel                                               │
│  •  run it in an elevated window                         │
│                                                          │
│   ↵  choose   ^G  cancel                                 │
╰──────────────────────────────────────────────────────────╯
```

Three things worth knowing:

- **Two of winget's codes say the opposite** — an installer that *refuses* an elevated
  context, and an action forbidden from an admin context on a user-scope package. jigger
  offers nothing on those: it tells you to try again from an ordinary terminal.
- **Which route it takes is announced before you answer.** If Windows 11's `sudo` is
  enabled (Settings → System → For developers), jigger uses it; otherwise it opens a
  separate elevated console — an elevated process cannot attach to the console of one that
  isn't, which is a system boundary. Either way jigger waits for the end and returns the
  code where you typed.
- **No terminal, no question.** `jg install … | tee`, a script, a scheduled task: jigger
  prints the exact line to re-run and returns the original code. A pipeline must never
  block on a prompt.

The reasoning lives in [ADR-0004](docs/adr/0004-elevation-constatee.md): jigger observes,
it doesn't intercept. **macOS and Linux** are not covered — no manager there publishes an
equivalent exit code (A-22).

### The PM column

```
$ jg source
PACKAGE
asmvik/formulae
felixkratz/formulae
jandedobbeleer/oh-my-posh
koekeishiya/formulae
nikitabobko/tap
yves/cocktails
```

(captured for real — the taps actually configured on this machine; the listing above and
the `outdated` example further up are in the same situation.) The `PM` column only shows up
when **several** managers contributed to a table — a column that's always identical
teaches nothing. On this machine, only brew answers: no PM column. With both winget and
scoop present, `jg outdated` would show a third column distinguishing the two origins.

### What the facade doesn't change

Native commands — `brew install fd`, `winget search Git`, `scoop info 7zip` — keep
working exactly as before, live popup included: the facade **only adds**, it never
replaces. `jg`/`jigger` is one more path, not a mandatory one.

**What isn't there yet:**

- The winget and scoop columns of the table above remain **unverified in practice**
  (see the warning above) — only brew has actually run for real.

## Prompt block

A block in the prompt: the **manager's version**, and **pending upgrades**, counted
separately. Ready to paste for **oh-my-posh** and for **starship**; everything goes
through environment variables, so any other prompt can read them too.

```
 yves@MacBook  ~/git/jigger   main  🍺 6.0.17  🔬 7  📦 2 ❯      ← macOS
 PS D:\jigger  💻 1.29.280  📦 48  🥄 1 ❯                        ← Windows
```

On macOS: a **beer** for brew, a **microscope** for formulae, a **package** for casks.
On Windows: a **laptop** for winget's version, a **package** for winget packages to
upgrade, a **spoon** for scoop applications.

Each counter disappears once it hits zero — `💻 1.29.280  🥄 1` if only scoop remains,
`💻 1.29.280` alone once everything is up to date. A counter that's **never** shown at
zero means its mere presence says "needs an update": no arrow or letter to add.

These are **emoji**: no special font is required. The choice is not free, though. macOS's
`wcwidth()` does not know the width of any emoji added after Unicode 8 and reports zero: zsh
then counts zero columns where the terminal draws two, its cursor arithmetic drifts, and the
command line loses a character **on screen** as soon as it reaches the right margin. The
buffer stays intact — the command that runs is the right one, which makes the failure all
the more confusing. The test tube `\U0001F9EA` and the window `\U0001FA9F` were two of
those, and have been replaced. The mirror trap exists too: glyphs with **text presentation**
by default, such as `\U0001F5A5` (desktop computer) or `\U0001F6E0` (hammer and wrench),
which zsh counts as two columns and the terminal draws on one.

Before picking another emoji, check it: `${(m)#c}` under zsh must match what the terminal
draws. On macOS 25.5, 848 of the 1171 wide glyphs pass. To sidestep the question entirely,
every segment file lists the matching **Nerd Font** glyphs, one column wide and never
ambiguous.

Either way, write them as **escapes** (`\u21E1`, `\U0001F37A`) rather than literally:
it's the only form that comes through editors, copy-paste, and Unicode-normalizing
tools unscathed. Both JSON and TOML accept it, and the themes shipped with
oh-my-posh do the same.

Counting upgrades costs one to five seconds — `brew outdated` as much as
`winget list --upgrade-available`: it is therefore **kept off the prompt's critical
path**. jigger runs it in the background and drops the result into a one-line file,
which the hook reads back using nothing but shell primitives — **no process spawned,
no waiting**. The value shown is that of the last computation; past
`JIGGER_PROMPT_TTL`, a refresh is fired off detached and the next prompt is up to date.

That TTL alone would let the counter lie for half an hour after a `winget upgrade`.
jigger therefore watches for the commands that change state — `install`, `upgrade`,
`uninstall`, `update`, `bucket`… — as they leave, and refreshes **right after**: the
prompt following the upgrade is already accurate. It's the only time the prompt waits
for anything (usually under a second, after a command that took far longer);
`JIGGER_PROMPT_SYNC=0` makes that refresh detached instead, at the cost of an accurate
counter only on the prompt *after that one*.

Only commands typed in the shell are detected: an upgrade launched elsewhere — another
tab, a GUI app — only gets picked up once the TTL expires.

Everything goes through **environment variables**: oh-my-posh has had no `command`
segment since v26, and having starship spawn a process on every prompt would go
against the rule above. The hook exports, the prompt reads — a `text` segment on the
oh-my-posh side, `env_var` modules on the starship side.

**1. Enable the hook** — *before* loading the plugin:

```sh
JIGGER_PROMPT=1                                    # ~/.zshrc
source /path/to/jigger/shell/jigger.plugin.zsh
```

```powershell
$env:JIGGER_PROMPT = '1'                           # $PROFILE
Import-Module $HOME\git\jigger\shell\jigger.psm1
```

Under PowerShell, `prompt` is the only "precmd" available: jigger **wraps** whatever
is already in place. So import jigger **after** oh-my-posh or starship, or the block
would always lag one prompt behind. (Under zsh, the order of `source` calls doesn't
matter: the hook places itself at the front of `precmd_functions` on its own, so
before the prompt gets rendered.)

**2a. The oh-my-posh segment** — the themes shipped with oh-my-posh get overwritten on
every update: work on a copy.

```sh
mkdir -p ~/.config/oh-my-posh
cp "$(brew --prefix oh-my-posh)/themes/catppuccin_mocha.omp.json" \
   ~/.config/oh-my-posh/my-theme.omp.json
```

Paste the content of [`shell/oh-my-posh/brew.segment.json`](shell/oh-my-posh/brew.segment.json)
— or of [`pacman.segment.json`](shell/oh-my-posh/pacman.segment.json) or
[`windows.segment.json`](shell/oh-my-posh/windows.segment.json) — into the
`segments` array of the block you want, then point your profile at your copy:

```sh
eval "$(oh-my-posh init zsh --config ~/.config/oh-my-posh/my-theme.omp.json)"
```

**2b. The starship segment** — nothing to copy beforehand: starship has only one
config file, yours. Append the content of
[`shell/starship/brew.toml`](shell/starship/brew.toml) — or of
[`pacman.toml`](shell/starship/pacman.toml) or
[`windows.toml`](shell/starship/windows.toml) — to the end of
`~/.config/starship.toml`:

```sh
cat /path/to/jigger/shell/starship/brew.toml >> ~/.config/starship.toml
```

These are three [`env_var`](https://starship.rs/config/#environment-variable)
modules, one per variable. starship's default `format` (`$all`) already contains
`$env_var`: the block shows up without anything more. To place it elsewhere, write an
explicit `format` and put `${env_var}` in it — or, module by module,
`${env_var.JIGGER_BREW_VERSION}`.

Where the oh-my-posh segment ties the whole block to the manager's version, the three
starship modules are independent: on a machine that only has scoop, `🥄 1` shows up
alone rather than nothing at all.

The block shows **nothing** until the cache exists — it appears on the second prompt,
once the first refresh has finished.

### Settings

```sh
JIGGER_PROMPT=1        # enables the block (default 0)
JIGGER_PROMPT_TTL=1800 # cache age, in seconds, before a refresh (default 30 min)
JIGGER_PROMPT_SYNC=1   # after a mutating command, refresh before showing the
                       # prompt (default); 0 = detached, only on the next prompt
JIGGER_CACHE_DIR=…     # cache location (default ~/Library/Caches/jigger on macOS,
                       # %LOCALAPPDATA%\jigger on Windows)
```

### Exposed variables

| Variable | Content |
|---|---|
| `JIGGER_BREW_VERSION` | brew's version, without the commit suffix: `6.0.17` |
| `JIGGER_BREW_FORMULAE` | outdated formulae |
| `JIGGER_BREW_CASKS` | outdated casks |
| `JIGGER_BREW_OUTDATED` | total of the two |
| `JIGGER_WINGET_VERSION` | winget's version: `1.29.280` |
| `JIGGER_WINGET_OUTDATED` | winget packages to upgrade |
| `JIGGER_SCOOP_OUTDATED` | scoop applications to upgrade |
| `JIGGER_PACMAN_VERSION` | pacman's version: `7.1.0` |
| `JIGGER_PACMAN_REPOS` | repository packages to upgrade |
| `JIGGER_PACMAN_AUR` | AUR packages to upgrade |
| `JIGGER_PACMAN_OUTDATED` | total of the two |
| `JIGGER_OUTDATED` | total of the two |

A counter is **left unset** when it's zero. On the oh-my-posh side, the template
reduces to a plain `{{ if .Env.JIGGER_WINGET_OUTDATED }}`, no string comparison
needed; on the starship side, an `env_var` module with no variable simply doesn't
show, and there's no condition to write. On both sides, the block clears itself when
there's nothing to say.

To show a single figure instead of the per-manager breakdown, replace the two
oh-my-posh template blocks with:

```
{{ if .Env.JIGGER_OUTDATED }} <#F9E2AF>\u21e1{{ .Env.JIGGER_OUTDATED }}</>{{ end }}
```

… or the last two starship modules with:

```toml
[env_var.JIGGER_OUTDATED]
symbol = "\u21E1 "
style  = "#F9E2AF"
format = "[$symbol$env_value]($style) "
```

Nothing stops you from using these variables outside these two prompts (a homemade
prompt, a hand-rolled `PS1`…). Under PowerShell, `Update-JiggerPrompt` is exported on
purpose: call it from your own `prompt` function.

## Under the hood (CLI)

The plugin builds on these subcommands; usable on their own:

```sh
jigger <verb> [--pm <manager>] [--json] [--yes] [arguments…]
                                 # the facade, see § One syntax — install, uninstall,
                                 # upgrade, list, outdated, search, info, source[ add|rm],
                                 # pin, unpin, cleanup, doctor
jigger render --line "winget install Git." --sel 0 --cols 80   # one frame of the live popup
                                 # 1st line: count=… sel=… exec=… left=<completed line>
                                 # --focus=true: the popup holds the keyboard (see § Usage)
jigger complete "install fire"   # candidates, one per line (classic completion)
jigger pick "scoop uninstall 7z" # interactive picker; prints the new line
                                 # exit code: 0 = insert, 10 = execute, 2 = cancelled
jigger demo                      # static, colored preview of the picker
jigger prompt                    # cached state: version⇥counter1⇥counter2⇥epoch
jigger prompt --refresh          # queries the manager and rewrites the cache (slow)
jigger prompt --path             # path to the cache file
jigger warm                      # rebuilds stale catalogs (slow, detached)
jigger warm --installed          # only the installed-package lists
jigger warm --all                # everything, stale or not
```

`render`, `complete`, `pick`, `prompt`, `warm`, and `demo` are **reserved words**: the
first word of the line is a facade verb as soon as it isn't one of these. A permanent
constraint going forward — no future internal use will ever be able to reuse a
canonical verb's name; if an internal "jigger list" were ever needed, it would be that
one that gets renamed, not the verb.

`render` is **stateless**: the selected index lives on the shell side and comes back
through `--sel`. That's what lets the plugin stay in charge of the keyboard — the
shell keeps its line, jigger only prints a frame.

**Nothing slow is ever on a render's critical path** (~8 ms of work, ~30 ms end to end
on Windows, where spawning a process costs the most). Each manager gets there its own
way:

| | catalog | installed | outdated |
|---|---|---|---|
| **brew** | `brew formulae` / `casks`, cached 24 h | read from `Cellar`/`Caskroom` (~1 ms) | `brew outdated --json=v2`, in the background |
| **scoop** | read from `buckets/*/bucket/*.json` | read from `apps/<name>/<version>` | manifest comparison, on disk |
| **winget** | `winget search .`, cached 24 h | `winget list`, cached 10 min | `winget list --upgrade-available`, in the background |

scoop needs **no cache at all**: everything jigger asks it for is already on disk, in
a tree structure that looks a lot like Homebrew's `Cellar`. Even the count of pending
upgrades is read from there — that's what `scoop status` does, but without starting
PowerShell or touching the network.

winget is the opposite: no machine-readable output (only fixed-width tables, with
headers that are **translated** — hence a split at column boundaries, and test
fixtures in French), and close to two seconds per call. Its two lists are therefore
kept in a cache and rebuilt by `jigger warm`, which `render` launches detached the
moment it finds them stale. The catalog is obtained by searching for `.`: the dot that
separates publisher from package in every identifier from the official source
(`Git.Git`, `Microsoft.PowerShell`) — that is, here, 14,401 names.

## Tests

```sh
make test-all     # Go tests + the platform's shell suite
```

The zsh widget can only be tested in a real pseudo-terminal: `tests/zpty.zsh` launches
an interactive zsh under `zpty`, types a sequence of keys, and checks what's actually
written to the screen. `JIGGER_TEST_PLUGINS=1` adds zsh-autosuggestions and
zsh-syntax-highlighting to the mix, to prove they coexist.

On Windows, `tests/conpty` plays the same role: it launches a pwsh in a
**pseudo-terminal** (ConPTY), types a sequence of keys, and **renders the screen** as
you'd actually see it, frame included. `tests/pty.ps1` (`make test-pty`) draws its
assertions from that, in the situation that matters — prompt on the terminal's last
line, as in a terminal in active use.

```sh
go run ./tests/conpty -rc setup.ps1 -keys 'winget ins\t' -screen   # the final screen
```

`tests/smoke.ps1` covers what doesn't need a console: the keys actually relayed, the
parsing of `render`'s output, the show/clear sequences, mutating-command detection,
and exporting the prompt's variables from a manufactured cache.

That suite also runs **on macOS's or Linux's pwsh**:

```sh
go build -o jigger . && pwsh -NoProfile -File tests/smoke.ps1
```

The PowerShell module is therefore developed in the same loop as everything else,
without spinning up a Windows machine — which stays indispensable for the on-screen
popup (`tests/conpty`) and for everything that goes through the real winget and scoop
CLIs. `GOOS=windows go build` and `GOOS=windows go vet ./...` check, along the way and
from any platform, that the Windows code compiles.

## Roadmap

- **Verify the winget and scoop columns** of the verb table against the real CLIs
  (`internal/winget/verbs.go`, `internal/scoop/verbs.go`) — written from memory, never
  checked against a Windows machine.
- **fish** and **bash** completion.
- Command wrapper: offer to **chain** onto the commands suggested by the manager
  ("To install …, run: …").
- Preview pane (`brew desc`, `winget show`) in the picker **and** in `jg search` /
  `jg info`.
- A **winget** package. The Homebrew tap and the scoop bucket exist; winget's
  submission process is a different animal and hasn't been started.

Deliberate non-goals of facade phase 1 — left out on purpose, not forgotten:

- **Singular verbs** (`brew services`, `winget export`, `scoop reset`…) — one table
  row each, the day one of them turns out to be missing.
- **Automatic tie-breaking** (a `JIGGER_PM_ORDER` that would choose for you between
  two managers that know the same name) — a silent arbitration between two `git`
  packages that aren't the same software would break trust in the facade. `--pm`
  remains the only escape hatch.
- **Third-party managers via subprocess** (apt, pacman… wired in without recompiling)
  — deserves its own ADR.
- **New managers** themselves — phase 1 proves the mechanism on three, not on five.

## Contributing

Bug reports are welcome on either the GitHub mirror or GitLab; code goes through GitLab,
which is the only source of truth. [CONTRIBUTING.md](CONTRIBUTING.md) explains why, and
what the code expects. For a vulnerability, see [SECURITY.md](SECURITY.md) rather than a
public issue.

## License

Apache-2.0.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
