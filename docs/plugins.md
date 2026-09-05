# Plugins

*Read this in [French](fr/plugins.md).*

A plugin teaches jigger a package manager it does not ship with — without recompiling it.
You write a binary, drop a `config.json` next to it, and `brew`, `winget` and yours behave
the same way in the popup and behind `jg`.

The design is [the plugin injection plan](plans/2026-09-04-plugins-injection.md); the way
a plugin is *run* is [ADR-0008](adr/0008-execution-des-plugins.md).

## Installing one

A plugin is a folder holding a descriptor and, usually, its binary:

```
~/.config/jigger/plugins/<name>/
├── config.json      the descriptor
└── jigger-<name>    the binary (optional — $PATH works too)
```

Three folders are searched, in this order — the first descriptor for a given name wins,
so what you put in your own config always overrides a system-wide plugin:

| Folder | What it is for |
|---|---|
| `~/.config/jigger/plugins/` | yours (honours `$XDG_CONFIG_HOME`) |
| `/usr/local/lib/jigger-plugins/` | system-wide, read-only |
| `<jigger cache>/plugins/` | installed by a third party |

Then fill the caches once:

```sh
jigger warm --all
```

A plugin whose binary cannot be found is skipped in silence, and picked up again as soon
as it appears. A plugin that takes the name of a built-in manager is refused, with a line
on stderr: `brew` is `brew`.

That `warm` is not only about caches: it is what **arms the popup** on the plugin's own
word. It drops the discovered words into `plugin-commands`, deep in the cache, and the
shell plugins — zsh and PowerShell alike — read it when they load. No default could know
them: they depend on what is installed on *this* machine. Open a new shell after the
`warm`, and `git ⇥` answers.

To keep a plugin *unarmed*, without rewriting `JIGGER_COMMANDS` in full:

```sh
JIGGER_PLUGIN_COMMANDS=0
```

`git` is a command you may legitimately want left alone. On lines that are none of the
plugin's business — `git status`, `git checkout a-branch` — jigger stays quiet by itself:
the word is not one of its verbs, so it has nothing to say and draws no frame.

## The `git` plugin

It ships with jigger, in [`packaging/plugins/git/`](../packaging/plugins/git). It sees
**your local clones as packages**: installing is cloning, uninstalling is deleting the
clone, upgrading is pulling.

```sh
cp -r packaging/plugins/git ~/.config/jigger/plugins/
go build -o ~/.config/jigger/plugins/git/jigger-git ./cmd/jigger-git
jigger warm --all
```

```console
$ jg list --pm git
PACKAGE        CURRENT                SOURCE
config         feat/clavier-macarchy  https://gitlab.yg-devworks.com/yves/config.git
jigger         main                   https://gitlab.yg-devworks.com/yves/jigger.git
omarchy        fix/sddm-greeter…      https://github.com/Yves848/omarchy.git
```

The **version** of a repository is its current branch — that is what tells two states of
the same clone apart. Its **source** is the origin URL.

### Where it looks

`$JIGGER_GIT_ROOTS` (a `$PATH`-style list) has the last word. Without it: `~/git`,
`~/Projets`, `~/Code`, `~/dev`, `~/src`. It descends two levels, so both `~/git/project`
and `~/git/client/project` are found — and it stops at any folder holding a `.git`, so
submodules and worktrees do not show up as packages of their own.

### The verbs

| Command | What it does |
|---|---|
| `jg list --pm git` | the clones it found |
| `jg outdated --pm git [--fetch]` | clones behind their upstream |
| `jg search --pm git <pattern>` | filters the catalogue |
| `jg install --pm git <name>` | clones it |
| `jg uninstall --pm git <name> [--force]` | deletes the clone |
| `jg upgrade --pm git [<name>…]` | `git pull --ff-only`, all of them if you name none |

`outdated` reads the tracking ref, which only moves on a fetch: with no network it reports
what the last synchronisation knows, and can well answer *nothing* about a repository that
has moved on. `--fetch` goes and asks. It is not the default on purpose — `outdated` can
cover dozens of repositories, and a read should not go to the network unasked.

`upgrade` pulls `--ff-only`: an update must not quietly build a merge commit, nor leave
you in the middle of a conflict.

### Deleting is guarded

`uninstall` removes a folder for good, so it refuses while the clone still holds work that
deleting would lose: uncommitted changes, unpushed commits, or no remote at all. `--force`
lifts the guard, but you have to write it.

