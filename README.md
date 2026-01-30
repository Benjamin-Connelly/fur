# lookit

Beautiful local development server for browsing code, markdown, and files.

Browse your project files with syntax highlighting, markdown rendering, and directory listings. No configuration needed.

## Features

**📝 Markdown Rendering**
- GitHub-style markdown with syntax highlighting
- Code blocks, tables, task lists, and more
- Beautiful typography optimized for reading

**💻 Code Viewing**
- Syntax highlighting for 50+ languages
- JavaScript, TypeScript, Python, Go, Rust, Java, and more
- Auto-detects language from file extension

**📁 Smart Directory Listings**
- Emoji file icons for quick recognition
- Respects `.gitignore` by default
- Human-readable sizes and dates
- Sorted: directories first, alphabetically second

**🔒 Secure by Default**
- HTTPS with local certificates (optional)
- Path traversal protection
- No directory listing outside served directory

**🎨 Modern Dark Theme**
- Clean, professional interface
- Responsive mobile-friendly design
- Breadcrumb navigation

## Installation

### Global Install (Recommended)

```bash
npm install -g lookit
```

### Run Without Installing

```bash
npx lookit
```

## Usage

### Basic Usage

```bash
# Serve current directory
lookit

# Serve specific directory
lookit /path/to/project

# Use custom port
lookit --port 8080

# Auto-open browser
lookit --open
```

The server starts on **port 7777** 🍀 by default.

### Command Options

```
--port <number>      Port to listen on (default: 7777)
--host <address>     Host to bind to (default: 127.0.0.1)
--open               Open browser automatically
--all                Show hidden files (including .gitignore matches)
--no-https           Use HTTP only, skip HTTPS
--https-only         Require HTTPS, fail if certificates not found
--no-dirlist         Disable directory listings
-h, --help           Show help message
```

## File Type Support

| Type | Extensions | Features |
|------|-----------|----------|
| **Markdown** | `.md`, `.mdx` | GitHub-style rendering with syntax highlighting |
| **Code** | `.js`, `.ts`, `.py`, `.go`, `.rs`, `.java`, etc. | Syntax highlighting for 50+ languages |
| **Images** | `.png`, `.jpg`, `.gif`, `.svg`, `.webp` | Native browser display |
| **Videos** | `.mp4`, `.webm`, `.mov` | Native browser playback |
| **Audio** | `.mp3`, `.wav`, `.ogg`, `.flac` | Native browser playback |
| **PDFs** | `.pdf` | Native browser viewer |
| **Binary** | All others | Download with preview card |

## HTTPS Setup (Optional)

For local HTTPS with trusted certificates:

### 1. Install mkcert

```bash
# Ubuntu/Debian
sudo apt install -y mkcert libnss3-tools

# macOS
brew install mkcert

# Windows
choco install mkcert
```

### 2. Install Local CA

```bash
mkcert -install
```

### 3. Generate Certificates

```bash
mkdir -p ~/.config/lookit
mkcert -cert-file ~/.config/lookit/localhost.pem \
       -key-file ~/.config/lookit/localhost-key.pem \
       localhost 127.0.0.1 ::1
```

### 4. Restart lookit

```bash
lookit
```

Now visit **https://localhost:7777** with no browser warnings.

## Examples

### Browse Project Documentation

```bash
cd ~/my-project
lookit docs/
```

### View Markdown with Code Examples

```bash
lookit README.md
```

### Check Configuration Files

```bash
lookit config/
```

### Quick File Review

```bash
# Open browser automatically
lookit --open

# Show all files including .gitignore matches
lookit --all
```

## Use Cases

**📖 Documentation Review**
- Read project documentation locally
- Preview markdown before committing
- View code examples with syntax highlighting

**🔍 Code Browsing**
- Quickly browse project structure
- View files without opening editor
- Share read-only view with team

**📝 Writing & Editing**
- Live markdown preview while writing
- Check formatting and code blocks
- View final rendered output

**🎓 Learning & Teaching**
- Browse code examples
- Share code with students
- View tutorials and guides

## Why lookit?

**Zero Configuration**
- No setup files
- No dependencies to install
- Works out of the box

**Fast & Lightweight**
- Starts instantly
- Minimal memory footprint
- No build process

**Beautiful by Default**
- Modern dark theme
- Clean typography
- Professional appearance

**Respects Your Workflow**
- Honors `.gitignore` files
- Secure path handling
- Non-intrusive

## Development

### Run from Source

```bash
git clone https://github.com/yourusername/lookit.git
cd lookit
npm install
node bin/lookit.js
```

### Run Tests

```bash
npm test
```

### Project Structure

```
lookit/
├── bin/
│   └── lookit.js          # CLI entry point
├── src/
│   ├── index.js           # Main server
│   ├── fileHandler.js     # File routing
│   ├── utils.js           # Utilities
│   ├── styles.js          # CSS styles
│   └── templates/         # HTML templates
│       ├── base.js
│       ├── code.js
│       ├── markdown.js
│       ├── directory.js
│       └── binary.js
└── test/
    └── fixtures/          # Test files
```

## Security

**Path Traversal Protection**
- Blocks `../` and absolute paths
- Only serves files within specified directory
- No access to parent directories

**HTTPS Support**
- Optional TLS encryption
- Local trusted certificates
- No browser warnings with mkcert

**Safe File Handling**
- Binary files show preview, not content
- No code execution
- Read-only access

## Troubleshooting

### Port Already in Use

```bash
# Use different port
lookit --port 8080
```

### HTTPS Certificate Warnings

```bash
# Use HTTP instead
lookit --no-https

# Or follow HTTPS setup guide above
```

### Files Not Showing

```bash
# Show hidden/ignored files
lookit --all
```

### Server Won't Start

```bash
# Check if port is available
lsof -i :7777

# Kill existing process
pkill -f lookit
```

## Configuration

lookit looks for HTTPS certificates in `~/.config/lookit/`:
- `localhost.pem` - Certificate file
- `localhost-key.pem` - Private key file

You can specify custom certificate paths:

```bash
lookit --cert /path/to/cert.pem --key /path/to/key.pem
```

## Roadmap

- [ ] Custom themes
- [ ] Watch mode with auto-reload
- [ ] Search within files
- [ ] File editing (optional)
- [ ] Multiple directory support
- [ ] Bookmark favorite paths

## Contributing

Contributions welcome! Please feel free to submit a Pull Request.

## License

MIT © Benjamin Connelly

## Acknowledgments

Built with:
- [markdown-it](https://github.com/markdown-it/markdown-it) - Markdown parser
- [highlight.js](https://highlightjs.org/) - Syntax highlighting
- [ignore](https://github.com/kaelzhang/node-ignore) - .gitignore support

---

**Made with ❤️ for developers who love clean, simple tools.**

Start browsing: `npx lookit`
