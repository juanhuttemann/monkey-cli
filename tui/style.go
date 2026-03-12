package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// UserMessageStyle returns the styling for user messages
func UserMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorUserBorder)).
		Width(width-4).
		Padding(0, 1)
}

// AssistantMessageStyle returns the styling for assistant messages
func AssistantMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorAssistantBorder)).
		Width(width-4).
		Padding(0, 1)
}

// ErrorMessageStyle returns the styling for error messages
func ErrorMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorErrorBorder)).
		Width(width-4).
		Padding(0, 1)
}

// ToolMessageStyle returns the styling for tool call messages (cyan border)
func ToolMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorToolBorder)).
		Width(width-4).
		Padding(0, 1)
}

// SystemMessageStyle returns the styling for system notices (e.g. retry messages)
func SystemMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorSystemBorder)).
		Width(width-4).
		Padding(0, 1)
}

// InputStyle returns the styling for the input textarea.
// Uses ColorAccent so the active input zone is clearly visible all session long.
func InputStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorAccent)).
		Width(width - 4).
		Height(height)
}

// SpinnerStyle returns the styling for the loading spinner
func SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorSystemBorder))
}

// TimerStyle returns the styling for the elapsed time indicator
func TimerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGrayLight))
}

// WaitingStyle returns the styling for the idle/waiting prompt shown after cancellation.
func WaitingStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorAccent)).
		Italic(true)
}

// FilePickerStyle returns the styling for the file picker dropdown border.
func FilePickerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorGrayDark)).
		Width(width-4).
		Padding(0, 1)
}

// FilePickerCursorStyle returns the styling for the highlighted file picker row.
// Uses ColorPickerCursor (accent) so "selected" reads as a warm highlight,
// distinct from ColorAssistantBorder which means "assistant message".
func FilePickerCursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPickerCursor)).
		Bold(true)
}

// ApeModeActiveStyle returns accent-color styling for the ape mode indicator when enabled.
func ApeModeActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAccent))
}

// ApeModeInactiveStyle returns grey styling for the ape mode indicator when disabled.
func ApeModeInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayDeep))
}

// StatusBarModelStyle renders the current model name in the status bar.
func StatusBarModelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayLight))
}

// StatusBarSepStyle renders the "|" separator between status bar segments.
func StatusBarSepStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayDeep))
}

// StatusBarTokenStyle renders the token count in the status bar.
func StatusBarTokenStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayLight))
}

// ToolApprovalModelStyle returns styling for the model name in the approval dialog.
// Uses ColorApprovalModel (green) — the assistant is the one requesting action.
func ToolApprovalModelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorApprovalModel)).
		Bold(true)
}

// ToolApprovalPreviewStyle returns styling for the bash command preview line.
func ToolApprovalPreviewStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGrayMid)).
		Bold(true).
		Padding(0, 1)
}

// ToolApprovalToolStyle returns styling for the tool name in the approval dialog.
// Uses ColorApprovalTool (cyan) — consistent with tool call borders.
func ToolApprovalToolStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorApprovalTool)).
		Bold(true)
}

// ToolApprovalCanceledLabelStyle returns muted gray italic styling for the "Canceled by user" label.
func ToolApprovalCanceledLabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGrayMid)).
		Italic(true)
}

// ToolApprovalDeniedToolStyle returns muted styling for the tool name in a canceled dialog.
func ToolApprovalDeniedToolStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGrayMid)).
		Bold(true)
}

// ToolApprovalDeniedPreviewStyle returns faint gray styling for the preview in a canceled dialog.
func ToolApprovalDeniedPreviewStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGrayDeep)).
		Faint(true)
}

// MessageTimestampStyle returns the styling for per-message timestamps: gray, right-aligned.
func MessageTimestampStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width - 2).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color(ColorGrayDeep))
}

// SearchMatchStyle returns the styling for the search-match indicator shown above matching messages.
func SearchMatchStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorAccent)).
		Bold(true)
}
