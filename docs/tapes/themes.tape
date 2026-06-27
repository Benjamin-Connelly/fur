# fur theme switching across dark palettes. ~14s.
Output docs/assets/fur-themes.gif

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
Sleep 2.2s
Type "/"
Sleep 300ms
Type "field"
Sleep 800ms
Enter
Sleep 500ms
Enter
Sleep 1.6s
Type ":theme dracula"
Enter
Sleep 1.6s
Type ":theme nord"
Enter
Sleep 1.6s
Type ":theme gruvbox"
Enter
Sleep 1.6s
Type ":theme tokyonight-night"
Enter
Sleep 1.8s
Hide
Type "q"
Enter
