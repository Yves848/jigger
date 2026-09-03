BINARY := jigger
PREFIX ?= $(HOME)/.local
# Le shell dépend de la plateforme, et avec lui le greffon à installer et la suite à
# lancer : zsh + Homebrew d'un côté, PowerShell + winget/scoop de l'autre.
GREFFON := Ajoute dans ~/.zshrc :  source $(CURDIR)/shell/jigger.plugin.zsh
# test-golden n'est pas de la partie : sa référence est celle d'une machine et d'une
# version (cf. l'en-tête de tests/render-golden.sh). Il se lance à la main.
TEST_SHELL := test-shell

ifeq ($(OS),Windows_NT)
  BINARY := jigger.exe
  GREFFON := Ajoute dans $$PROFILE :  Import-Module $(CURDIR)/shell/jigger.psm1
  TEST_SHELL := test-shell-ps test-pty
endif

build:
	go build -o $(BINARY) .

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "Installé : $(PREFIX)/bin/$(BINARY)"
	@echo "$(GREFFON)"

test:
	go test ./...

# Le widget zsh ne se teste que dans un vrai pseudo-terminal (cf. tests/zpty.zsh).
test-shell: build
	./tests/zpty.zsh --suite

# Le popup ne doit pas bouger d'un octet en français (cf. docs/plans/2026-08-16-i18n.md).
# Hors de test-all, et à lancer à la main : la référence dépend de la version affichée
# dans la bannière et du catalogue Homebrew de la machine qui l'a capturée. `--capturer`
# avant un chantier de rendu, `--verifier` après.
test-golden: build
	./tests/render-golden.sh --verifier

# Le module PowerShell : tout ce qui se teste sans console (cf. tests/smoke.ps1).
test-shell-ps: build
	pwsh -NoProfile -File tests/smoke.ps1

# Le popup lui-même, dans un vrai pseudo-terminal (cf. tests/pty.ps1). Lent — quelques
# secondes par cas, le temps qu'un pwsh démarre — mais c'est le seul juge de ce que
# l'utilisateur voit.
test-pty: build
	pwsh -NoProfile -File tests/pty.ps1

test-all: test $(TEST_SHELL)

# Les images et les enregistrements de la documentation. Demande vhs, ffmpeg et tmux —
# tmux n'est pas un confort ici : sans lui le popup ne s'affiche pas du tout sous un
# enregistreur, faute de réponse assez rapide à l'interrogation du curseur. Voir
# docs/captures.md.
#
# La plateforme est déduite d'uname : cette cible ne produit que ce que la machine
# courante peut produire. Windows a son propre script, docs/media/capturer.ps1.
media:
	./docs/media/capturer.sh

# Réécrit les tapes VHS à partir du générateur. À lancer après avoir touché un
# scénario, jamais pour corriger un tape à la main.
media-tapes:
	./docs/media/generer-tapes.sh

clean:
	rm -f jigger jigger.exe conpty-test.exe

.PHONY: build install test test-shell test-golden test-shell-ps test-pty test-all clean media media-tapes
