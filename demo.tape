Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 800
Set Theme "Dracula"
Set Padding 20
Set PlaybackSpeed 1

# Build and launch the TUI
Hide
Type "go build -o /tmp/monkey-demo . && /tmp/monkey-demo"
Enter
Sleep 10s
Show
Sleep 4s

# Enable auto-approve mode so tool calls run without confirmation
Type "/ape"
Sleep 1s
Ctrl+Enter
Sleep 1s

Type "fix all bugs. you have 5 minutes or no bananas for you."
Sleep 500ms
Ctrl+Enter
Sleep 25s

# Quit
Escape
Sleep 500ms
