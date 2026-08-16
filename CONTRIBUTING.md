# Contributing to jigger

Thanks for taking the time. This project is small, and that shows in how it is run —
here is the short version of where things go.

## Where the project lives

jigger is developed on a **self-hosted GitLab**, which is the only source of truth:

<https://gitlab.yg-devworks.com/yves/jigger>

The GitHub repository is a **mirror**, pushed from GitLab. It is there so people can find
the project and report problems where they already are.

| | GitHub mirror | GitLab |
|---|---|---|
| Read the code | yes | yes |
| Report a bug, ask a question | **yes** | yes |
| Propose code | no — see below | **yes** |

## Reporting a bug

Open an issue on either side. What helps, in rough order of usefulness:

1. The **exact command line** you typed, and what appeared instead of what you expected.
2. `jigger --version`, your OS, your shell (zsh or PowerShell), and which package
   managers are installed.
3. `jigger doctor`, which reports what jigger sees of your setup.
4. For anything involving the popup: whether it appears at all, and whether `jigger
   render --line "brew ins" --cols 80` prints a frame on its own.

If jigger did something to your system that you did not ask for, say so plainly and it
gets priority over everything else.

## Proposing code

**Pull requests on GitHub cannot be merged** — the mirror is pushed from GitLab, so
anything committed there is overwritten on the next sync. GitHub offers no way to turn
pull requests off, so an automation closes them with a pointer here. That is not a
judgement on the contribution; it is the only honest thing the mirror can do.

To propose code, open a merge request on GitLab. Anyone can register on the instance.

Before you write much, open an issue describing what you have in mind. This project has
a written design behind most of its parts (see `docs/specs/`), and it is a poor use of
your evening to discover after the fact that a decision was already argued somewhere.

## What the code expects

- **Go 1.26.5 or newer** — the version in `go.mod`, not the one in your distribution.
- `make test` for the Go tests, `make test-all` to add the shell harnesses.
- The CI runs `go vet ./...` and `go test ./...` on every tag. Tests must not depend on
  what is installed on the machine running them: a test that passes only where Homebrew
  exists is a broken test, and there is a helper in `internal/complete` showing the way
  around it.
- Comments and commit messages are in **French**; everything a user can see is in
  **English and French**, through `internal/i18n/catalogue.go`. A user-visible string
  that is not in the catalogue will not be accepted.

## Translations

jigger speaks English and French. Both live side by side in
`internal/i18n/catalogue.go`, one line per key. A missing translation falls back to
English rather than showing a blank or a raw key.

Adding a third language means widening that table and its `nbLangues` constant. If you
want to do it, open an issue first — the design document
(`docs/specs/2026-08-16-i18n-design.md`) explains what else the change touches.

## Security

Please do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).
