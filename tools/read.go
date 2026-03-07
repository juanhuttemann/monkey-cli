package tools

import (
	"fmt"
	"os"
	"strings"

	"monkey/api"
)

// ReadTool returns the Tool definition for reading file contents.
func ReadTool() api.Tool {
	return api.Tool{
		Name:        "read",
		Description: "Read the contents of a file and return them with line numbers. Use offset and limit to read a specific range.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to read.",
				},
				"offset": {
					Type:        "integer",
					Description: "1-based line number to start reading from (default: 1).",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of lines to return (default: all).",
				},
			},
			Required: []string{"path"},
		},
	}
}

// ReadExecutor implements api.ToolExecutor for the read tool.
type ReadExecutor struct{}

// ExecuteTool reads the file at input["path"] and returns its contents with
// line numbers in cat-n format. Optional offset (1-based) and limit control
// which lines are returned.
func (r ReadExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("read: missing or empty path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if len(data) == 0 {
		return "", nil
	}

	lines := strings.Split(string(data), "\n")
	// Trim the phantom empty element that Split adds when file ends with \n.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	start := 0
	if off := toInt(input["offset"]); off > 0 {
		start = off - 1 // convert to 0-based
	}
	end := len(lines)
	if lim := toInt(input["limit"]); lim > 0 {
		if start+lim < end {
			end = start + lim
		}
	}

	if start >= len(lines) {
		return "", nil
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%4d\t%s\n", i+1, lines[i])
	}
	return sb.String(), nil
}

// toInt extracts an integer from a map value that may be int or float64
// (JSON numbers decode as float64 in map[string]any).
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}
