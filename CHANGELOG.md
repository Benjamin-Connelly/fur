# Changelog

## Unreleased

### Security
- **SSH config exec directives are never honored (regression-guarded).** fur
  reads only `User`, `Hostname`, `Port`, and `IdentityFile` from `~/.ssh/config`
  and dials directly over TCP. `ProxyCommand`, `ProxyJump`, `LocalCommand`, and
  `Match exec` are ignored, so a planted `~/.ssh/config` cannot turn a
  remote-browse into command execution (audit Chain E). A new test guards the
  key allowlist and the absence of any exec/proxy path.
- **Strict script CSP.** `script-src` no longer allows `'unsafe-inline'`. All
  of fur's own scripts (theme toggle, live-reload, Mermaid bootstrap, link
  graph) are now external files under `/__static`, so an injected inline
  `<script>` in rendered content cannot execute (audit Chain D). The Mermaid
  bootstrap also pins `securityLevel: 'strict'`, sanitizing diagram HTML and
  disabling click/script handlers. `cdn.jsdelivr.net` and `d3js.org` remain
  allowlisted for the Mermaid and D3 libraries.
- **Web server refuses non-loopback binds by default.** `fur serve` now
  errors out if `server.host` resolves to a non-loopback address (`0.0.0.0`,
  a LAN IP, an external hostname) unless `--listen-public` is passed, which
  also prints a reachability warning. Previously any `server.host` value
  (including via `FUR_SERVER_HOST`) bound silently, exposing the file,
  search, and document APIs — and thus the whole browsed tree — to other
  hosts and users on the network (audit Chain C).
- **Symlink containment.** The indexer no longer surfaces symlinks whose
  target resolves outside the browse root. Previously a directory adversary
  could plant `notes.md -> ~/.ssh/id_rsa` inside a browsed tree and have the
  TUI preview or `fur serve` read the out-of-root target (audit Chain B). Pass
  `--follow-symlinks` (or set `follow_symlinks: true`) to restore the old
  behavior. `Index.ValidatePath` now also returns the symlink-resolved path so
  callers open exactly the bytes that were validated.
- **Per-project config is now restricted to a display/UX allowlist.** A
  `.fur.{toml,yaml,yml}` discovered by walking up from CWD may only override
  `theme`, `keymap`, `show_hidden`, `ignore`, `scrolloff`, `reading_guide`,
  and `mouse`. Runtime-sensitive keys (`server.*`, `git.*`, `remotes.*`,
  `root`, `debug`) are silently ignored from per-project sources. Previously a
  checked-out hostile repository could ship a `.fur.yaml` that pivoted
  `server.custom_css` onto an attacker-controlled stylesheet, rebound the web
  listener, or injected SSH remotes (audit Chain A).

### Added
- **Named theme system.** 19 built-in palettes — `auto`, `dark`, `light`, `ascii`, plus the Catppuccin (mocha/macchiato/frappe/latte), Gruvbox (dark/light), Dracula, Nord, Solarized (dark/light), Rosé Pine (main/moon/dawn), and TokyoNight (night/storm/moon/day) families. Each palette drives the glamour markdown body, Chroma code highlighting, and lipgloss TUI chrome from one color set. `ctrl+t` cycles through all themes at runtime; `:theme <name>` jumps to a specific one. Any theme name is valid in config and the `--theme` flag. See [docs/themes](docs/themes/).
- `show_hidden` config key and `--show-hidden` persistent flag. When set, dotfiles and dotdirs are surfaced in listings, search, the link graph, and the file watcher. `.git`, `.hg`, `.svn`, and `.bzr` remain filtered regardless.
- `fur config init` writes `~/.config/fur/config.yaml` from a documented template (use `--force` to overwrite; the existing file is preserved as `config.yaml.bak`).
- `fur config path` prints the resolved config path; `fur config show` prints the active merged config.

