package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- GlobTool definition tests ---

func TestGlobTool_Name(t *testing.T) {
	tool := GlobTool()
	if tool.Name != "glob" {
		t.Errorf("GlobTool().Name = %q, want %q", tool.Name, "glob")
	}
}

func TestGlobTool_HasDescription(t *testing.T) {
	tool := GlobTool()
	if tool.Description == "" {
		t.Error("GlobTool().Description should not be empty")
	}
}

func TestGlobTool_InputSchemaType(t *testing.T) {
	tool := GlobTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestGlobTool_HasPatternProperty(t *testing.T) {
	tool := GlobTool()
	prop, ok := tool.InputSchema.Properties["pattern"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'pattern' key")
	}
	if prop.Type != "string" {
		t.Errorf("pattern property Type = %q, want %q", prop.Type, "string")
	}
	if prop.Description == "" {
		t.Error("pattern property Description should not be empty")
	}
}

func TestGlobTool_HasPathProperty(t *testing.T) {
	tool := GlobTool()
	prop, ok := tool.InputSchema.Properties["path"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'path' key")
	}
	if prop.Type != "string" {
		t.Errorf("path property Type = %q, want %q", prop.Type, "string")
	}
}

func TestGlobTool_PatternIsRequired(t *testing.T) {
	tool := GlobTool()
	for _, r := range tool.InputSchema.Required {
		if r == "pattern" {
			return
		}
	}
	t.Error("'pattern' should be in InputSchema.Required")
}

func TestGlobTool_PathIsNotRequired(t *testing.T) {
	tool := GlobTool()
	for _, r := range tool.InputSchema.Required {
		if r == "path" {
			t.Error("'path' should not be in InputSchema.Required")
		}
	}
}

// --- GlobExecutor tests ---

func TestGlobExecutor_MissingPatternReturnsError(t *testing.T) {
	exec := GlobExecutor{}
	_, err := exec.ExecuteTool("glob", map[string]any{})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'pattern' is missing")
	}
}

func TestGlobExecutor_EmptyPatternReturnsError(t *testing.T) {
	exec := GlobExecutor{}
	_, err := exec.ExecuteTool("glob", map[string]any{"pattern": ""})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'pattern' is empty")
	}
}

func TestGlobExecutor_MatchesSingleWildcard(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "")
	writeFile(t, dir, "b.txt", "")
	writeFile(t, dir, "c.go", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "*.txt",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 2 {
		t.Errorf("got %d matches, want 2: %v", len(lines), lines)
	}
	for _, l := range lines {
		if !strings.HasSuffix(l, ".txt") {
			t.Errorf("unexpected match %q", l)
		}
	}
}

func TestGlobExecutor_MatchesDoubleStarRecursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "")
	writeFile(t, dir, "sub/util.go", "")
	writeFile(t, dir, "sub/deep/helper.go", "")
	writeFile(t, dir, "sub/readme.md", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "**/*.go",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 3 {
		t.Errorf("got %d matches, want 3: %v", len(lines), lines)
	}
	for _, l := range lines {
		if !strings.HasSuffix(l, ".go") {
			t.Errorf("unexpected match %q", l)
		}
	}
}

func TestGlobExecutor_DoubleStarAloneMatchesAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "")
	writeFile(t, dir, "sub/b.go", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "**",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 2 {
		t.Errorf("got %d matches, want 2: %v", len(lines), lines)
	}
}

func TestGlobExecutor_QuestionMarkWildcard(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a1.txt", "")
	writeFile(t, dir, "a2.txt", "")
	writeFile(t, dir, "ab.txt", "")
	writeFile(t, dir, "abc.txt", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "a?.txt",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 3 {
		t.Errorf("got %d matches, want 3 (a1, a2, ab): %v", len(lines), lines)
	}
}

func TestGlobExecutor_CharacterClass(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "")
	writeFile(t, dir, "b.txt", "")
	writeFile(t, dir, "c.txt", "")
	writeFile(t, dir, "d.txt", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "[ab].txt",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 2 {
		t.Errorf("got %d matches, want 2 (a, b): %v", len(lines), lines)
	}
}

func TestGlobExecutor_NoMatchReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "*.ts",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestGlobExecutor_DefaultsToCurrentDirectory(t *testing.T) {
	exec := GlobExecutor{}
	_, err := exec.ExecuteTool("glob", map[string]any{"pattern": "*.go"})
	if err != nil {
		t.Errorf("ExecuteTool() without path should not error: %v", err)
	}
}

func TestGlobExecutor_CustomPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "x.txt", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "*.txt",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if result == "" {
		t.Error("expected match in custom path, got empty result")
	}
}

func TestGlobExecutor_SortedByModTimeNewestFirst(t *testing.T) {
	dir := t.TempDir()

	paths := []string{
		filepath.Join(dir, "old.txt"),
		filepath.Join(dir, "mid.txt"),
		filepath.Join(dir, "new.txt"),
	}
	base := time.Now()
	for i, p := range paths {
		os.WriteFile(p, []byte("x"), 0o644)
		ts := base.Add(time.Duration(i) * time.Second)
		os.Chtimes(p, ts, ts)
	}

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "*.txt",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(lines), lines)
	}
	if !strings.HasSuffix(lines[0], "new.txt") {
		t.Errorf("first result should be newest (new.txt), got %q", lines[0])
	}
	if !strings.HasSuffix(lines[2], "old.txt") {
		t.Errorf("last result should be oldest (old.txt), got %q", lines[2])
	}
}

func TestGlobExecutor_PatternWithSubdirPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/main.go", "")
	writeFile(t, dir, "src/util.go", "")
	writeFile(t, dir, "test/main_test.go", "")

	exec := GlobExecutor{}
	result, err := exec.ExecuteTool("glob", map[string]any{
		"pattern": "src/*.go",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}

	lines := splitLines(result)
	if len(lines) != 2 {
		t.Errorf("got %d matches, want 2: %v", len(lines), lines)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "src/") {
			t.Errorf("unexpected match outside src/: %q", l)
		}
	}
}

// helpers

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", rel, err)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
