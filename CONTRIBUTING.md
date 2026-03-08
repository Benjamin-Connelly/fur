# Contributing to Lookit

Thanks for your interest in contributing! Whether it's a bug fix, new feature, or documentation improvement, we appreciate the help.

## Development Setup

```bash
# Clone
git clone https://github.com/Benjamin-Connelly/lookit.git
cd lookit

# Build (either works)
make build
go build -o lookit ./cmd/lookit

# Test
make test
go test ./...

# Run
./lookit .
```

## Requirements

- Go 1.24+
- No CGO dependencies

## Architecture

- `cmd/lookit/main.go` — CLI entry point (Cobra commands)
- `internal/tui/` — Bubble Tea TUI (split-pane, preview, keys, links, panels)
- `internal/web/` — stdlib net/http server (Goldmark, SSE, go:embed)
- `internal/index/` — File walker, fuzzy search, full-text search (Bleve), link graph, watcher
- `internal/render/` — Glamour (TUI) and Chroma (syntax) wrappers, heading extraction
- `internal/git/` — go-git integration, permalink generation
- `internal/config/` — Viper config loader, per-project config discovery
- `internal/export/` — Markdown to HTML export
- `internal/doctor/` — Environment diagnostics
- `internal/plugin/` — YAML hook system
- `internal/tasks/` — TODO extraction

## Guidelines

- Pure Go, no CGO. Must cross-compile to linux/darwin x amd64/arm64.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Table-driven tests where applicable.
- Keep commits focused: one logical change per commit.
- Use conventional commit format: `feat(scope): summary`

## Testing

```bash
go test ./...          # Run all tests (122 tests across 8 packages)
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
