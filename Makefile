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

clean:
	rm -f $(BINARY)

.PHONY: build install test clean
