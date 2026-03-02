package tui

import "github.com/charmbracelet/lipgloss"

// Style constants
const (
	UserBorderColor      = "#04B575" // Green
	AssistantBorderColor = "#7D56F4" // Purple
	ErrorBorderColor     = "#FF6B6B" // Red
	UserBackground       = "#1A1A1A" // Darker
	AssistantBackground  = "#0D0D0D" // Default
	ErrorBackground      = "#3D1A1A" // Dark red
)

// UserMessageStyle returns the styling for user messages
func UserMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(UserBorderColor)).
		Background(lipgloss.Color(UserBackground)).
		Width(width - 4).
		Padding(0, 1)
}

// AssistantMessageStyle returns the styling for assistant messages
func AssistantMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(AssistantBorderColor)).
		Background(lipgloss.Color(AssistantBackground)).
		Width(width - 4).
		Padding(0, 1)
}

// ErrorMessageStyle returns the styling for error messages
func ErrorMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ErrorBorderColor)).
		Background(lipgloss.Color(ErrorBackground)).
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
