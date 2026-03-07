package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SlashCommand represents a supported slash command.
type SlashCommand struct {
	Name string // without leading slash, e.g. "exit"
	Desc string
}

// slashCommands is the authoritative list of supported commands.
var slashCommands = []SlashCommand{
	{Name: "exit", Desc: "quit monkey"},
	{Name: "clear", Desc: "start a fresh session"},
	{Name: "model", Desc: "switch model"},
	{Name: "ape", Desc: "toggle unrestricted mode (skip tool approval)"},
	{Name: "copy", Desc: "copy last assistant response to clipboard"},
}

// CommandPicker is a keyboard-navigable dropdown for slash command completion.
type CommandPicker struct {
	filtered []SlashCommand
	cursor   int
	active   bool
	width    int
}

// NewCommandPicker returns an inactive command picker.
func NewCommandPicker(width int) CommandPicker {
	return CommandPicker{width: width}
}

// Activate makes the picker visible.
func (cp *CommandPicker) Activate() { cp.active = true }

// Deactivate hides the picker.
func (cp *CommandPicker) Deactivate() { cp.active = false }

// IsActive reports whether the picker is visible.
func (cp CommandPicker) IsActive() bool { return cp.active }

// SetWidth updates the display width.
func (cp *CommandPicker) SetWidth(w int) { cp.width = w }

// SetQuery filters commands by fuzzy-matching against the query.
func (cp *CommandPicker) SetQuery(query string) {
	if query == "" {
		cp.filtered = slashCommands
		cp.cursor = 0
		return
	}
	cp.filtered = nil
	for _, cmd := range slashCommands {
		if fuzzyMatch(query, cmd.Name) {
			cp.filtered = append(cp.filtered, cmd)
		}
	}
	cp.cursor = 0
}

// SelectedCommand returns the full slash command string (e.g. "/exit") for the
// currently highlighted entry, or "" when inactive or no match.
func (cp CommandPicker) SelectedCommand() string {
	if !cp.active || len(cp.filtered) == 0 || cp.cursor < 0 || cp.cursor >= len(cp.filtered) {
		return ""
	}
	return "/" + cp.filtered[cp.cursor].Name
}

// Update handles Up/Down cursor navigation.
func (cp CommandPicker) Update(msg tea.Msg) (CommandPicker, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return cp, nil
	}
	switch key.Type {
	case tea.KeyDown:
		if cp.cursor < len(cp.filtered)-1 {
			cp.cursor++
		}
	case tea.KeyUp:
		if cp.cursor > 0 {
			cp.cursor--
		}
	}
	return cp, nil
}

// View renders the command picker dropdown. Returns "" when inactive.
func (cp CommandPicker) View() string {
	if !cp.active {
		return ""
	}
	if len(cp.filtered) == 0 {
		return FilePickerStyle(cp.width).Render("  no commands found")
	}
	var sb strings.Builder
	for i, cmd := range cp.filtered {
		line := "/" + cmd.Name + "  " + cmd.Desc
		if i == cp.cursor {
			sb.WriteString(FilePickerCursorStyle().Render("> " + line))
		} else {
			sb.WriteString("  " + line)
		}
		if i < len(cp.filtered)-1 {
			sb.WriteByte('\n')
		}
	}
	return FilePickerStyle(cp.width).Render(sb.String())
}

// detectCommandQuery returns the partial command name being typed when input
// starts with "/" on a single line with no spaces. Active means the picker
// should be shown.
func detectCommandQuery(input string) (query string, active bool) {
	if !strings.HasPrefix(input, "/") || strings.Contains(input, "\n") {
		return "", false
	}
	after := input[1:]
	if strings.ContainsAny(after, " \t") {
		return "", false
	}
	return after, true
}

// parseSlashCommand extracts the slash command from input (e.g. "/exit").
// Returns ok=true when input is a single-line string starting with "/".
func parseSlashCommand(input string) (cmd string, ok bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\n") {
		return "", false
	}
	// Take the first word after the slash
	parts := strings.Fields(trimmed[1:])
	if len(parts) == 0 {
		return "/", true
	}
	return "/" + parts[0], true
}
