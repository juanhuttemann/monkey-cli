package api

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ReadTool definition tests ---

func TestReadTool_Name(t *testing.T) {
	tool := ReadTool()
	if tool.Name != "read" {
		t.Errorf("ReadTool().Name = %q, want %q", tool.Name, "read")
	}
}

func TestReadTool_HasDescription(t *testing.T) {
	tool := ReadTool()
	if tool.Description == "" {
		t.Error("ReadTool().Description should not be empty")
	}
}

func TestReadTool_InputSchemaType(t *testing.T) {
	tool := ReadTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestReadTool_HasPathProperty(t *testing.T) {
	tool := ReadTool()
	prop, ok := tool.InputSchema.Properties["path"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'path' key")
	}
	if prop.Type != "string" {
		t.Errorf("path property Type = %q, want %q", prop.Type, "string")
	}
	if prop.Description == "" {
		t.Error("path property Description should not be empty")
	}
}

func TestReadTool_PathIsRequired(t *testing.T) {
	tool := ReadTool()
	for _, r := range tool.InputSchema.Required {
		if r == "path" {
			return
		}
	}
	t.Error("'path' should be in InputSchema.Required")
}

// --- ReadExecutor tests ---

func TestReadExecutor_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("ExecuteTool() = %q, want %q", result, "hello world")
	}
}

func TestReadExecutor_ReadsMultilineFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.txt")
	content := "line1\nline2\nline3\n"
	os.WriteFile(path, []byte(content), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != content {
		t.Errorf("ExecuteTool() = %q, want %q", result, content)
	}
}

func TestReadExecutor_MissingPathReturnsError(t *testing.T) {
	exec := ReadExecutor{}
	_, err := exec.ExecuteTool("read", map[string]any{})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'path' is missing")
	}
}

func TestReadExecutor_EmptyPathReturnsError(t *testing.T) {
	exec := ReadExecutor{}
	_, err := exec.ExecuteTool("read", map[string]any{"path": ""})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'path' is empty")
	}
}

func TestReadExecutor_NonExistentFileReturnsError(t *testing.T) {
	exec := ReadExecutor{}
	_, err := exec.ExecuteTool("read", map[string]any{"path": "/nonexistent/path/file.txt"})
	if err == nil {
		t.Error("ExecuteTool() should return error for non-existent file")
	}
}

func TestReadExecutor_ReadsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "" {
		t.Errorf("ExecuteTool() = %q, want empty string", result)
	}
}
