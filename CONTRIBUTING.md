# Contributing to Lookit

## Development Setup

```bash
# Clone
git clone https://github.com/Benjamin-Connelly/lookit.git
cd lookit

# Build
go build -o lookit ./cmd/lookit

# Test
go test ./...

# Run
./lookit .
```

## Requirements

- Go 1.21+
- No CGO dependencies

## Architecture

- `cmd/lookit/main.go` — CLI entry point (Cobra commands)
- `internal/tui/` — Bubble Tea TUI (split-pane, preview, keys, links)
- `internal/web/` — stdlib net/http server (Goldmark, SSE, go:embed)
- `internal/index/` — File walker, fuzzy search, link graph, watcher
- `internal/render/` — Glamour (TUI) and Chroma (syntax) wrappers
- `internal/git/` — go-git integration, permalink generation
- `internal/config/` — Viper config loader

## Guidelines

- Pure Go, no CGO. Must cross-compile to linux/darwin x amd64/arm64.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Table-driven tests where applicable.
- Keep commits focused: one logical change per commit.
- Use conventional commit format: `feat(scope): summary`

## Testing

```bash
go test ./...          # Run all tests
go test -race ./...    # Race detector
go vet ./...           # Static analysis
```

## Pull Requests

1. Fork the repo and create a feature branch
2. Write tests for new functionality
3. Ensure `go test ./...` and `go vet ./...` pass
4. Submit a PR with a clear description of the change
