# fur — full-text search (/ then Tab for content mode). ~12s.
Output assets/fur-search.gif

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set Framerate 24
Set TypingSpeed 55ms

Hide
Type "mkdir -p /tmp/furbin && go build -o /tmp/furbin/fur ./cmd/fur && export PATH=/tmp/furbin:$PATH && clear"
Enter
Sleep 5s
Show
Sleep 700ms
Type "fur demo/"
Enter
Sleep 2s
Type "/"
Sleep 500ms
Tab
Sleep 600ms
Type "mermaid"
Sleep 1.6s
Enter
Sleep 900ms
Enter
Sleep 2.6s
Hide
Type "q"
Enter
