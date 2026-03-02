package api

import (
	"fmt"
	"os/exec"
)

// BashTool returns the Tool definition for executing bash commands.
func BashTool() Tool {
	return Tool{
		Name:        "bash",
		Description: "Execute a bash command and return its combined stdout and stderr output.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDef{
				"command": {
					Type:        "string",
					Description: "The bash command to execute.",
				},
			},
			Required: []string{"command"},
		},
	}
}

// BashExecutor implements ToolExecutor for the bash tool.
type BashExecutor struct{}

// ExecuteTool runs the bash command from input["command"] and returns combined output.
// Returns an error on missing/empty command or non-zero exit code (output is still returned).
func (b BashExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("bash: missing or empty command")
	}

	cmd := exec.Command("bash", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("bash: command failed: %w", err)
	}
	return string(out), nil
}
