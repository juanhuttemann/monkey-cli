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
		Width(width-4).
		Padding(0, 1)
}

// AssistantMessageStyle returns the styling for assistant messages
func AssistantMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(AssistantBorderColor)).
		Width(width-4).
		Padding(0, 1)
}

// ErrorMessageStyle returns the styling for error messages
func ErrorMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ErrorBorderColor)).
		Width(width-4).
		Padding(0, 1)
}

// ToolMessageStyle returns the styling for tool call messages (cyan border)
func ToolMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ToolBorderColor)).
		Width(width-4).
		Padding(0, 1)
}

// SystemMessageStyle returns the styling for system notices (e.g. retry messages)
func SystemMessageStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(SystemBorderColor)).
		Width(width-4).
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

// WaitingStyle returns the styling for the idle/waiting prompt shown after cancellation.
func WaitingStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4A017")).
		Italic(true)
}

// IntroBorderColor is the dark brown color used for the intro block border and title.
const IntroBorderColor = "#7B4F2E"

// RenderIntroBlock renders content inside a bordered block with the title
// and optional version embedded in the top border: ╭─ Title v0.1.0 ──────╮
// The block is split: 3/5 left (content) │ 2/5 right ("Type ? for help").
func RenderIntroBlock(width int, title, version, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(IntroBorderColor))
	verFg := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// Split inner area: 3/5 left, 1 divider, 2/5 right.
	innerW := width - 2 // space between the two outer │ borders
	leftW := innerW * 3 / 5
	rightW := innerW - leftW - 1 // -1 for the divider │

	// Center the art block as a unit within the left panel.
	// Find the widest art line, compute a uniform left offset, then render.
	artLines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	maxArtW := 0
	for _, l := range artLines {
		if w := lipgloss.Width(l); w > maxArtW {
			maxArtW = w
		}
	}
	contentW := leftW - 2 // inner content area (Width() is outer, Padding uses 1 each side)
	leftOff := max(0, (contentW-maxArtW)/2)
	pad := strings.Repeat(" ", leftOff)
	var centeredArt strings.Builder
	for i, l := range artLines {
		if i > 0 {
			centeredArt.WriteByte('\n')
		}
		centeredArt.WriteString(pad + l)
	}
	leftLines := strings.Split(strings.TrimRight(
		lipgloss.NewStyle().Width(leftW).Padding(0, 1).Render(centeredArt.String()),
		"\n",
	), "\n")
	nLines := len(leftLines)

	// Build right lines: "Type ? for help" vertically centered.
	helpLine := lipgloss.NewStyle().
		Width(rightW).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#666666")).
		Render("Type ? for help")
	emptyRight := strings.Repeat(" ", rightW)
	rightLines := make([]string, nLines)
	for i := range rightLines {
		if i == nLines/2 {
			rightLines[i] = helpLine
		} else {
			rightLines[i] = emptyRight
		}
	}

	// Top border: ╭─ Title v0.1.0 ──────────────────────────────────────────╮
	prefix := "╭─ "
	suffix := " "
	titleSection := prefix + title
	if version != "" {
		titleSection += " " + version
	}
	titleSection += suffix
	// innerW+2 accounts for the two │ side chars; -1 for ╮
	dashLen := max(0, innerW+2-lipgloss.Width(titleSection)-1)

	topLine := bdr.Render(prefix + title)
	if version != "" {
		topLine += bdr.Render(" ") + verFg.Render(version)
	}
	topLine += bdr.Render(suffix + strings.Repeat("─", dashLen) + "╮")

	var sb strings.Builder
	sb.WriteString(topLine)
	for i, ll := range leftLines {
		sb.WriteByte('\n')
		sb.WriteString(bdr.Render("│"))
		sb.WriteString(ll)
		sb.WriteString(bdr.Render("│"))
		sb.WriteString(rightLines[i])
		sb.WriteString(bdr.Render("│"))
	}
	// Bottom border: ╰──────────────────────────────────────────────────────╯
	sb.WriteByte('\n')
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerW) + "╯"))

	return sb.String()
}

