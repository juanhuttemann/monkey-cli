package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestViewportHeight_BaseAtRest checks the baseline height when idle.
func TestViewportHeight_BaseAtRest(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	want := 24 - 9
	if got := m.GetViewportHeight(); got != want {
		t.Errorf("at rest: viewport height = %d, want %d", got, want)
	}
}

// TestViewportHeight_LoadingState checks that the viewport shrinks by 1
// when the status (spinner) line appears, keeping the input at a fixed position.
func TestViewportHeight_LoadingState(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m.SetLoading(true)
	loadingHeight := m.GetViewportHeight()

	if loadingHeight != restHeight-1 {
		t.Errorf("loading: viewport height = %d, want %d (rest-1)", loadingHeight, restHeight-1)
	}
}

// TestViewportHeight_WasCancelled checks that viewport shrinks when the
// "What should monkey do?" line is shown.
func TestViewportHeight_WasCancelled(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m.SetLoading(true)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cancelledHeight := m2.(Model).GetViewportHeight()

	if cancelledHeight != restHeight-1 {
		t.Errorf("cancelled: viewport height = %d, want %d (rest-1)", cancelledHeight, restHeight-1)
	}
}

// TestViewportHeight_RestoresAfterResponse checks that viewport grows back
// to base height after a response arrives (lastElapsed shown for 1 tick then clears,
// but at minimum it should NOT still be loading-size).
func TestViewportHeight_LoadingThenResponse(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	m.SetLoading(true)
	loadingHeight := m.GetViewportHeight()

	// Response arrives → state = Ready, lastElapsed set
	m2, _ := m.Update(PromptResponseMsg{Response: "hi"})
	responseHeight := m2.(Model).GetViewportHeight()

	// lastElapsed > 0 still shows a status line, so height stays restHeight-1
	if responseHeight != loadingHeight {
		t.Errorf("after response (elapsed shown): viewport height = %d, want %d (same as loading)", responseHeight, loadingHeight)
	}
}

// TestViewportHeight_ApprovalDialogShrinksViewport checks that when an approval
// dialog is shown, the viewport shrinks by the dialog's height.
func TestViewportHeight_ApprovalDialogShrinksViewport(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	ch := make(chan bool, 1)
	m2, _ := m.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", Input: map[string]any{"command": "ls"}, ResponseCh: ch})
	dialogHeight := m2.(Model).GetViewportHeight()

	if dialogHeight >= restHeight {
		t.Errorf("with dialog: viewport height = %d, want < %d (rest)", dialogHeight, restHeight)
	}
}

// TestViewportHeight_DeniedDialogShrinksViewport checks that when the denied
// dialog is shown, the viewport shrinks beyond just the status-line row —
// i.e. the denied dialog's own height is also reserved.
func TestViewportHeight_DeniedDialogShrinksViewport(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	ch := make(chan bool, 1)
	m1, _ := m.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", Input: map[string]any{"command": "ls"}, ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})   // move to No
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm No → denied view

	// After denial: wasCancelled=true adds 1 row (status line), plus the
	// denied dialog view itself adds more rows. So the viewport must be
	// strictly less than restHeight-1 (not just less than restHeight).
	deniedHeight := m3.(Model).GetViewportHeight()
	if deniedHeight >= restHeight-1 {
		t.Errorf("denied dialog: viewport height = %d, want < %d (rest-1), denied view not being reserved", deniedHeight, restHeight-1)
	}
}

// TestViewportHeight_CommandPickerShrinksViewport checks that when the command
// picker (slash command list) is visible, the viewport shrinks accordingly.
func TestViewportHeight_CommandPickerShrinksViewport(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	// Activate command picker directly
	m.commandPicker.Activate()
	m.syncViewportHeight()

	if m.GetViewportHeight() >= restHeight {
		t.Errorf("command picker active: viewport height = %d, want < %d", m.GetViewportHeight(), restHeight)
	}
}

// TestViewportHeight_CommandPickerGrowsBackAfterDeactivate checks that the
// viewport restores when the picker is dismissed.
func TestViewportHeight_CommandPickerGrowsBackAfterDeactivate(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m.commandPicker.Activate()
	m.syncViewportHeight()
	withPicker := m.GetViewportHeight()

	m.commandPicker.Deactivate()
	m.syncViewportHeight()
	after := m.GetViewportHeight()

	if after <= withPicker {
		t.Errorf("after deactivate: viewport height = %d, want > %d", after, withPicker)
	}
	if after != restHeight {
		t.Errorf("after deactivate: viewport height = %d, want %d (rest)", after, restHeight)
	}
}

// TestViewportHeight_TypingSlashActivatesPicker checks that typing "/" triggers
// the command picker and the viewport shrinks to accommodate it.
func TestViewportHeight_TypingSlashActivatesPicker(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	withPicker := m1.(Model).GetViewportHeight()

	if withPicker >= restHeight {
		t.Errorf("typing /: viewport height = %d, want < %d (command picker not reserved)", withPicker, restHeight)
	}
}

// TestViewportHeight_TypingAtActivatesFilePicker checks that typing "@" triggers
// the file picker and the viewport shrinks to accommodate it.
func TestViewportHeight_TypingAtActivatesFilePicker(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	withPicker := m1.(Model).GetViewportHeight()

	if withPicker >= restHeight {
		t.Errorf("typing @: viewport height = %d, want < %d (file picker not reserved)", withPicker, restHeight)
	}
}

// TestViewportHeight_DeletingSlashDeactivatesPicker checks that deleting the "/"
// dismisses the command picker and the viewport grows back to rest height.
func TestViewportHeight_DeletingSlashDeactivatesPicker(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	restHeight := m.GetViewportHeight()

	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyBackspace})
	afterDelete := m2.(Model).GetViewportHeight()

	if afterDelete != restHeight {
		t.Errorf("after deleting /: viewport height = %d, want %d (rest)", afterDelete, restHeight)
	}
}

// TestViewportHeight_DialogGrowsBackAfterApproval checks that after approving
// a tool, the viewport returns to near the rest height (minus status row).
func TestViewportHeight_DialogGrowsBackAfterApproval(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)

	ch := make(chan bool, 1)
	m1, _ := m.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", Input: map[string]any{"command": "ls"}, ResponseCh: ch})
	withDialog := m1.(Model).GetViewportHeight()

	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm Yes
	afterApproval := m2.(Model).GetViewportHeight()

	if afterApproval <= withDialog {
		t.Errorf("after approval: viewport height = %d, want > %d (with dialog)", afterApproval, withDialog)
	}
}
