package tools

import (
	"os"
	"path/filepath"
	"strings"
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

func TestReadTool_HasOffsetAndLimitProperties(t *testing.T) {
	tool := ReadTool()
	for _, key := range []string{"offset", "limit"} {
		prop, ok := tool.InputSchema.Properties[key]
		if !ok {
			t.Fatalf("InputSchema.Properties should have %q key", key)
		}
		if prop.Type != "integer" {
			t.Errorf("%s property Type = %q, want %q", key, prop.Type, "integer")
		}
		if prop.Description == "" {
			t.Errorf("%s property Description should not be empty", key)
		}
	}
}

// --- ReadExecutor tests ---

func TestReadExecutor_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	_ = os.WriteFile(path, []byte("hello world"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	want := "   1\thello world\n"
	if result != want {
		t.Errorf("ExecuteTool() = %q, want %q", result, want)
	}
}

func TestReadExecutor_ReadsMultilineFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.txt")
	_ = os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	want := "   1\tline1\n   2\tline2\n   3\tline3\n"
	if result != want {
		t.Errorf("ExecuteTool() = %q, want %q", result, want)
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
	_ = os.WriteFile(path, []byte{}, 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "" {
		t.Errorf("ExecuteTool() = %q, want empty string", result)
	}
}

func TestReadExecutor_LineNumbersUseRealLineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nums.txt")
	_ = os.WriteFile(path, []byte("a\nb\nc\n"), 0o644)

	exec := ReadExecutor{}
	result, _ := exec.ExecuteTool("read", map[string]any{"path": path})
	if !strings.Contains(result, "   1\ta") {
		t.Errorf("expected line 1 to be numbered, got: %q", result)
	}
	if !strings.Contains(result, "   3\tc") {
		t.Errorf("expected line 3 to be numbered, got: %q", result)
	}
}

func TestReadExecutor_WithOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path, "offset": 3})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	// offset=3 means start at line 3 (1-based), so lines 3 and 4
	if strings.Contains(result, "   1\t") || strings.Contains(result, "   2\t") {
		t.Errorf("offset=3 should skip lines 1 and 2, got: %q", result)
	}
	if !strings.Contains(result, "   3\tc") {
		t.Errorf("offset=3 should include line 3, got: %q", result)
	}
}

func TestReadExecutor_WithLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path, "limit": 2})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	// limit=2 means only lines 1-2
	if strings.Contains(result, "   3\t") || strings.Contains(result, "   4\t") {
		t.Errorf("limit=2 should exclude lines 3 and 4, got: %q", result)
	}
	want := "   1\ta\n   2\tb\n"
	if result != want {
		t.Errorf("ExecuteTool() = %q, want %q", result, want)
	}
}

func TestReadExecutor_WithOffsetAndLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("a\nb\nc\nd\ne\n"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path, "offset": 2, "limit": 2})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	// offset=2, limit=2 → lines 2 and 3
	want := "   2\tb\n   3\tc\n"
	if result != want {
		t.Errorf("ExecuteTool() = %q, want %q", result, want)
	}
}

func TestReadExecutor_OffsetBeyondEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("a\nb\n"), 0o644)

	exec := ReadExecutor{}
	result, err := exec.ExecuteTool("read", map[string]any{"path": path, "offset": 100})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "" {
		t.Errorf("offset beyond EOF should return empty, got %q", result)
	}
}
