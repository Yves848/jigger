BINARY := jigger
PREFIX ?= $(HOME)/.local

build:
	go build -o $(BINARY) .

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "Installé : $(PREFIX)/bin/$(BINARY)"
	@echo "Ajoute dans ~/.zshrc :  source $(CURDIR)/shell/jigger.plugin.zsh"

test:
	go test ./...

# Le widget zsh ne se teste que dans un vrai pseudo-terminal (cf. tests/zpty.zsh).
test-shell: build
	./tests/zpty.zsh --suite

test-all: test test-shell

clean:
	rm -f $(BINARY)

.PHONY: build install test test-shell test-all clean