```console
$ jg uninstall --pm git jigger
jigger-git : jigger: some commits are not pushed — run again with --force to delete anyway
```

### Where clone URLs come from

Nothing is guessed. jigger will not build `https://github.com/<name>.git` out of a word
and clone whatever answers. A name resolves in this order:

1. **a URL** you passed — `jigger-git run install https://…` (any form git clones);
2. **`depots.json`**, the table you write by hand, next to the descriptor;
3. **`connus.json`**, the origins jigger remembers from the clones it has already seen.

That third one is what makes the model whole: without it, a repository deleted by
`uninstall` could never be cloned back, though jigger had just shown you its URL.

```jsonc
// ~/.config/jigger/plugins/git/depots.json
{
  "jigger":  "https://gitlab.yg-devworks.com/yves/jigger.git",
  "omarchy": "https://github.com/Yves848/omarchy.git"
}
```

`jg install` only accepts catalogue names — the guard that catches a typo before it clones
something. To clone a URL that is in neither table, either add it to `depots.json`, or
call the binary directly: `jigger-git run install <url>`.

## Writing a plugin

A plugin is a program that answers in JSON. Read verbs write **one** document on stdout
and nothing else; write verbs relay their tool as-is, terminal included, so that a
password prompt or a progress bar reaches the user.

```console
$ jigger-mine catalog
{"names":["foo","bar"],"badges":{"foo":"R","bar":"X"}}

$ jigger-mine list
[{"name":"foo","version":"1.2.3","kind":"R","source":"…"}]

$ jigger-mine run install foo        # relays, exit code says how it went
```

A package carries `name`, `version`, `available`, `kind` and `source`; only `name` is
required. `kind` is a badge: `R` for the ordinary class, `X` for the other one — the popup
paints them differently.

### The descriptor

```jsonc
{
  "name": "mine",              // the word typed in the terminal
  "version": "1.0.0",
  "cmd": "jigger-mine",        // the binary — NOT the same thing as name
  "platforms": ["linux", "darwin", "windows"],

  "verbs": {
    // "native" is the COMPLETE argv handed to the binary: jigger prefixes nothing.
    "list":      {"native": ["list"],                       "pool": "aucun"},
    "search":    {"native": ["search", "{args}"],           "pool": "aucun"},
    "install":   {"native": ["run", "install", "{args}"],   "pool": "catalogue"},
    "uninstall": {"native": ["run", "uninstall", "{args}"], "pool": "installees"}
  },

  "warmup": {
    "catalog":   {"cmd": "jigger-mine", "args": ["catalog"]},
    "installed": {"cmd": "jigger-mine", "args": ["list"]}
  },

  "parse": {"package_fields": ["name", "version", "kind", "source"], "encoding": "utf-8"}
}
```

`name` and `cmd` are **not** the same thing. `name` is the word on the line, `cmd` is the
program to run. Confusing them in a `git` plugin would run the real git.

**`pool`** says where the candidates for a verb come from, and jigger checks the arguments
against it:

| `pool` | Candidates | Effect on arguments |
|---|---|---|
| `catalogue` | everything known | must be a catalogue name |
| `installees` | installed only | must be installed |
| `aucun` | none | passed through as-is |

A search term is not a package name: `search` takes `aucun`, otherwise jigger would refuse
to search for a word that is precisely not yet a known name.

**Markers** in `native`: `{args}` spreads every argument into a single call — the usual
case; `{arg}` makes **one call per argument**, for a manager that installs one package at
a time. Flags typed on the line are prepended to the arguments, so `{arg}` would send them
in a call of their own: prefer `{args}` unless you need otherwise.

**Which verbs get parsed** is decided by the verb, not by the pool: `list`, `outdated`,
`search` and `source` have their output captured and read as JSON, everything else is
relayed. `install` draws its candidates from the catalogue but it is a write, and relaying
it is what lets an authentication prompt through.

### Rules a plugin lives by

- **Never in the render path.** `jigger render` runs on every keystroke. A plugin is
  launched by `jigger warm` and to run a verb — never to complete a line. Completion reads
  the caches `warmup` filled.
- **30 seconds, then it is killed.** A plugin that never returns must not leave `jigger
  warm` stuck behind its lock.
- **Failing is allowed, lying is not.** A plugin that exits non-zero leaves the previous
  cache in place rather than overwriting it with nothing, and what it wrote on stderr comes
  back in jigger's error message.
- **A plugin does not rewrite the line.** `Insert` returns the name unchanged: insertion
  fixes are façade bugs, not a power granted to a third party.
