package tools

import (
	"fmt"
	"os"

	"monkey/api"
)

// ReadTool returns the Tool definition for reading file contents.
func ReadTool() api.Tool {
	return api.Tool{
		Name:        "read",
		Description: "Read the contents of a file and return them as a string.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to read.",
				},
			},
			Required: []string{"path"},
		},
	}
}

// ReadExecutor implements api.ToolExecutor for the read tool.
type ReadExecutor struct{}

// ExecuteTool reads the file at input["path"] and returns its contents.
func (r ReadExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("read: missing or empty path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return string(data), nil
}
