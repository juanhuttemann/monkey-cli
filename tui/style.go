package tui

import (
	"strings"

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

// colorizeArt applies lipgloss foreground colors to the three Unicode block
// shading characters used in the monkey pixel art:
//
//	█  →  ColorMonkeyDark  (outline)
//	▒  →  ColorMonkeyMid   (body)
//	░  →  ColorMonkeyLight (face / underbelly)
//
// All other runes (spaces, newlines) are passed through unchanged.
func colorizeArt(s string) string {
	dark := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMonkeyDark))
	mid := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMonkeyMid))
	light := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMonkeyLight))

	var sb strings.Builder
	sb.Grow(len(s) * 2) // rough over-alloc for ANSI sequences
	for _, r := range s {
		switch r {
		case '█':
			sb.WriteString(dark.Render(string(r)))
		case '▒':
			sb.WriteString(mid.Render(string(r)))
		case '░':
			sb.WriteString(light.Render(string(r)))
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// RenderIntroBlock renders a two-panel block split 3/5 left (ASCII art) and
// 2/5 right (title + version at top, "Type ? for help" centered below).
func RenderIntroBlock(width int, title, version, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary))
	verFg := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayMid))

	// Split inner area: 3/5 left, 1 divider, 2/5 right.
	innerW := width - 2 // space between the two outer │ borders
	leftW := innerW * 3 / 5
	rightW := innerW - leftW - 1 // -1 for the divider │

	// Colorize the pixel-art block characters before layout so that
	// lipgloss.Width() (ANSI-aware) still measures correctly during centering.
	content = colorizeArt(content)

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

	// Build right lines: title+version at top, "Type ? for help" centered in remaining rows.
	emptyRight := strings.Repeat(" ", rightW)

	// Row 0: title (and version) centered in the right panel.
	titleText := title
	titleVisW := lipgloss.Width(title)
	if version != "" {
		titleVisW += 1 + lipgloss.Width(version)
	}
	titleLPad := max(0, (rightW-titleVisW)/2)
	titleRPad := max(0, rightW-titleVisW-titleLPad)
	titleLine := strings.Repeat(" ", titleLPad) +
		lipgloss.NewStyle().Bold(true).Render(titleText)
	if version != "" {
		titleLine += " " + verFg.Render(version)
	}
	titleLine += strings.Repeat(" ", titleRPad)

	helpLine := lipgloss.NewStyle().
		Width(rightW).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(ColorGrayDark)).
		Render("Type ? for help")
	rightLines := make([]string, nLines)
	titleRow := min(1, nLines-1) // 1 row top padding; clamp for very short content
	rightLines[titleRow] = titleLine
	// Center "Type ? for help" among rows below the title.
	helpRow := titleRow + 1 + (nLines-1-titleRow)/2
	if helpRow >= nLines {
		helpRow = titleRow // fallback: no room below title
	}
	for i := range rightLines {
		switch {
		case i == titleRow:
			// title already set above
		case i == helpRow:
			rightLines[i] = helpLine
		default:
			rightLines[i] = emptyRight
		}
	}

	var sb strings.Builder
	for i, ll := range leftLines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(ll)
		sb.WriteString(bdr.Render("│"))
		sb.WriteString(rightLines[i])
	}

	return sb.String()
}

// RenderAssistantBlock renders an assistant message inside a green rounded border
// with the model name embedded in the top border: ╭─ claude-sonnet-4-5 ──────╮
func RenderAssistantBlock(width int, modelName, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAssistantBorder))
	modelFg := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayMid))

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
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorUserBorder))
	labelFg := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayMid))

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
// with the tool name embedded in the top border: ╭─ bash ──────╮
func RenderToolBlock(width int, toolName, content string) string {
	bdr := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorToolBorder))

	innerW := width - 4
	paddedContent := lipgloss.NewStyle().Width(innerW).Padding(0, 1).Render(content)
	contentLines := strings.Split(paddedContent, "\n")

	label := toolName
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
