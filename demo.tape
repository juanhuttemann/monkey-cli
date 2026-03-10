Output demo.gif

Require monkey

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 800
Set Theme "Dracula"
Set Padding 20
Set PlaybackSpeed 1

# Launch the TUI
Type "monkey"
Sleep 500ms
Enter
Sleep 4s

# Enable auto-approve mode so tool calls run without confirmation
Type "/ape"
Sleep 1s
Ctrl+Enter
Sleep 1s

Type "fix all bugs. you have 5 minutes."
Sleep 500ms
Ctrl+Enter
Sleep 15s

# Quit
Escape
Sleep 500ms
