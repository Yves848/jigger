# Changelog

*This changelog is kept in English only. The rest of the documentation exists in
[French](README.fr.md) too.*

All notable changes to `jigger` are recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/1.1.0/) and
[SemVer](https://semver.org/). Versions before `v0.1.6` predate this log; their detail
lives in the git history.

## [v0.18.0] — 2026-09-05

Plugins were shipped in `v0.16.0` able to describe one thing only: another package manager.
That is not what a plugin is for. A plugin exists to make an **existing command** friendlier
— to show, in the popup, the subcommands, options and arguments that command actually
accepts. This release makes the descriptor able to say so, and rewrites the `git` plugin as
what it should always have been.

### Added

- **A plugin descriptor can describe a command helper.** A verb may draw from a **named
  pool** and carry its own **options**, where it previously had a single machine-wide pool
  (`catalogue`, `installees`, `aucun`) and no options at all.

  ```jsonc
  "verbs": {
    "checkout": {"native": ["checkout", "{args}"], "pool": "branches",
                 "options": ["-b", "--detach"]}
  },
  "pools": {
    "branches": {"regime": "direct", "args": ["branch", "--format=%(refname:short)"]}
  }
  ```

  A pool declares its **regime**: `cache`, warmed by `jigger warm` like the catalogue, when
  it is large and slow to produce; `direct`, asked of the plugin's binary **as you type and
  in the current directory**, when it is small and contextual. A repository's branches are
  three lines long and change by the minute — caching them would make them *wrong*, not
  fast. The reasoning, and what the decision costs, is in
  [ADR-0009](docs/adr/0009-viviers-de-plugin-par-verbe.md). (#151)

  The binary asked is always the plugin's own; a pool declares arguments only. A descriptor
  must not be able to have any program run on every keystroke.

### Changed

- **The shipped `git` plugin is now a helper for git**, and no longer a package manager
  wearing git's name. The old one saw your clones as packages: `git ⇥` offered `install`,
  `list`, `upgrade`… — none of them git commands — and the completed line read like git
  while running something else entirely. It has been withdrawn and rewritten. (#149, #152)

  ```console
  $ git ⇥          add branch checkout commit diff fetch log merge pull push
                   rebase remote restore stash status switch tag
  $ git checkout ⇥ feat/site-refonte  main
  $ git push ⇥     github  origin
  $ git add ⇥      the files you have actually modified, right now, right here
  $ git tag ⇥      v0.17.1  v0.17.0  v0.16.0   (most recent first)
  ```

  It knows only the seventeen verbs it declares; on `git bisect`, `git worktree` or
  `git submodule`, jigger stays quiet. Your ordinary git lines are never in its way.

  It also has **no binary of its own**: it declares `git` as its command, so real git both
  runs the verbs and fills the pools. Installing it is a `cp` and a `warm` — nothing to
  compile — and `cmd/jigger-git`, 950 lines, is gone with the old design.

### Fixed

- **Publishing a release no longer loses a race against the mirror.** The `github:` job
  starts within a second of the tag, while GitLab → GitHub replication is asynchronous:
  GitHub refused with a `422` to create a release for a tag it did not know yet, and the tag
  pipeline came out red. `tools/publier-github.sh` now waits for the tag — a minute at most,
  then an explicit failure telling you how to force the mirror to sync. `v0.17.0` had won
  the very same race six hours before `v0.17.1` lost it: the chain was not reliable, it had
  been lucky. (#148)

## [v0.17.1] — 2026-09-05

### Fixed

- **The `git` plugin no longer confuses two clones of the same repository.** With
  `~/git/app` and `~/git/bsca/app` side by side, `jg list --pm git` printed two rigorously
  identical lines, and `jg uninstall app` **deleted one of them for good** with nothing
  saying which. The existing guard — refusing while a clone holds uncommitted work,
  unpushed commits, or has no remote at all — only narrowed the damage to a *clean* clone,
  which is precisely the one you delete without thinking.

  Same-named clones now carry a name qualified by their path relative to the root, **root
  included** — `git/app` and `git/bsca/app`. Including the root is not cosmetic: without
  it the clone sitting at the root would keep the bare name `app` and would still be the
  one silently picked. `uninstall` and `upgrade` accept a qualified name or a path, and
  refuse a bare ambiguous name by naming the candidates rather than answering "not
  cloned", which would be both false and puzzling when jigger has just shown two of them.
  Names that are already unique are left alone. (#145)

  ```console
  $ jg uninstall Moteur_IT_Csharp.Net --pm git
  Moteur_IT_Csharp.Net désigne 2 clones : git/Moteur_IT_Csharp.Net,
  git/bsca/Moteur_IT_Csharp.Net — donnez celui que vous visez
  ```

- **A repository with no commits keeps its branch.** Freshly `git init`-ed, it came out
  with no version at all, though it really is on `master` — and a `git` package's *version*
  **is** its current branch. The branch is now read off the *reference* rather than off the
  commit: `rev-parse --abbrev-ref HEAD` requires `HEAD` to point at a commit, which an
  unborn branch does not. Detached heads still show their short revision. (#145)

## [v0.17.0] — 2026-09-05

Third-party plugins shipped in `v0.16.0` were complete on the binary side and invisible on
the shell side: the popup never opened on a plugin's own command word. This release closes
that gap, and fixes the two things that made closing it unsafe.

### Added

- **A plugin arms the popup on its own word.** Install the `git` plugin, run
  `jigger warm --all`, open a new shell, and `git ⇥` answers — in zsh and in PowerShell
  alike. Until now the shell plugins watched a **static** list of commands, and no default
  could ever hold a word that depends on which plugins are installed on *this* machine.
  `warm` now drops the discovered words into `plugin-commands` in the cache, and the shell
  plugins read that file when they load — with a shell builtin, never by spawning jigger,
  since neither side will pay a fork at shell startup. The file is rewritten on every
  `warm`, empty included, so an uninstalled plugin stops being armed. (#143)

- **`JIGGER_PLUGIN_COMMANDS=0`** turns that off without forcing you to rewrite
  `JIGGER_COMMANDS` in full. This is what [ADR-0005](docs/adr/0005-completion-sans-facade.md)
  asks for: `git` is a command you may legitimately want left alone, and taking that choice
  away would be a defect. (#143)

### Fixed

- **A verb a plugin does not declare no longer offers packages.** With the `git` plugin,
  `git checkout ` was served the list of cloned repositories — **marked executable**, so ⏎
  would have completed *and run* `git checkout <some repository>`. Same on
  `git push origin ` and inside the message of `git commit -m `. The cause was in the
  completion path: whether a word was a package argument depended only on its **position**,
  never on whether the previous word was one of the manager's verbs. (#142)

  The fix deliberately does **not** tighten every manager. `Subcommands()` does not carry
  the same weight everywhere: a plugin's verbs come from its `config.json` and the list is
  complete by construction, whereas Homebrew declares twenty-five hand-picked subcommands
  out of a hundred — `brew fetch fir` is valid and its argument really is a formula. The
  new `pm.Exhaustif` contract is optional, like `Bindings` and `Elevateur`: whoever does
  not declare it keeps the permissive behaviour. Only plugins declare it. (#142)

- **The first word of a plugin no longer draws an empty frame.** `git st`, `git status`,
  `git push` opened an "no candidates" frame on *every keystroke* — the very defect
  [ADR-0006](docs/adr/0006-silence-sur-catalogue-vide.md) had already fixed for `ssh`. A
  manager with exhaustive verbs, none of which start with the word being typed, now stays
  quiet. A built-in still shows its frame: `winget zzz` has to say it doesn't know. (#143)

## [v0.16.0] — 2026-09-04

### Added

- **jigger accepts third-party package managers**, shipped as a separate binary and
  discovered through a `config.json` descriptor — no recompiling jigger. Three folders are
  searched — `~/.config/jigger/plugins/`, `/usr/local/lib/jigger-plugins/`, the cache — the
  first descriptor found for a given name winning, so what you put in your own
  configuration always overrides a system-wide plugin. A discovered plugin behaves like a
  built-in one: it shows up in the popup, completes from its caches, and answers the
  façade's verbs. A plugin taking the name of a built-in manager is refused with a line on
  stderr — two `brew` in the list would run the line twice, and routing has no way to tell
  them apart. (#125)

  How a plugin is *run* was the one thing the injection plan had left too thin, and it is
  now settled by [ADR-0008](docs/adr/0008-execution-des-plugins.md): the façade substitutes
  the binary into the **native execution path** rather than wrapping output in a JSON
  envelope. The envelope was tempting — one format across the whole protocol — but it
  cancels the terminal relay, on which authentication prompts, progress bars and UAC
  elevation all depend. A plugin therefore inherits the relay, the exit-code reading and
  the elevated `Rejeu` of ADR-0004 for free, and the plugin-specific branch in
  `ExecuterAvec` went from forty lines to four. (#124)

- **A `git` plugin**, the first one shipped, in `packaging/plugins/git/`. It sees your
  local clones as packages: installing is cloning, uninstalling is deleting the clone,
  upgrading is pulling. A repository's *version* is its current branch — what tells two
  states of the same clone apart — and its *source* is the origin URL. Roots come from
  `$JIGGER_GIT_ROOTS`, otherwise the usual places, and the walk stops at any folder holding
  a `.git` so submodules and worktrees do not surface as packages of their own. (#126)

  Two guards carry most of the care. **No URL is ever guessed**: building
  `https://github.com/<name>.git` out of a word amounts to cloning whatever answers, so a
  name resolves through an explicit URL, then `depots.json` written by hand, then
  `connus.json` where the plugin remembers the origins of clones it has seen — that last
  one being what closes the `uninstall` → `install` loop. And **`uninstall` refuses while
  the clone still holds work that deleting would lose** — uncommitted changes, unpushed
  commits, or no remote at all; `--force` lifts the guard, but you have to write it.

  `outdated` reads the tracking ref, which only moves on a fetch: with no network it
  reports what the last synchronisation knows, and `--fetch` goes and asks. Not the default
  on purpose — a read can cover dozens of repositories and should not go to the network
  unasked. `upgrade` pulls `--ff-only`, so an update never quietly builds a merge commit.

- **Plugin documentation**, [`docs/plugins.md`](docs/plugins.md) and its French mirror:
  installing one, writing one — descriptor, JSON protocol, `pool`, argv markers — and the
  rules a plugin lives by, starting with the one that matters most: never in the render
  path.

- **The documentation shows jigger instead of describing it.** Until now every frame in
  the guides was ASCII art, hand-kept in step with the binary. There are now real
  recordings and real screenshots, produced by a protocol that is written down and
  reproducible: `docs/captures.md`. Three scenarios — the native manager, the `jg`
  facade, the SSH picker — captured in a decor frozen down to the palette, so the macOS
  image and the Arch one differ by the manager answering and by nothing else. One pass
  renders the GIF for the guides, the MP4 for the site, and the still frame extracted
  from the recording itself, at an instant computed from the script rather than chosen
  by hand.

- **The `^R` regex toggle is documented and shown.** The key existed and worked; the
  documentation only ever asserted it, and the single image of it was a Windows one — so
  a reader on macOS, the site's default, saw the feature claimed and never demonstrated.
  There is now a macOS recording (`macos-04-regex`, and an Arch tape waiting for a
  reachable machine), and the READMEs' key tables — which omitted `^R` entirely — list it.

  What the toggle actually does is written down for the first time, because none of it is
  guessable from the frame: the pattern is **not anchored**, so switching often *widens*
  the list before you narrow it (`fire` starts matching `arrayfire`); the case is ignored
  in both modes; it applies to **package names only**, verbs, subcommands and flags
  keeping prefix matching; and a pattern that does not compile matches **nothing**, the
  frame saying so rather than spilling 16 000 entries over a missing bracket. The
  full-screen picker makes the opposite choice on that last point — an invalid pattern
  keeps every line and the filter line carries the warning — and the contrast is now
  stated instead of being left for a reader to discover twice.

- **The capture protocol is executable, not prose.** `docs/media/generer-tapes.sh` writes
  the nine VHS tapes from a single preamble — the three platforms cannot drift apart by a
  manual copy — and the tapes it produces are standalone: one file to carry onto the
  target machine, no `Source` directive, nothing but VHS to install. `make media` and
  `make media-tapes` drive it; `docs/media/capturer.sh` deduces the platform from
  `uname`, and `docs/media/capturer.ps1` covers Windows.

  The one surprise is documented where it bites: **the live popup does not survive a
  recorder.** It asks the terminal for the cursor position (`ESC[6n`) and waits 150 ms;
  after two misses `_jigger_row` turns the live popup off for the session. VHS goes
  through ttyd and an `xterm.js` in a headless Chrome, and the round trip does not fit.
  A capture made naively therefore shows a terminal with **no popup at all**, and reads
  as jigger doing nothing. The fix is to stack `VHS → tmux → zsh`: tmux is a real local
  emulator and answers from its own state. Windows needs none of this — the PowerShell
  module reads the cursor through PSReadLine's `GetBufferState`, never through the
  terminal.

- **A turnkey installation page**, [`docs/installation.md`](docs/installation.md) and its
  French mirror: three self-contained procedures, one per platform, from a machine that
  has never heard of jigger to a popup that answers. No cross-references — that is what
  makes them turnkey — and each ends on how to remove it again. The guides keep the
  explanations and point at it for readers in a hurry.

- **A page for the SSH picker**, [`docs/ssh.md`](docs/ssh.md) and its French mirror. It
  was scattered across a table row, a README bullet and a paragraph of § 7; it now has
  the `Include` resolution, the patterns left out (`*`, `?`, `!`), what a `Match` block
  does to the `Host` block before it, why `scp` gets a colon and the other two don't, and
  a list of what the picker deliberately does not do — no `known_hosts`, no remote paths,
  no options, and never a connection opened.

- **The SSH captures show invented servers.** A picture of the SSH picker is a picture of
  someone's `~/.ssh/config`, hence of their infrastructure.
  `docs/media/fixtures/home/.ssh/config` carries six fictional hosts and `capturer.sh`
  points `HOME` at it for the duration. The fixture doubles as a test: the host `atelier`
  comes from an `Include` and must appear, `Host *.exemple.net` and `Host *` are patterns
  and must not.

### Fixed

- **The `brew install fire` and `jg` frames were finally captured on a Mac.** v0.15.0
  recorded that both were still wanting one — brew had to be installed for the first, the
  twelve verbs declared for the second — and updated only the version banner on the
  grounds that a real recapture was owed. It is done, and they are photographs now rather
  than drawings.

## [v0.15.0] — 2026-09-02

### Added

- **The site now shows how jigger works.** A `#fonctionnement` section on
  <https://jigger.yg-devworks.com/> carries a hand-written SVG: the shell on top, jigger
  in the middle, the six managers at the bottom, and the three channels that link them —
  completion reading catalogues, the facade running a translated command, the prompt
  counting upgrades in the background. Every arrow is labelled, the colour says which
  channel, and the dashes say what is asynchronous. No image, no script: it is styled by
  classes rather than by attribute, because the vhost's CSP sets `style-src 'self'`, and
  every label carries a `data-i18n` key — so `verifier.sh` checks the diagram's French
  exactly as it checks the prose. (#99)

- **The SSH picker reached the site**, one version after reaching the binary: a
  paragraph in the popup section says that `ssh`, `scp` and `sftp` offer the hosts of
  `~/.ssh/config`, that jigger never opens the connection, and that it shows nothing at
  all when it has no host to offer. (#99)

### Changed

- **The site had stayed at three managers.** It now names pacman alongside Homebrew,
  winget and scoop — page title, social metadata, hero — and the facade section becomes
  "four dialects, one vocabulary", with a line on [ADR-0007](docs/adr/0007-pacman-lit-yay-pilote.md):
  pacman and yay are two doors onto the same database, so `jg` lists your packages once,
  never twice. The prompt section stops promising a Homebrew block alone, now that brew,
  pacman and Windows segments all ship. (#99)

### Fixed

- **The frames in the documentation and on the site show the current version.** They
  announced `jigger 0.10.0`, and the Open Graph image `jigger 0.9.0` — a lag reported but
  left alone at v0.13.0, v0.14.0 and v0.14.1, each time on the same grounds: a real
  recapture, not a touch-up of the number. That held as long as nobody knew whether the
  frame itself had moved. It was measured: rendered at the documents' exact width
  (`--cols 60`), `jigger render` produces a frame *identical to the byte* to the 0.10.0
  one — border, title line, badge column, blank row, footer. Only the banner was wrong, so
  only the banner changed. § 5 of both guides also gains a frame genuinely captured on an
  Arch machine, `yay -S visual-studio`: ◆ a repository package with its version and the ●
  that says it is installed, ▣ an AUR one, and the **qualified** insert
  `omarchy/visual-studio-code-bin`. The `brew install fire` and `jg` frames still want a
  Mac — brew has to be installed for one, and the twelve verbs have to be declared for the
  other. (#95)

- **`jigger demo` shows the machine's manager, not the operating system's.** On Arch it
  announced "brew install" and listed six Homebrew formulae — a manager that is not there.
  `runDemo` picked between two previews on `runtime.GOOS == "windows"` and fell back to
  brew everywhere else; the v0.14.0 pacman module made that preview wrong **without ever
  touching one of its lines**. The rule already existed elsewhere: `managers.Default()`
  answers exactly the question `runDemo` was asking, so the preview now asks it instead of
  restating it. pacman's title is `pacman -S`, not `pacman install` — its operations are
  flags, and a preview showing a grammar the popup does not show would lie twice. (#97)

## [v0.14.1] — 2026-09-02

### Fixed

- **The "Getting started" guide documents pacman and yay.** v0.14.0 shipped the module
  and updated both READMEs, but not the guide: a reader on Arch still found a
  "macOS, Linux → Homebrew" table, an installation section that did not apply to them,
  and not a word about the two managers jigger was in fact completing on their machine.
  Both languages are back in step, and four things the guide had never said are now in
  it. **There is no jigger package on Arch** — neither in the repositories nor in the
  AUR — so the route is Go or the sources, and the `$(brew --prefix jigger)` paths have
  no equivalent there. **yay drives, pacman only reads**
  ([ADR-0007](docs/adr/0007-pacman-lit-yay-pilote.md)) — which is what explains, from the
  user's side, why `jg install --pm pacman` does not exist while yay is installed.
  **`shell/oh-my-posh/pacman.segment.json` and `shell/starship/pacman.toml`** shipped in
  v0.14.0 without a single document pointing at them, along with the `JIGGER_PACMAN_*`
  naming they depend on. And **`JIGGER_CACHE_DIR` announced no path on Linux** at all.
  (#93)

## [v0.14.0] — 2026-09-02

### Added

- **pacman and yay are completed — repositories and the AUR.** jigger's first Linux
  package manager. Type `pacman ` or `yay ` and the popup offers the operations (`-S`,
  `-Syu`, `-Rns`, `-Qu`…); type one and the packages follow — the whole catalogue behind
  `-S`, only what is installed behind `-R` and `-Q`. A diamond marks a repository
  package, a square an AUR one.

  pacman is the only manager whose operations are **flags**, so the provider declares the
  same list on both sides of the engine's `-` test: `pacman ⇥` and `pacman -⇥` open the
  same list. Installed packages are read straight from `/var/lib/pacman/local` — no
  subprocess on the keystroke path, the same trick as Homebrew's Cellar. (#89)

- **`yay -S` inserts a shared name qualified** — `yay -S omarchy/1password`. Without it,
  yay opens its "repository or AUR?" menu in the middle of what jigger just inserted. 121
  names are carried by both on an Omarchy machine, nearly all of them from the `omarchy`
  repository. (#89)

- **`jg install`, `jg outdated`, `jg list`, `jg search` work on Arch.** Where yay is
  installed it drives everything, including the mutating verbs — it handles its own
  `sudo`. Where it is not, pacman declares the four read verbs, the ones that do not need
  root. jigger still elevates nothing. See
  [ADR-0007](docs/adr/0007-pacman-lit-yay-pilote.md). (#89)

- **The prompt block counts repository and AUR upgrades separately** —
  `JIGGER_PACMAN_VERSION`, `JIGGER_PACMAN_REPOS`, `JIGGER_PACMAN_AUR`. The split is the
  point: AUR upgrades are rebuilt from source, repository ones are downloaded, and seeing
  them apart tells you whether `yay -Syu` will take ten seconds or ten minutes. Ready-made
  segments in `shell/oh-my-posh/pacman.segment.json` and `shell/starship/pacman.toml`.
  Counting is done by `checkupdates` when it exists — never by `pacman -Sy`, which would
  leave an Arch install in the state that breaks the next install. (#89, #90)

### Changed

- **`JIGGER_COMMANDS` now defaults to `brew pacman yay ssh scp sftp` in zsh.** The default
  does not depend on the distribution: a `pacman` typed on macOS is still completed, at
  worst against an empty catalogue — exactly like a `brew` typed on Windows. (#90)

- **The prompt block names its variables after the manager that actually filled them.** On
  a machine with pacman the figures used to come out as `JIGGER_BREW_*`: calling a count
  of repositories "formulae" is a lie in someone's prompt. The status file format is
  unchanged, and neither the PowerShell hook nor the brew and windows segments are
  touched. (#90)

- **The catalogue is merged, deduplicated and sorted at warm time** rather than on every
  keystroke, and `pm.NewCatalogDe` sizes the tables up front. On a machine with the AUR
  catalogue loaded — 134 000 names — this takes `jigger render` from 70 ms to 28 ms per
  keystroke, pacman alone from 13 ms to 7 ms, and `warm --all` from 1 058 ms to 710 ms.
  Nothing changes for brew, winget or scoop. (#89)

- **Subcommand completion now matches case-insensitively**, as option completion already
  did. Without it a `-S` operation was never found behind a word the engine had
  lowercased to `-s`. (#89)

### Fixed

- **The popup header showed the subcommand lowercased** — `pacman -rns`, which is not a
  valid pacman command. It now shows what was typed; the lookup key stays lowercased,
  which is what the managers' tables expect. (#89)

## [v0.13.0] — 2026-09-02

### Added

- **The popup now offers the servers of your `~/.ssh/config`.** Type `ssh`, `scp` or
  `sftp` and the hosts appear, with the address to the right of the name — where package
  managers show the version. `Include` fragments are followed. Names come before
  addresses, and addresses are ordered numerically among themselves. `scp` appends the
  colon on insert: without it, `scp file archlight` would silently copy to a **local**
  file named `archlight`. jigger never *runs* `ssh` — it completes the line you will run
  yourself. See [ADR-0005](docs/adr/0005-completion-sans-facade.md): the completion
  contract is not reserved for package managers (#82).
- **A PowerShell profile can take `^R` back without losing the regex toggle.**
  `Invoke-JiggerRegex` switches the popup to regex mode **and says whether it took the
  key**, so a profile that binds `^R` after the import — an fzf history search,
  typically — offers it the key first and keeps its own fallback for every other line.
  `Update-JiggerPopup` puts the frame back in agreement with the line, or clears it:
  that is what the keys which take over the screen and hand the line back (`^U` to a file
  explorer, `^D` to a drive picker) were missing — their handler replaced ours and the
  frame stayed behind. Both are safe to call unconditionally and never throw. The module
  cannot do this by itself: PSReadLine keeps only the **last** handler bound to a chord,
  and hands back a script block's description, never the block. Under zsh, where
  `bindkey` names the widget, the plugin has always given `^R` back on its own — this
  closes the asymmetry, and with it the A-19 promise that nothing is confiscated from
  your shell (#85).

### Changed

- **`JIGGER_COMMANDS` now includes `ssh`, `scp` and `sftp` in its default value.** If you
  override the variable in your profile, nothing changes until you lengthen it yourself —
  the same rule the `jg` alias followed in v0.12.0. Under zsh the list of intercepted
  commands also moved to the `: "${JIGGER_COMMANDS:=…}"` idiom of the other settings; it
  used to be hard-coded, and nothing could turn interception off (#83).

### Fixed

- **A `~/.ssh/config` reduced to a single `Host *` block** — what Apple's documentation
  has you write on macOS — left a "no match" frame redrawing on **every keystroke** of
  any `ssh`, `scp` or `sftp` line. Silence was decided on the file existing; it is now
  decided on the catalog being empty ([ADR-0006](docs/adr/0006-silence-sur-catalogue-vide.md))
  (#84).
- **A `Match` block gave its `HostName` to the preceding `Host` block**, so the popup
  could name one machine for another (#84).
- **An end-of-line comment** — `Host pve  # the living-room proxmox` — offered `living`,
  `room`, `proxmox`, `the` and `#` as hosts. OpenSSH 10.2p1's behaviour was measured with
  `ssh -G` before the fix: `#` opens a comment only where it **opens a word** (#84).

### Security

- **A GitHub token was committed in clear** in `docs/historique/2026-08-16.md`, in a
  public repository — `repo` and `workflow` scopes on the `Yves848` account. The
  `UserPromptSubmit` hook copies every prompt into the day's journal, and the token,
  pasted into the conversation on 16 August to set the GitHub mirror up, was written
  there as-is. It was found while preparing the mirror sync, which was suspended rather
  than carry the token to GitHub. **The token is revoked**, and removed from the file by
  an ordinary commit. It remains reachable in `955e60f`, the only commit that introduces
  it: published history is not rewritten — revocation is what makes a token harmless, not
  its removal. Nothing filters secrets at capture time yet; that is A-23 (#76).

## [v0.12.0] — 2026-08-17

### Added

- **`jigger config`** — a settings screen in three groups: what takes effect right away,
  what waits for the next shell, and what jigger reads without owning. Every line shows
  **where its value comes from**. A configuration file in `key = value` form now lives
  where your system expects it; `--path` prints its location, `--list` shows the table
  without the screen (#62).
- **Catalogue lifetimes are yours to set.** brew and winget declare theirs and the screen
  displays them without knowing anything about either. They used to be hard-coded to 24
  hours (#62).
- **A command refused for lack of rights offers to replay itself elevated.** jigger
  intercepts nothing: it lets the command run through, reads its exit code afterwards,
  names the cause and asks. It **never elevates without an explicit yes** — the line open
  by default is "cancel". With no terminal — a pipe, a script, a scheduled task — there is
  no question: the exact line to replay is printed and the original exit code is returned.
  Windows only for now (#67).
- **The `jg` alias exists under PowerShell too.** `Remove-Module` takes it back, which is
  the way out for anyone already attached to their own. It points at the binary the module
  resolved, not at the word "jigger" — with `JIGGER_BIN` set to a build tree, `jg` and the
  popup would otherwise talk about two different executables (#71).

### Changed

- **`↵` completes the line and runs it in one keystroke**, as long as a candidate is
  highlighted: `winget li ↵` runs `winget list`. Pressing `↵` means "go" — the line leaves,
  completed if something was highlighted, as typed otherwise. jigger doesn't judge whether
  it is correct in your stead, and `^G` still runs exactly what you typed (#66).
- The frame's footer now carries four pills — `⇥` insert, `↩` run, `↓` browse, `^G` close.
  Two distinct gestures deserve two labels (#66).
- **The environment still has the last word** over the configuration file. That is why the
  screen shows provenance: without it, it would display a value your machine doesn't
  apply. The plugins ask the binary for the settings (`config --export`) instead of reading
  the file, so **precedence has a single implementation**, in Go (#62).
- The PowerShell façade arms the popup for `jg` and `jigger` in all cases, instead of
  relying on the default value of `JIGGER_COMMANDS` — extending that default would have
  changed nothing for anyone who copied "winget,scoop" into their profile, as the
  documentation showed them to for three versions (#71).
- **The Homebrew prompt block is back to emojis** — beer, microscope, package — after a
  detour through Nerd Font glyphs. The glyphs fixed the width defect below but demanded a
  font: without it the prompt shows boxes. Only one of the three was at fault, so one
  substitution was enough (#65).

### Fixed

- **The Homebrew prompt block made a character disappear from your command line** on
  macOS. `jigger --version` showed up as `jigger -version` while the command actually run
  was the right one — a purely visual failure, and one that only appeared when there were
  formulae to upgrade. The cause was the glyph jigger shipped, not jigger: macOS's
  `wcwidth()` predates Unicode 11 and reports zero columns for the test tube `U+1F9EA`,
  which the terminal draws on two. zsh's cursor arithmetic drifted by two columns and ZLE
  wrote in the wrong place at the right edge (#63).
- **The Windows prompt block carried the same defect** — the window `U+1FA9F` counted for
  zero, drawn on two. It becomes a laptop. Not observed under PowerShell, whose width
  arithmetic differs from ZLE's, but the segment is text nothing stops you from pasting
  into another prompt (#64).
- `$script:Commands` was fed a pipeline in the PowerShell module, so a bare string when
  `JIGGER_COMMANDS` held a single name — the façade's `+=` would have concatenated it
  instead of extending the list (#71).

## [v0.11.0] — 2026-08-17

### Added

- **Long listings became navigable.** `list`, `outdated`, `search` and `source` now open
  in a paged view when the output is a terminal **and** the rows don't fit on screen.
  Type to filter, `⇥` to select a row, `^A` to select everything the filter leaves, `↵` to
  print what you picked, `^G` to leave with nothing (#56, #60).
- **`--select`** opens that view even when the output goes down a pipe, drawing on the
  terminal and sending only the chosen names — one per line. `jg install $(jg search fd
  --select)` works the way you would hope. `JIGGER_PAGER=0` turns the automatic view off
  (#56).
- **A regex filter, on `^R`.** Every surface that filters — the live popup, the
  full-screen picker, the paged view — switches between plain-text and regular-expression
  matching on `^R`, and shows which mode it is in (#58, #61).

  The key is **not taken away from your shell**: outside a package-manager line, `^R`
  still runs your shell's reverse history search. That is the same rule the arrow keys
  have always followed.
- **`JIGGER_BIN`** tells the zsh plugin which binary to call — the PowerShell module has
  always had it. Homebrew's `bin` usually comes before `~/.local/bin`, so without this a
  freshly built jigger could never be the one that runs (#60).

### Changed

- **The plugins now require a 0.11.0 binary.** They pass `render --regex`, which older
  binaries reject — a popup would vanish without a word. Plugin and binary go together:
  upgrade both, or the plugin says so at shell startup and stands down.
- Nothing changes when the output isn't a terminal. `jg list | grep` prints the same plain
  table it always has, byte for byte, and `--json` is never paged — it is a machine
  contract.
- **Windows installs through scoop.** `scoop bucket add jigger …` then
  `scoop install jigger` — a prebuilt binary, nothing to compile. The documentation used
  to say no scoop package existed; it does now (#52).
- The captured frames in the README and the guide show the version actually published
  (#51).

## [v0.10.0] — 2026-08-16

### Added

- **Prebuilt binaries with every release.** A tag now builds four targets — macOS on
  Apple Silicon and Intel, Windows x64, Linux x64 — and attaches them to the GitLab
  release along with a `SHA256SUMS`. Installing jigger no longer requires having Go
  (#48). Each archive carries the binary, the Apache-2.0 licence and the README.
- **A site of its own**, at <https://jigger.yg-devworks.com/> — what the popup does, the
  twelve verbs of `jg`, and the three ways to install it, in English and French (#47).
  Its source lives in this repository under `website/`, so the page and the documentation
  change under the same review.
- **`CONTRIBUTING.md` and `SECURITY.md`** — where bug reports go, where code goes, what
  the code expects, and how to report a vulnerability without opening a public issue
  (#50).

### Fixed

- Three completion tests passed on a machine with Homebrew installed and failed
  everywhere else: they asked the machine instead of the facade. With no package manager
  present, jigger proposes no verbs — the behaviour was right, the tests were not (#49).
  Found by the first CI run, in a container with no package manager at all.
- The README and the getting-started guide claimed **Go ≥ 1.24**; `go.mod` has required
  **1.26.5** for a while. The claim now matches the file.

## [v0.9.0] — 2026-08-16

### Added

- **jigger speaks English and French.** Every string a user can see — the popup, the
  facade's messages, the table headers, the usage and flag help, the two plugins' own
  warnings — goes through a single catalogue
  ([`internal/i18n/catalogue.go`](internal/i18n/catalogue.go)) that carries both
  languages side by side, one line per key. A missing translation falls back to English
  rather than leaving a blank.
- **`JIGGER_LANG`** picks the language: `en` or `fr`. Unset, jigger reads `LC_ALL`,
  `LC_MESSAGES` and `LANG` — then, under Windows, the system's own language — and settles
  on English for anything it can't translate. This is the switch that gives French back
  to someone whose shell runs in English. The binary, the zsh plugin and the PowerShell
  module apply the same rule, so the popup and the plugin messages can't disagree; the
  two test suites assert that agreement.
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

- **English is now the default language.** A shell that says nothing about its locale
  used to get French; it now gets English. Nothing is lost: `JIGGER_LANG=fr`, or any of
  the usual locale variables set to French, restores exactly the previous wording — the
  French strings didn't move by a single byte, and a golden bench of 480 rendered popups
  ([`tests/render-golden.sh`](tests/render-golden.sh)) was written to prove it.
- **The documentation is published in English**, with the French kept in full alongside
  it: [`README.md`](README.md) and [`docs/getting-started.md`](docs/getting-started.md)
  are the English originals, [`README.fr.md`](README.fr.md) and
  [`docs/fr/getting-started.md`](docs/fr/getting-started.md) their French counterparts.
  Each page links to the other at the top. This changelog stays English-only.
- The README's "oh-my-posh block" section becomes **"Prompt block"** and covers both
  prompts (steps 2a/2b); the anchor is fixed in the guide, and the PowerShell ordering
  rule — import jigger **after** the prompt it wraps — now names starship as well as
  oh-my-posh.

### Fixed

- **`jg source` and `jg list` never showed anything from scoop.** Both parsers returned
  **zero rows without an error** — exit code 0, "nothing to report" — so on a Windows
  machine where winget and scoop live side by side, half the table went missing in
  silence. Two causes, and the first was foreseen by nobody: PowerShell colors the header
  and the dashes line of its tables **even when the output is redirected**, and it is the
  dashes line that marks the start of the table — wrapped in ANSI escapes, it no longer
  looked like dashes, so no parser ever entered the table. The second: `Format-Table` pads
  each column to its widest cell, so the widest row has **a single space** before the next
  column — splitting on "two spaces or more" read `git-interactive-rebase-tool 2.4.1` as
  one field, and got it wrong on exactly the row that sets each column's width. Columns
  are now cut at the positions read from the dashes line, and the test fixtures are real
  captures from a Windows machine rather than hand-written guesses.
- **The documentation sent Windows users into a wall.** The guide announced `make install`
  as "the route to take on Windows"; that target calls `install(1)`, a POSIX tool Windows
  doesn't have, and `make` isn't shipped there either. `install-windows.ps1` now does the
  job — build, then a **scoop shim** when scoop is around, or a **copy** into
  `%USERPROFILE%\bin` added to the user `PATH`. The PowerShell module was giving the same
  bad advice in its own version warning; it now points at the script.
- Three of the four caveats published with v0.8.0 are lifted: `winget pin add`/`remove`,
  `winget source add`/`list`/`remove` and scoop's `update *` are now checked against the
  real CLIs, on captured help output. `cleanup *` and `bucket rm` remain unproven.

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
