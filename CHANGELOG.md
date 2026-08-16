# Changelog

*This changelog is kept in English only. The rest of the documentation exists in
[French](README.fr.md) too.*

All notable changes to `jigger` are recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/1.1.0/) and
[SemVer](https://semver.org/). Versions before `v0.1.6` predate this log; their detail
lives in the git history.

## [Unreleased]

### Added

- **A starship segment**, alongside the oh-my-posh one:
  [`shell/starship/brew.toml`](shell/starship/brew.toml) and
  [`windows.toml`](shell/starship/windows.toml). Three `env_var` modules per platform, on
  the same variables, the same emoji, and the same colors as the existing segments.
  Nothing to add to `format`: starship's default `$all` already includes `$env_var`. The
  block only went through oh-my-posh, even though everything it shows was already sitting
  in the environment — any prompt can read it.
- Two properties fall out for free where oh-my-posh needed a template: a module whose
  variable isn't set **doesn't render**, with no condition to write (the rule "an unset
  counter isn't exported" already covers it); and because the three modules are
  independent, a machine with **only scoop** shows its own spoon alone, where the
  oh-my-posh segment — which ties everything to the winget version — would show nothing
  at all.

### Changed

- The README's "oh-my-posh block" section becomes **"Prompt block"** and covers both
  prompts (steps 2a/2b); the anchor is fixed in the guide, and the PowerShell ordering
  rule — import jigger **after** the prompt it wraps — now names starship as well as
  oh-my-posh.

## [v0.8.0] — 2026-08-15

Until now, jigger helped write the command for each manager. It can now speak one
vocabulary for all three: `jg install fd` reaches whichever one knows `fd`, without you
having to know which. The facade **adds to** the native popup — it replaces nothing.

### Added

- **A single syntax on top of Homebrew, winget, and scoop.** Twelve verbs, the same
  everywhere: `install`, `uninstall`, `upgrade`, `list`, `outdated`, `search`, `info`,
  `source`, `pin`, `unpin`, `cleanup`, `doctor`. Each manager **declares** what it can do
  in a verb → binding table; a verb missing from that table is one it can't render, and
  saying so is a capability message, not a silent failure.
- **The manager is found from the package name.** jigger searches every available
  catalog: if only one knows the name, it wins; if several do, the picker decides.
  **Never an automatic pick**, and no setting introduces one — two packages sharing a
  name aren't necessarily the same software. `--pm` forces the choice when needed (a
  script, CI, a package too recent for the catalog, a verb with no name).
- **A uniform output for anything tabular.** `list`, `outdated`, `search`, and `source`
  render a table with adaptive columns, or JSON with `--json`. The column naming the
  manager only appears when several of them answered.
- **Everything else is relayed as is**: winget's prompts, its progress bars, and its UAC
  elevation work exactly as if the command had been typed directly, precisely because
  jigger doesn't get in the way. `--yes` accepts winget's license agreements, and is
  never implicit.
- **The popup knows the new syntax.** `jg ⇥` offers the verbs, `jg source ⇥` offers `add`
  and `rm`, `jg install g` completes across every catalog at once, and each candidate
  says which manager it comes from.
- **The `jg` alias**, set up by the zsh plugin, which arms the popup on top of it.
- **A getting-started guide** — [`docs/getting-started.md`](docs/getting-started.md):
  from installation to the first completion, with two chapters that existed nowhere
  before, "checking that it works" and a troubleshooting table. Installing through the
  **Homebrew tap** is documented there, and in the README alongside it.
- **The `tests/smoke.ps1` suite runs on macOS and Linux pwsh**, with no workaround: the
  PowerShell module is now developed in the same loop as everything else, with no need
  to spin up a Windows machine.

### Fixed

- **The PowerShell plugin failed to load outside Windows.** Its default cache directory
  was `Join-Path $env:LOCALAPPDATA 'jigger'`, evaluated **before** `JIGGER_CACHE_DIR` was
  even consulted: on macOS and Linux that variable is null, and `Import-Module` stopped
  right there. The default now follows Go's `os.UserCacheDir()` on all three platforms,
  which is the only rule that matters — the module has to read exactly the file the
  binary writes.
- **The plugin/binary version check had stopped protecting anything.** The plugin sets
  up the `jg` alias and calls on the facade's verbs, but it still only required 0.7.0: a
  0.7.0 binary passed the check and then failed to recognize those verbs — exactly the
  silent failure this check exists to prevent. Both now require 0.8.0.

### Not yet there

- **The PowerShell plugin doesn't have the `jg` alias**: only the zsh plugin arms the
  facade.
- **The winget and scoop verb tables haven't been checked against the real CLIs** —
  development happened on macOS, and only the brew column has actually run for real.
  Both files carry the warning in their header, and scoop's `search` and `source`
  parsers target an output format that has since changed: they'll likely return zero
  rows on a real Windows machine, without crashing. That's the first item on the Windows
  pass.

## [v0.7.0] — 2026-08-15

The Windows port in the previous two versions fixed, along the way, defects that had
nothing to do with winget specifically: they were just as present in the zsh plugin,
where nobody had spotted them. This version fixes them there too — the Homebrew side
catches up with the PowerShell side.

### Fixed

- **The zsh popup practically never showed up.** It was only drawn if it fit below the
  command line — but in a terminal that's actually in use, the prompt sits on the last
  line of the screen, and there's nothing below it. The plugin now makes room by pushing
  the screen up, the way any overlay picker does (`fzf --height`), and puts the cursor
  back on the command line, which has moved up by the same amount. It's the counterpart
  to 0.6.0's `New-JiggerRoom`; zle doesn't need to know any of this — it moves relative
  to wherever it left the cursor.
- **⇥ would open the full-screen picker by surprise, under zsh too.** When the popup had
  nothing to offer — no match, or no room to draw itself — ⇥ fell through to `jigger
  pick`, which drew over the prompt and waited for a key. ⇥ now hands off to the shell's
  own completion; the full-screen picker stays what you get with `JIGGER_LIVE=0` — that
  is, when you've asked for it.
- **A keystroke could hang waiting on brew.** Past 24 hours, the catalog was rebuilt
  right in the render path: `brew formulae` then `brew casks` — a good second's wait —
  the first time you typed `brew i…` in the day. The catalog is now read from the cache
  and nothing else; a stale cache is used as is and triggers a detached warm-up, which
  redoes it for the next keystroke — the rule winget already followed. On the very first
  use, the frame says so ("building the Homebrew catalog…") instead of announcing "no
  match".
- `brew list --versions` no longer runs in a keystroke's path either: this unexpected
  reliance on Homebrew's own listing has moved into `jigger warm`.

### Added

- **The zsh plugin checks the binary's version** on `source`, and reports a binary
  missing from the `PATH`. Plugin and binary go together: the plugin passes `jigger
  render` options an older binary doesn't know about — it then exits with an error, and
  the popup never shows, without a word. This is the counterpart to the check the
  PowerShell module has made since 0.6.0.
- **The zsh plugin warms the catalog**: on load (a detached `jigger warm`, just like on
  import of the PowerShell module), and after a `brew update`, `tap`, or `untap` — the
  only subcommands that change the list of known names. Detecting these mutating
  commands no longer lives in the oh-my-posh block: the catalog serves completion, not
  the prompt, so it shouldn't depend on `JIGGER_PROMPT`.
- The zpty harness **can now simulate scrolling** — until now it invariably answered
  "row 3" to the cursor-position query, which made the first bug above strictly
  invisible. Two more cases: the popup with the prompt at the bottom of the screen, and
  ⇥ with no match.

## [v0.6.0] — 2026-08-14

### Changed

- **The arrow keys drive the popup, but only once it owns the keyboard.** `↓` enters the
  list, `↑` steps back out of it on the first candidate; until the popup has focus, `↑`
  and `↓` stay the shell's history, whether the popup is open or not. Opening a list of
  candidates therefore doesn't cost access to the previous command — which was the whole
  reason for leaving the arrow keys alone until now.
- The current line **shows** the focus state: underlined in the accent color when the
  popup owns the keyboard, plain on the badges' background when it doesn't. The footer
  changes with it — `↓ browse` then `↑↓ navigate`. Without this cue, the rule would be
  invisible.
- `^N`/`^P` follow the exact same rule and remain available.
- Both plugins hand the key back to whatever it did before them: if the arrow keys are
  already held by another plugin (prefix search in history…), that plugin regains
  control outside of focus.

### Fixed

- **The screen flickered when PSReadLine shows its predictions as a list.** `ListView`
  draws in exactly the spot where the popup draws: the two fought over the same lines on
  every keystroke. jigger tucks the view away for as long as the frame is showing — the
  prediction switches back to `InlineView`, which takes up no lines of its own — and
  hands it back afterward.
- **A frame line could overflow, and the terminal would wrap it.** A package name was
  truncated to a fixed width, without accounting for the space taken on the right by the
  version and the installed dot — and a winget identifier outside the catalog, followed
  by a four-part version (`ARP\Machine\X64\{226CEF88…  6.4.0.3079  ●`), overshoots that
  by a wide margin. The wrapped line made the popup occupy twice the lines it announced:
  the frame redrew itself lower on every keystroke, and the screen filled up with
  stacked frames. The name is now sized against what's actually left, and no line ever
  leaves the frame — a guard rail cuts it short if the math gets it wrong.
- **A freshly installed package was missing from `uninstall` completion.** Refreshing the
  list of installed packages was locked inside the prompt block: without
  `JIGGER_PROMPT=1` it never ran, and the cache stayed wrong until it expired. It no
  longer depends on it.
- **The popup practically never showed up under PowerShell.** It was only drawn if it
  fit below the command line — but in a terminal that's actually in use, the prompt sits
  on the last line of the screen, and there's nothing below it. jigger now makes room by
  pushing the screen up, the way any overlay picker does, and shifts PSReadLine's anchor
  by the same number of lines: without that, the command line would redraw further
  down, preceded by that much empty space.
- **⇥ would open the full-screen picker by surprise.** When the popup had nothing to
  offer — no match, or no room to draw itself — ⇥ fell through to `jigger pick`, which
  drew over the prompt and waited for a key. ⇥ now hands off to the shell's own
  completion; the full-screen picker stays what you get with `JIGGER_LIVE=0` — that is,
  when you've asked for it.
- **`winget` or `scoop` alone announced "no match"** instead of offering the
  subcommands: since the word being typed was the command name itself, it was looked up
  among the subcommands — and none of them is called "winget".

### Added

- `jigger render --focus=true|false`: focus lives shell-side, like the selected index,
  and is passed in on every render. `render` stays stateless.
- The PowerShell module **checks the binary's version** on import. Module and binary go
  together: an older binary doesn't know the options the module passes it, exits with an
  error, and the popup never shows — without a word. It now says so.
- **A pseudo-terminal harness for Windows** (`tests/conpty`) and the suite that comes
  with it (`tests/pty.ps1`, `make test-pty`). It launches a pwsh inside a ConPTY, types
  keys, and **renders the screen** exactly as you'd see it — frame included. It's the
  counterpart to `tests/zpty.zsh`, and exists for the same reason: none of the three bugs
  above showed up anywhere but on screen; none would have been found otherwise.

## [v0.5.0] — 2026-08-14

### Added

- **Windows: winget and scoop**, with the same command line and the same popup as
  Homebrew. The first word of the line now names the manager — `brew`, `winget`, or
  `scoop` — each bringing its own subcommands, options, catalog, and insertion
  corrections. A new `internal/pm` package carries this contract; `internal/brew`,
  `internal/winget`, and `internal/scoop` fill it in.
- **PowerShell module** (`shell/jigger.psm1`), the counterpart to the zsh plugin: live
  popup, ⇥, `^N`/`^P`/`^G`, and the prompt block. Since PSReadLine offers no hook called
  on every keystroke, jigger re-registers the keys that change the line behind a relay
  that calls the original PSReadLine function before redrawing: no editing behavior gets
  rewritten. `JIGGER_KEYS_EXTRA` covers non-ASCII keys ("éèçàù" on an AZERTY keyboard).
- **`jigger warm`** rebuilds slow catalogs outside a render's path. `--installed`
  rebuilds only the installed-package lists — what changes after an install — `--all`
  rebuilds everything. `render` launches it detached the moment it finds a stale cache: a
  keystroke never waits on winget.
- **Per-manager insertion corrections**: scoop qualifies with its bucket a name that
  lives in several of them (`main/flux`, where scoop itself just prints a warning before
  picking one for you); winget wraps an identifier containing spaces in quotes.
- Windows oh-my-posh segment (`shell/oh-my-posh/windows.segment.json`) and the
  `JIGGER_WINGET_VERSION`, `JIGGER_WINGET_OUTDATED`, `JIGGER_SCOOP_OUTDATED`,
  `JIGGER_OUTDATED` variables.
- `tests/smoke.ps1`: the PowerShell module's assertion suite, which `make test-all` runs
  in place of the zsh suite on Windows.

### Changed

- Candidates are sorted **case-insensitively**. A raw sort would have put every
  capitalized winget identifier ahead of the rest — `Microsoft.PowerShell` well ahead of
  `mailspring` — while the filter itself ignores case: the list would have looked
  scrambled.
- `prompt.Status` names its two counters `Primary` and `Secondary`: they carry formulae
  and casks under Homebrew, winget and scoop under Windows. The cache file's format
  itself doesn't change — both shells' hooks read the same line.

### Notes

- The winget catalog is obtained by searching for `.`: the dot that separates publisher
  from package in every identifier from the official source. Since winget has no
  machine-readable output, its tables are split at column boundaries read from the
  header — the only method that survives translated headers and identifiers containing
  spaces. The test fixtures are real captured output, in French.
- scoop needs no cache at all: catalog, installed packages, and pending updates are all
  read straight from disk, in a few milliseconds.

## [v0.4.3] — 2026-08-14

### Fixed

- **The oh-my-posh block stayed frozen after a `brew upgrade`.** The cache only
  refreshed on TTL expiry: the counter could keep announcing ten pending upgrades for
  half an hour after they'd all been installed. jigger now spots, in `preexec`, the brew
  commands that change state (`install`, `upgrade`, `uninstall`, `update`, `tap`,
  `pin`…) and refreshes before showing the next prompt — the only wait the plugin allows
  itself, and only where the cache is *known* to be wrong. `JIGGER_PROMPT_SYNC=0` makes
  that refresh detached instead, at the cost of an accurate counter only on the prompt
  after that.
- jigger's `precmd` hook now places itself **at the front** of `precmd_functions`:
  loaded after oh-my-posh, it used to export its counters once the prompt had already
  been computed, and the block stayed one step behind no matter what else happened.

### Added

- `jigger prompt --refresh --wait` waits for the lock instead of giving up on it. This
  is what the forced refresh now uses: a lazy refresh started during the upgrade holds
  the lock for as long as brew takes to answer, and giving up there would have left the
  counter wrong in the very case this fixes.

## [v0.4.2] — 2026-08-14

### Changed

- The oh-my-posh block uses **emoji** rather than Nerd Font glyphs:
  `🍺 6.0.17  🧪 9  📦 1`. They don't depend on any particular font — the block now
  shows up everywhere, where a private-use-area glyph collapsed into an empty square
  without Nerd Font. The trade-off: an emoji carries its own color, only the counters
  follow the segment's. The Nerd Font glyphs stay documented in the README and in the
  fragment, for anyone who prefers monochrome.

## [v0.4.1] — 2026-08-14

### Fixed

- **The Homebrew icon in the oh-my-posh block never showed up.** The glyph, written
  literally in the file, didn't survive successive edits to the template: the block had
  opened on a blank since v0.3.0. Every glyph is now written as a **JSON escape**
  (`\uf0fc`…), the only form that survives editors, copy-paste, and tools that normalize
  Unicode unscathed — which is, incidentally, what oh-my-posh's own bundled themes do.

### Changed

- Each package type gets its own **icon** instead of a letter: a **flask** for formulae,
  a **box** for casks. ` 6.0.17   9   1` replaces ` 6.0.17 ⇡9F ⇡1C`. The arrow
  disappears too: since a counter never shows at zero, its mere presence already says
  "needs an update".

## [v0.4.0] — 2026-08-14

### Added

- The oh-my-posh block now counts **formulae and casks separately**:
  ` 6.0.17 ⇡7F ⇡2C`. The `F`/`C` badges are the picker's own. Two new variables,
  `JIGGER_BREW_FORMULAE` and `JIGGER_BREW_CASKS`, exported under the same rule as the
  other counters — **unset when they're zero** — so each half of the block clears itself
  on its own.

### Changed

- The segment shipped in `shell/oh-my-posh/brew.segment.json` shows the F/C breakdown.
  The total stays available in `JIGGER_BREW_OUTDATED`: the original, single-digit
  template is preserved in the README and in the file itself.

## [v0.3.0] — 2026-08-14

### Added

- **oh-my-posh block**: brew's version and the number of pending upgrades show up in the
  prompt. `JIGGER_PROMPT=1` enables a `precmd` hook that exports `JIGGER_BREW_VERSION`
  and `JIGGER_BREW_OUTDATED`; the `text` segment to paste in ships in
  `shell/oh-my-posh/brew.segment.json`. `JIGGER_BREW_OUTDATED` stays **unset** when
  everything is up to date, which hides the counter with no comparison needed in the
  template.
- `jigger prompt`: reads Homebrew's cached state (`--refresh` queries it and rewrites the
  cache, `--path` prints the file).

