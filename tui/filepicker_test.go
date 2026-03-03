package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- fuzzyMatch ---

func TestFuzzyMatch_Subsequence(t *testing.T) {
	if !fuzzyMatch("mai", "src/main.go") {
		t.Error("fuzzyMatch('mai', 'src/main.go') = false, want true")
	}
}

func TestFuzzyMatch_ExactMatch(t *testing.T) {
	if !fuzzyMatch("main.go", "main.go") {
		t.Error("fuzzyMatch('main.go', 'main.go') = false, want true")
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	if fuzzyMatch("xyz", "main.go") {
		t.Error("fuzzyMatch('xyz', 'main.go') = true, want false")
	}
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	if !fuzzyMatch("", "main.go") {
		t.Error("fuzzyMatch('', 'main.go') = false, want true (empty matches everything)")
	}
}

func TestFuzzyMatch_EmptyTarget(t *testing.T) {
	if fuzzyMatch("x", "") {
		t.Error("fuzzyMatch('x', '') = true, want false")
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	if !fuzzyMatch("MAIN", "src/main.go") {
		t.Error("fuzzyMatch('MAIN', 'src/main.go') = false, want true (case-insensitive)")
	}
}

func TestFuzzyMatch_PathSeparator(t *testing.T) {
	if !fuzzyMatch("tmod", "tui/model.go") {
		t.Error("fuzzyMatch('tmod', 'tui/model.go') = false, want true")
	}
}

// --- detectMentionQuery ---

func TestDetectMentionQuery_AtOnly(t *testing.T) {
	query, active := detectMentionQuery("@")
	if !active {
		t.Error("active = false, want true")
	}
	if query != "" {
		t.Errorf("query = %q, want ''", query)
	}
}

func TestDetectMentionQuery_AtWithQuery(t *testing.T) {
	query, active := detectMentionQuery("@main")
	if !active {
		t.Error("active = false, want true")
	}
	if query != "main" {
		t.Errorf("query = %q, want 'main'", query)
	}
}

func TestDetectMentionQuery_TrailingSpace(t *testing.T) {
	_, active := detectMentionQuery("@main ")
	if active {
		t.Error("active = true after trailing space, want false")
	}
}

func TestDetectMentionQuery_NoAt(t *testing.T) {
	_, active := detectMentionQuery("hello world")
	if active {
		t.Error("active = true with no @, want false")
	}
}

func TestDetectMentionQuery_Empty(t *testing.T) {
	_, active := detectMentionQuery("")
	if active {
		t.Error("active = true for empty input, want false")
	}
}

func TestDetectMentionQuery_MidSentence(t *testing.T) {
	query, active := detectMentionQuery("check @src/mai")
	if !active {
		t.Error("active = false for mid-sentence @, want true")
	}
	if query != "src/mai" {
		t.Errorf("query = %q, want 'src/mai'", query)
	}
}

func TestDetectMentionQuery_MultipleAt_LastWins(t *testing.T) {
	query, active := detectMentionQuery("@foo @bar")
	if !active {
		t.Error("active = false for '@foo @bar', want true")
	}
	if query != "bar" {
		t.Errorf("query = %q, want 'bar'", query)
	}
}

func TestDetectMentionQuery_CompletedThenNew(t *testing.T) {
	query, active := detectMentionQuery("@foo.go @bar")
	if !active {
		t.Error("active = false for '@foo.go @bar', want true")
	}
	if query != "bar" {
		t.Errorf("query = %q, want 'bar'", query)
	}
}

// --- replaceCurrentMention ---

func TestReplaceCurrentMention_Basic(t *testing.T) {
	got := replaceCurrentMention("@mai", "src/main.go")
	want := "@src/main.go "
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReplaceCurrentMention_MidText(t *testing.T) {
	got := replaceCurrentMention("check @mai", "src/main.go")
	want := "check @src/main.go "
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReplaceCurrentMention_EmptyQuery(t *testing.T) {
	got := replaceCurrentMention("@", "main.go")
	want := "@main.go "
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- expandMentions ---

func TestExpandMentions_NoMentions(t *testing.T) {
	got := expandMentions("hello world")
	if got != "hello world" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestExpandMentions_NonExistentFile(t *testing.T) {
	got := expandMentions("check @__nonexistent_file__.go")
	if got != "check @__nonexistent_file__.go" {
		t.Errorf("non-existent file: got %q, want original text", got)
	}
}

func TestExpandMentions_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := expandMentions("check @" + path)
	if !strings.Contains(got, "package main") {
		t.Errorf("file contents not included, got: %q", got)
	}
	if !strings.Contains(got, "File: "+path) {
		t.Errorf("file label not included, got: %q", got)
	}
}

func TestExpandMentions_DuplicateMentions_IncludedOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.go")
	if err := os.WriteFile(path, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := expandMentions("@" + path + " and @" + path)
	count := strings.Count(got, "File: "+path)
	if count != 1 {
		t.Errorf("duplicate mention included %d times, want 1", count)
	}
}

func TestExpandMentions_PreservesOriginalText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.go")
	if err := os.WriteFile(path, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := expandMentions("see @" + path + " please")
	if !strings.HasPrefix(got, "see @"+path+" please") {
		t.Errorf("original text not preserved at start, got: %q", got)
	}
}

// --- NewFilePicker ---

func TestNewFilePicker_Inactive(t *testing.T) {
	fp := NewFilePicker(80)
	if fp.IsActive() {
		t.Error("NewFilePicker should be inactive by default")
	}
}

func TestNewFilePicker_NoSelectedFile(t *testing.T) {
	fp := NewFilePicker(80)
	if got := fp.SelectedFile(); got != "" {
		t.Errorf("SelectedFile = %q, want ''", got)
	}
}

// --- SetFiles / SetQuery ---

func TestFilePicker_SetFiles_FirstSelected(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go", "c.go"})
	fp.Activate()
	if fp.SelectedFile() != "a.go" {
		t.Errorf("SelectedFile = %q, want 'a.go'", fp.SelectedFile())
	}
}

func TestFilePicker_SetQuery_Filters(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"main.go", "tui/model.go", "api/client.go"})
	fp.Activate()
	fp.SetQuery("model")
	if got := fp.SelectedFile(); got != "tui/model.go" {
		t.Errorf("SelectedFile = %q, want 'tui/model.go'", got)
	}
}

func TestFilePicker_SetQuery_EmptyShowsAll(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go"})
	fp.SetQuery("b")
	fp.SetQuery("") // reset
	fp.Activate()
	if fp.SelectedFile() != "a.go" {
		t.Errorf("SelectedFile after empty query = %q, want 'a.go'", fp.SelectedFile())
	}
}

func TestFilePicker_SetQuery_NoMatch(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"main.go", "model.go"})
	fp.SetQuery("zzzzz")
	if got := fp.SelectedFile(); got != "" {
		t.Errorf("SelectedFile = %q, want '' (no match)", got)
	}
}

// --- Activate / Deactivate ---

func TestFilePicker_ActivateDeactivate(t *testing.T) {
	fp := NewFilePicker(80)
	fp.Activate()
	if !fp.IsActive() {
		t.Error("After Activate, IsActive = false, want true")
	}
	fp.Deactivate()
	if fp.IsActive() {
		t.Error("After Deactivate, IsActive = true, want false")
	}
}

// --- Navigation ---

func TestFilePicker_NavigateDown(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go", "c.go"})
	fp.Activate()
	first := fp.SelectedFile()
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyDown})
	second := fp.SelectedFile()
	if first == second {
		t.Errorf("Down did not move cursor: before=%q, after=%q", first, second)
	}
}

