package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestToolApprovalDialog_InactiveByDefault(t *testing.T) {
	d := NewToolApprovalDialog(80)
	if d.IsActive() {
		t.Error("NewToolApprovalDialog should be inactive by default")
	}
}

func TestToolApprovalDialog_ActivateDeactivate(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("claude-sonnet", "bash", "", ch)
	if !d.IsActive() {
		t.Error("After Activate, IsActive = false, want true")
	}
	d.Deactivate()
	if d.IsActive() {
		t.Error("After Deactivate, IsActive = true, want false")
	}
}

func TestToolApprovalDialog_DefaultCursorIsYes(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	if !d.IsApproved() {
		t.Error("Default cursor should be on Yes (IsApproved = true)")
	}
}

func TestToolApprovalDialog_NavigateDown_MovesToNo(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	if d.IsApproved() {
		t.Error("After Down, cursor should be on No (IsApproved = false)")
	}
}

func TestToolApprovalDialog_NavigateUp_AtYes_IsNoop(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyUp})
	if !d.IsApproved() {
		t.Error("Up at Yes should be noop (IsApproved stays true)")
	}
}

func TestToolApprovalDialog_NavigateDown_AtNo_IsNoop(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	if d.IsApproved() {
		t.Error("Down at No should be noop (cursor stays at No)")
	}
}

func TestToolApprovalDialog_Confirm_SendsTrueWhenYes(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d.Confirm()
	select {
	case approved := <-ch:
		if !approved {
			t.Error("Confirm with cursor on Yes should send true")
		}
	default:
		t.Error("Confirm should have sent on responseCh")
	}
}

func TestToolApprovalDialog_Confirm_SendsFalseWhenNo(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d.Confirm()
	select {
	case approved := <-ch:
		if approved {
			t.Error("Confirm with cursor on No should send false")
		}
	default:
		t.Error("Confirm should have sent on responseCh")
	}
}

func TestToolApprovalDialog_Confirm_DeactivatesDialog(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d.Confirm()
	if d.IsActive() {
		t.Error("After Confirm, dialog should be deactivated")
	}
}

func TestToolApprovalDialog_Deny_SendsFalse(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d.Deny()
	select {
	case approved := <-ch:
		if approved {
			t.Error("Deny should send false on responseCh")
		}
	default:
		t.Error("Deny should have sent on responseCh")
	}
}

func TestToolApprovalDialog_Deny_Deactivates(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d.Deny()
	if d.IsActive() {
		t.Error("After Deny, dialog should be deactivated")
	}
}

func TestToolApprovalDialog_View_InactiveReturnsEmpty(t *testing.T) {
	d := NewToolApprovalDialog(80)
	if got := d.View(); got != "" {
		t.Errorf("View when inactive = %q, want ''", got)
	}
}

func TestToolApprovalDialog_View_ContainsPrompt(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("claude-sonnet-4", "bash", "", ch)
	view := d.View()
	if !containsSubstring(view, "claude-sonnet-4") {
		t.Errorf("View should contain model name 'claude-sonnet-4': %q", stripANSI(view))
	}
	if !containsSubstring(view, "bash") {
		t.Errorf("View should contain tool name 'bash': %q", stripANSI(view))
	}
	if !containsSubstring(view, "wants to run") {
		t.Errorf("View should contain 'wants to run': %q", stripANSI(view))
	}
}

func TestToolApprovalDialog_View_ContainsYesNo(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	view := d.View()
	if !containsSubstring(view, "Yes") {
		t.Errorf("View should contain 'Yes': %q", stripANSI(view))
	}
	if !containsSubstring(view, "No") {
		t.Errorf("View should contain 'No': %q", stripANSI(view))
	}
}

func TestToolApprovalDialog_View_CursorOnYes_Default(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	view := stripANSI(d.View())
	if !strings.Contains(view, "> Yes") {
		t.Errorf("Default view should show '> Yes': %q", view)
	}
	if !strings.Contains(view, "  No") {
		t.Errorf("Default view should show '  No': %q", view)
	}
}

func TestToolApprovalDialog_View_WithPreview_ShowsSplitPanels(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "edit", "@@ -1,1 +1,1 @@\n-old\n+new\n", ch)
	view := d.View()
	stripped := stripANSI(view)
	if !strings.Contains(stripped, "old") {
		t.Errorf("View with preview should contain deleted content: %q", stripped)
	}
	if !strings.Contains(stripped, "new") {
		t.Errorf("View with preview should contain added content: %q", stripped)
	}
	// ColorDiffDel = ColorErrorBorder = #BA3F28 → emits 186;63;40
	if !strings.Contains(view, "186;63;40") {
		t.Errorf("View with preview should have burnt-orange border for left (before) panel")
	}
	// ColorDiffAdd = ColorAssistantBorder = #729B2F → emits 113;155;47
	if !strings.Contains(view, "113;155;47") {
		t.Errorf("View with preview should have leaf-green border for right (after) panel")
	}
}

func TestToolApprovalDialog_View_EmptyPreview_NoExtraContent(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	view := stripANSI(d.View())
	if !strings.Contains(view, "wants to run") {
		t.Errorf("View should still show prompt: %q", view)
	}
}

func TestToolApprovalDialog_View_CursorOnNo_AfterDown(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	view := stripANSI(d.View())
	if !strings.Contains(view, "  Yes") {
		t.Errorf("After Down, view should show '  Yes': %q", view)
	}
	if !strings.Contains(view, "> No") {
		t.Errorf("After Down, view should show '> No': %q", view)
	}
}

func TestToolApprovalDialog_IsDenied_FalseByDefault(t *testing.T) {
	d := NewToolApprovalDialog(80)
	if d.IsDenied() {
		t.Error("IsDenied should be false by default")
	}
}

func TestToolApprovalDialog_Confirm_No_SetsDenied(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "ls -la", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown}) // move to No
	d.Confirm()
	if !d.IsDenied() {
		t.Error("After Confirm with cursor on No, IsDenied should be true")
	}
}

func TestToolApprovalDialog_Confirm_Yes_NotDenied(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d.Confirm() // cursor on Yes
	if d.IsDenied() {
		t.Error("After Confirm with cursor on Yes, IsDenied should be false")
	}
}

func TestToolApprovalDialog_Activate_ClearsDenied(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d.Confirm() // denied
	ch2 := make(chan bool, 1)
	d.Activate("model", "bash", "", ch2)
	if d.IsDenied() {
		t.Error("Re-activating dialog should clear denied state")
	}
}

func TestToolApprovalDialog_DeniedView_WhenNotDenied_ReturnsEmpty(t *testing.T) {
	d := NewToolApprovalDialog(80)
	if got := d.DeniedView(); got != "" {
		t.Errorf("DeniedView when not denied = %q, want ''", got)
	}
}

func TestToolApprovalDialog_DeniedView_ContainsToolNameAndCanceled(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d.Confirm()
	view := stripANSI(d.DeniedView())
	if !strings.Contains(view, "bash") {
		t.Errorf("DeniedView should contain tool name 'bash': %q", view)
	}
	if !strings.Contains(view, "Canceled by user") {
		t.Errorf("DeniedView should contain 'Canceled by user': %q", view)
	}
}

func TestToolApprovalDialog_DeniedView_ContainsPreviewText(t *testing.T) {
	d := NewToolApprovalDialog(80)
	ch := make(chan bool, 1)
	d.Activate("model", "bash", "ls -la /tmp", ch)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d.Confirm()
	view := stripANSI(d.DeniedView())
	if !strings.Contains(view, "ls -la /tmp") {
		t.Errorf("DeniedView should contain preview text: %q", view)
	}
}
