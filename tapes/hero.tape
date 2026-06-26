# fur — hero demo: keyboard-driven inter-document navigation.
# One clean story, no banner cards. ~18s.
Output assets/fur-hero.gif

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set Framerate 24
Set TypingSpeed 55ms

# --- hidden setup: build fur, put it on PATH ---
Hide
Type "mkdir -p /tmp/furbin && go build -o /tmp/furbin/fur ./cmd/fur && export PATH=/tmp/furbin:$PATH && clear"
Enter
Sleep 5s
Show
Sleep 700ms

# Open the docs tree
Type "fur demo/"
Enter
Sleep 2s

# Fuzzy-filter to the README and open it
Type "/"
Sleep 500ms
Type "readme"
Sleep 1.1s
Enter
Sleep 700ms
Enter
Sleep 2.2s

# Follow a link to another document (link picker)
Type "f"
Sleep 1.6s
Down
Sleep 700ms
Enter
Sleep 2.4s

# Jump back through history
Backspace
Sleep 2s

# Reveal the backlinks panel
Type "b"
Sleep 2.4s
Escape
Sleep 1.4s

Hide
Type "q"
Enter
