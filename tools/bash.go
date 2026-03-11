package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/juanhuttemann/monkey-cli/api"
)

const (
	defaultBashTimeout = 30 * time.Second
	maxBashOutputBytes = 50 * 1024
)

// truncateOutput caps raw command output at maxBashOutputBytes and appends a
// notice when the limit is exceeded.
func truncateOutput(out []byte) string {
	if len(out) <= maxBashOutputBytes {
		return string(out)
	}
	return string(out[:maxBashOutputBytes]) + fmt.Sprintf("\n[output truncated: %d bytes total, showing first %d]", len(out), maxBashOutputBytes)
}

// BashTool returns the Tool definition for executing bash commands.
func BashTool() api.Tool {
	return api.Tool{
		Name:        "bash",
		Description: "Execute a bash command and return its combined stdout and stderr output.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"command": {
					Type:        "string",
					Description: "The bash command to execute.",
				},
			},
			Required: []string{"command"},
		},
	}
}

// BashExecutor implements api.ToolExecutor for the bash tool.
// Timeout controls the execution deadline; zero means 30s default.
type BashExecutor struct {
	Timeout time.Duration
}

func (b BashExecutor) timeout() time.Duration {
	if b.Timeout > 0 {
		return b.Timeout
	}
	return defaultBashTimeout
}

// ExecuteTool runs the bash command from input["command"] and returns combined output.
// Returns an error on missing/empty command, timeout, or non-zero exit code.
func (b BashExecutor) ExecuteTool(ctx context.Context, _ string, input map[string]any) (string, error) {
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("bash: missing or empty command")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, b.timeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	out, err := cmd.CombinedOutput()
	result := truncateOutput(out)
	if err != nil {
		if ctx.Err() != nil {
			return result, fmt.Errorf("bash: command timed out after %s", b.timeout())
		}
		return result, fmt.Errorf("bash: command failed: %w", err)
	}
	return result, nil
}
