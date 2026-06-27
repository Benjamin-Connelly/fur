# Contributing

## Development Setup

```bash
git clone https://github.com/Benjamin-Connelly/fur
cd fur
go build -o fur ./cmd/fur
```

## Running Tests

```bash
go test ./...
```

## Code Style

- Pure Go, no CGO
- Idiomatic error handling
- Table-driven tests
- No external web frameworks

## Pull Requests

1. Fork the repo
2. Create a feature branch
3. Write tests
4. Submit PR against `master`

Back to [project docs](README.md).
