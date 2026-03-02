package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Style constants
const (
	UserBorderColor      = "#4B2DA8" // Dark purple
	AssistantBorderColor = "#04B575" // Green
	ErrorBorderColor     = "#FF6B6B" // Red
	ToolBorderColor      = "#00AACC" // Cyan
	SystemBorderColor    = "#FFA500" // Orange
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

// SystemMessageStyle returns the styling for system notices (e.g. retry messages)
func SystemMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(SystemBorderColor)).
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

// IntroBorderColor is the dark brown color used for the intro block border and title.
const IntroBorderColor = "#7B4F2E"

// RenderIntroBlock renders content inside a bordered block with the title
// and optional version embedded in the top border: ╭─ Title v0.1.0 ──────╮
func RenderIntroBlock(width int, title, version, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(IntroBorderColor))
	verFg := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// Render the inner content with padding (no border).
	inner := lipgloss.NewStyle().
		Width(width - 4).
		Padding(0, 1).
		Render(content)

	lines := strings.Split(strings.TrimRight(inner, "\n"), "\n")
	innerLineWidth := lipgloss.Width(lines[0])

	// Top border: ╭─ Title v0.1.0 ──────────────────────────────────────────╮
	prefix := "╭─ "
	suffix := " "
	titleSection := prefix + title
	if version != "" {
		titleSection += " " + version
	}
	titleSection += suffix
	// +2 accounts for the two │ side chars; -1 for ╮
	dashLen := max(0, innerLineWidth+2-lipgloss.Width(titleSection)-1)

	topLine := bdr.Render(prefix+title)
	if version != "" {
		topLine += bdr.Render(" ") + verFg.Render(version)
	}
	topLine += bdr.Render(suffix+strings.Repeat("─", dashLen)+"╮")

	var sb strings.Builder
	sb.WriteString(topLine)
	for _, line := range lines {
		sb.WriteByte('\n')
		sb.WriteString(bdr.Render("│"))
		sb.WriteString(line)
		sb.WriteString(bdr.Render("│"))
	}
	// Bottom border: ╰──────────────────────────────────────────────────────╯
	sb.WriteByte('\n')
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerLineWidth) + "╯"))

	return sb.String()
}

// MessageTimestampStyle returns the styling for per-message timestamps: gray, right-aligned.
func MessageTimestampStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width - 2).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("#555555"))
}