### Changed
- Markdown rendering now uses a palette-driven glamour style: inline code is a distinct color with **no background block** (previously a padded highlight), and bold/italic/code are visually separated.
- List items now reflow to the pane width and have a blank line between them. Glamour preserves source soft-breaks inside list items and renders lists tight; fur unwraps soft-wrapped block text before rendering and spaces items in post-processing.
- `cat` and piped stdin wrap markdown to the terminal width instead of a fixed 80 columns (falling back to 80 only when output is not a TTY).
- The TUI preview reflows on terminal resize and side-panel toggles without a manual reload, preserving scroll position.
- `ctrl+u` / `ctrl+d` now scroll a full page; `u` / `d` remain half-page.
- Default behavior unifies dotfile and dotdir filtering. Previously, dotdirs were hidden but dotfiles like `.gitignore` appeared at root. Now both are filtered by default. Set `show_hidden: true` (or pass `--show-hidden`) to restore visibility.
- SFTP remote walker no longer applies its own dotfile filter — the indexer is the single source of truth, so local and remote sessions behave identically.

## v1.0.1

Maintenance release. CI drift cleanup, demo improvements, and one latent bug fix.

### Fixed
- `navigateToPath` now applies the `scroll` argument — history back and bookmark restore were silently losing scroll position
- Help overlay header now reads `fur - Key Bindings` (was stuck on the old `Lookit` name)
- `TestManPagesUpToDate` no longer compares cobra's non-deterministic `.SH HISTORY` date, so CI runs on a different calendar day than the last regen pass cleanly
- CI workflows (`ci.yml`, `release.yml`) updated to build `./cmd/fur` and publish `fur-*` artifacts (were still referencing the pre-rename `lookit` paths)
- gofmt drift across seven files
- `errcheck`, `unparam`, and `staticcheck` findings from the accumulated lint backlog

### Changed
- Demo GIF rewritten as a captioned five-chapter walkthrough with a title card, explicit keybind labels, and an end card — previously showed only basic navigation

## v1.0.0

Initial stable release of fur — a dual-mode markdown navigator with TUI and web interfaces.

### Features
- **TUI mode**: Split-pane Bubble Tea interface with fuzzy file search, markdown rendering (Glamour), syntax highlighting (Chroma), and inter-document link navigation
- **Web mode**: stdlib `net/http` server with Goldmark markdown rendering, SSE live reload, security headers, ETag caching
- **MCP server**: Model Context Protocol server exposing 5 tools for AI agent integration
- **Remote browsing**: SSH/SFTP support with ssh-agent, key files, and `~/.ssh/config` integration
- **Link graph**: Bidirectional link tracking with backlinks, broken link detection, and DOT/JSON graph output
- **Full-text search**: Bleve-based search with BM25 scoring, plus fuzzy filename matching
- **Task extraction**: TODO/checkbox extraction with priority markers, tags, and due dates
- **Plugin system**: YAML-based hooks for content transformation (prepend/append/replace)
- **50+ language highlighting**: Chroma-powered syntax highlighting in both TUI and web modes
- **Git integration**: go-git for status, branches, log, and permalink generation (GitHub/GitLab/Bitbucket/Gitea/Codeberg)
- **Man pages**: Embedded man page installer for all subcommands
- **Shell completions**: Bash, Zsh, and Fish completion generation
- **Per-project config**: `.fur.toml`/`.fur.yaml` with automatic discovery (walks up from CWD)
- **Environment diagnostics**: `fur doctor` with 9 checks and colored output

### Distribution
- Homebrew tap: `brew install Benjamin-Connelly/fur/fur`
- Nix flake: `nix run github:Benjamin-Connelly/fur`
- Go install: `go install github.com/Benjamin-Connelly/fur/cmd/fur@v1.0.0`
- Pure Go, no CGO — cross-compiles to linux/darwin on amd64/arm64

### Security
- Path traversal protection via `Index.ValidatePath()` (shared by web and MCP)
- Content Security Policy headers
- Input sanitization on all API endpoints
- No shell-outs (pure Go throughout)
