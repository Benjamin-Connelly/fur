# Contributing to fur

Thanks for your interest in contributing! Whether it's a bug fix, new feature, or documentation improvement, we appreciate the help.

This project follows a [Code of Conduct](CODE_OF_CONDUCT.md) — by participating, you're expected to uphold it. Security issues go through [SECURITY.md](SECURITY.md), not public issues.

## Development Setup

```bash
# Clone
git clone https://github.com/Benjamin-Connelly/fur.git
cd fur

# Build
go build -o fur ./cmd/fur

# Test
go test ./...

# Run
./fur .
```

## Requirements

- Go 1.25+
- No CGO dependencies

## Architecture

- `cmd/fur/main.go` — CLI entry point (Cobra commands)
- `internal/tui/` — Bubble Tea TUI (split-pane, preview, keys, links, panels)
- `internal/web/` — stdlib net/http server (Goldmark, SSE, go:embed)
- `internal/mcp/` — MCP server for AI agent integration
- `internal/index/` — File walker, fuzzy search, full-text search (Bleve), link graph, watcher
- `internal/render/` — Glamour (TUI) and Chroma (syntax) wrappers, heading extraction, image protocols
- `internal/git/` — go-git integration, permalink generation
- `internal/remote/` — SSH/SFTP remote file browsing
- `internal/config/` — Viper config loader, per-project config discovery
- `internal/export/` — Markdown to HTML/PDF export
- `internal/doctor/` — Environment diagnostics
- `internal/plugin/` — YAML hook system
- `internal/tasks/` — TODO extraction
- `internal/manpages/` — Embedded man page installer

## Guidelines

- Pure Go, no CGO. Must cross-compile to linux/darwin x amd64/arm64.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Table-driven tests where applicable.
- Keep commits focused: one logical change per commit.
- Use conventional commit format: `feat(scope): summary`

## Testing

```bash
go test ./...          # Run all tests (560 tests across 14 packages)
go test -race ./...    # Race detector
go vet ./...           # Static analysis
```

## Pull Requests

1. Fork the repo and create a feature branch
2. Write tests for new functionality
3. Ensure `go test ./...` and `go vet ./...` pass
4. Submit a PR with a clear description of what changed and why

## Project Structure

See the [README](README.md) for full feature documentation and keybinding reference.
