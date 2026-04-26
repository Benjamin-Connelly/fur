# Changelog

## Unreleased

### Added
- `show_hidden` config key and `--show-hidden` persistent flag. When set, dotfiles and dotdirs are surfaced in listings, search, the link graph, and the file watcher. `.git`, `.hg`, `.svn`, and `.bzr` remain filtered regardless.
- `fur config init` writes `~/.config/fur/config.yaml` from a documented template (use `--force` to overwrite; the existing file is preserved as `config.yaml.bak`).
- `fur config path` prints the resolved config path; `fur config show` prints the active merged config.

### Changed
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
