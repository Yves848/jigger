# Faire connaître jigger — brouillons

Document interne, en français comme `captures.md` ; les textes à publier, eux,
sont en anglais et se lisent tels quels.

**Rien ici n'a été publié.** Ce sont des brouillons à relire, à modifier et à
poster vous-même. Aucune de ces plateformes n'accepte bien qu'on poste à la
place de quelqu'un, et un lancement qui n'est pas porté par son auteur se voit.

## L'état des lieux, en septembre 2026

| | |
|---|---|
| Dépôt | GitLab auto-hébergé, public, clonable anonymement — 0 étoile, 0 fork |
| Miroir | `github.com/Yves848/jigger`, push mirror automatique, en phase |
| Site | 4 pages, bilingue, 14 enregistrements — indexable depuis le 4 septembre |
| Paquets | Homebrew (tap perso), scoop (bucket perso). **Ni AUR ni winget** |

Le frein principal n'est pas la qualité : c'est que jigger n'est visible nulle
part où l'on cherche un outil de terminal. Le second est que la commande
d'installation demande de faire confiance à un domaine inconnu — un `brew tap`
vers `gitlab.yg-devworks.com` arrête beaucoup de monde.

## L'ordre qui a du sens

1. **Les paquets d'abord** (`packaging/`). Être dans l'AUR et dans winget, c'est
   être trouvable par des gens qui ne cherchaient pas jigger, et c'est répondre
   à la question « je dois donner accès à quoi ? » par « à rien ».
2. **Les annonces ensuite**, quand `yay -S jigger` et `winget install jigger`
   marchent. Une annonce dont la première étape est un `tap` vers un domaine
   inconnu convertit mal.
3. **Les annuaires en dernier**, une fois qu'il y a des retours à citer.

## Show HN

> **Show HN: Jigger – a live package-manager popup inside your real shell**
>
> I kept typing `brew install` and then stopping, because I could not remember
> whether the formula was `fd`, `fd-find` or `sharkdp/fd`. Autocomplete existed,
> but it meant pressing Tab and reading a wall of names, and it stopped at the
> boundary of each manager.
>
> Jigger is a small Go binary wired into zsh and PowerShell. You type the
> command you already typed — `brew install fire`, `winget install fire`,
> `yay -S visual-studio` — and a popup appears under the prompt on its own,
> narrowing with every letter. Tab inserts, Enter completes and runs in one
> keystroke, Ctrl-R switches the filter to a regex. If you ignore it, your line
> runs exactly as before: it never replaces your commands.
>
> Three things I would want to know if someone showed me this:
>
> - **It never chooses a manager for you.** When two of them know the same name,
>   the picker opens and you decide. There is no setting that introduces an
>   automatic choice.
> - **It never touches the network.** Candidates come from what brew, pacman,
>   yay, winget or scoop already has on disk, so it does not slow your prompt.
> - **It is not only package managers.** Type `ssh ` and the same popup offers
>   the hosts of your `~/.ssh/config`. The completion contract was written
>   before the SSH picker existed, and SSH is what proved it held.
>
> It works on macOS, Windows and Arch, with the same frame and the same keys on
> all three. Recordings of each are on the site — they are real captures, not
> mock-ups, and the protocol that produces them is written down.
>
> Site and guided tour: https://jigger.yg-devworks.com/parcours.html
> Code (GitLab is the source of truth, GitHub is a mirror):
> https://github.com/Yves848/jigger
>
> Happy to answer anything, including why the popup does not survive a screen
> recorder without tmux in front of it — that one cost me an evening.

**À vérifier avant de poster :** le titre commence bien par `Show HN:`, il est
posté un jour de semaine en matinée côte est, et vous êtes disponible les deux
heures qui suivent — un Show HN sans son auteur dans les commentaires retombe.

## r/commandline

> **Jigger: a live package-manager popup in zsh and PowerShell**
>
> I wrote this because I never remembered whether the package was called `fd`,
> `fd-find` or something else, and `brew search` in another window had become a
> reflex.
>
> You type `brew install fire` — nothing else, no key to press — and a frame
> appears under the prompt, narrowing with every letter. Tab inserts the
> highlighted candidate, Enter completes and runs the line in one keystroke,
> Ctrl-R switches the filter to a regex so you can write `(bird|fly)` when you
> do not know how the name starts.
>
> Same frame and same keys for Homebrew, pacman, yay, winget and scoop — and for
> `ssh`, `scp` and `sftp`, where the candidates are the hosts of your
> `~/.ssh/config` instead of packages.
>
> Two deliberate limits: it never picks a manager for you when two of them know
> the same name, and it never opens the network — candidates come from what the
> manager already has on disk.
>
> https://jigger.yg-devworks.com/parcours.html

## r/zsh (angle greffon)

> **A zsh plugin that shows package candidates as you type, without pressing Tab**
>
> The plugin hooks the line editor and asks a Go binary for candidates as the
> line changes, then draws a frame under the prompt. What it does *not* do
> matters as much: while the popup does not hold the keyboard, `↑` and `↓`
> remain your history, and it hands the key back to whatever plugin held it
> before — if you already bind arrows for prefix history search, that keeps
> working.
>
> `^G` closes it for the current line, `JIGGER_LIVE=0` turns the live popup off
> entirely and puts everything behind Tab, and `JIGGER_COMMANDS` decides which
> commands are intercepted at all.
>
> https://jigger.yg-devworks.com/

## r/PowerShell (angle module)

> **A PowerShell module that completes winget and scoop as you type**
>
> PSReadLine gives you the buffer; this module draws a popup under the prompt
> and fills it with winget and scoop candidates, live, without pressing Tab.
>
> The honest caveat first: PSReadLine keeps only the **last** handler bound to a
> chord. If your profile binds `^R` after importing jigger, it takes the key back
> and the regex toggle becomes unreachable. `Invoke-JiggerRegex` exists exactly
> for that — have your own handler call it first, and it returns `$true` when the
> popup took the key.
>
> https://jigger.yg-devworks.com/utiliser.html

## r/archlinux — **seulement une fois le paquet AUR publié**

> **jigger: pacman and yay completion in a live popup**
>
> `yay -S visual-studio` and the repositories and the AUR come back in a single
> list — ◆ for a repository package, ▣ for an AUR one, ● for what is already
> installed. Since pacman and yay are two doors onto the same database, `jg`
> lists your packages once, never twice.
>
> Nothing is fetched over the network: candidates come from the sync databases
> pacman already downloaded and from yay's AUR cache.
>
> `yay -S jigger` (or `jigger-bin`)

**Ne pas poster sur r/archlinux avant que le paquet existe.** Un « il n'y a pas
encore de paquet, clonez le dépôt » y est mal reçu, et à raison.

## Annuaires, quand il y aura des retours

- `awesome-shell`, `awesome-cli-apps`, `unixorn/awesome-zsh-plugins`
- Terminal Trove
- Le fil hebdomadaire de r/commandline

## Ce qu'il ne faut pas faire

- Poster les cinq textes le même jour : les mêmes gens lisent plusieurs de ces
  espaces, et une salve simultanée se lit comme du spam.
- Annoncer avant que `packaging/` soit soumis. Le premier commentaire sera
  « c'est dans l'AUR ? » ; y répondre « pas encore » coûte l'essentiel de
  l'attention gagnée.
- Écrire un chiffre qu'on ne peut pas montrer. Le dépôt s'est déjà fait prendre
  à écrire « quatre gestionnaires » là où le code en enregistrait cinq ; une
  annonce est exactement l'endroit où ce genre d'approximation se paie.
