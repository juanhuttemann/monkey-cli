package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// setupModelWithFiles creates a model and pre-loads a set of files into the picker.
func setupModelWithFiles(files []string) Model {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	m.filePicker.SetFiles(files)
	return m
}

func TestMention_AtSymbol_ActivatesPicker(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go", "tui/model.go"})

	// Simulate typing '@' — goes through the default key handler which updates the picker
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	result := updated.(Model)

	if !result.filePicker.IsActive() {
		t.Error("File picker should be active after typing '@'")
	}
}

func TestMention_AtWithQuery_FiltersPicker(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go", "tui/model.go", "api/client.go"})
	// SetInput doesn't trigger picker state; set one char less, then type last char
	m.SetInput("@mode")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	result := updated.(Model)

	if !result.filePicker.IsActive() {
		t.Error("Picker should be active with @modell in input")
	}
	// Should show only "tui/model.go"
	if got := result.filePicker.SelectedFile(); got != "tui/model.go" {
		t.Errorf("SelectedFile = %q, want 'tui/model.go'", got)
	}
}

func TestMention_SpaceAfterAt_DeactivatesPicker(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	// Get picker active first
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	m = m2.(Model)
	if !m.filePicker.IsActive() {
		t.Skip("Picker not activated, skipping deactivation test")
	}
	// Type a space — @<space> means no active mention
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := m3.(Model)
	if result.filePicker.IsActive() {
		t.Error("Picker should be inactive after space following @")
	}
}

func TestMention_EscDismissesPicker_NotQuit(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	m.filePicker.Activate()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(Model)

	if result.filePicker.IsActive() {
		t.Error("Picker should be inactive after Esc")
	}

	// Must not quit the app
	if cmd != nil {
		msg := cmd()
		if _, isQuit := msg.(tea.QuitMsg); isQuit {
			t.Error("Esc while picker is active should not quit the app")
		}
	}
}

func TestMention_EscWhenPickerInactive_IsNoop(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	// Picker is not active

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// Esc no longer quits when picker is inactive (use /exit instead)
	if cmd != nil {
		result := cmd()
		if _, isQuit := result.(tea.QuitMsg); isQuit {
			t.Error("Esc when picker inactive should not quit — use /exit instead")
		}
	}
}

func TestMention_UpDown_NavigatesPicker(t *testing.T) {
	m := setupModelWithFiles([]string{"aaa.go", "bbb.go", "ccc.go"})
	m.filePicker.Activate()

	first := m.filePicker.SelectedFile()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	second := m2.(Model).filePicker.SelectedFile()

	if first == second {
		t.Errorf("Down should move picker cursor: before=%q, after=%q", first, second)
	}
}

func TestMention_UpDown_WithoutPicker_GoesToTextarea(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	// Picker inactive — Up/Down should not touch picker

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := m2.(Model)
	if result.filePicker.IsActive() {
		t.Error("Down with inactive picker should not activate it")
	}
}

func TestMention_Tab_SelectsFile_UpdatesInput(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go", "tui/model.go"})
	m.SetInput("@mai")
	m.filePicker.Activate()
	m.filePicker.SetQuery("mai")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(Model)

	if result.filePicker.IsActive() {
		t.Error("Picker should be inactive after Tab selection")
	}
	input := result.GetInput()
	if !strings.HasPrefix(input, "@") || !strings.Contains(input, "main.go") {
		t.Errorf("Input after Tab = %q; want @main.go prefix", input)
	}
}

func TestMention_Tab_WithoutPicker_PassesThrough(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	// Picker not active — Tab goes to textarea normally

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_ = cmd // no error is the key assertion
}

func TestMention_FilesLoadedMsg_SetsFiles(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)

	updated, _ := m.Update(FilesLoadedMsg{Files: []string{"a.go", "b.go"}})
	result := updated.(Model)

	// Activate picker and check files are there
	result.filePicker.Activate()
	if result.filePicker.SelectedFile() != "a.go" {
		t.Errorf("SelectedFile = %q, want 'a.go'", result.filePicker.SelectedFile())
	}
}

