package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
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
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})  // move to No
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

func TestUpdate_ToolApprovalRequestMsg_EditTool_ShowsDiffInDialog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(path, []byte("hello world\n"), 0o644)

	model := NewModel(nil)
	ch := make(chan bool, 1)
	msg := ToolApprovalRequestMsg{
		ModelName: "m",
		ToolName:  "edit",
		Input: map[string]any{
			"path":       path,
			"old_string": "hello",
			"new_string": "goodbye",
		},
		ResponseCh: ch,
	}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	view := stripANSI(m.approvalDialog.View())
	if !strings.Contains(view, "hello") || !strings.Contains(view, "goodbye") {
		t.Errorf("edit approval dialog should show before/after content in view: %q", view)
	}
}

func TestUpdate_ToolApprovalRequestMsg_BashTool_ShowsCommandInDialog(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	msg := ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "bash",
		Input:      map[string]any{"command": "ls -la /tmp"},
		ResponseCh: ch,
	}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	view := stripANSI(m.approvalDialog.View())
	if !strings.Contains(view, "$ ls -la /tmp") {
		t.Errorf("bash approval dialog should show command with $ prefix in view: %q", view)
	}
}

func TestUpdate_ToolApprovalRequestMsg_EditTool_BadFile_StillActivates(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	msg := ToolApprovalRequestMsg{
		ModelName: "m",
		ToolName:  "edit",
		Input: map[string]any{
			"path":       "/nonexistent/file.txt",
			"old_string": "x",
			"new_string": "y",
		},
		ResponseCh: ch,
	}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	if !m.approvalDialog.IsActive() {
		t.Error("dialog should still activate even if diff computation fails")
	}
}

func TestUpdate_ToolCallMsg_EmptyOutput_IsNotAddedToHistory(t *testing.T) {
	model := NewModel(nil)
	before := len(model.GetHistory())
	// Simulate a declined tool: Output is empty (as when ApprovingExecutor returns error)
	updated, _ := model.Update(ToolCallMsg{ToolCall: api.ToolCallResult{Name: "edit", Output: ""}})
	m := updated.(Model)
	if len(m.GetHistory()) > before {
		t.Error("ToolCallMsg with empty output should not add an empty block to message history")
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

func TestView_ToolApprovalDialog_DeniedAfterNo_ShowsCanceled(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", Input: map[string]any{"command": "ls -la"}, ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})  // move to No
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm No
	view := stripANSI(m3.(Model).View())
	if !strings.Contains(view, "Canceled by user") {
		t.Errorf("View after No should contain 'Canceled by user': %q", view)
	}
	if !strings.Contains(view, "bash") {
		t.Errorf("View after No should contain tool name 'bash': %q", view)
	}
}

func TestUpdate_ToolApprovalDoneMsg_DismissesActiveDialog(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	if !m1.(Model).approvalDialog.IsActive() {
		t.Fatal("precondition: dialog should be active after ToolApprovalRequestMsg")
	}
	m2, _ := m1.(Model).Update(toolApprovalDoneMsg{})
	if m2.(Model).approvalDialog.IsActive() {
		t.Error("toolApprovalDoneMsg should deactivate the approval dialog")
	}
}

func TestView_ToolApprovalDialog_ApprovedAfterYes_NoCanceledMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", Input: map[string]any{"command": "ls"}, ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm Yes
	view := stripANSI(m2.(Model).View())
	if strings.Contains(view, "Canceled by user") {
		t.Errorf("View after Yes should not contain 'Canceled by user': %q", view)
	}
}

// --- ToolApprovalDialog: No with cancelFn set ---

func TestUpdate_ToolApprovalDialog_No_CallsCancelFn(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	var called bool
	model.cancelFn = func() { called = true }
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	m2, _ := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyDown}) // move to No
	_, _ = m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM})  // confirm No
	if !called {
		t.Error("confirming No in approval dialog should invoke cancelFn")
	}
}

// --- ToolApprovalDialog: Yes with approvalCh ---

func TestUpdate_ToolApprovalDialog_Yes_WithApprovalCh_ReturnsWaitCmd(t *testing.T) {
	model := NewModel(nil)
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	model.approvalCh = approvalCh
	ch := make(chan bool, 1)
	m1, _ := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	_, cmd := m1.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlM}) // confirm Yes (default)
	if cmd == nil {
		t.Error("confirming Yes with approvalCh set should return a wait cmd")
	}
}

// --- ToolApprovalRequestMsg: ape mode auto-approves ---

