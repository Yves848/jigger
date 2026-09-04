# Installing jigger, end to end

*Read this in [French](fr/installation.md).*

Three turnkey procedures — one per platform — from a machine that has never heard of
jigger to a popup that answers under your prompt. Each is **self-contained**: no
cross-references, nothing to read elsewhere, copy-paste from top to bottom.

If you'd rather understand what you're typing as you go, read
[Getting started](getting-started.md) instead; it covers the same ground and explains
why. This page is for when you just want it working.

| | Time | Compiles? | Requires |
|---|---|---|---|
| [macOS](#macos) | ~3 min | yes, via the tap | Homebrew, zsh |
| [Omarchy / Arch](#omarchy--arch-linux) | ~2 min | yes | Go, zsh, pacman or yay |
| [Windows](#windows) | ~2 min | no | scoop, PowerShell 7 |

---

## macOS

**Homebrew, zsh, and the `brew` completions.**

### 1. Install

```sh
brew tap yves/cocktails https://gitlab.yg-devworks.com/yves/homebrew-cocktails.git
brew install jigger
```

The tap lives on the project's GitLab, hence the explicit URL. The formula builds the
binary (`go` comes in as a build dependency and goes away after), installs the zsh
plugin under `share/`, and sets up `brew jigger …`.

### 2. Wire it into the shell

```sh
echo 'source "$(brew --prefix jigger)/share/jigger/jigger.plugin.zsh"' >> ~/.zshrc
exec zsh
```

The order of `source` lines in `~/.zshrc` does not matter — the plugin places itself
where it needs to in zsh's hooks.

### 3. Check

```sh
jigger --version          # → jigger 0.16.0, or newer
which -a jigger           # exactly one line: two binaries on the PATH is the
                          # single most painful failure to diagnose
```

Then type — do not press anything — `brew install fire`. The frame appears on its own:

![The popup under a brew install line, on macOS](media/out/macos-01-gestionnaire-natif.png)

`↓` enters the list, `⇥` inserts, `⏎` inserts and runs, `^G` closes.

### 4. Optional — the prompt block

Homebrew's version and pending upgrades, in the prompt, counted in the background:

```sh
# oh-my-posh: merge shell/oh-my-posh/brew.segment.json into your theme
# starship:   append shell/starship/brew.toml to ~/.config/starship.toml
```

Import jigger **after** oh-my-posh or starship in `~/.zshrc`.

### Removing it

```sh
brew uninstall jigger && brew untap yves/cocktails
# then drop the `source` line from ~/.zshrc
```

---

## Omarchy / Arch Linux

**pacman and yay, zsh.** There is **no jigger package** — neither in the repositories
nor in the AUR — so the route is Go. It is not a lesser route: jigger has no runtime
dependency beyond the managers themselves.

### 1. Install

```sh
sudo pacman -S --needed go git zsh
go install gitlab.yg-devworks.com/yves/jigger@latest
```

The binary lands in `$GOBIN`, `~/go/bin` by default. Put it on the `PATH` if it isn't:

```sh
grep -q 'go/bin' ~/.zshrc || echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
```

### 2. Wire it into the shell

The plugin is not in the Go module — it comes from the repository:

```sh
git clone https://gitlab.yg-devworks.com/yves/jigger.git ~/git/jigger
echo 'source ~/git/jigger/shell/jigger.plugin.zsh' >> ~/.zshrc
exec zsh
```

Keep the clone: `git -C ~/git/jigger pull` is how the plugin gets updated, and
`go install …@latest` how the binary does. The two travel together — the plugin refuses
to load against a binary older than 0.11.0 and says so.

### 3. Check

```sh
jigger --version          # → jigger 0.16.0, or newer
which -a jigger           # exactly one line
```

Then type — do not press anything — `yay -S visual-studio`. The frame appears on its
own, `◆` for a repository package, `▣` for an AUR one, `●` for what is already
installed:

![The popup under a yay -S line, on Omarchy](media/out/omarchy-01-gestionnaire-natif.png)

**pacman and yay are two doors onto the same database**, so jigger lists your packages
once, never twice. yay drives, pacman only reads
([ADR-0007](adr/0007-pacman-lit-yay-pilote.md)) — which is why `jg install --pm pacman`
does not exist while yay is installed.

### 4. Optional — the prompt block

```sh
# oh-my-posh: merge shell/oh-my-posh/pacman.segment.json into your theme
# starship:   append shell/starship/pacman.toml to ~/.config/starship.toml
```

### Removing it

```sh
rm ~/go/bin/jigger && rm -rf ~/git/jigger
# then drop the `source` line from ~/.zshrc
```

---

## Windows

**winget and scoop, PowerShell 7.** Nothing compiles: the releases carry prebuilt
binaries.

### 1. Install

```powershell
scoop bucket add jigger https://gitlab.yg-devworks.com/yves/scoop-jigger.git
scoop install jigger
```

`scoop bucket add` takes **two** arguments — the local name you choose, then the
repository. Passing only a name makes scoop look it up in its own directory of known
buckets, and it answers `unknown bucket`.

### 2. Wire it into the shell

The bucket installs the **binary** only. The module — the part that draws the popup —
comes from the repository:

```powershell
git clone https://gitlab.yg-devworks.com/yves/jigger.git $HOME\git\jigger
Add-Content $PROFILE "`nImport-Module $HOME\git\jigger\shell\jigger.psm1"
. $PROFILE
```

If `$PROFILE` does not exist yet: `New-Item -ItemType File -Path $PROFILE -Force`.

**One ordering constraint, and it is real:** if you use oh-my-posh or starship, import
jigger **after** it.

### 3. Check

```powershell
jigger --version              # → jigger 0.16.0, or newer
Get-Command jigger -All       # exactly one line
```

Then type — do not press anything — `winget install fire`. The frame appears on its own,
`◆` for a catalog package, `▣` for an application detected outside it.

![The popup under a winget install line, on Windows](media/out/windows-01-gestionnaire-natif.png)

### 4. If you use `^R`, `^U` or other screen-taking keys

PSReadLine keeps only the **last** handler bound to a chord, so a profile that binds
`^R` after the import takes the key back and the regex toggle becomes unreachable. Offer
it the key first — it says whether it wants it:

```powershell
Set-PSReadLineKeyHandler -Chord Ctrl+r -ScriptBlock {
    if (Invoke-JiggerRegex) { return }   # $true: the popup took it
    Invoke-FzfHistory                    # $false: the key is yours
}
```

Keys that take over the screen and hand the line back want `Update-JiggerPopup`
afterwards, otherwise the frame is left behind, stale or orphaned:

```powershell
Set-PSReadLineKeyHandler -Chord Ctrl+u -ScriptBlock {
    yazi
    [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
    Update-JiggerPopup
}
```

Call it with no precaution: off a package-manager line it erases the frame and returns,
and it never throws.

### 5. Optional — the prompt block

```powershell
# oh-my-posh: merge shell\oh-my-posh\windows.segment.json into your theme
# starship:   append shell\starship\windows.toml to your starship.toml
```

### Removing it

```powershell
scoop uninstall jigger ; scoop bucket rm jigger
Remove-Item -Recurse $HOME\git\jigger
# then drop the Import-Module line from $PROFILE
```

---

## Common to all three

### Choosing which commands trigger the popup

`JIGGER_COMMANDS` decides, in both shells. `jigger` and `jg` are always added.

```sh
JIGGER_COMMANDS='brew pacman ssh'                   # ~/.zshrc, before the source
```
```powershell
$env:JIGGER_COMMANDS = 'winget,scoop,ssh,scp,sftp'  # $PROFILE, before the import
```

This is also how you turn the [SSH picker](ssh.md) off — drop `ssh`, `scp` and `sftp`
from the list.

### When it doesn't work

| Symptom | Cause, most often |
|---|---|
| No frame, ever | two binaries on the `PATH`; check with `which -a jigger` |
| The frame appears once, then never again | your terminal does not answer the cursor query fast enough; the plugin turns the live popup off for the session and `⇥` still works |
| The frame is cut off | terminal under 30 columns (`JIGGER_MIN_COLUMNS`) |
| `^R` does nothing (Windows) | another handler took the chord — see § 4 above |
| The plugin says the binary is too old | `brew upgrade jigger`, `go install …@latest`, or `scoop update jigger` |

[Getting started § 9](getting-started.md#9-when-it-doesnt-work) goes further.
