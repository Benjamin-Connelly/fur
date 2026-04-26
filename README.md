# fur

**Dual-mode markdown navigator with inter-document link navigation.**

Zero config. TUI and web modes. Full-text search. Syntax highlighting for 50+ languages. Git-aware. Broken link detection. Backlinks. Interactive link graph.

## Demo

![fur demo](demo/fur-demo.gif)

## Install

```bash
# Homebrew (macOS/Linux)
brew install Benjamin-Connelly/fur/fur

# From source (requires Go 1.26+)
go install github.com/Benjamin-Connelly/fur/cmd/fur@latest

# Or clone and build
git clone https://github.com/Benjamin-Connelly/fur.git
cd fur
go build -o fur ./cmd/fur

# Nix
nix run github:Benjamin-Connelly/fur
```

Pure Go, no CGO. Cross-compiles to linux/darwin on amd64/arm64.

## Quick Start

```bash
fur                          # TUI mode — browse current directory
fur ~/docs                   # TUI mode — browse specific directory
fur myhost:~/docs            # SSH remote — browse files on a remote host
fur serve                    # Web mode — localhost:7777
fur serve --port 3000 --open # Web mode — custom port, auto-open browser
fur cat README.md            # Render markdown to terminal
fur export --format html     # Export all markdown to standalone HTML
fur doctor                   # Environment diagnostics
```

## Why fur?

| Feature | `glow` | `mdcat` | `frogmouth` | **fur** |
|---------|:------:|:-------:|:-----------:|:----------:|
| TUI file browser | Stash only | No | Single-pane | **Split-pane tree + preview** |
| SSH remote browsing | No | No | No | **`host:/path` — browse remote docs** |
| Full-text search | No | No | No | **Bleve BM25 index** |
| Inter-document links | No | No | No | **History, backlinks, `[[wikilinks]]`** |
| Broken link detection | No | No | No | **Files + `#heading` anchors** |
| Web server | No | No | No | **SSE live reload, D3 graph** |
| Syntax highlighting | Markdown | Limited | Markdown | **50+ languages (TUI + web)** |
| Git integration | No | No | No | **Status, branches, permalinks** |
| Visual line select | No | No | No | **V mode, GitHub permalinks** |
| Data files (JSON/CSV) | No | No | No | **Pretty-print + tables** |
| Keybinding presets | No | No | Vim-like | **Default, vim, emacs** |

## Features

### TUI Mode

Split-pane layout: collapsible file tree (left) + rendered preview (right). Side panels for TOC, backlinks, bookmarks, and git info.

- **Fuzzy search** — `/` to filter files instantly, Enter to freeze results
- **Full-text search** — Tab toggles to content search (Bleve BM25 index with snippets)
- **Preview search** — `/` in preview pane for in-document search, `n`/`N` for next/prev match, Ctrl+R for regex
- **Search history** — Up/Down cycles through previous search queries
- **Link navigation** — follow markdown links and `[[wikilinks]]` with history stack, `#heading` anchor scrolling
- **Link cursor** — Tab/Shift-Tab to cycle links in preview, Enter to follow
- **Global heading jump** — Ctrl+G fuzzy picks any heading across all files
- **Visual line select** — `V` to select lines, `y` to copy GitHub permalink for range
- **Vim-style marks** — `m{a-z}` to set, `'{a-z}` to jump back
- **Cursor line** — gutter marker tracks position, `H` toggles reading guide bar
- **Side panels** — `t` TOC, `b` backlinks, `M` bookmarks, `i` git info
- **Command palette** — `:` opens command mode, `:N` jumps to line N
- **Data preview** — JSON pretty-print, CSV/TSV tables, YAML frontmatter cards
- **Image info cards** — dimensions, size, format; `e` to open in system viewer
- **Keybinding presets** — default, vim, emacs
- **Themes** — light, dark, auto, ascii (no color); Ctrl+T cycles at runtime
- **Recent files** — persistent history across sessions
- **Stdin pipe** — `echo '# Hello' | fur` renders piped markdown
- **Mouse** — wheel scrolling (enable with `mouse: true` in config)

### Web Mode

Lightweight HTTP server with live reload.

- **GitHub-style markdown** — GFM extensions, emoji, syntax highlighting
- **Mermaid diagrams** — renders `mermaid` fenced code blocks client-side
- **Interactive link graph** — D3.js force-directed graph at `/graph` with drag, zoom, click-to-navigate
- **Directory listings** — git status badges, file icons, breadcrumbs
- **Code viewing** — 50+ languages with line numbers and language badges
- **Search** — Ctrl+K overlay with fuzzy file search and full-text content search (Bleve)
- **Live reload** — SSE-based, updates on file save
- **SSH detection** — prints URL instead of opening browser when connected via SSH
- **Print stylesheet** — clean layout when printing (Ctrl+P), hides navigation
- **Custom CSS** — `--css path/to/custom.css` or `custom_css` in config
- **Themes** — light/dark toggle, CSS custom properties
- **Security headers** — CSP, X-Frame-Options, Referrer-Policy, Permissions-Policy
- **ETag caching** — MD5-based for HTML, size+mtime for static

