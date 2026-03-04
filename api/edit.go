package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/aymanbagabas/go-udiff"
)

// EditTool returns the Tool definition for editing a file by replacing a string.
func EditTool() Tool {
	return Tool{
		Name:        "edit",
		Description: "Replace the first occurrence of old_string with new_string in a file.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDef{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to edit.",
				},
				"old_string": {
					Type:        "string",
					Description: "The exact string to find and replace (must be unique enough to identify the location).",
				},
				"new_string": {
					Type:        "string",
					Description: "The string to replace old_string with.",
				},
			},
			Required: []string{"path", "old_string", "new_string"},
		},
	}
}

// DiffEdit computes a unified diff of replacing the first occurrence of oldStr
// with newStr in the file at path, without modifying the file.
func DiffEdit(path, oldStr, newStr string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	original := string(data)
	if !strings.Contains(original, oldStr) {
		return "", fmt.Errorf("edit: old_string not found in %s", path)
	}
	updated := strings.Replace(original, oldStr, newStr, 1)
	return udiff.Unified(path, path, original, updated), nil
}

// EditExecutor implements ToolExecutor for the edit tool.
type EditExecutor struct{}

// ExecuteTool replaces the first occurrence of old_string with new_string in the file at path.
// Returns a unified diff of the change on success.
func (e EditExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("edit: missing or empty path")
	}
	oldStr, ok := input["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("edit: missing old_string")
	}
	newStr, _ := input["new_string"].(string)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	original := string(data)

	if !strings.Contains(original, oldStr) {
		return "", fmt.Errorf("edit: old_string not found in %s", path)
	}

	updated := strings.Replace(original, oldStr, newStr, 1)

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}

	diff := udiff.Unified(path, path, original, updated)
	return diff, nil
}
