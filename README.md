# lookit

**Dual-mode markdown navigator with inter-document link navigation.**

Zero config. TUI and web modes. Syntax highlighting for 50+ languages. Git-aware. Broken link detection. Backlinks. Fuzzy search.

<!-- ## Screenshots -->
<!-- TODO: Add terminal screenshots -->

## Install

```bash
# From source
go install github.com/Benjamin-Connelly/lookit/cmd/lookit@latest

# Or clone and build
git clone https://github.com/Benjamin-Connelly/lookit.git
cd lookit
go build -o lookit ./cmd/lookit
```

Requires Go 1.24+. Pure Go, no CGO — cross-compiles to linux/darwin on amd64/arm64.

## Quick Start

```bash
lookit                          # TUI mode — browse current directory
lookit ~/docs                   # TUI mode — browse specific directory
lookit serve                    # Web mode — localhost:7777
lookit serve --port 3000 --open # Web mode — custom port, auto-open browser
lookit cat README.md            # Render markdown to terminal
lookit export --format html     # Export all markdown to standalone HTML
lookit doctor                   # Environment diagnostics
```

## Why lookit?

| Feature | `python -m http.server` | `http-server` | `glow` | **lookit** |
|---------|:-----------------------:|:-------------:|:------:|:----------:|
| TUI file browser | No | No | Single file | **Split-pane, tree view, fuzzy search** |
| Web server | Yes | Yes | No | **Yes, with SSE live reload** |
| Inter-document links | No | No | No | **History, backlinks, broken detection** |
| Syntax highlighting | No | No | Yes | **50+ languages (TUI + web)** |
| Git integration | No | No | No | **Status, branches, permalinks** |
| Preview search | No | No | No | **/ search, n/N navigation** |
| Link cursor | No | No | No | **Tab/Shift-Tab cycle, Enter follow** |
| Visual line select | No | No | No | **V mode, copy GitHub permalinks** |
| .gitignore aware | No | No | No | **Yes** |

## Features

### TUI Mode

Split-pane layout: collapsible file tree (left) + rendered preview (right). Side panels for TOC, backlinks, bookmarks, and git info.

- **Fuzzy search** — `/` to filter files instantly, Enter to freeze results
- **Preview search** — `/` in preview pane for in-document search, `n`/`N` for next/prev match
- **Link navigation** — follow markdown links and `[[wikilinks]]` with history stack
- **Link cursor** — Tab/Shift-Tab to cycle links in preview, Enter to follow
- **Visual line select** — `V` to select lines, `y` to copy GitHub permalink for range
- **Cursor line** — gutter marker tracks position, `H` toggles reading guide bar
- **Side panels** — `t` TOC, `b` backlinks, `M` bookmarks, `i` git info
- **Command palette** — `:` opens command mode, `:N` jumps to line N
- **Keybinding presets** — default, vim, emacs
- **Themes** — light, dark, auto (detects terminal)
- **Mouse** — wheel scrolling (enable with `mouse: true` in config)

### Web Mode

Lightweight HTTP server with live reload.

- **GitHub-style markdown** — GFM extensions, emoji, syntax highlighting
- **Directory listings** — git status badges, file icons, breadcrumbs
- **Code viewing** — 50+ languages with line numbers and language badges
- **Search** — Ctrl+K overlay with fuzzy file search and content search (git grep)
- **Live reload** — SSE-based, updates on file save
- **Themes** — light/dark toggle, CSS custom properties
- **Security headers** — CSP, X-Frame-Options, Referrer-Policy, Permissions-Policy
- **ETag caching** — MD5-based for HTML, size+mtime for static

### Shared

