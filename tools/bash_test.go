package tools

import (
	"strings"
	"testing"
	"time"
)

// --- BashTool definition tests ---

func TestBashTool_Name(t *testing.T) {
	tool := BashTool()
	if tool.Name != "bash" {
		t.Errorf("BashTool().Name = %q, want %q", tool.Name, "bash")
	}
}

func TestBashTool_HasDescription(t *testing.T) {
	tool := BashTool()
	if tool.Description == "" {
		t.Error("BashTool().Description should not be empty")
	}
}

func TestBashTool_InputSchemaType(t *testing.T) {
	tool := BashTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestBashTool_HasCommandProperty(t *testing.T) {
	tool := BashTool()
	prop, ok := tool.InputSchema.Properties["command"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'command' key")
	}
	if prop.Type != "string" {
		t.Errorf("command property Type = %q, want %q", prop.Type, "string")
	}
	if prop.Description == "" {
		t.Error("command property Description should not be empty")
	}
}

func TestBashTool_CommandIsRequired(t *testing.T) {
	tool := BashTool()
	for _, r := range tool.InputSchema.Required {
		if r == "command" {
			return
		}
	}
	t.Error("'command' should be in InputSchema.Required")
}

// --- BashExecutor tests ---

func TestBashExecutor_RunsCommand(t *testing.T) {
	exec := BashExecutor{}
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "echo hello"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("ExecuteTool() = %q, want output containing %q", result, "hello")
	}
}

func TestBashExecutor_CapturesStdout(t *testing.T) {
	exec := BashExecutor{}
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "printf 'line1\nline2\n'"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "line1\nline2\n" {
		t.Errorf("ExecuteTool() = %q, want %q", result, "line1\nline2\n")
	}
}

func TestBashExecutor_CapturesStderr(t *testing.T) {
	exec := BashExecutor{}
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "echo oops >&2"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "oops") {
		t.Errorf("ExecuteTool() = %q, expected stderr to be captured", result)
	}
}

func TestBashExecutor_NonZeroExitReturnsOutput(t *testing.T) {
	exec := BashExecutor{}
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "echo fail && exit 1"})
	if err == nil {
		t.Error("ExecuteTool() should return an error for non-zero exit code")
	}
	if !strings.Contains(result, "fail") {
		t.Errorf("ExecuteTool() = %q, expected output to contain 'fail'", result)
	}
}

func TestBashExecutor_MissingCommandReturnsError(t *testing.T) {
	exec := BashExecutor{}
	_, err := exec.ExecuteTool("bash", map[string]any{})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'command' is missing")
	}
}

func TestBashExecutor_EmptyCommandReturnsError(t *testing.T) {
	exec := BashExecutor{}
	_, err := exec.ExecuteTool("bash", map[string]any{"command": ""})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'command' is empty")
	}
}

func TestBashExecutor_TimesOut(t *testing.T) {
	exec := BashExecutor{Timeout: 100 * time.Millisecond}
	_, err := exec.ExecuteTool("bash", map[string]any{"command": "sleep 10"})
	if err == nil {
		t.Error("ExecuteTool() should return error when command exceeds timeout")
	}
}

func TestBashExecutor_DefaultTimeoutAllowsFastCommands(t *testing.T) {
	exec := BashExecutor{}
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "echo fast"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "fast") {
		t.Errorf("ExecuteTool() = %q, want output containing 'fast'", result)
	}
}

func TestBashExecutor_MultilineScript(t *testing.T) {
	exec := BashExecutor{}
	script := "x=5\necho $x"
	result, err := exec.ExecuteTool("bash", map[string]any{"command": script})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "5") {
		t.Errorf("ExecuteTool() = %q, want output containing '5'", result)
	}
}

func TestBashExecutor_TruncatesLargeOutput(t *testing.T) {
	exec := BashExecutor{}
	// Generate ~60KB of output (exceeds the 50KB cap).
	result, err := exec.ExecuteTool("bash", map[string]any{"command": "python3 -c \"print('x' * 60000)\""})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	const maxOutputBytes = 50 * 1024
	if len(result) <= maxOutputBytes {
		t.Errorf("expected truncation for large output, got %d bytes (under cap)", len(result))
	}
	if !strings.Contains(result, "[output truncated") {
		t.Errorf("expected truncation notice in output, got: %q", result[:min(200, len(result))])
	}
	// The raw content before the notice must not exceed the cap.
	idx := strings.Index(result, "\n[output truncated")
	if idx < 0 {
		idx = strings.Index(result, "[output truncated")
	}
	if idx > maxOutputBytes {
		t.Errorf("content before truncation notice is %d bytes, want <= %d", idx, maxOutputBytes)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