func TestFilePicker_NavigateUp_AtTop_Noop(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go"})
	fp.Activate()
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if fp.SelectedFile() != "a.go" {
		t.Errorf("Up at top: SelectedFile = %q, want 'a.go'", fp.SelectedFile())
	}
}

func TestFilePicker_NavigateDown_AtBottom_Noop(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go"})
	fp.Activate()
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyDown})
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyDown}) // try to go past last
	if fp.SelectedFile() != "b.go" {
		t.Errorf("Down at bottom: SelectedFile = %q, want 'b.go'", fp.SelectedFile())
	}
}

func TestFilePicker_NavigateDownThenUp(t *testing.T) {
	fp := NewFilePicker(80)
	fp.SetFiles([]string{"a.go", "b.go", "c.go"})
	fp.Activate()
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyDown})
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyDown})
	fp, _ = fp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if fp.SelectedFile() != "b.go" {
		t.Errorf("Down↓↓↑: SelectedFile = %q, want 'b.go'", fp.SelectedFile())
	}
}

// --- LoadFilesCmd ---

func TestLoadFilesCmd_ReturnsFilesMsg(t *testing.T) {
	msg := LoadFilesCmd()()
	loaded, ok := msg.(FilesLoadedMsg)
	if !ok {
		t.Fatalf("got %T, want FilesLoadedMsg", msg)
	}
	if len(loaded.Files) == 0 {
		t.Error("LoadFilesCmd found no files in project directory")
	}
}

func TestLoadFilesCmd_SkipsGitDir(t *testing.T) {
	msg := LoadFilesCmd()()
	loaded := msg.(FilesLoadedMsg)
	for _, f := range loaded.Files {
		if strings.HasPrefix(filepath.ToSlash(f), ".git/") {
			t.Errorf("LoadFilesCmd included .git file: %s", f)
		}
	}
}

func TestLoadFilesCmd_SkipsHiddenDirs(t *testing.T) {
	msg := LoadFilesCmd()()
	loaded := msg.(FilesLoadedMsg)
	for _, f := range loaded.Files {
		parts := strings.Split(filepath.ToSlash(f), "/")
		for _, part := range parts[:len(parts)-1] { // check dir components only
			if strings.HasPrefix(part, ".") {
				t.Errorf("LoadFilesCmd included file in hidden dir: %s", f)
			}
		}
	}
}
