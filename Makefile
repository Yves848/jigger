BINARY := jigger
PREFIX ?= $(HOME)/.local
# Le shell dépend de la plateforme, et avec lui le greffon à installer et la suite à
# lancer : zsh + Homebrew d'un côté, PowerShell + winget/scoop de l'autre.
GREFFON := Ajoute dans ~/.zshrc :  source $(CURDIR)/shell/jigger.plugin.zsh
TEST_SHELL := test-shell test-golden

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

clean:
	rm -f jigger jigger.exe conpty-test.exe

.PHONY: build install test test-shell test-golden test-shell-ps test-pty test-all clean
