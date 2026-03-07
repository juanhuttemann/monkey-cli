package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

func makeToolContent(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString("line content here\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// --- ToolCallMsg adds collapsed message for long output ---

func TestToolCallMsg_ShortOutput_NotCollapsed(t *testing.T) {
	m := NewModel(nil)
	tc := api.ToolCallResult{
		Name:   "bash",
		Input:  map[string]any{"command": "echo hi"},
		Output: makeToolContent(5), // short — below threshold
	}
	updated, _ := m.Update(ToolCallMsg{ToolCall: tc})
	result := updated.(Model)

	history := result.GetHistory()
	if len(history) == 0 {
		t.Fatal("no messages after ToolCallMsg")
	}
	last := history[len(history)-1]
	if last.Role != "tool" {
		t.Fatalf("last.Role = %q, want %q", last.Role, "tool")
	}
	if last.Collapsed {
		t.Error("short tool output should not be collapsed by default")
	}
}

func TestToolCallMsg_LongOutput_AutoCollapsed(t *testing.T) {
	m := NewModel(nil)
	tc := api.ToolCallResult{
		Name:   "bash",
		Input:  map[string]any{"command": "git log"},
		Output: makeToolContent(toolCollapseLines + 5),
	}
	updated, _ := m.Update(ToolCallMsg{ToolCall: tc})
	result := updated.(Model)

	history := result.GetHistory()
	if len(history) == 0 {
		t.Fatal("no messages after ToolCallMsg")
	}
	last := history[len(history)-1]
	if last.Role != "tool" {
		t.Fatalf("last.Role = %q, want %q", last.Role, "tool")
	}
	if !last.Collapsed {
		t.Error("long tool output (>20 lines) should be auto-collapsed")
	}
}

// --- ctrl+t expands the last collapsed tool message ---

func TestCtrlT_NoToolMessages_Noop(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("user", "hello")
	ctrlT := tea.KeyMsg{Type: tea.KeyCtrlT}
	updated, _ := m.Update(ctrlT)
	result := updated.(Model)
	// Nothing should change; no tool messages
	if len(result.GetHistory()) != 1 {
		t.Errorf("history len changed after ctrl+t with no tool messages")
	}
}

func TestCtrlT_ExpandsLastCollapsedTool(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("tool", makeToolContent(toolCollapseLines+5))
	// Manually mark it collapsed (as auto-collapse would)
	m.messages[0].Collapsed = true

	ctrlT := tea.KeyMsg{Type: tea.KeyCtrlT}
	updated, _ := m.Update(ctrlT)
	result := updated.(Model)

	history := result.GetHistory()
	if history[0].Collapsed {
		t.Error("ctrl+t should expand a collapsed tool message")
	}
}

func TestCtrlT_NoopWhenNothingCollapsed(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("tool", makeToolContent(toolCollapseLines+5))
	// Start expanded — ctrl+t only expands, so this should be a noop
	m.messages[0].Collapsed = false

	ctrlT := tea.KeyMsg{Type: tea.KeyCtrlT}
	updated, _ := m.Update(ctrlT)
	result := updated.(Model)

	history := result.GetHistory()
	if history[0].Collapsed {
		t.Error("ctrl+t should not collapse an already-expanded tool message")
	}
}

func TestCtrlT_ExpandsLastCollapsedToolWhenMultiple(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("tool", makeToolContent(toolCollapseLines+5))
	m.AddMessage("user", "next prompt")
	m.AddMessage("tool", makeToolContent(toolCollapseLines+5))
	m.messages[0].Collapsed = true
	m.messages[2].Collapsed = true // last tool

	ctrlT := tea.KeyMsg{Type: tea.KeyCtrlT}
	updated, _ := m.Update(ctrlT)
	result := updated.(Model)

	history := result.GetHistory()
	// Only the last collapsed tool message should be expanded
	if history[2].Collapsed {
		t.Error("ctrl+t should expand the last collapsed tool message (index 2)")
	}
	if !history[0].Collapsed {
		t.Error("first tool message (index 0) should remain collapsed")
	}
}

// --- Rendering: collapsed tool shows summary, not full content ---

func TestRenderMessages_CollapsedTool_ShowsSummary(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)
	content := makeToolContent(toolCollapseLines + 5)
	m.AddMessage("tool", content)
	m.messages[0].Collapsed = true

	rendered := m.renderMessages()
	// Should contain a hint that content is hidden
	if !strings.Contains(rendered, "lines") {
		t.Errorf("collapsed tool render should mention 'lines', got:\n%s", rendered[:min(300, len(rendered))])
	}
	// Should NOT contain all the line content
	lineCount := strings.Count(rendered, "line content here")
	if lineCount > 3 {
		t.Errorf("collapsed tool render should not show all lines, got %d occurrences", lineCount)
	}
}

func TestRenderMessages_ExpandedTool_ShowsFullContent(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)
	content := makeToolContent(toolCollapseLines + 5)
	m.AddMessage("tool", content)
	m.messages[0].Collapsed = false

	rendered := m.renderMessages()
	lineCount := strings.Count(rendered, "line content here")
	if lineCount < toolCollapseLines {
		t.Errorf("expanded tool render should show all %d lines, got %d occurrences", toolCollapseLines+5, lineCount)
	}
}

