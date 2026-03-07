package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- EditTool definition tests ---

func TestEditTool_Name(t *testing.T) {
	tool := EditTool()
	if tool.Name != "edit" {
		t.Errorf("EditTool().Name = %q, want %q", tool.Name, "edit")
	}
}

func TestEditTool_HasDescription(t *testing.T) {
	tool := EditTool()
	if tool.Description == "" {
		t.Error("EditTool().Description should not be empty")
	}
}

func TestEditTool_InputSchemaType(t *testing.T) {
	tool := EditTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestEditTool_HasRequiredProperties(t *testing.T) {
	tool := EditTool()
	for _, key := range []string{"path", "old_string", "new_string"} {
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

func TestEditTool_AllFieldsAreRequired(t *testing.T) {
	tool := EditTool()
	required := map[string]bool{}
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	for _, key := range []string{"path", "old_string", "new_string"} {
		if !required[key] {
			t.Errorf("%q should be in InputSchema.Required", key)
		}
	}
}

// --- EditExecutor tests ---

func TestEditExecutor_ReplacesString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "hello",
		"new_string": "goodbye",
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "goodbye world\n" {
		t.Errorf("file content = %q, want %q", string(data), "goodbye world\n")
	}
}

func TestEditExecutor_ReturnsDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	exec := EditExecutor{}
	result, err := exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "hello",
		"new_string": "goodbye",
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "---") || !strings.Contains(result, "+++") {
		t.Errorf("expected unified diff output, got: %q", result)
	}
	if !strings.Contains(result, "-hello") || !strings.Contains(result, "+goodbye") {
		t.Errorf("diff should show removed and added lines, got: %q", result)
	}
}

func TestEditExecutor_AmbiguousOldStringReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("foo foo foo\n"), 0o644)

	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "foo",
		"new_string": "bar",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error when old_string matches multiple locations")
	}
	if err != nil && !strings.Contains(err.Error(), "3") {
		t.Errorf("error should mention match count, got: %v", err)
	}
}

func TestEditExecutor_AmbiguousOldString_FileUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	original := "foo foo foo\n"
	_ = os.WriteFile(path, []byte(original), 0o644)

	exec := EditExecutor{}
	_, _ = exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "foo",
		"new_string": "bar",
	})

	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Errorf("file should be unchanged on ambiguous match, got %q", string(data))
	}
}

func TestEditExecutor_OldStringNotFoundReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "nothere",
		"new_string": "x",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error when old_string is not found")
	}
}

func TestEditExecutor_FileNotFoundReturnsError(t *testing.T) {
	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"path":       "/nonexistent/file.txt",
		"old_string": "foo",
		"new_string": "bar",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error for non-existent file")
	}
}

func TestEditExecutor_MissingPathReturnsError(t *testing.T) {
	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"old_string": "foo",
		"new_string": "bar",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'path' is missing")
	}
}

func TestEditExecutor_MissingOldStringReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello\n"), 0o644)

	exec := EditExecutor{}
	_, err := exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"new_string": "bar",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'old_string' is missing")
	}
}

// --- DiffEdit tests ---

func TestDiffEdit_ReturnsDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	diff, err := DiffEdit(path, "hello", "goodbye")
	if err != nil {
		t.Fatalf("DiffEdit() returned error: %v", err)
	}
	if !strings.Contains(diff, "---") || !strings.Contains(diff, "+++") {
		t.Errorf("expected unified diff output, got: %q", diff)
	}
	if !strings.Contains(diff, "-hello") || !strings.Contains(diff, "+goodbye") {
		t.Errorf("diff should show removed and added lines, got: %q", diff)
	}
}

func TestDiffEdit_DoesNotModifyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	original := "hello world\n"
	_ = os.WriteFile(path, []byte(original), 0o644)

	_, _ = DiffEdit(path, "hello", "goodbye")

	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Errorf("DiffEdit should not modify the file; got %q", string(data))
	}
}

func TestDiffEdit_FileNotFoundReturnsError(t *testing.T) {
	_, err := DiffEdit("/nonexistent/file.txt", "foo", "bar")
	if err == nil {
		t.Error("DiffEdit should return error for non-existent file")
	}
}

func TestDiffEdit_AmbiguousOldStringReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("foo foo foo\n"), 0o644)

	_, err := DiffEdit(path, "foo", "bar")
	if err == nil {
		t.Error("DiffEdit should return error when old_string matches multiple locations")
	}
	if err != nil && !strings.Contains(err.Error(), "3") {
		t.Errorf("error should mention match count, got: %v", err)
	}
}

func TestDiffEdit_OldStringNotFoundReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	_, err := DiffEdit(path, "nothere", "x")
	if err == nil {
		t.Error("DiffEdit should return error when old_string is not found")
	}
}

func TestEditExecutor_DoesNotModifyFileOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	original := "hello world\n"
	_ = os.WriteFile(path, []byte(original), 0o644)

	exec := EditExecutor{}
	_, _ = exec.ExecuteTool("edit", map[string]any{
		"path":       path,
		"old_string": "nothere",
		"new_string": "x",
	})

	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Errorf("file should be unchanged on error, got %q", string(data))
	}
}
