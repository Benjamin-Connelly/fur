# Lookit

Dual-mode markdown navigator: TUI (Bubble Tea) and web (stdlib net/http). Inter-document link navigation with history, backlinks, and broken link detection. Syntax highlighting (50+ languages), git-aware, zero-config.

**Version:** v0.1.0

## Tech Stack

- **Language:** Go (pure, no CGO). Cross-compiles to linux/darwin × amd64/arm64.
- **Module:** `github.com/Benjamin-Connelly/lookit`
- **TUI:** charmbracelet/bubbletea + lipgloss + glamour + bubbles
- **Web:** stdlib `net/http`, yuin/goldmark for markdown, go:embed for static assets
- **Syntax:** alecthomas/chroma/v2 (terminal256 for TUI, HTML classes for web)
- **Git:** go-git/go-git/v5 (no shelling out)
- **CLI:** spf13/cobra + spf13/viper
- **Config:** `~/.config/lookit/config.yaml`

## Directory Structure

```
cmd/lookit/main.go              # CLI entry: Cobra commands (root, serve, cat, export, doctor, version, completion)
internal/
  config/config.go              # Viper config loader, validation, watch, defaults
  index/
    index.go                    # File walker, .gitignore parsing, in-memory index
    fuzzy.go                    # Fuzzy search via sahilm/fuzzy
    links.go                    # Bidirectional link graph, wikilink resolution
    watcher.go                  # fsnotify with 100ms debounce
  tui/
    model.go                    # Root Bubble Tea model, split-pane layout
    filelist.go                 # File list panel with fuzzy filter
    preview.go                  # Scrollable preview pane (Glamour + Chroma)
    statusbar.go                # Mode indicator, path, key hints
    keys.go                     # Keybinding system (default/vim/emacs)
    links.go                    # Link navigation with history stack
    panels.go                   # TOC, backlinks, git info, bookmarks
    commands.go                 # Command palette (:command mode)
    images.go                   # Image protocol detection (stub)
  web/
    server.go                   # HTTP server, SSE live reload, security headers, ETag
    handlers.go                 # Route handlers (dir, markdown, code, API)
    templates/                  # Go HTML templates (go:embed)
    static/                     # CSS + JS (go:embed), light/dark themes
  render/
    markdown.go                 # Glamour wrapper, heading extraction
    code.go                     # Chroma wrapper (terminal + HTML)
  git/
    git.go                      # go-git: repo, status, branches, log, remotes
    permalink.go                # URL generation (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
  remote/
    remote.go                   # SCP-style path parsing, Target type
    conn.go                     # SSH connection (ssh-agent, key files, ~/.ssh/config)
    sync.go                     # SFTP sync/cache with periodic polling
  export/export.go              # Markdown → HTML with Chroma highlighting
  doctor/doctor.go              # 8 environment checks with colored output
  plugin/plugin.go              # YAML hook system (prepend/append/replace)
  tasks/tasks.go                # TODO extraction (priority, tags, due dates)
```

## Architecture

**TUI mode** (default): Bubble Tea app with split-pane layout. Left panel is a fuzzy-searchable file list, right panel is a rendered preview (Glamour for markdown, Chroma terminal256 for code). Side panels for TOC, backlinks, bookmarks, git info. Command palette via `:`. Link navigation with history stack.

**Web mode** (`lookit serve`): stdlib `net/http` server. Goldmark renders markdown to HTML with GFM extensions. Chroma provides syntax highlighting with CSS classes. SSE endpoint (`/__events`) for live reload. API endpoints: `/__api/files` (fuzzy search), `/__api/search` (git grep). Security headers, ETag caching, request logging.

**Index**: In-memory file tree with `.gitignore` parsing (manual, no external dep). Bidirectional link graph tracks forward links and backlinks between markdown files. Supports standard `[text](target)` and `[[wikilink]]` syntax. fsnotify watcher with 100ms debounce rebuilds index and link graph on changes.

**Config**: Viper reads from `~/.config/lookit/config.yaml`, env vars (`LOOKIT_*`), and CLI flags (flags win). PersistentPreRunE on root command merges all sources. Live reload via `viper.WatchConfig()`.

## Conventions

- Pure Go, no CGO. Must cross-compile cleanly.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Idiomatic Go: small interfaces, explicit error returns, table-driven tests.
- YAGNI -- only build what's needed.

## Quick Reference

```bash
# Build
go build -o lookit ./cmd/lookit

# Run TUI (default mode)
./lookit [path]

# Run web server
./lookit serve [path]
./lookit serve --port 3000 --open

# Remote browsing (SSH)
./lookit myhost:/path/to/docs       # SCP-style remote path
./lookit user@host:/path            # with explicit user
./lookit --remote myhost /path      # flag-style alternative
./lookit @docs                      # named remote from config

# Utilities
./lookit cat README.md              # render markdown to terminal
./lookit export --format html       # export markdown to HTML
./lookit doctor                     # environment diagnostics
./lookit version                    # print version

# Config
./lookit --theme dark               # override theme
./lookit --keymap vim               # override keybindings
./lookit -c /path/to/config.yaml    # custom config file

# Shell completion
source <(lookit completion bash)

# Test
go test ./...                       # 48 tests across 7 packages

# Cross-compile
GOOS=linux GOARCH=arm64 go build -o lookit-linux-arm64 ./cmd/lookit
GOOS=darwin GOARCH=arm64 go build -o lookit-darwin-arm64 ./cmd/lookit
```

## Gotchas

- `FuzzySearch` uses variadic maxResults: `FuzzySearch(query string, maxResults ...int)`.
- Web mode uses Goldmark (not Glamour) for markdown → HTML. TUI uses Glamour.
- go-git repo instances are cached via `sync.Mutex`-guarded map in `git.Open()`.
- SSE endpoint: `/__events`. File API: `/__api/files`. Search API: `/__api/search`.
- Templates and static assets use `go:embed` in `internal/web/templates/` and `internal/web/static/`.
- `.gitignore` parsing is manual (supports `**`, negation, dir-only patterns) -- no external dependency.
- Permalink generation detects forge style from remote URL (GitHub/GitLab/Bitbucket/Gitea/Codeberg).
- Plugin hooks loaded from `~/.config/lookit/plugins/*.yaml`.
- Task extraction recognizes `!high`/`!medium`/`!low` priority, `#tag`, `@due(YYYY-MM-DD)`.
- Remote mode caches files to `~/.cache/lookit/remote/<hash>/`. Git features disabled for remote.
- SSH auth: ssh-agent → key files → ~/.ssh/config. TOFU for unknown host keys.
- Remote polling interval: 15 seconds. No real-time change notification (SFTP limitation).