// RenderAssistantBlock renders an assistant message inside a green rounded border
// with the model name embedded in the top border: ╭─ claude-sonnet-4-5 ──────╮
func RenderAssistantBlock(width int, modelName, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(AssistantBorderColor))
	modelFg := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// innerW matches the Width() param used by AssistantMessageStyle so the block is the same size.
	innerW := width - 4

	// Render content with the same padding as AssistantMessageStyle (no border here).
	paddedContent := lipgloss.NewStyle().Width(innerW).Padding(0, 1).Render(content)
	contentLines := strings.Split(paddedContent, "\n")

	// Top border: ╭─ modelName ──────────────────────────────────────────╮
	// "─ " prefix + modelName + " " suffix take up (2 + len(model) + 1) of innerW.
	titleW := 2 + lipgloss.Width(modelName) + 1
	dashLen := max(0, innerW-titleW)
	topLine := bdr.Render("╭─ ") + modelFg.Render(modelName) + bdr.Render(" "+strings.Repeat("─", dashLen)+"╮")

	var sb strings.Builder
	sb.WriteString(topLine)
	for _, line := range contentLines {
		sb.WriteByte('\n')
		sb.WriteString(bdr.Render("│") + line + bdr.Render("│"))
	}
	sb.WriteByte('\n')
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerW) + "╯"))

	return sb.String()
}

// RenderUserBlock renders a user message inside a purple rounded border
// with "You" embedded in the top border: ╭─ You ──────╮
func RenderUserBlock(width int, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(UserBorderColor))
	labelFg := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	innerW := width - 4
	paddedContent := lipgloss.NewStyle().Width(innerW).Padding(0, 1).Render(content)
	contentLines := strings.Split(paddedContent, "\n")

	label := "You"
	titleW := 2 + lipgloss.Width(label) + 1
	dashLen := max(0, innerW-titleW)
	topLine := bdr.Render("╭─ ") + labelFg.Render(label) + bdr.Render(" "+strings.Repeat("─", dashLen)+"╮")

	var sb strings.Builder
	sb.WriteString(topLine)
	for _, line := range contentLines {
		sb.WriteByte('\n')
		sb.WriteString(bdr.Render("│") + line + bdr.Render("│"))
	}
	sb.WriteByte('\n')
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerW) + "╯"))

	return sb.String()
}

// RenderToolBlock renders a tool call message inside a cyan rounded border
// with a wrench emoji embedded in the top border: ╭─ 🔧 ──────╮
func RenderToolBlock(width int, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(ToolBorderColor))

	innerW := width - 4
	paddedContent := lipgloss.NewStyle().Width(innerW).Padding(0, 1).Render(content)
	contentLines := strings.Split(paddedContent, "\n")

	label := "🔧"
	titleW := 2 + lipgloss.Width(label) + 1
	dashLen := max(0, innerW-titleW)
	topLine := bdr.Render("╭─ " + label + " " + strings.Repeat("─", dashLen) + "╮")

	var sb strings.Builder
	sb.WriteString(topLine)
	for _, line := range contentLines {
		sb.WriteByte('\n')
		sb.WriteString(bdr.Render("│") + line + bdr.Render("│"))
	}
	sb.WriteByte('\n')
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerW) + "╯"))

	return sb.String()
}

// FilePickerStyle returns the styling for the file picker dropdown border.
func FilePickerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666666")).
		Width(width-4).
		Padding(0, 1)
}

// FilePickerCursorStyle returns the styling for the highlighted file picker row.
func FilePickerCursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(AssistantBorderColor)).
		Bold(true)
}

// ApeModeActiveStyle returns banana-yellow styling for the ape mode indicator when enabled.
func ApeModeActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
}

// ApeModeInactiveStyle returns grey styling for the ape mode indicator when disabled.
func ApeModeInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
}

// ToolApprovalModelStyle returns flashy vivid-violet bold styling for the model name in the approval dialog.
func ToolApprovalModelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C084FC")).
		Bold(true)
}

// ToolApprovalPreviewStyle returns a dark navy background style for the bash command preview line.
func ToolApprovalPreviewStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)
}

// ToolApprovalToolStyle returns flashy amber-gold bold styling for the tool name in the approval dialog.
func ToolApprovalToolStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Bold(true)
}

// MessageTimestampStyle returns the styling for per-message timestamps: gray, right-aligned.
func MessageTimestampStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width - 2).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("#555555"))
}
