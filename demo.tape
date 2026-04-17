Output demo/fur-demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1280
Set Height 760
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 24
Set TypingSpeed 35ms

# -------------------------------------------------------------
# Setup (hidden): build binary, load banner helpers
# -------------------------------------------------------------
Hide
Type "export PATH=/usr/local/go/bin:$PATH"
Enter
Type "go build -o /tmp/fur-demo ./cmd/fur"
Enter
Sleep 4s
Type "source demo-banner.sh && clear"
Enter
Sleep 200ms

# Title card
Type "title_card"
Enter
Show
Sleep 3.5s

# =============================================================
# Chapter 1 — navigation: filter, link picker, history
# =============================================================
Hide
Type "banner 'Fuzzy filter · link picker · history' '/ query Enter  —  f pick  —  Backspace back'"
Enter
Show
Sleep 3s
Hide
Type "/tmp/fur-demo demo/"
Enter
Show
Sleep 2s

# fuzzy filter to README.md
Type "/"
Sleep 300ms
Type "read"
Sleep 1s
Enter
Sleep 700ms
Enter
Sleep 1.8s
Down@60ms 8
Sleep 800ms

# link picker — f surfaces every link, Down/Enter to follow
Type "f"
Sleep 1.6s
Down
Sleep 500ms
Enter
Sleep 2s

# history — Backspace walks the stack
Backspace
Sleep 1.8s
Hide
Type "q"
Sleep 300ms

# =============================================================
# Chapter 2 — panels and command palette
# =============================================================
Type "banner 'TOC · backlinks · command palette' 't  —  b  —  : broken'"
Enter
Show
Sleep 3s
Hide
Type "/tmp/fur-demo demo/"
Enter
Show
Sleep 2s

# open architecture.md
Type "/"
Sleep 300ms
Type "arch"
Sleep 900ms
Enter
Sleep 700ms
Enter
Sleep 1.8s

# TOC panel — Enter jumps preview to heading
Type "t"
Sleep 1.8s
Down
Sleep 400ms
Down
Sleep 500ms
Enter
Sleep 1.2s
Escape
Sleep 500ms

# backlinks panel
Type "b"
Sleep 2.2s
Escape
Sleep 500ms

# command palette — : broken lists broken link targets
Type ":"
Sleep 400ms
Type "broken"
Sleep 1.1s
Enter
Sleep 2.5s
Hide
Type "q"
Sleep 300ms

# =============================================================
# Chapter 3 — data and code previews
# =============================================================
Type "banner 'Data previews · syntax highlighting' 'CSV tables  ·  JSON pretty-print  ·  50+ languages'"
Enter
Show
Sleep 3s
Hide
Type "/tmp/fur-demo demo/"
Enter
Show
Sleep 2s

# CSV as a table
Type "/"
Sleep 300ms
Type "metrics"
Sleep 900ms
Enter
Sleep 700ms
Enter
Sleep 2.5s

# JSON pretty-printed
Escape
Sleep 250ms
Type "/"
Sleep 300ms
Type "config"
Sleep 900ms
Enter
Sleep 700ms
Enter
Sleep 2.5s

# Go source with syntax highlighting
Escape
Sleep 250ms
Type "/"
Sleep 300ms
Type "server"
Sleep 900ms
Enter
Sleep 700ms
Enter
Sleep 2.5s
Hide
Type "q"
Sleep 300ms

# =============================================================
# Chapter 4 — search and jump
# =============================================================
Type "banner 'Global heading jump · full-text search' 'Ctrl+G fuzzy  —  /  then Tab (content mode)'"
Enter
Show
Sleep 3s
Hide
Type "/tmp/fur-demo demo/"
Enter
Show
Sleep 2s

# global heading jump
Ctrl+G
Sleep 500ms
Type "contrib"
Sleep 1s
Enter
Sleep 2.2s

# full-text search via filter + Tab
Escape
Sleep 250ms
Type "/"
Sleep 300ms
Tab
Sleep 400ms
Type "mermaid"
Sleep 1.4s
Enter
Sleep 800ms
Enter
Sleep 2.2s
Hide
Type "q"
Sleep 300ms

# =============================================================
# Chapter 5 — themes and help
# =============================================================
Type "banner 'Themes · help overlay' 'Ctrl+T cycle  —  ? keybindings'"
Enter
Show
Sleep 3s
Hide
Type "/tmp/fur-demo demo/"
Enter
Show
Sleep 2s

# open any file
Type "/"
Sleep 300ms
Type "read"
Sleep 800ms
Enter
Sleep 500ms
Enter
Sleep 1.5s

# cycle themes
Ctrl+T
Sleep 1.4s
Ctrl+T
Sleep 1.4s
Ctrl+T
Sleep 1.2s

# help overlay
Type "?"
Sleep 3s
Escape
Sleep 500ms
Hide
Type "q"
Sleep 300ms

# End card
Type "end_card"
Enter
Show
Sleep 4s
