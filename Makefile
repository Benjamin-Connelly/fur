.PHONY: help build test man install clean

BIN     ?= fur
PREFIX  ?= /usr/local
MANDIR  ?= $(PREFIX)/share/man/man1
BINDIR  ?= $(PREFIX)/bin

help:
	@echo "Targets:"
	@echo "  build    Build the fur binary"
	@echo "  test     Run the test suite"
	@echo "  man      Regenerate man pages from cobra commands"
	@echo "  install  Install binary to \$$PREFIX/bin and man pages to \$$PREFIX/share/man/man1"
	@echo "  clean    Remove built binary"
	@echo ""
	@echo "Overrides: BIN=$(BIN) PREFIX=$(PREFIX)"

build:
	go build -o $(BIN) ./cmd/fur

test:
	go test ./...

man: build
	./$(BIN) gen-man ./docs/man/man1

install: build man
	install -m 755 $(BIN) $(BINDIR)/
	install -d $(MANDIR)/
	install -m 644 docs/man/man1/*.1 $(MANDIR)/

clean:
	rm -f $(BIN)
