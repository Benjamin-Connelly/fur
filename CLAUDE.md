# fur

Dual-mode markdown navigator: TUI (Bubble Tea) and web (stdlib net/http). Inter-document link navigation with history, backlinks, and broken link detection. Syntax highlighting (50+ languages), git-aware, zero-config.

**Version:** v0.4.0-dev

## Tech Stack

- **Language:** Go (pure, no CGO). Cross-compiles to linux/darwin × amd64/arm64.
- **Module:** `github.com/Benjamin-Connelly/fur`
- **TUI:** charmbracelet/bubbletea + lipgloss + glamour + bubbles
- **Web:** stdlib `net/http`, yuin/goldmark for markdown, go:embed for static assets
- **Syntax:** alecthomas/chroma/v2 (terminal256 for TUI, HTML classes for web)
- **Git:** go-git/go-git/v5 (no shelling out)
- **CLI:** spf13/cobra + spf13/viper
- **Config:** `~/.config/fur/config.yaml`
- **Search:** blevesearch/bleve/v2 for fulltext, sahilm/fuzzy for filename

## Directory Structure

```
cmd/fur/main.go                 # CLI entry: Cobra commands (root, serve, cat, export, graph, tasks, doctor, mcp, version, completion, gen-man)
internal/
  config/
    config.go                   # Viper config loader, validation, watch, defaults, config migration
    recent.go                   # Recent files list, per-project config (.fur.toml/.yaml)
  index/
    index.go                    # File walker, .gitignore parsing, in-memory index, ValidatePath
    fuzzy.go                    # Fuzzy search via sahilm/fuzzy
    links.go                    # Bidirectional link graph, wikilink resolution
    dot.go                      # DOT graph output for link visualization
    fulltext.go                 # Bleve fulltext search integration
    watcher.go                  # fsnotify with 100ms debounce
  tui/
    model.go                    # Root Bubble Tea model, split-pane layout
    dispatch.go                 # Update() central message dispatcher
    handle_normal.go            # handleNormalKey() — normal mode keybindings
    handle_filelist.go          # handleFileListKey() — file list navigation
    handle_preview.go           # handlePreviewKey(), search, visual mode, scrollToLink
    handle_filter.go            # handleFilterKey(), applyFilter()
    handle_command.go           # handleCommandKey() — command palette input
    handle_links.go             # handleLinkSelectKey(), handleFollowLink()
    handle_panels.go            # handleSidePanelKey() — TOC/backlinks/bookmarks/git
    handle_heading.go           # handleHeadingJumpKey(), headingJumpView, filterHeadingJump
    handle_util.go              # navigateToPath(), openInEditor(), clearStatusAfter()
    navigation.go               # Link follow, heading jump, theme cycling
    preview_load.go             # loadPreview(), file type handlers
    filelist.go                 # File list panel with fuzzy filter
    statusbar.go                # Mode indicator, path, key hints
    keys.go                     # Keybinding system (default/vim/emacs)
    links.go                    # Link navigation with history stack
    panels.go                   # TOC, backlinks, git info, bookmarks
    commands.go                 # Command palette (:command mode)
    images.go                   # Image protocol detection (iTerm2/Kitty/Sixel)
    datapreview.go              # Data file preview (JSON, CSV)
  web/
    server.go                   # HTTP server, SSE live reload, security headers, ETag, Goldmark instance
    handlers.go                 # Route handlers (dir, markdown, code, API)
    templates/                  # Go HTML templates (go:embed)
    static/                     # CSS + JS (go:embed), light/dark themes
  render/
    markdown.go                 # Glamour wrapper, heading extraction, Slugify
    code.go                     # Chroma wrapper (terminal + HTML)
    image.go                    # Image protocol rendering (iTerm2, Kitty, Sixel)
  mcp/
    server.go                   # MCP server: 5 tools (search, get_document, related, health, structure)
  git/
    git.go                      # go-git: repo, status, branches, log, remotes
    permalink.go                # URL generation (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
  remote/
    remote.go                   # SCP-style path parsing, Target type
    conn.go                     # SSH connection (ssh-agent, key files, ~/.ssh/config)
    sftpfs.go                   # afero.Fs implementation over SFTP
  manpages/
    manpages.go                 # Embedded man page installer
    pages/                      # go:embed man pages
  export/export.go              # Markdown → HTML/PDF with Chroma highlighting
  doctor/doctor.go              # 9 environment checks with colored output
  plugin/plugin.go              # YAML hook system (prepend/append/replace)
  tasks/tasks.go                # TODO extraction (priority, tags, due dates)
```

## Architecture

