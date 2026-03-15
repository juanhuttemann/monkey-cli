package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestPickerHeight_InactiveReturnsZero verifies all pickers/dialogs return 0
// when inactive, so syncViewportHeight never reserves space for hidden widgets.
func TestPickerHeight_InactiveReturnsZero(t *testing.T) {
	if h := NewCommandPicker(80).Height(); h != 0 {
		t.Errorf("inactive CommandPicker.Height() = %d, want 0", h)
	}
	if h := NewFilePicker(80).Height(); h != 0 {
		t.Errorf("inactive FilePicker.Height() = %d, want 0", h)
	}
	if h := NewModelPicker(80).Height(); h != 0 {
		t.Errorf("inactive ModelPicker.Height() = %d, want 0", h)
	}
	if h := NewToolApprovalDialog(80).Height(); h != 0 {
		t.Errorf("inactive ToolApprovalDialog.Height() = %d, want 0", h)
	}
	if h := NewToolApprovalDialog(80).DeniedHeight(); h != 0 {
		t.Errorf("non-denied ToolApprovalDialog.DeniedHeight() = %d, want 0", h)
	}
}

// TestCommandPickerHeight_MatchesView verifies that CommandPicker.Height()
// returns the same value as lipgloss.Height(cp.View()) for all relevant states.
func TestCommandPickerHeight_MatchesView(t *testing.T) {
	cases := []string{
		"",    // all commands visible
		"e",   // filtered subset
		"zzz", // no match → empty list
	}
	for _, query := range cases {
		cp := NewCommandPicker(80)
		cp.Activate()
		cp.SetQuery(query)

		got := cp.Height()
		want := lipgloss.Height(cp.View())
		if got != want {
			t.Errorf("CommandPicker query=%q: Height()=%d, lipgloss.Height(View())=%d", query, got, want)
		}
	}
}

// TestFilePickerHeight_MatchesView verifies that FilePicker.Height()
// matches lipgloss.Height(fp.View()) for empty, small, and capped lists.
func TestFilePickerHeight_MatchesView(t *testing.T) {
	cases := []struct {
		name  string
		files []string
	}{
		{"empty", nil},
		{"one", []string{"a.go"}},
		{"few", []string{"a.go", "b.go", "c.go"}},
		{"at-max", makeFiles(filePickerMaxVisible)},
		{"over-max", makeFiles(filePickerMaxVisible + 3)},
	}
	for _, tc := range cases {
		fp := NewFilePicker(80)
		fp.Activate()
		fp.SetFiles(tc.files)

		got := fp.Height()
		want := lipgloss.Height(fp.View())
		if got != want {
			t.Errorf("FilePicker %s: Height()=%d, lipgloss.Height(View())=%d", tc.name, got, want)
		}
	}
}

// TestModelPickerHeight_MatchesView verifies that ModelPicker.Height()
// matches lipgloss.Height(mp.View()) for empty and non-empty lists.
func TestModelPickerHeight_MatchesView(t *testing.T) {
	cases := []struct {
		name   string
		models []string
	}{
		{"empty", nil},
		{"one", []string{"gpt-4o"}},
		{"three", []string{"gpt-4o", "claude-3-5-sonnet", "gemini-pro"}},
	}
	for _, tc := range cases {
		mp := NewModelPicker(80)
		mp.Activate()
		mp.SetModels(tc.models)

		got := mp.Height()
		want := lipgloss.Height(mp.View())
		if got != want {
			t.Errorf("ModelPicker %s: Height()=%d, lipgloss.Height(View())=%d", tc.name, got, want)
		}
	}
}

// TestToolApprovalDialogHeight_MatchesView verifies that Height() matches
// lipgloss.Height(View()) for dialogs with and without preview text.
func TestToolApprovalDialogHeight_MatchesView(t *testing.T) {
	cases := []struct {
		name    string
		preview string
	}{
		{"no preview", ""},
		{"short preview", "ls -la"},
		{"multiline preview", "line1\nline2\nline3"},
	}
	for _, tc := range cases {
		d := NewToolApprovalDialog(80)
		ch := make(chan bool, 1)
		d.Activate("claude", "bash", tc.preview, ch)

		got := d.Height()
		want := lipgloss.Height(d.View())
		if got != want {
			t.Errorf("ToolApprovalDialog %s: Height()=%d, lipgloss.Height(View())=%d", tc.name, got, want)
		}
	}
}

// TestToolApprovalDialogDeniedHeight_MatchesView verifies that DeniedHeight()
// matches lipgloss.Height(DeniedView()) after the user denies the tool.
func TestToolApprovalDialogDeniedHeight_MatchesView(t *testing.T) {
	cases := []struct {
		name    string
		preview string
	}{
		{"no preview", ""},
		{"short preview", "ls -la"},
	}
	for _, tc := range cases {
		d := NewToolApprovalDialog(80)
		ch := make(chan bool, 1)
		d.Activate("claude", "bash", tc.preview, ch)
		d.cursor = 1 // select No
		d.Confirm()  // → denied state

		got := d.DeniedHeight()
		want := lipgloss.Height(d.DeniedView())
		if got != want {
			t.Errorf("ToolApprovalDialog denied %s: DeniedHeight()=%d, lipgloss.Height(DeniedView())=%d", tc.name, got, want)
		}
	}
}

// TestSyncViewportHeight_UsesPickerHeight verifies that syncViewportHeight
// produces the same viewport height whether using Height() or lipgloss.Height(View()).
// This is the integration test ensuring the refactoring is a no-op.
func TestSyncViewportHeight_UsesPickerHeight(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)

	// Activate command picker with all commands visible.
	m.commandPicker.Activate()
	m.commandPicker.SetQuery("")
	m.syncViewportHeight()
	withPicker := m.GetViewportHeight()

	// The picker height should equal lipgloss.Height(View()).
	pickerH := lipgloss.Height(m.commandPicker.View())
	// viewport = 40 - reservedBaseRows - pickerH
	wantVP := 40 - reservedBaseRows - pickerH
	if withPicker != wantVP {
		t.Errorf("syncViewportHeight with command picker: got %d, want %d (40-%d-%d)", withPicker, wantVP, reservedBaseRows, pickerH)
	}
}

// makeFiles returns a slice of n unique fake file paths.
func makeFiles(n int) []string {
	files := make([]string, n)
	for i := range files {
		files[i] = "file" + string(rune('a'+i%26)) + ".go"
	}
	return files
}