### SSH Remote Browsing

Browse files on remote hosts over SSH — no installation required on the remote machine.

```bash
fur myhost:~/docs                # SSH config alias with ~ expansion
fur user@192.168.1.50:/var/docs  # Explicit user and IP
fur --remote myhost ~/docs       # Flag-style alternative
fur @docs                        # Named remote from config
```

- **Zero remote setup** — uses SFTP, nothing to install on the server
- **SSH config support** — respects `~/.ssh/config` (Host aliases, Hostname, User, Port, IdentityFile)
- **Auth chain** — ssh-agent → key files → SSH config (TOFU for unknown host keys)
- **Sync/cache model** — files cached locally at `~/.cache/fur/remote/`, polled every 15s for changes
- **Status bar** — shows connection state (Connected/Reconnecting/Disconnected) and last sync time
- **Auto-reconnect** — exponential backoff on connection loss
- **Named remotes** — configure aliases in `config.yaml`:
  ```yaml
  remotes:
    docs:
      host: myserver
      user: deploy
      path: /home/deploy/docs
  ```

### Shared

- **Full-text search** — Bleve persistent index with BM25 scoring, field boosting, and highlighted snippets
- **Link graph** — bidirectional tracking of `[text](target)` and `[[wikilink]]` links with fragment validation
- **Broken link detection** — identifies links to nonexistent files and invalid `#heading` anchors
- **File watcher** — fsnotify with 100ms debounce, auto-rebuilds file and search indexes
- **Git integration** — go-git for status, branches, log, permalinks (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
- **Per-project config** — `.fur.toml` / `.fur.yaml` discovered by walking up from CWD
- **Plugin hooks** — YAML-defined hooks for content transformation (prepend/append/replace)
- **Task extraction** — finds TODOs with priority (`!high`), tags (`#tag`), due dates (`@due(...)`)
- **Export** — markdown to standalone HTML or PDF with embedded CSS and syntax highlighting
- **Graph export** — `fur graph` outputs DOT format for Graphviz visualization
- **Doctor** — 9 environment checks with colored output
- **Man pages** — `fur gen-man` generates troff man pages

### MCP Server

fur exposes its documentation index as an [MCP](https://modelcontextprotocol.io/) server for AI agents.

```bash
fur mcp ~/docs     # Start MCP server on stdin/stdout
```

**Tools:**

| Tool | Description |
|------|-------------|
| `search_docs` | Fuzzy filename or Bleve fulltext search |
| `get_document` | Read file content (with size guard and binary detection) |
| `get_related_docs` | Forward links, backlinks, or both for a file |
| `check_doc_health` | Broken links, broken anchors, and pending tasks |
| `get_doc_structure` | Heading tree with anchor slugs and line numbers |

**Claude Code configuration** (`.claude/settings.json`):

```json
{
  "mcpServers": {
    "fur": {
      "command": "fur",
      "args": ["mcp", "."]
    }
  }
}
```

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
| `m` | File list | Bookmark current file |
| `m{a-z}` | Preview | Set mark at current position |
| `'{a-z}` | Preview | Jump to mark |
| `M` | Any | Toggle/focus bookmarks panel |
| `i` | Any | Toggle/focus git info panel |
| `c` | Preview | Copy file to clipboard |
| `r` | Preview | Reload file |
| `e` | Any | Open in `$EDITOR` (images: system viewer) |
| `ctrl+g` | Any | Global heading jump (fuzzy picker) |
| `ctrl+t` | Any | Cycle theme (auto → dark → light) |
| `ctrl+r` | Search | Toggle regex search mode |
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
| `tab` | Toggle filename / content search |
| `enter` | Freeze filter results |
| `esc` | Clear filter |
| `ctrl+u` | Clear input |
| `ctrl+w` | Delete last word |
| `up` / `down` | Cycle search history |

## Commands

```
fur [path]                    # TUI mode (default)
fur host:/path                # SSH remote (SCP-style)
fur @alias                    # Named remote from config
  --remote <host>                # Remote host (SSH config alias or user@host)
  --remote-port <port>           # Remote SSH port
  --keymap vim|emacs|default     # Keybinding preset
  --theme dark|light|auto|ascii  # Color theme
  --no-color                     # Alias for --theme ascii
  --show-hidden                  # Surface dotfiles/dotdirs (.git/.hg/.svn/.bzr always hidden)
  --version, -V                  # Print version
fur serve [path]              # Web server
  --port, -p <port>              # Server port (default: 7777)
  --open                         # Open browser after starting
  --no-https                     # Disable HTTPS
  --css <path>                   # Custom CSS file
fur config init               # Write ~/.config/fur/config.yaml from a documented template
  --force                        # Overwrite existing config (creates config.yaml.bak)
fur config path               # Print resolved config file path
fur config show               # Print active merged config (file + env + flags)
fur cat <file>                # Render markdown or image to terminal
fur export [path]             # Export markdown to HTML
  --format html|pdf              # Output format (PDF requires chromium or wkhtmltopdf)
  --output, -o <dir>             # Output directory
fur graph [path]              # Output link graph in DOT format
fur doctor                    # Environment diagnostics
fur completion [shell]        # Shell completions (bash/zsh/fish/powershell)
  --install                      # Auto-install without prompts
cat file.md | fur             # Render piped markdown
```

## Configuration

Global config at `~/.config/fur/config.yaml`:

```yaml
theme: auto          # light, dark, auto, ascii
keymap: default      # default, vim, emacs
mouse: false         # enable mouse wheel scrolling
reading_guide: false # persistent reading guide bar
scrolloff: 5         # cursor margin (lines above/below)
show_hidden: false   # surface dotfiles/dotdirs; .git/.hg/.svn/.bzr always hidden

server:
  port: 7777
  host: localhost
  no_https: false
  open: false
  custom_css: ""     # path to custom CSS file

git:
  enabled: true
  show_status: true
  remote: origin

ignore:
  - "*.tmp"
  - "vendor/"

remotes:                # Named remote hosts for SSH browsing
  docs:
    host: myserver      # SSH config alias or hostname
    user: deploy        # SSH user (optional)
    path: /home/deploy/docs
```

**Per-project config:** Place `.fur.toml` or `.fur.yaml` in your project root. fur walks up from the current directory and merges the first one found over the global config.

CLI flags override config: `--theme dark`, `--keymap vim`, `-c /path/to/config.yaml`.

Environment variables: `LOOKIT_THEME`, `LOOKIT_SERVER_PORT`, etc.

## Development

```bash
git clone https://github.com/Benjamin-Connelly/fur.git
cd fur

make build                            # Build binary
make test                             # Run tests
make man                              # Generate man pages
go vet ./...                          # Lint

# Or without make:
go build -o fur ./cmd/fur
go test ./...

# Cross-compile
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o fur-darwin-arm64 ./cmd/fur
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o fur-linux-arm64 ./cmd/fur
```

Requires Go 1.26+. Pure Go, no CGO.

## Contributing

Contributions welcome. Open an issue or submit a PR.

## Acknowledgments

Lookit exists because generous people write extraordinary software and give it away. We stand on the shoulders of giants, and we're deeply grateful to every maintainer, contributor, and community member behind these projects.

**Inspiration**
- [Glow](https://github.com/charmbracelet/glow) by Charmbracelet — the beautiful terminal markdown viewer that started this whole idea

**TUI Framework**
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the brilliant Elm Architecture for terminals that makes TUI development a joy
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling so good it feels like cheating
- [Glamour](https://github.com/charmbracelet/glamour) — gorgeous terminal markdown rendering
- [Bubbles](https://github.com/charmbracelet/bubbles) — polished, composable TUI components

**Syntax & Markdown**
- [Chroma](https://github.com/alecthomas/chroma) — syntax highlighting for 50+ languages, one import away
- [Goldmark](https://github.com/yuin/goldmark) — rock-solid CommonMark parser that powers our web mode
- [goldmark-emoji](https://github.com/yuin/goldmark-emoji) — because docs deserve personality

**Search**
- [Bleve](https://github.com/blevesearch/bleve) — full-text search and indexing in pure Go, no compromises
- [fuzzy](https://github.com/sahilm/fuzzy) — fast, intuitive fuzzy matching

**Git**
- [go-git](https://github.com/go-git/go-git) — a complete Git implementation in pure Go — no shelling out, no CGO, just works

**CLI & Config**
- [Cobra](https://github.com/spf13/cobra) — the CLI framework that powers kubectl, hugo, and now fur
- [Viper](https://github.com/spf13/viper) — effortless config from files, env vars, and flags

**Visualization**
- [D3.js](https://d3js.org) — the gold standard for data visualization on the web
- [Mermaid](https://mermaid.js.org) — diagrams from text, rendered beautifully in the browser

**SSH & Networking**
- [x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) — pure Go SSH client from the Go team
- [sftp](https://github.com/pkg/sftp) — SFTP client that makes remote file access feel local
- [ssh_config](https://github.com/kevinburke/ssh_config) — OpenSSH config parser (maintained by Tailscale)
- [knownhosts](https://github.com/skeema/knownhosts) — SSH host key verification with known_hosts support

**Utilities**
- [clipboard](https://github.com/atotto/clipboard) — cross-platform clipboard access
- [fsnotify](https://github.com/fsnotify/fsnotify) — cross-platform file system notifications
- [x/term](https://pkg.go.dev/golang.org/x/term) — terminal size detection from the Go team

Open source is a gift economy. If you use and enjoy any of these projects, consider starring their repos, sponsoring their maintainers, or contributing back. The ecosystem thrives when we pay it forward.

## License

MIT © Benjamin Connelly
