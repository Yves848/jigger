# Journal des passes de test

Une entrée par exécution complète de la suite, sur l'une ou l'autre plateforme. Les
entrées sont ajoutées par les scripts — `tests/passe-macos.sh` et
`tests/passe-windows.ps1` — jamais à la main, et la plus récente est en tête.

**À quoi ça sert.** À répondre, dans trois semaines, à la question qui n'avait aucune
réponse jusqu'ici : *ce commit a-t-il été éprouvé sur Windows, et quand ?* C'est
précisément ce trou qui a laissé les analyseurs scoop rendre zéro ligne depuis la v0.8.0
sans que personne ne le sache — le code passait tous les tests, sur la seule machine où
on les lançait.

Le détail complet de la **dernière** passe de chaque plateforme vit à côté, dans
`tests/captures/derniers-tests-macos.md` et `tests/captures/derniers-tests-windows.md`.
Ce journal-ci, lui, garde la trace.

<!-- nouvelles passes ici -->

## 2026-08-16 11:18 — macOS — `eba2d59` — tout passe

macOS 26.5.2 · zsh 5.9 · go1.26.6

- **ok** — go build · go test · zpty.zsh (vrai pseudo-terminal) · smoke.ps1 · banc de rendu (français figé)
- durée totale : 148 s


## 2026-08-16 11:15 — macOS — `5a54e20` — tout passe

macOS 26.5.2 · zsh 5.9 · go1.26.6

- **ok** — go build · go test · zpty.zsh (vrai pseudo-terminal) · smoke.ps1
- durée totale : 133 s


## 2026-08-16 08:57 — macOS — `3af3efb` — tout passe

macOS 26.5.2 · zsh 5.9 · go1.26.6

- **ok** — go build · go test · zpty.zsh (vrai pseudo-terminal) · smoke.ps1
- durée totale : 144 s


## 2026-08-16 08:51 — Windows — `eb3d131` — tout passe

Microsoft Windows NT 10.0.26200.0 · pwsh 7.6.3 · go version go1.26.5 windows/amd64 · captures rafraîchies

- **ok** — captures scoop et winget · go build · go test · smoke.ps1 · pty.ps1 (vraie console)


## 2026-08-16 08:11 — macOS — `047f11c` — tout passe

macOS 26.5.2 · zsh 5.9 · go1.26.6

- **ok** — go build · go test · zpty.zsh (vrai pseudo-terminal) · smoke.ps1

