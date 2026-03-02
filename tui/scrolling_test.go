package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestScrolling_ViewportOffsetAtBottomAfterNewMessage verifies that after a
// PromptResponseMsg, the model's viewport.YOffset is set to the bottom position
// (not 0), confirming SetContent + GotoBottom are called in Update, not just View.
func TestScrolling_ViewportOffsetAtBottomAfterNewMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 10) // viewport height = 10 - 6 = 4 lines

	for i := 0; i < 15; i++ {
		model.AddMessage("user", fmt.Sprintf("user message number %d", i))
	}

	response := PromptResponseMsg{Response: "assistant reply"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	if m.viewport.YOffset == 0 {
		t.Error("After PromptResponseMsg with tall content, viewport.YOffset should be > 0 (SetContent + GotoBottom must be called in Update)")
	}
}

// TestScrolling_MouseWheelUp_DecreasesYOffset verifies that mouse scroll up
// actually changes the viewport offset (requires content set in the model).
func TestScrolling_MouseWheelUp_DecreasesYOffset(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 10)

	for i := 0; i < 15; i++ {
		model.AddMessage("user", fmt.Sprintf("user message number %d", i))
	}

	response := PromptResponseMsg{Response: "assistant reply"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	if m.viewport.YOffset == 0 {
		t.Skip("Viewport not scrollable - content may not exceed viewport height")
	}

	initialOffset := m.viewport.YOffset
	mouseUp := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	updated2, _ := m.Update(mouseUp)
	m2 := updated2.(Model)

	if m2.viewport.YOffset >= initialOffset {
		t.Errorf("After mouse wheel up, YOffset = %d, want < %d", m2.viewport.YOffset, initialOffset)
	}
}

// TestScrolling_ScrolledUp_ViewShowsScrolledContent verifies that after scrolling
// to the top, View() renders earlier messages, not always the bottom.
func TestScrolling_ScrolledUp_ViewShowsScrolledContent(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 10)

	for i := 0; i < 15; i++ {
		model.AddMessage("user", fmt.Sprintf("user message number %d", i))
	}

	response := PromptResponseMsg{Response: "assistant reply"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	if m.viewport.YOffset == 0 {
		t.Skip("Content fits in viewport - scrolling not needed")
	}

	// Scroll all the way to the top
	mouseUp := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	for i := 0; i < 200; i++ {
		updated2, _ := m.Update(mouseUp)
		m = updated2.(Model)
	}

	topView := m.View()
	if !strings.Contains(topView, "user message number 0") {
		t.Error("After scrolling to top, first message should be visible in View()")
	}
}

// TestScrolling_NewMessage_SetsScrollToBottomTrue verifies that a new message
// resets the scrollToBottom flag so the next View() call goes to the bottom.
func TestScrolling_NewMessage_SetsScrollToBottomTrue(t *testing.T) {
	model := NewModel(nil)
	model.scrollToBottom = false // simulate user scrolled up

	response := PromptResponseMsg{Response: "new message"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	if !m.scrollToBottom {
		t.Error("After PromptResponseMsg, scrollToBottom should be true")
	}
}

// TestScrolling_MouseWheelUp_SetsScrollToBottomFalse verifies that mouse wheel up
// clears scrollToBottom so subsequent renders don't snap back to the bottom.
func TestScrolling_MouseWheelUp_SetsScrollToBottomFalse(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 10)

	for i := 0; i < 15; i++ {
		model.AddMessage("user", fmt.Sprintf("message %d", i))
	}
	response := PromptResponseMsg{Response: "reply"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	if !m.scrollToBottom {
		t.Fatal("scrollToBottom should be true after new message")
	}

	mouseUp := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	updated2, _ := m.Update(mouseUp)
	m2 := updated2.(Model)

	if m2.scrollToBottom {
		t.Error("After mouse wheel up, scrollToBottom should be false")
	}
}
