package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ToolApprovalDialog is a Yes/No prompt shown when the model wants to run a tool.
type ToolApprovalDialog struct {
	modelName   string
	toolName    string
	previewText string
	cursor      int // 0 = Yes, 1 = No
	active      bool
	denied      bool // true after user explicitly selects "No"
	width       int
	responseCh  chan<- bool
}

// NewToolApprovalDialog returns an inactive approval dialog.
func NewToolApprovalDialog(width int) ToolApprovalDialog {
	return ToolApprovalDialog{width: width}
}

// Activate shows the dialog for the given model/tool pair with a response channel.
// previewText is optional additional content shown above the Yes/No prompt (e.g. a diff).
// The cursor is reset to Yes on each activation.
func (d *ToolApprovalDialog) Activate(modelName, toolName, previewText string, responseCh chan<- bool) {
	d.modelName = modelName
	d.toolName = toolName
	d.previewText = previewText
	d.cursor = 0
	d.responseCh = responseCh
	d.active = true
	d.denied = false
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

// IsDenied reports whether the user explicitly selected "No" to cancel the tool.
func (d ToolApprovalDialog) IsDenied() bool { return d.denied }

// Confirm sends the user's decision on responseCh and deactivates the dialog.
// If the cursor is on "No", sets the denied state so DeniedView can render.
func (d *ToolApprovalDialog) Confirm() {
	approved := d.cursor == 0
	if !approved {
		d.denied = true
	}
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

// DeniedView renders a grayed-out version of the dialog after the user cancels a tool.
// Returns "" when not in the denied state.
func (d ToolApprovalDialog) DeniedView() string {
	if !d.denied {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(ToolApprovalDeniedToolStyle().Render(d.toolName) +
		" " + ToolApprovalCanceledLabelStyle().Render("Canceled by user"))
	if d.previewText != "" {
		sb.WriteString("\n\n")
		sb.WriteString(ToolApprovalDeniedPreviewStyle().Render("$ " + d.previewText))
	}
	return FilePickerStyle(d.width).Render(sb.String())
}

// View renders the approval dialog. Returns "" when inactive.
func (d ToolApprovalDialog) View() string {
	if !d.active {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(ToolApprovalModelStyle().Render(d.modelName) +
		" wants to run " +
		ToolApprovalToolStyle().Render(d.toolName) +
		" tool")
	if d.previewText != "" {
		sb.WriteString("\n\n")
		if d.toolName == "edit" {
			sb.WriteString(RenderSplitDiff(d.previewText, d.width))
		} else {
			sb.WriteString(ToolApprovalPreviewStyle().Render("$ " + d.previewText))
		}
	}
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
