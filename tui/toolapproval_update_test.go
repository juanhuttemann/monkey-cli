package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdate_ToolApprovalRequestMsg_ActivatesDialog(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	msg := ToolApprovalRequestMsg{ModelName: "claude-sonnet-4", ToolName: "bash", ResponseCh: ch}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	if !m.approvalDialog.IsActive() {
		t.Error("ToolApprovalRequestMsg should activate the approval dialog")
	}
}

func TestUpdate_ToolApprovalRequestMsg_StoresModelAndToolName(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	updatedModel, _ := model.Update(ToolApprovalRequestMsg{ModelName: "claude-opus", ToolName: "bash", ResponseCh: ch})
	m := updatedModel.(Model)
	view := stripANSI(m.approvalDialog.View())
	if !strings.Contains(view, "claude-opus") {
		t.Errorf("dialog view should contain model name: %q", view)
	}
	if !strings.Contains(view, "bash") {
		t.Errorf("dialog view should contain tool name: %q", view)
	}
}

func TestUpdate_ToolApprovalDialog_Enter_ApprovesWhenYes(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := m2.(Model)
	if m.approvalDialog.IsActive() {
		t.Error("Dialog should be deactivated after Enter")
	}
	select {
	case approved := <-ch:
		if !approved {
			t.Error("Enter with cursor on Yes should send true (approved)")
		}
	default:
		t.Error("Enter should have sent on responseCh")
	}
}

func TestUpdate_ToolApprovalDialog_Down_Then_Enter_Denies(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := m3.(Model)
	if m.approvalDialog.IsActive() {
		t.Error("Dialog should be deactivated after Enter")
	}
	select {
	case approved := <-ch:
		if approved {
			t.Error("Enter with cursor on No should send false (denied)")
		}
	default:
		t.Error("Enter should have sent on responseCh")
	}
}

func TestUpdate_ToolApprovalDialog_No_CancelsRequest(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})   // move to No
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm No
	m := m3.(Model)
	if m.IsLoading() {
		t.Error("Selecting No should stop loading (cancel the request)")
	}
}

func TestUpdate_ToolApprovalDialog_Up_RestoresToYes(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown}) // move to No
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyUp})   // move back to Yes
	if !m3.(Model).approvalDialog.IsApproved() {
		t.Error("After Down then Up, cursor should be back on Yes")
	}
}

func TestUpdate_ToolApprovalDialog_Esc_WhileLoading_Denies(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := m2.(Model)
	if m.approvalDialog.IsActive() {
		t.Error("Esc while loading should deactivate the approval dialog")
	}
	select {
	case approved := <-ch:
		if approved {
			t.Error("Esc should send false (deny) on responseCh")
		}
	default:
		t.Error("Esc should have sent on responseCh")
	}
}

func TestUpdate_ToolApprovalDialog_CtrlC_WhileLoading_Denies(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m := m2.(Model)
	if m.approvalDialog.IsActive() {
		t.Error("Ctrl+C while loading should deactivate the approval dialog")
	}
	select {
	case approved := <-ch:
		if approved {
			t.Error("Ctrl+C should send false (deny) on responseCh")
		}
	default:
		t.Error("Ctrl+C should have sent on responseCh")
	}
}

func TestView_ToolApprovalDialog_Active_ShowsInView(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	ch := make(chan bool, 1)
	updatedModel, _ := model.Update(ToolApprovalRequestMsg{ModelName: "claude-sonnet", ToolName: "bash", ResponseCh: ch})
	m := updatedModel.(Model)
	view := m.View()
	if !containsSubstring(view, "wants to run") {
		t.Errorf("View with active dialog should contain 'wants to run': %q", stripANSI(view))
	}
	if !containsSubstring(view, "bash") {
		t.Errorf("View with active dialog should contain tool name 'bash': %q", stripANSI(view))
	}
	if !containsSubstring(view, "Yes") {
		t.Errorf("View with active dialog should contain 'Yes': %q", stripANSI(view))
	}
	if !containsSubstring(view, "No") {
		t.Errorf("View with active dialog should contain 'No': %q", stripANSI(view))
	}
}

func TestView_ToolApprovalDialog_Inactive_NotInView(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	view := model.View()
	if containsSubstring(view, "wants to run") {
		t.Error("View without active dialog should not contain 'wants to run'")
	}
}
