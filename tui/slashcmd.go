package tui

import "strings"

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
	{Name: "compact", Desc: "summarize and compress conversation history"},
}

// parseSlashCommand extracts the slash command from input (e.g. "/exit").
// Returns ok=true when input is a single-line string starting with "/" followed
// by at least one word. A lone "/" returns ok=false.
func parseSlashCommand(input string) (cmd string, ok bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\n") {
		return "", false
	}
	// Take the first word after the slash.
	// A lone "/" with no word following is not a valid command.
	parts := strings.Fields(trimmed[1:])
	if len(parts) == 0 {
		return "", false
	}
	return "/" + parts[0], true
}