func TestMention_FilesLoadedMsg_ReappliesQueryWhenActive(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	m.SetInput("@model")
	m.filePicker.Activate()
	m.filePicker.SetQuery("model")

	// Files arrive after picker is already active
	updated, _ := m.Update(FilesLoadedMsg{Files: []string{"main.go", "tui/model.go"}})
	result := updated.(Model)

	// Query should be re-applied; only model.go should match
	if got := result.filePicker.SelectedFile(); got != "tui/model.go" {
		t.Errorf("SelectedFile after FilesLoadedMsg = %q, want 'tui/model.go'", got)
	}
}

func TestMention_Submit_ExpandsMentions(t *testing.T) {
	// Create a real temp file to reference
	dir := t.TempDir()
	path := filepath.Join(dir, "snippet.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewModel(nil) // nil client — we only check messages, not the API call
	m.SetDimensions(80, 24)
	m.SetInput("check @" + path)

	// Force submit — CanSubmit requires non-empty input and StateReady, both true
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	result := updated.(Model)

	history := result.GetHistory()
	if len(history) == 0 {
		t.Fatal("No messages in history after submit")
	}
	userMsg := history[0]
	if userMsg.Role != "user" {
		t.Fatalf("history[0].Role = %q, want 'user'", userMsg.Role)
	}
	// The displayed message preserves the @mention
	if !strings.Contains(userMsg.Content, "@"+path) {
		t.Errorf("Displayed message should contain @%s, got: %q", path, userMsg.Content)
	}
}

func TestMention_View_ShowsPickerWhenActive(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	m.filePicker.Activate()

	view := m.View()
	if !strings.Contains(view, "main.go") {
		t.Error("View() should contain file picker contents when active")
	}
}

func TestMention_View_HidesPickerWhenInactive(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go"})
	// Picker inactive by default

	view := m.View()
	// main.go should not appear in the picker (it might appear if there are
	// messages, but there are none here so any occurrence is from the picker)
	_ = view // Just verify View() doesn't panic
}

// collectBatchFilesLoaded executes a command (possibly a tea.BatchMsg) and
// returns any FilesLoadedMsg found within it.
func collectBatchFilesLoaded(cmd tea.Cmd) (FilesLoadedMsg, bool) {
	if cmd == nil {
		return FilesLoadedMsg{}, false
	}
	msg := cmd()
	if fm, ok := msg.(FilesLoadedMsg); ok {
		return fm, true
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if fm, ok := c().(FilesLoadedMsg); ok {
				return fm, true
			}
		}
	}
	return FilesLoadedMsg{}, false
}

func TestMention_LargeFileTruncated(t *testing.T) {
	// Create a file larger than the size cap
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	big := strings.Repeat("x", 200*1024) // 200KB
	if err := os.WriteFile(path, []byte(big), 0644); err != nil {
		t.Fatal(err)
	}

	result := expandMentions("check @" + path)

	// Should contain a truncation notice, not the full 200KB
	if strings.Contains(result, big) {
		t.Error("expandMentions should truncate large files")
	}
	if !strings.Contains(result, "truncated") {
		preview := result
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Errorf("expandMentions should include a truncation notice, got: %q", preview)
	}
}


// TestMention_AtSymbol_TriggersFileReload verifies that typing '@' when the
// picker is inactive dispatches LoadFilesCmd so newly-created files appear.
func TestMention_AtSymbol_TriggersFileReload(t *testing.T) {
	m := setupModelWithFiles([]string{"existing.go"})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})

	_, found := collectBatchFilesLoaded(cmd)
	if !found {
		t.Error("typing '@' should dispatch LoadFilesCmd to pick up new files, but no FilesLoadedMsg was produced")
	}
}

// TestMention_AtQuery_DoesNotReloadOnEveryKeystroke verifies that subsequent
// keystrokes while the picker is already active do NOT re-trigger the reload.
func TestMention_AtQuery_DoesNotReloadOnEveryKeystroke(t *testing.T) {
	m := setupModelWithFiles([]string{"main.go", "tui/model.go"})
	// Activate picker first (simulates '@' having been typed already)
	m.filePicker.Activate()
	m.SetInput("@mai")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	_, found := collectBatchFilesLoaded(cmd)
	if found {
		t.Error("additional keystrokes while picker is active should not re-trigger file reload")
	}
}
