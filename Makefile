.PHONY: help build test man install clean capture-web

BIN          ?= fur
PREFIX       ?= /usr/local
MANDIR       ?= $(PREFIX)/share/man/man1
BINDIR       ?= $(PREFIX)/bin
CAPTURE_PORT ?= 7799

help:
	@echo "Targets:"
	@echo "  build        Build the fur binary"
	@echo "  test         Run the test suite"
	@echo "  man          Regenerate man pages from cobra commands"
	@echo "  install      Install binary to \$$PREFIX/bin and man pages to \$$PREFIX/share/man/man1"
	@echo "  capture-web  Regenerate docs/assets/web-*.webp from a headless browser"
	@echo "  clean        Remove built binary"
	@echo ""
	@echo "Overrides: BIN=$(BIN) PREFIX=$(PREFIX) CAPTURE_PORT=$(CAPTURE_PORT)"

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

# capture-web regenerates the README web-mode screenshots. It serves the demo
# tree on a loopback port, drives a headless browser (e2e/capture, which finds
# a Chrome/Chromium incl. the Playwright/rod caches), then converts the PNGs to
# webp. The graph shot pulls d3 from its CDN, so this needs outbound network.
capture-web: build
	@bash -euo pipefail -c '\
	  ./$(BIN) serve docs/demo/ --port $(CAPTURE_PORT) >/tmp/fur-capture-serve.log 2>&1 & \
	  sv=$$!; trap "kill $$sv 2>/dev/null || true" EXIT; \
	  for i in $$(seq 1 50); do curl -sf http://127.0.0.1:$(CAPTURE_PORT)/ >/dev/null 2>&1 && break || sleep 0.2; done; \
	  ( cd e2e && go run ./capture -url http://127.0.0.1:$(CAPTURE_PORT) -out ../docs/assets ); \
	  for n in web-cover web-reading web-graph; do \
	    python3 -c "from PIL import Image; Image.open(\"docs/assets/$$n.png\").convert(\"RGB\").save(\"docs/assets/$$n.webp\", \"WEBP\", quality=82, method=6)"; \
	    rm -f docs/assets/$$n.png; \
	  done; \
	  echo "regenerated docs/assets/web-*.webp"'

clean:
	rm -f $(BIN)
