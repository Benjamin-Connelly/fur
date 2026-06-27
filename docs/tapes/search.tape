# fur full-text search (/ then Tab). ~12s.
Output docs/assets/fur-search.gif

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
Sleep 7s
Show
Sleep 700ms
Type "fur docs/demo/"
Enter
Sleep 2.4s
Type "/"
Sleep 400ms
Tab
Sleep 600ms
Type "trillions"
Sleep 1.6s
Enter
Sleep 800ms
Enter
Sleep 2.6s
Hide
Type "q"
Enter
