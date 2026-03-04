package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ToolApprovalDialog is a Yes/No prompt shown when the model wants to run a tool.
type ToolApprovalDialog struct {
	modelName  string
	toolName   string
	cursor     int // 0 = Yes, 1 = No
	active     bool
	width      int
	responseCh chan<- bool
}

// NewToolApprovalDialog returns an inactive approval dialog.
func NewToolApprovalDialog(width int) ToolApprovalDialog {
	return ToolApprovalDialog{width: width}
}

// Activate shows the dialog for the given model/tool pair with a response channel.
// The cursor is reset to Yes on each activation.
func (d *ToolApprovalDialog) Activate(modelName, toolName string, responseCh chan<- bool) {
	d.modelName = modelName
	d.toolName = toolName
	d.cursor = 0
	d.responseCh = responseCh
	d.active = true
}

// Deactivate hides the dialog and clears the response channel.
func (d *ToolApprovalDialog) Deactivate() {
	d.active = false
	d.responseCh = nil
}

// IsActive reports whether the dialog is visible.
func (d ToolApprovalDialog) IsActive() bool { return d.active }

// SetWidth updates the display width.
func (d *ToolApprovalDialog) SetWidth(w int) { d.width = w }

// IsApproved reports whether the cursor is on "Yes".
func (d ToolApprovalDialog) IsApproved() bool { return d.cursor == 0 }

// Confirm sends the user's decision on responseCh and deactivates the dialog.
func (d *ToolApprovalDialog) Confirm() {
	approved := d.cursor == 0
	if d.responseCh != nil {
		d.responseCh <- approved
	}
	d.Deactivate()
}

// Deny sends false on responseCh (non-blocking) and deactivates the dialog.
// Used for cancellation paths where the executor may have already timed out.
func (d *ToolApprovalDialog) Deny() {
	if d.responseCh != nil {
		select {
		case d.responseCh <- false:
		default:
		}
	}
	d.Deactivate()
}

// Update handles Up/Down cursor navigation.
func (d ToolApprovalDialog) Update(msg tea.Msg) (ToolApprovalDialog, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, nil
	}
	switch key.Type {
	case tea.KeyDown:
		if d.cursor < 1 {
			d.cursor++
		}
	case tea.KeyUp:
		if d.cursor > 0 {
			d.cursor--
		}
	}
	return d, nil
}

// View renders the approval dialog. Returns "" when inactive.
func (d ToolApprovalDialog) View() string {
	if !d.active {
		return ""
	}
	prompt := d.modelName + " wants to run " + d.toolName
	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n")
	options := []string{"Yes", "No"}
	for i, opt := range options {
		if i == d.cursor {
			sb.WriteString(FilePickerCursorStyle().Render("> " + opt))
		} else {
			sb.WriteString("  " + opt)
		}
		if i < len(options)-1 {
			sb.WriteByte('\n')
		}
	}
	return FilePickerStyle(d.width).Render(sb.String())
}
