package tui

import (
	"strings"
	"testing"
)

// --- messageStyleWidth ---

func TestMessageStyleWidth_StandardTerminal_Unchanged(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)

	if got := m.messageStyleWidth(); got != 80 {
		t.Errorf("messageStyleWidth() = %d, want 80 on 80-wide terminal", got)
	}
}

func TestMessageStyleWidth_NarrowTerminal_Unchanged(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(40, 24)

	if got := m.messageStyleWidth(); got != 40 {
		t.Errorf("messageStyleWidth() = %d, want 40 on 40-wide terminal", got)
	}
}

func TestMessageStyleWidth_WideTerminal_Capped(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(200, 24)

	got := m.messageStyleWidth()
	if got > 126 {
		t.Errorf("messageStyleWidth() = %d on 200-wide terminal, want ≤126", got)
	}
}

// TestRenderMessages_WideTerminal_BubbleWidthCapped verifies that on a very wide
// terminal message bubbles don't expand to full terminal width.
func TestRenderMessages_WideTerminal_BubbleWidthCapped(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(200, 24)
	m.AddMessage("user", strings.Repeat("word ", 40))
	m.AddMessage("assistant", strings.Repeat("word ", 40))

	content := stripANSI(m.renderMessages())

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if len([]rune(trimmed)) > 130 {
			t.Errorf("Message line too wide on 200-char terminal (%d display chars), want ≤130",
				len([]rune(trimmed)))
		}
	}
}

// TestRenderMessages_NarrowTerminal_WidthPreserved verifies that on a narrow
// terminal the bubble still uses the available width (no regression).
func TestRenderMessages_NarrowTerminal_WidthPreserved(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(50, 24)
	m.AddMessage("user", "Hello")

	content := stripANSI(m.renderMessages())

	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "╭") {
			displayWidth := len([]rune(strings.TrimRight(line, " ")))
			if displayWidth < 44 {
				t.Errorf("Bubble too narrow on 50-char terminal: %d display chars", displayWidth)
			}
			return
		}
	}
}

// --- inputWithCursor ---

func TestRender_InputWithCursor_ContainsCursorMarker(t *testing.T) {
	m := NewModel(nil)
	m.SetInput("hello")
	if !strings.Contains(m.inputWithCursor(), "▌") {
		t.Error("inputWithCursor() should contain cursor marker ▌")
	}
}

func TestRender_InputWithCursor_EmptyInput(t *testing.T) {
	m := NewModel(nil)
	if !strings.Contains(m.inputWithCursor(), "▌") {
		t.Error("inputWithCursor() on empty input should contain cursor marker ▌")
	}
}
