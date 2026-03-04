package api

import (
	"fmt"
	"os"
)

// WriteTool returns the Tool definition for writing file contents.
func WriteTool() Tool {
	return Tool{
		Name:        "write",
		Description: "Write content to a file, creating or overwriting it.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDef{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to write.",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file.",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// WriteExecutor implements ToolExecutor for the write tool.
type WriteExecutor struct{}

// ExecuteTool writes input["content"] to the file at input["path"].
func (w WriteExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("write: missing or empty path")
	}
	content, _ := input["content"].(string)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}
