package tui

import "github.com/charmbracelet/lipgloss"

// Style constants
const (
	UserBorderColor      = "#4B2DA8" // Dark purple
	AssistantBorderColor = "#04B575" // Green
	ErrorBorderColor     = "#FF6B6B" // Red
	ToolBorderColor      = "#00AACC" // Cyan
)

// UserMessageStyle returns the styling for user messages
func UserMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(UserBorderColor)).
		Width(width - 4).
		Padding(0, 1)
}

// AssistantMessageStyle returns the styling for assistant messages
func AssistantMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(AssistantBorderColor)).
		Width(width - 4).
		Padding(0, 1)
}

// ErrorMessageStyle returns the styling for error messages
func ErrorMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ErrorBorderColor)).
		Width(width - 4).
		Padding(0, 1)
}

// ToolMessageStyle returns the styling for tool call messages (cyan border)
func ToolMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ToolBorderColor)).
		Width(width - 4).
		Padding(0, 1)
}

// InputStyle returns the styling for the input textarea
func InputStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#888888")).
		Width(width - 4).
		Height(height)
}

// SpinnerStyle returns the styling for the loading spinner
func SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500"))
}

// TimerStyle returns the styling for the elapsed time indicator
func TimerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))
}

// MessageTimestampStyle returns the styling for per-message timestamps: gray, right-aligned.
func MessageTimestampStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width - 2).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("#555555"))
}
