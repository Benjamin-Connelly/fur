# Quickstart Guide

## Installation

```bash
brew tap Benjamin-Connelly/fur
brew install fur
```

## Basic Usage

Launch the TUI in any directory:

```bash
fur ~/docs
```

## Web Mode

Start the web server for browser-based viewing:

```bash
fur serve --port 3000 --open
```

## Key Bindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate file list |
| `Enter` | Open file |
| `/` | Filter files |
| `Tab` | Toggle side panel |
| `t` | Cycle theme |
| `q` | Quit |

## Next Steps

- Read the [architecture overview](architecture.md)
- Check the [API reference](api.md)
