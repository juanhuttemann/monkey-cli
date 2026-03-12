package tui

import (
	"strings"
	"testing"

	"github.com/juanhuttemann/monkey-cli/api"
)

func TestFormatToolCall_BashWithOutput(t *testing.T) {
	tc := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "ls"}, Output: "file.txt"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "$ ls") || !strings.Contains(got, "file.txt") {
		t.Errorf("formatToolCall bash = %q, want command and output", got)
	}
}

func TestFormatToolCall_BashNoOutput(t *testing.T) {
	tc := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "rm x"}, Output: ""}
	got := formatToolCall(tc)
	if got != "$ rm x" {
		t.Errorf("formatToolCall bash with no output = %q, want %q", got, "$ rm x")
	}
}

func TestFormatToolCall_ReadTool(t *testing.T) {
	tc := api.ToolCallResult{Name: "read", Input: map[string]any{"path": "/tmp/f.txt"}, Output: "content"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "/tmp/f.txt") || !strings.Contains(got, "content") {
		t.Errorf("formatToolCall read = %q, want path and output", got)
	}
}

func TestFormatToolCall_ReadTool_NoOutput(t *testing.T) {
	tc := api.ToolCallResult{Name: "read", Input: map[string]any{"path": "/tmp/f.txt"}, Output: ""}
	got := formatToolCall(tc)
	if got != "/tmp/f.txt" {
		t.Errorf("formatToolCall read no output = %q, want path only", got)
	}
}

func TestFormatToolCall_EditTool(t *testing.T) {
	tc := api.ToolCallResult{Name: "edit", Input: map[string]any{"path": "/tmp/e.txt"}, Output: "done"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "/tmp/e.txt") {
		t.Errorf("formatToolCall edit = %q, want path", got)
	}
}

func TestFormatToolCall_WriteTool(t *testing.T) {
	tc := api.ToolCallResult{Name: "write", Input: map[string]any{"path": "/tmp/w.txt"}, Output: "ok"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "/tmp/w.txt") {
		t.Errorf("formatToolCall write = %q, want path", got)
	}
}

func TestFormatToolCall_GlobTool_WithPath(t *testing.T) {
	tc := api.ToolCallResult{Name: "glob", Input: map[string]any{"pattern": "**/*.go", "path": "/src"}, Output: "a.go"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "**/*.go") || !strings.Contains(got, "in /src") {
		t.Errorf("formatToolCall glob = %q, want pattern and path", got)
	}
}

func TestFormatToolCall_GlobTool_NoPath(t *testing.T) {
	tc := api.ToolCallResult{Name: "glob", Input: map[string]any{"pattern": "*.go"}, Output: "a.go"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "*.go") {
		t.Errorf("formatToolCall glob no path = %q, want pattern", got)
	}
}

func TestFormatToolCall_GrepTool_WithPath(t *testing.T) {
	tc := api.ToolCallResult{Name: "grep", Input: map[string]any{"pattern": "func main", "path": "/src"}, Output: "match"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "func main") || !strings.Contains(got, "in /src") {
		t.Errorf("formatToolCall grep = %q, want pattern and path", got)
	}
}

func TestFormatToolCall_GrepTool_NoPath(t *testing.T) {
	tc := api.ToolCallResult{Name: "grep", Input: map[string]any{"pattern": "TODO"}, Output: "match"}
	got := formatToolCall(tc)
	if !strings.Contains(got, "TODO") {
		t.Errorf("formatToolCall grep no path = %q, want pattern", got)
	}
}

func TestFormatToolCall_UnknownTool_NoHeader(t *testing.T) {
	tc := api.ToolCallResult{Name: "other", Input: map[string]any{}, Output: "raw output"}
	got := formatToolCall(tc)
	if got != "raw output" {
		t.Errorf("formatToolCall unknown = %q, want raw output", got)
	}
}
