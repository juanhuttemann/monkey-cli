package tui

import (
	"strings"

	"github.com/juanhuttemann/monkey-cli/api"
)

// formatToolCall formats an api.ToolCallResult for display in the conversation.
// For bash: "$ <command>\n<output>". For other tools: shows the key input + output.
func formatToolCall(tc api.ToolCallResult) string {
	if cmd, ok := tc.Input["command"].(string); ok {
		content := "$ " + cmd
		if tc.Output != "" {
			content += "\n" + strings.TrimRight(tc.Output, "\n")
		}
		return content
	}
	// For read/write/edit/glob/grep: prefix with the primary input parameter.
	var header string
	switch tc.Name {
	case "read", "write", "edit":
		if path, ok := tc.Input["path"].(string); ok && path != "" {
			header = path
		}
	case "glob":
		if pat, ok := tc.Input["pattern"].(string); ok && pat != "" {
			header = pat
			if p, ok := tc.Input["path"].(string); ok && p != "" {
				header += " in " + p
			}
		}
	case "grep":
		if pat, ok := tc.Input["pattern"].(string); ok && pat != "" {
			header = pat
			if p, ok := tc.Input["path"].(string); ok && p != "" {
				header += " in " + p
			}
		}
	}
	if header == "" {
		return tc.Output
	}
	if tc.Output == "" {
		return header
	}
	return header + "\n" + strings.TrimRight(tc.Output, "\n")
}