- **Link graph** — bidirectional tracking of `[text](target)` and `[[wikilink]]` links
- **Broken link detection** — identifies links to nonexistent files
- **File watcher** — fsnotify with 100ms debounce, auto-rebuilds index
- **Git integration** — go-git for status, branches, log, permalinks (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
- **Plugin hooks** — YAML-defined hooks for content transformation (prepend/append/replace)
- **Task extraction** — finds TODOs with priority (`!high`), tags (`#tag`), due dates (`@due(...)`)
- **Export** — markdown to standalone HTML with embedded CSS and syntax highlighting
- **Doctor** — 8 environment checks with colored output

## Keybindings

### Default / Vim

| Key | Context | Action |
|-----|---------|--------|
| `j` / `k` | File list | Navigate up/down |
| `j` / `k` | Preview | Move cursor (with scrolloff) |
| `enter` / `l` | File list | Open file / expand directory |
| `h` | File list | Collapse directory |
| `g` / `G` | Any | Go to top / bottom |
| `u` / `d` | Preview | Half-page up / down |
| `tab` | Preview | Next link |
| `shift+tab` | Preview | Previous link |
| `enter` | Preview | Follow highlighted link |
| `/` | File list | Start fuzzy filter |
| `/` | Preview | Start preview search |
| `n` / `N` | Preview | Next / previous search match |
| `V` | Preview | Enter visual line select |
| `y` | Preview | Copy permalink (cursor line) |
| `y` | Visual | Copy permalink (selected range) |
| `H` | Preview | Toggle reading guide bar |
| `f` | Any | Follow link (single) or show link picker |
| `t` | Any | Toggle/focus TOC panel |
| `b` | Any | Toggle/focus backlinks panel |
| `m` | Any | Bookmark current file |
| `M` | Any | Toggle/focus bookmarks panel |
| `i` | Any | Toggle/focus git info panel |
| `c` | Preview | Copy file to clipboard |
| `r` | Preview | Reload file |
| `e` | Any | Open in `$EDITOR` |
| `:` | Any | Command palette |
| `?` | Any | Toggle help overlay |
| `backspace` | Any | Navigate back (history) |
| `L` | Any | Navigate forward (history) |
| `esc` | Any | Close panel / clear filter / go back |
| `q` | Any | Quit |

### Emacs Differences

| Key | Replaces | Action |
|-----|----------|--------|
| `ctrl+p` | `k` | Up |
| `ctrl+n` | `j` | Down |
| `ctrl+s` | `/` | Search |
| `ctrl+b` | `backspace` | Back |

### Visual Mode

| Key | Action |
|-----|--------|
| `V` | Enter visual line select |
| `j` / `k` | Extend selection |
| `g` / `G` | Select to top / bottom |
| `y` | Copy permalink for selection |
| `esc` / `V` | Cancel selection |

### Filter Mode

| Key | Action |
|-----|--------|
| Type | Fuzzy filter files |
| `enter` | Freeze filter results |
| `esc` | Clear filter |
| `ctrl+u` | Clear input |
| `ctrl+w` | Delete last word |

## Commands

```
lookit [path]                    # TUI mode (default)
lookit serve [path]              # Web server
  --port, -p <port>              # Server port (default: 7777)
  --open                         # Open browser after starting
  --no-https                     # Disable HTTPS
lookit cat <file>                # Render markdown to terminal
lookit export [path]             # Export to HTML
  --format html                  # Output format
  --output, -o <dir>             # Output directory
lookit doctor                    # Environment diagnostics
lookit version                   # Print version
lookit completion [shell]        # Shell completions (bash/zsh/fish/powershell)
  --install                      # Auto-install without prompts
```

## Configuration

Config file at `~/.config/lookit/config.yaml`:

```yaml
theme: auto          # light, dark, auto
keymap: default      # default, vim, emacs
mouse: false         # enable mouse wheel scrolling
reading_guide: false # persistent reading guide bar
scrolloff: 5         # cursor margin (lines above/below)

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

## Development

```bash
git clone https://github.com/Benjamin-Connelly/lookit.git
cd lookit

go build -o lookit ./cmd/lookit      # Build
go test ./...                         # Test (48 tests across 7 packages)
go vet ./...                          # Lint

# Cross-compile
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o lookit-darwin-arm64 ./cmd/lookit
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o lookit-linux-arm64 ./cmd/lookit
```

Requires Go 1.24+. Pure Go, no CGO.

## Contributing

Contributions welcome. Open an issue or submit a PR.

## Acknowledgments

lookit is built on the shoulders of excellent open source projects:

**Inspiration**
- [Glow](https://github.com/charmbracelet/glow) by Charmbracelet — the terminal markdown viewer that inspired this project

**TUI Framework**
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — terminal UI framework (The Elm Architecture for Go)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling and layout
- [Glamour](https://github.com/charmbracelet/glamour) — terminal markdown rendering
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components library

**Syntax & Markdown**
- [Chroma](https://github.com/alecthomas/chroma) — syntax highlighting engine (50+ languages)
- [Goldmark](https://github.com/yuin/goldmark) — CommonMark markdown parser for web mode
- [goldmark-emoji](https://github.com/yuin/goldmark-emoji) — emoji extension for Goldmark

**Git**
- [go-git](https://github.com/go-git/go-git) — pure Go git implementation (no shelling out)

**CLI & Config**
- [Cobra](https://github.com/spf13/cobra) — CLI framework with subcommands and completions
- [Viper](https://github.com/spf13/viper) — configuration management (YAML, env vars, flags)

**Utilities**
- [fuzzy](https://github.com/sahilm/fuzzy) — fuzzy string matching
- [clipboard](https://github.com/atotto/clipboard) — cross-platform clipboard access
- [fsnotify](https://github.com/fsnotify/fsnotify) — cross-platform file system notifications

Thank you to all the maintainers and contributors of these projects.

## License

MIT © Benjamin Connelly
