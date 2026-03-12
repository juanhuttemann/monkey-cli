package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// colorizeArt applies lipgloss foreground colors to the Unicode block
// shading characters used in the monkey pixel art:
//
//	█▄▀  →  ColorMonkeyDark  (outline)
//	▓  →  ColorMonkeyMid   (body)
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
		case '█', '▄', '▀':
			sb.WriteString(dark.Render(string(r)))
		case '▓', '▒':
			sb.WriteString(mid.Render(string(r)))
		case '░':
			sb.WriteString(light.Render(string(r)))
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// RenderIntroBlock renders a two-panel block: left panel sized to fit the ASCII
// art, right panel taking the remaining space (title + version + help text).
func RenderIntroBlock(width int, title, version, content string) string {
	verFg := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGrayMid))

	innerW := width - 2 // space between the two outer │ borders

	// Colorize the pixel-art block characters before layout so that
	// lipgloss.Width() (ANSI-aware) still measures correctly during centering.
	content = colorizeArt(content)

	// Size the left panel to fit the art exactly (padding: 3 left, 1 right).
	artLines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	maxArtW := 0
	for _, l := range artLines {
		if w := lipgloss.Width(l); w > maxArtW {
			maxArtW = w
		}
	}
	leftW := maxArtW + 6 // 3 left padding + 3 right padding
	rightW := innerW - leftW - 1 // -1 for the divider │

	leftLines := strings.Split(strings.TrimRight(
		lipgloss.NewStyle().Width(leftW).Padding(0, 3, 0, 3).Render(strings.TrimRight(content, "\n")),
		"\n",
	), "\n")
	nLines := len(leftLines)

	// Build right lines: title+version at top, "Type ? for help" below.
	emptyRight := strings.Repeat(" ", rightW)

	const rightPad = "   " // 3-space left padding for right panel content

	titleLine := rightPad + lipgloss.NewStyle().Bold(true).Render(title)
	if version != "" {
		titleLine += " " + verFg.Render(version)
	}
	titleVisW := lipgloss.Width(titleLine)
	titleLine += strings.Repeat(" ", max(0, rightW-titleVisW))

	helpLine := lipgloss.NewStyle().
		Width(rightW).
		Foreground(lipgloss.Color(ColorGrayDark)).
		Render(rightPad + "Type ? for help")
	rightLines := make([]string, nLines)
	titleRow := min(1, nLines-1) // 1 row top padding; clamp for very short content
	rightLines[titleRow] = titleLine
	// Center "Type ? for help" among rows below the title.
	helpRow := titleRow + 1 + (nLines-1-titleRow)/2
	if helpRow >= nLines {
		helpRow = titleRow // fallback: no room below title
	}
	for i := range rightLines {
		switch i {
		case titleRow:
			// title already set above
		case helpRow:
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
		sb.WriteString("│")
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
