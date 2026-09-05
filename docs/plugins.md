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
`warm`, and the plugin's word answers.

To keep a plugin *unarmed*, without rewriting `JIGGER_COMMANDS` in full:

```sh
JIGGER_PLUGIN_COMMANDS=0
```

A plugin may take the name of a command you type a hundred times a day and would rather
leave alone. On lines that are none of its business, jigger stays quiet by itself: the word
is not one of its verbs, so it has nothing to say and draws no frame.

## The `git` plugin

Ships with jigger, in [`packaging/plugins/git/`](../packaging/plugins/git). It does not
replace git — **it makes it convenient**. `git ⇥` offers the real subcommands, and each verb
offers what that verb expects.

```console
$ git ⇥
  add  branch  checkout  commit  diff  fetch  log  merge  pull  push
  rebase  remote  restore  stash  status  switch  tag

$ git checkout ⇥                         $ git tag ⇥
  feat/site-refonte      2 days ago          v0.18.0   27 min ago · Version 0.…
  main                5 minutes ago          v0.17.1    6 hours ago · Version 0.1…
  vieille    [behind 3] 2 days ago          v0.17.0   11 hours ago · Version 0.…

$ git push ⇥           $ git add ⇥
  github                 docs/historique/2026-09-05.md
  origin                 packaging/plugins/git/config.json

$ git commit -⇥
  -m  -a  --amend  --no-edit  --no-verify  --fixup  -S
```

Candidates are **computed in the current directory, as you type**: these are the branches of
*this* repository, not the ones a cache warmed this morning.

### It has no binary

It is a descriptor and nothing else. The binary it declares is **git itself**: jigger runs
real git to execute a verb, and real git to fill a pool (`git branch --format=…`,
`git remote`, `git ls-files --modified --others`). A helper needs no program of its own when
the command it assists already knows how to answer.

### Installing it

```sh
cp -r packaging/plugins/git ~/.config/jigger/plugins/
jigger warm --all
```

Then open a new shell. Nothing to compile.

### What it does not do

It knows only the seventeen verbs it declares. On everything else — `git bisect`,
`git worktree`, `git submodule` — jigger **stays quiet**: it has nothing to say, and an empty
frame would be worth less than silence. Your ordinary git lines are never in its way.


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

**Per-verb pools, for a command helper.** The three `pool` values above describe a package
manager. A helper — an existing command made friendlier — needs something else: *branches*
behind `checkout`, *modified files* behind `add`, *remotes* behind `push`. A verb can
therefore draw from a **named pool**, declared separately:

```jsonc
"verbs": {
  "checkout": {"native": ["checkout", "{args}"], "pool": "branches",
               "options": ["-b", "--detach"]}
},

"pools": {
  // "direct": asked for as you type, in the current directory.
  "branches": {"regime": "direct", "args": ["pools", "branches"]}
}
```

Two regimes, and the choice is not one of convenience:

| `regime` | When | What it costs |
|---|---|---|
| `cache` | pool is **large and slow** to produce, stable from hour to hour | warmed by `jigger warm`, like the catalogue |
| `direct` | pool is **small and contextual**, wrong the moment it is cached | one subprocess **per keystroke**, capped at 200 ms |

The binary asked is **always the plugin's own**: a pool declares arguments only. A
descriptor must not be able to have any program run on every keystroke.

A `direct` pool prints **one candidate per line** on standard output: `name`,
`name<TAB>badge`, or `name<TAB>badge<TAB>context` — the same convention as the installed
cache. The **context** is the popup's right-hand column, and it is what separates a helper
from plain completion: your shell already completes a branch name, it does not tell you
which one is behind or when it last moved. It is truncated at half the row: the name stays whole,
since the name is what gets inserted. If it fails or overruns the deadline it returns nothing and **nothing is drawn**: in
the render path, one error per keystroke would be worse than silence.

**`options`** lists the flags offered behind `-` for that verb. The descriptor is the only
source: jigger has no way to guess what a third-party command accepts.

The full reasoning — and what the decision costs — is in
[ADR-0009](../adr/0009-viviers-de-plugin-par-verbe.md).

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
