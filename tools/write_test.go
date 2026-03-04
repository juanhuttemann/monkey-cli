package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// --- WriteTool definition tests ---

func TestWriteTool_Name(t *testing.T) {
	tool := WriteTool()
	if tool.Name != "write" {
		t.Errorf("WriteTool().Name = %q, want %q", tool.Name, "write")
	}
}

func TestWriteTool_HasDescription(t *testing.T) {
	tool := WriteTool()
	if tool.Description == "" {
		t.Error("WriteTool().Description should not be empty")
	}
}

func TestWriteTool_InputSchemaType(t *testing.T) {
	tool := WriteTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestWriteTool_HasPathAndContentProperties(t *testing.T) {
	tool := WriteTool()
	for _, key := range []string{"path", "content"} {
		prop, ok := tool.InputSchema.Properties[key]
		if !ok {
			t.Fatalf("InputSchema.Properties should have %q key", key)
		}
		if prop.Type != "string" {
			t.Errorf("%s property Type = %q, want %q", key, prop.Type, "string")
		}
		if prop.Description == "" {
			t.Errorf("%s property Description should not be empty", key)
		}
	}
}

func TestWriteTool_PathAndContentAreRequired(t *testing.T) {
	tool := WriteTool()
	required := map[string]bool{}
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	for _, key := range []string{"path", "content"} {
		if !required[key] {
			t.Errorf("%q should be in InputSchema.Required", key)
		}
	}
}

// --- WriteExecutor tests ---

func TestWriteExecutor_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"path": path, "content": "hello"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", string(data), "hello")
	}
}

func TestWriteExecutor_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	os.WriteFile(path, []byte("old content"), 0o644)

	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"path": path, "content": "new content"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "new content" {
		t.Errorf("file content = %q, want %q", string(data), "new content")
	}
}

func TestWriteExecutor_WritesEmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"path": path, "content": ""})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Errorf("file content = %q, want empty", string(data))
	}
}

func TestWriteExecutor_ReturnsSuccessMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	exec := WriteExecutor{}
	result, err := exec.ExecuteTool("write", map[string]any{"path": path, "content": "hi"})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result == "" {
		t.Error("ExecuteTool() should return a non-empty success message")
	}
}

func TestWriteExecutor_MissingPathReturnsError(t *testing.T) {
	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"content": "hi"})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'path' is missing")
	}
}

func TestWriteExecutor_EmptyPathReturnsError(t *testing.T) {
	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"path": "", "content": "hi"})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'path' is empty")
	}
}

func TestWriteExecutor_InvalidPathReturnsError(t *testing.T) {
	exec := WriteExecutor{}
	_, err := exec.ExecuteTool("write", map[string]any{"path": "/nonexistent/dir/file.txt", "content": "hi"})
	if err == nil {
		t.Error("ExecuteTool() should return error when parent directory does not exist")
	}
}
