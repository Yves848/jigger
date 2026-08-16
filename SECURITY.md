# Security policy

## What jigger does on your machine

Worth stating plainly, because it bounds what a vulnerability here could mean:

- jigger **runs package-manager commands on your behalf** — `brew`, `winget`, `scoop` —
  and relays their output untouched. A flaw that lets someone influence which command is
  run is therefore serious.
- It reads and writes a **cache** of package catalogues under your user directory
  (`~/Library/Caches/jigger`, `%LOCALAPPDATA%\jigger`, or `$JIGGER_CACHE_DIR`).
- It **never** downloads anything itself, never phones home, and holds no credentials.

## Reporting a vulnerability

Email **yves.godart@yg-devworks.com** with what you found and how to reproduce it. Please
do not open a public issue first.

You will get an acknowledgement. This is a one-person project, so the honest expectation
is days rather than hours, and no bounty.

If you would rather report anonymously, that is fine — a report with no name still gets
fixed.

## Supported versions

The latest release only. jigger is a single binary that people rebuild or reinstall in
one command; there is no branch of older versions to backport to.

## Disclosure

Once a fix is released, the problem is described in `CHANGELOG.md` under **Security**,
with credit if you want it.