func TestUpdate_ToolApprovalRequestMsg_ApeMode_AutoApproves(t *testing.T) {
	model := NewModel(nil)
	model.apeMode = true
	ch := make(chan bool, 1)
	_, _ = model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	select {
	case approved := <-ch:
		if !approved {
			t.Error("ape mode should auto-approve tool requests")
		}
	default:
		t.Error("ape mode should have sent approval on responseCh immediately")
	}
}

// --- ToolApprovalRequestMsg with approvalCh: returns wait cmd ---

func TestUpdate_ToolApprovalRequestMsg_WithApprovalCh_ReturnsWaitCmd(t *testing.T) {
	model := NewModel(nil)
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	model.approvalCh = approvalCh
	ch := make(chan bool, 1)
	_, cmd := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	if cmd == nil {
		t.Error("ToolApprovalRequestMsg with approvalCh set should return a wait cmd")
	}
}

// --- Tool approval previews: read, write, glob, grep ---

func TestUpdate_ToolApprovalRequestMsg_ReadTool_ShowsPath(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "read",
		Input:      map[string]any{"path": "/tmp/file.txt"},
		ResponseCh: ch,
	})
	if !strings.Contains(stripANSI(m.(Model).approvalDialog.View()), "/tmp/file.txt") {
		t.Error("read tool approval preview should contain the file path")
	}
}

func TestUpdate_ToolApprovalRequestMsg_WriteTool_ShowsPreview(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "write",
		Input:      map[string]any{"path": "/tmp/out.txt", "content": "line1\nline2\nline3"},
		ResponseCh: ch,
	})
	if !strings.Contains(stripANSI(m.(Model).approvalDialog.View()), "/tmp/out.txt") {
		t.Error("write tool approval preview should contain the file path")
	}
}

func TestUpdate_ToolApprovalRequestMsg_WriteTool_TruncatesLongContent(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	content := "l1\nl2\nl3\nl4\nl5\nl6\nl7"
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "write",
		Input:      map[string]any{"path": "/tmp/out.txt", "content": content},
		ResponseCh: ch,
	})
	if !strings.Contains(stripANSI(m.(Model).approvalDialog.View()), "...") {
		t.Error("write tool approval with >5 lines should show truncation marker '...'")
	}
}

func TestUpdate_ToolApprovalRequestMsg_GlobTool_ShowsPatternAndPath(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "glob",
		Input:      map[string]any{"pattern": "**/*.go", "path": "/src"},
		ResponseCh: ch,
	})
	view := stripANSI(m.(Model).approvalDialog.View())
	if !strings.Contains(view, "**/*.go") || !strings.Contains(view, "/src") {
		t.Errorf("glob approval preview should contain pattern and path: %q", view)
	}
}

func TestUpdate_ToolApprovalRequestMsg_GlobTool_DefaultsPathToDot(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "glob",
		Input:      map[string]any{"pattern": "*.go"},
		ResponseCh: ch,
	})
	view := stripANSI(m.(Model).approvalDialog.View())
	if !strings.Contains(view, " in .") {
		t.Errorf("glob approval preview with no path should default to '.': %q", view)
	}
}

func TestUpdate_ToolApprovalRequestMsg_GrepTool_ShowsPatternAndPath(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "grep",
		Input:      map[string]any{"pattern": "func main", "path": "/src"},
		ResponseCh: ch,
	})
	view := stripANSI(m.(Model).approvalDialog.View())
	if !strings.Contains(view, "func main") || !strings.Contains(view, "/src") {
		t.Errorf("grep approval preview should contain pattern and path: %q", view)
	}
}

func TestUpdate_ToolApprovalRequestMsg_GrepTool_DefaultsPathToDot(t *testing.T) {
	model := NewModel(nil)
	ch := make(chan bool, 1)
	m, _ := model.Update(ToolApprovalRequestMsg{
		ModelName:  "m",
		ToolName:   "grep",
		Input:      map[string]any{"pattern": "TODO"},
		ResponseCh: ch,
	})
	view := stripANSI(m.(Model).approvalDialog.View())
	if !strings.Contains(view, " in .") {
		t.Errorf("grep approval preview with no path should default to '.': %q", view)
	}
}

// --- ToolApprovalRequestMsg: ape mode with approvalCh ---

func TestUpdate_ToolApprovalRequestMsg_ApeMode_WithApprovalCh_ReturnsWaitCmd(t *testing.T) {
	model := NewModel(nil)
	model.apeMode = true
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	model.approvalCh = approvalCh
	ch := make(chan bool, 1)
	_, cmd := model.Update(ToolApprovalRequestMsg{ModelName: "m", ToolName: "bash", ResponseCh: ch})
	if cmd == nil {
		t.Error("ape mode with approvalCh set should return a wait cmd")
	}
}
