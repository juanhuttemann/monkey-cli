package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- GrepTool definition tests ---

func TestGrepTool_Name(t *testing.T) {
	tool := GrepTool()
	if tool.Name != "grep" {
		t.Errorf("GrepTool().Name = %q, want %q", tool.Name, "grep")
	}
}

func TestGrepTool_HasDescription(t *testing.T) {
	tool := GrepTool()
	if tool.Description == "" {
		t.Error("GrepTool().Description should not be empty")
	}
}

func TestGrepTool_HasRequiredPatternProperty(t *testing.T) {
	tool := GrepTool()
	prop, ok := tool.InputSchema.Properties["pattern"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'pattern' key")
	}
	if prop.Type != "string" {
		t.Errorf("pattern property Type = %q, want %q", prop.Type, "string")
	}
	required := map[string]bool{}
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["pattern"] {
		t.Error("'pattern' should be in InputSchema.Required")
	}
}

func TestGrepTool_HasOptionalPathAndGlobProperties(t *testing.T) {
	tool := GrepTool()
	for _, key := range []string{"path", "glob"} {
		if _, ok := tool.InputSchema.Properties[key]; !ok {
			t.Errorf("InputSchema.Properties should have %q key", key)
		}
	}
}

// --- GrepExecutor tests ---

func TestGrepExecutor_FindsMatch(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\nfoo bar\n"), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "hello",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("result should contain 'hello', got: %q", result)
	}
}

func TestGrepExecutor_OutputFormat(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("line one\nline two\n"), 0o644)

	exec := GrepExecutor{}
	result, _ := exec.ExecuteTool("grep", map[string]any{
		"pattern": "line two",
		"path":    dir,
	})
	// Format: file:line:content
	if !strings.Contains(result, ":2:") {
		t.Errorf("result should contain line number ':2:', got: %q", result)
	}
	if !strings.Contains(result, "line two") {
		t.Errorf("result should contain matched text, got: %q", result)
	}
}

func TestGrepExecutor_NoMatchReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello world\n"), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "zzznomatch",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "" {
		t.Errorf("ExecuteTool() = %q, want empty for no matches", result)
	}
}

func TestGrepExecutor_InvalidRegexReturnsError(t *testing.T) {
	exec := GrepExecutor{}
	_, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "[invalid",
		"path":    ".",
	})
	if err == nil {
		t.Error("ExecuteTool() should return error for invalid regex")
	}
}

func TestGrepExecutor_MissingPatternReturnsError(t *testing.T) {
	exec := GrepExecutor{}
	_, err := exec.ExecuteTool("grep", map[string]any{"path": "."})
	if err == nil {
		t.Error("ExecuteTool() should return error when 'pattern' is missing")
	}
}

func TestGrepExecutor_GlobFilter(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("func main() {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("func not here\n"), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "func",
		"path":    dir,
		"glob":    "*.go",
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "main.go") {
		t.Errorf("result should contain main.go, got: %q", result)
	}
	if strings.Contains(result, "readme.txt") {
		t.Errorf("result should not contain readme.txt when glob=*.go, got: %q", result)
	}
}

func TestGrepExecutor_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("needle\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "c.txt"), []byte("haystack\n"), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "needle",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "a.txt") || !strings.Contains(result, "b.txt") {
		t.Errorf("result should contain both a.txt and b.txt, got: %q", result)
	}
	if strings.Contains(result, "c.txt") {
		t.Errorf("result should not contain c.txt (no match), got: %q", result)
	}
}

func TestGrepExecutor_CapsAt200Results(t *testing.T) {
	dir := t.TempDir()
	// Write a file with 300 matching lines
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("match line\n")
	}
	_ = os.WriteFile(filepath.Join(dir, "big.txt"), []byte(sb.String()), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": "match",
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) > 201 { // 200 results + optional truncation notice
		t.Errorf("result should be capped at 200 matches, got %d lines", len(lines))
	}
}

func TestGrepExecutor_RegexPattern(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "code.go"), []byte("func Foo() {}\nfunc Bar() {}\nvar x = 1\n"), 0o644)

	exec := GrepExecutor{}
	result, err := exec.ExecuteTool("grep", map[string]any{
		"pattern": `^func`,
		"path":    dir,
	})
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if !strings.Contains(result, "Foo") || !strings.Contains(result, "Bar") {
		t.Errorf("regex ^func should match func lines, got: %q", result)
	}
	if strings.Contains(result, "var") {
		t.Errorf("regex ^func should not match var lines, got: %q", result)
	}
}