### Implementation details

- Since `brew outdated` costs one to five seconds, it **never** runs in the prompt's
  path: `jigger prompt --refresh` is launched detached once the cache is older than
  `JIGGER_PROMPT_TTL` (30 min by default), and the hook itself just rereads a line using
  zsh builtins — **0.03 ms per prompt, no fork**.
- Atomic cache writes (temp file + `rename`) and a refresh lock: ten open terminals
  trigger only a single call to brew. A lock older than 5 minutes is considered
  abandoned.
- If brew can't be reached, the previous cache is **kept**: a prompt never shows an
  error.
- `JIGGER_CACHE_DIR` lets you relocate the cache (and is used by the tests).

## [v0.2.0] — 2026-08-13

### Added

- **Live popup**: as soon as the current line is a `brew` command, the picker shows up
  below the prompt and filters itself with every keystroke — no more pressing ⇥ to make
  it appear. `^N`/`^P` navigate, `⇥` inserts the current candidate, `^G` closes the
  popup for the current line (`⇥` reopens it). The `↑`/`↓` arrow keys are never
  hijacked: zsh's history stays intact. (#2)
- After `brew install`, where the word is empty and the catalog holds thousands of
  entries, the frame invites you to type a letter instead of listing everything; under
  300 candidates — the installed packages — the list shows up directly. (#2)
- Settings `JIGGER_LIVE` (0 = back to ⇥-only mode), `JIGGER_ROWS`,
  `JIGGER_MIN_COLUMNS`. The popup shrinks or clears itself if the terminal is too short,
  too narrow, or silent when asked for the cursor position. (#2)
- `jigger render` subcommand: one frame of the popup, with no state and no keyboard,
  preceded by a metadata line (`count`, `sel`, `exec`, `left`). (#2)
- `tests/zpty.zsh`: the widget's test suite in a real pseudo-terminal (`make
  test-shell`), replayable with zsh-autosuggestions and zsh-syntax-highlighting loaded
  (`JIGGER_TEST_PLUGINS=1`). (#2)

### Changed

- Installed packages are now read directly from `Cellar`/`Caskroom` (~1 ms) instead of
  `brew list --versions` (~300 ms): a full call drops from ~300 ms to **~8 ms**. This is
  what makes rendering on every keystroke tenable, and it also speeds up ⇥ mode. (#2)

## [v0.1.6] — 2026-08-13

### Changed

- ⇥ on a line whose completion leaves only one candidate inserts that candidate
  directly, without showing the picker — the way zsh's own completion does on a single
  match. The popup now only opens when there's an actual choice to make. (#1)

[v0.2.0]: https://gitlab.yg-devworks.com/yves/jigger/-/releases/v0.2.0
[v0.1.6]: https://gitlab.yg-devworks.com/yves/jigger/-/releases/v0.1.6