**TUI mode** (default): Bubble Tea app with split-pane layout. Left panel is a fuzzy-searchable file list, right panel is a rendered preview (Glamour for markdown, Chroma terminal256 for code). Side panels for TOC, backlinks, bookmarks, git info. Command palette via `:`. Link navigation with history stack.

**Web mode** (`fur serve`): stdlib `net/http` server. Goldmark renders markdown to HTML with GFM extensions (instance on Server struct). Chroma provides syntax highlighting with CSS classes. SSE endpoint (`/__events`) for live reload. API endpoints: `/__api/files` (fuzzy search), `/__api/search` (Bleve fulltext, fallback to git grep/grep), `/__api/graph`, `/__api/document`, `/__api/tasks`. Security headers, ETag caching, request logging.

**MCP mode** (`fur mcp`): Model Context Protocol server exposing the index and link graph as tools for AI agents. 5 tools: search_docs, get_document, get_related_docs, check_doc_health, get_doc_structure. Path validation shared with web via `Index.ValidatePath()`.

**Index**: In-memory file tree with `.gitignore` parsing (manual, no external dep). Bidirectional link graph tracks forward links and backlinks between markdown files. Supports standard `[text](target)` and `[[wikilink]]` syntax. fsnotify watcher with 100ms debounce rebuilds index and link graph on changes. Bleve fulltext index at `~/.cache/fur/index.bleve`.

**Config**: Viper reads from `~/.config/fur/config.yaml`, env vars (`FUR_*`), and CLI flags (flags win). Per-project config via `.fur.toml` / `.fur.yaml` (walks up from CWD). PersistentPreRunE on root command merges all sources. Live reload via `viper.WatchConfig()`. Auto-migrates from legacy `~/.config/lookit/` path.

## Conventions

- Pure Go, no CGO. Must cross-compile cleanly.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Idiomatic Go: small interfaces, explicit error returns, table-driven tests.
- YAGNI -- only build what's needed.

## Quick Reference

```bash
# Build
go build -o fur ./cmd/fur

# Run TUI (default mode)
./fur [path]

# Run web server
./fur serve [path]
./fur serve --port 3000 --open

# Remote browsing (SSH)
./fur myhost:/path/to/docs       # SCP-style remote path
./fur user@host:/path            # with explicit user
./fur --remote myhost /path      # flag-style alternative
./fur @docs                      # named remote from config

# Utilities
./fur cat README.md              # render markdown to terminal
./fur export --format html       # export markdown to HTML
./fur graph                      # link graph in DOT format
./fur graph --json               # link graph as JSON
./fur tasks                      # extract TODOs from markdown
./fur doctor                     # environment diagnostics
./fur version                    # version, commit, Go version, OS/arch
./fur mcp .                      # start MCP server

# Config
./fur --theme dark               # override theme
./fur --keymap vim               # override keybindings
./fur -c /path/to/config.yaml    # custom config file

# Shell completion
source <(fur completion bash)

# Test
go test ./...                    # 478 tests across 14 packages

# Cross-compile
GOOS=linux GOARCH=arm64 go build -o fur-linux-arm64 ./cmd/fur
GOOS=darwin GOARCH=arm64 go build -o fur-darwin-arm64 ./cmd/fur
```

## Gotchas

- `FuzzySearch` uses variadic maxResults: `FuzzySearch(query string, maxResults ...int)`.
- Web mode uses Goldmark (not Glamour) for markdown → HTML. TUI uses Glamour.
- Goldmark instance is on `Server` struct (initialized once, safe for concurrent use).
- go-git repo instances are cached via `sync.Mutex`-guarded map in `git.Open()`.
- SSE endpoint: `/__events`. File API: `/__api/files`. Search API: `/__api/search`.
- Additional web APIs: `/__api/graph`, `/__api/document`, `/__api/tasks`.
- Templates and static assets use `go:embed` in `internal/web/templates/` and `internal/web/static/`.
- `.gitignore` parsing is manual (supports `**`, negation, dir-only patterns) -- no external dependency.
- Permalink generation detects forge style from remote URL (GitHub/GitLab/Bitbucket/Gitea/Codeberg).
- Plugin hooks loaded from `~/.config/fur/plugins/*.yaml`.
- Task extraction recognizes `!high`/`!medium`/`!low` priority, `#tag`, `@due(YYYY-MM-DD)`.
- SSH auth: ssh-agent → key files → ~/.ssh/config. Agent connection tracked and closed properly.
- `render.Slugify()` is the single source of truth for anchor slugs (web and TUI both use it).
- `Index.ValidatePath()` is the shared path security check (web and MCP both delegate to it).
- Version is `var` not `const` (ldflags -X compatibility). Build info: `-X main.commit=... -X main.date=...`.
- Remote mode uses direct SFTP reads via SFTPFs.


## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
