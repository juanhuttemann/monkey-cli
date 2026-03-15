package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// messageStyleWidth returns the style parameter used for message bubbles.
// It caps the terminal width so text stays within ~75 chars for readability.
// Width(p-4) + Padding(0,1) = text area of p-6; at p=81 that is 75 chars.
func (m Model) messageStyleWidth() int {
	const maxStyleWidth = 126
	if m.width > maxStyleWidth {
		return maxStyleWidth
	}
	return m.width
}

// renderMessageEntry returns the fully-formatted string for messages[i],
// including a search-match prefix (when active) and a timestamp footer.
func (m Model) renderMessageEntry(sw, i int) string {
	msg := m.messages[i]
	rendered := m.renderSingleMessage(sw, msg)
	if m.searchBar.IsActive() && m.searchBar.IsMatch(i) {
		matchLabel := SearchMatchStyle().Render("▶ match")
		rendered = matchLabel + "\n" + rendered
	}
	var sb strings.Builder
	sb.WriteString(rendered)
	sb.WriteString("\n")
	sb.WriteString(MessageTimestampStyle(sw).Render(msg.Timestamp.Format("15:04")))
	sb.WriteString("\n")
	return sb.String()
}

// renderMessages returns the styled content string for all messages.
func (m Model) renderMessages() string {
	sw := m.messageStyleWidth()

	// Streaming fast path: prior messages were rendered once into renderedPrior on
	// the first token; only re-render the last (in-flight) message each tick.
	// Fall back to the full loop when search is active (every message needs a
	// match prefix) or when the cache is stale (e.g. after a window resize).
	if m.streaming && m.renderedPriorValid && !m.searchBar.IsActive() {
		n := len(m.messages)
		if n == 0 {
			return m.renderedPrior
		}
		var sb strings.Builder
		sb.WriteString(m.renderedPrior)
		sb.WriteString(m.renderMessageEntry(sw, n-1))
		return sb.String()
	}

	var sb strings.Builder
	for i := m.printedCount; i < len(m.messages); i++ {
		sb.WriteString(m.renderMessageEntry(sw, i))
	}
	return sb.String()
}

// commitUpTo prints messages[printedCount:n] to the terminal scrollback via
// tea.Println and advances printedCount to n. Returns nil if nothing to print.
func (m *Model) commitUpTo(n int) tea.Cmd {
	if n <= m.printedCount {
		return nil
	}
	sw := m.messageStyleWidth()
	var cmds []tea.Cmd
	for i := m.printedCount; i < n; i++ {
		msg := m.messages[i]
		rendered := m.renderSingleMessage(sw, msg)
		ts := msg.Timestamp.Format("15:04")
		text := rendered + "\n" + MessageTimestampStyle(sw).Render(ts)
		cmds = append(cmds, tea.Println(text))
	}
	m.printedCount = n
	return tea.Batch(cmds...)
}

// renderSingleMessage returns the styled string for one message (without timestamp).
func (m Model) renderSingleMessage(sw int, msg Message) string {
	switch msg.Role {
	case roleUser:
		return RenderUserBlock(sw, msg.Content)
	case roleAssistant:
		md := strings.TrimRight(RenderMarkdown(msg.Content, sw-8), "\n")
		modelName := ""
		if m.client != nil {
			modelName = m.client.GetModel()
		}
		if modelName != "" {
			return RenderAssistantBlock(sw, modelName, md)
		}
		return AssistantMessageStyle(sw).Render(md)
	case roleTool:
		content := msg.Content
		if msg.Collapsed {
			lines := strings.Split(content, "\n")
			content = fmt.Sprintf("%s\n[%d lines hidden — ctrl+t to expand]", lines[0], len(lines)-1)
		}
		return RenderToolBlock(sw, msg.ToolName, content)
	case roleSystem:
		return SystemMessageStyle(sw).Render(msg.Content)
	default:
		return ErrorMessageStyle(sw).Render(msg.Content)
	}
}

// renderStatusBar renders a 1-line footer: model | ape | tokens.
func (m Model) renderStatusBar() string {
	sep := StatusBarSepStyle().Render(" | ")

	model := ""
	if m.client != nil {
		model = m.client.GetModel()
	}
	modelSeg := StatusBarModelStyle().Render(model)

	var apeSeg string
	if m.autoApprove {
		apeSeg = ApeModeActiveStyle().Render("ape mode: on")
	} else {
		apeSeg = ApeModeInactiveStyle().Render("ape mode: off")
	}

	total := m.totalUsage.InputTokens + m.totalUsage.OutputTokens
	if total > 0 {
		tokenStr := fmt.Sprintf("%s tokens", formatTokenCount(total))
		if cost := formatCost(estimateCost(model, m.totalUsage)); cost != "" {
			tokenStr += "  " + cost
		}
		tokenSeg := StatusBarTokenStyle().Render(tokenStr)
		return modelSeg + sep + apeSeg + sep + tokenSeg
	}
	return modelSeg + sep + apeSeg
}

// inputWithCursor returns the input value with the ▌ cursor character inserted
// at the actual textarea cursor position (row/col), so the visual cursor tracks
// correctly when CursorUp/CursorDown moves within multiline input.
func (m Model) inputWithCursor() string {
	row := m.input.Line()
	info := m.input.LineInfo()
	col := info.StartColumn + info.ColumnOffset // raw rune index within row

	lines := strings.Split(m.input.Value(), "\n")
	if row < len(lines) {
		line := []rune(lines[row])
		if col > len(line) {
			col = len(line)
		}
		lines[row] = string(line[:col]) + "▌" + string(line[col:])
	}
	return strings.Join(lines, "\n")
}

// formatRetryLabel formats the retry attempt indicator shown in the status line.
// When reason is non-empty, the format is "retrying: <reason> (<attempt>)".
func formatRetryLabel(attempt int, reason string) string {
	if reason != "" {
		return fmt.Sprintf("retrying: %s (%d)", reason, attempt)
	}
	return fmt.Sprintf("retrying (%d)", attempt)
}
