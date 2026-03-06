# lookit

**Dual-mode markdown navigator with inter-document link navigation.**

Zero config. TUI and web modes. Syntax highlighting for 50+ languages. Git-aware. Broken link detection. Backlinks. Fuzzy search.

## Quick Start

```bash
# Build from source
go build -o lookit ./cmd/lookit

# TUI mode (default) — browse current directory
./lookit

# Web mode — serve on localhost:7777
./lookit serve
./lookit serve --port 3000 --open

# Render markdown to terminal
./lookit cat README.md

# Export markdown to HTML
./lookit export --format html

# Diagnostics
./lookit doctor
```

## Why lookit?

| Feature | `python -m http.server` | `http-server` | `glow` | **lookit** |
|---------|:-----------------------:|:-------------:|:------:|:----------:|
| TUI file browser | No | No | Single file | **Split-pane, fuzzy search** |
| Web server | Yes | Yes | No | **Yes, with SSE live reload** |
| Inter-document links | No | No | No | **History, backlinks, broken detection** |
| Syntax highlighting | No | No | Yes | **50+ languages (TUI + web)** |
| Git integration | No | No | No | **Status, branches, permalinks** |
| Markdown rendering | No | No | Yes | **Both terminal and HTML** |
| Fuzzy search | No | No | No | **Files + content (git grep)** |
| .gitignore aware | No | No | No | **Yes** |

## Features

### TUI Mode

Split-pane layout: file list (left) + rendered preview (right). Navigate with keyboard.

- **Fuzzy search** — type to filter files instantly
- **Link navigation** — follow markdown links between documents with history stack
- **Backlinks** — see which files link to the current document
- **TOC panel** — jump to headings
- **Bookmarks** — save frequently accessed files
- **Command palette** — `:` opens command mode
- **Keybinding presets** — default, vim, emacs
- **Themes** — light, dark, auto (detects terminal)

### Web Mode

Lightweight HTTP server with live reload.

- **GitHub-style markdown** — GFM extensions, emoji, syntax highlighting
- **Directory listings** — git status badges, file icons, breadcrumbs
- **Code viewing** — 50+ languages with line numbers and language badges
- **Search** — Ctrl+K overlay with fuzzy file search and content search (git grep)
- **Live reload** — SSE-based, updates on file save
- **Themes** — light/dark toggle, CSS custom properties
- **Security headers** — CSP, X-Frame-Options, etc.
- **ETag caching** — MD5-based for HTML, size+mtime for static

### Shared

- **Link graph** — bidirectional tracking of `[text](target)` and `[[wikilink]]` links
- **Broken link detection** — identifies links to nonexistent files
- **File watcher** — fsnotify with debounce, auto-rebuilds index
- **Git integration** — go-git for status, branches, log, permalinks (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
- **Plugin hooks** — YAML-defined hooks for content transformation
- **Task extraction** — finds TODOs with priority (`!high`), tags (`#tag`), due dates (`@due(...)`)
- **Export** — markdown to standalone HTML with embedded CSS and syntax highlighting
- **Config** — `~/.config/lookit/config.yaml`, env vars, CLI flags

## Configuration

Config file at `~/.config/lookit/config.yaml`:

```yaml
theme: auto          # light, dark, auto
keymap: default      # default, vim, emacs

server:
  port: 7777
  host: localhost
  no_https: false
  open: false

git:
  enabled: true
  show_status: true
  remote: origin

ignore:
  - "*.tmp"
  - "vendor/"
```

CLI flags override config: `--theme dark`, `--keymap vim`, `-c /path/to/config.yaml`.

Environment variables: `LOOKIT_THEME`, `LOOKIT_SERVER_PORT`, etc.

## Commands

```
lookit [path]                    # TUI mode (default)
lookit serve [path]              # Web server
lookit cat <file>                # Render markdown to terminal
lookit export [path]             # Export to HTML
  --format html|pdf
  --output <dir>
lookit doctor                    # Environment diagnostics
lookit version                   # Print version
lookit completion bash|zsh|fish  # Shell completions
```

## Development

```bash
git clone https://github.com/Benjamin-Connelly/lookit.git
cd lookit

# Build
go build -o lookit ./cmd/lookit

# Test (48 tests across 7 packages)
go test ./...

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o lookit-darwin-arm64 ./cmd/lookit
```

Requires Go 1.24+.

## Contributing

Contributions welcome. Open an issue or submit a PR.

## License

MIT © Benjamin Connelly
