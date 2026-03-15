package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestScrolling_ViewportOffsetAtBottomAfterNewMessage verifies that after a
// PromptResponseMsg, messages are committed to scrollback (printedCount advances).
// With scrollback mode, completed messages leave the viewport.
func TestScrolling_ViewportOffsetAtBottomAfterNewMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 10)

	for i := 0; i < 15; i++ {
		model.AddMessage("user", fmt.Sprintf("user message number %d", i))
	}

	response := PromptResponseMsg{Response: "assistant reply"}
	updated, _ := model.Update(response)
	m := updated.(Model)

	// All messages (15 user + 1 assistant) should be committed to scrollback.
	if m.printedCount != 16 {
		t.Errorf("After PromptResponseMsg, printedCount = %d, want 16", m.printedCount)
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

// TestScrolling_PageUp_SetsScrollToBottomFalse verifies that pressing PageUp
// clears scrollToBottom so subsequent renders don't snap back to the bottom.
func TestScrolling_PageUp_SetsScrollToBottomFalse(t *testing.T) {
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

	pgUp := tea.KeyMsg{Type: tea.KeyPgUp}
	updated2, _ := m.Update(pgUp)
	m2 := updated2.(Model)

	if m2.scrollToBottom {
		t.Error("After PageUp, scrollToBottom should be false")
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
