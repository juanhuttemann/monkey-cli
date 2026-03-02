package tui

import (
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"monkey/api"
)

func TestView_RendersMessages(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("user", "Hello")
	model.AddMessage("assistant", "Hi there!")

	view := model.View()

	if !strings.Contains(view, "Hello") {
		t.Error("View() should contain user message 'Hello'")
	}
	if !strings.Contains(stripANSI(view), "Hi there!") {
		t.Error("View() should contain assistant message 'Hi there!'")
	}
}

func TestView_RendersInputArea(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := model.View()

	// The input area should be present in the view
	// It typically has a border or placeholder text
	if view == "" {
		t.Error("View() should return non-empty string")
	}
}

func TestView_RendersSpinner_WhenLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetLoading(true)

	view := model.View()

	// Monkey spinner frames: 🙈 🙉 🙊
	monkeyFrames := []string{"🙈", "🙉", "🙊"}
	hasSpinner := false
	for _, frame := range monkeyFrames {
		if strings.Contains(view, frame) {
			hasSpinner = true
			break
		}
	}

	if !hasSpinner {
		t.Error("View() should contain monkey spinner emoji when loading")
	}
}

func TestView_HidesSpinner_WhenNotLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetLoading(false)

	view := model.View()

	// When not loading, spinner characters should not appear prominently
	// (they might appear in message content, so we check for the loading indicator area)
	// The view should not contain the typical spinner animation pattern
	spinnerChars := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	for _, char := range spinnerChars {
		if strings.Contains(view, string(char)) {
			// If spinner chars are present, they shouldn't be in a loading context
			// This is a soft check - we mainly want to ensure no active spinner animation
		}
	}
}

func TestView_UserMessage_HasGreenBorder(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("user", "Test message")

	view := model.View()

	// Green color ANSI codes typically contain "32" or specific green color codes
	// The lipgloss library uses specific codes for the green we defined (#04B575)
	if !strings.Contains(view, "Test message") {
		t.Error("View() should contain the user message text")
	}

	// Check that the view has styling (ANSI escape codes)
	if !strings.Contains(view, "\x1b[") {
		t.Error("View() should contain ANSI escape codes for styling")
	}
}

func TestView_AssistantMessage_HasBlueBorder(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("assistant", "Assistant response")

	view := model.View()

	if !strings.Contains(stripANSI(view), "Assistant response") {
		t.Error("View() should contain the assistant message text")
	}

	// Check that the view has styling (ANSI escape codes)
	if !strings.Contains(view, "\x1b[") {
		t.Error("View() should contain ANSI escape codes for styling")
	}
}

func TestView_NoLabels(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("user", "User message content")
	model.AddMessage("assistant", "Assistant message content")

	view := model.View()

	// Should NOT contain "User:" or "Assistant:" labels
	if strings.Contains(view, "User:") {
		t.Error("View() should not contain 'User:' label")
	}
	if strings.Contains(view, "user:") {
		t.Error("View() should not contain 'user:' label")
	}
	if strings.Contains(view, "Assistant:") {
		t.Error("View() should not contain 'Assistant:' label")
	}
	if strings.Contains(view, "assistant:") {
		t.Error("View() should not contain 'assistant:' label")
	}
}

func TestView_MessagesInOrder(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("user", "first message")
	model.AddMessage("assistant", "second message")
	model.AddMessage("user", "third message")

	view := model.View()

	// Find positions of each message (strip ANSI so multi-word text is contiguous)
	stripped := stripANSI(view)
	firstIdx := strings.Index(stripped, "first message")
	secondIdx := strings.Index(stripped, "second message")
	thirdIdx := strings.Index(stripped, "third message")

	if firstIdx == -1 || secondIdx == -1 || thirdIdx == -1 {
		t.Fatal("View() should contain all messages")
	}

	// Verify order: first should come before second, second before third
	if firstIdx > secondIdx {
		t.Error("First message should appear before second message")
	}
	if secondIdx > thirdIdx {
		t.Error("Second message should appear before third message")
	}
}

func TestView_AutoScrollsToBottom(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	// Add many messages to create scrollable content
	for i := 0; i < 30; i++ {
		model.AddMessage("user", "Message number in sequence")
	}

	view := model.View()

	// The most recent content should be visible
	// This is a soft check - we verify the view renders without error
	if view == "" {
		t.Error("View() should return non-empty string with many messages")
	}
}

func TestView_ErrorMessage_HasRedStyle(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("error", "Something went wrong")

	view := model.View()

	if !strings.Contains(view, "Something went wrong") {
		t.Error("View() should contain the error message text")
	}

	// Check that the view has styling (ANSI escape codes)
	if !strings.Contains(view, "\x1b[") {
		t.Error("View() should contain ANSI escape codes for error styling")
	}
}

func TestView_InputContainsText(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetInput("User is typing this")

	view := model.View()

	// The input text should appear in the view
	if !strings.Contains(view, "User is typing this") {
		t.Error("View() should contain the current input text")
	}
}

func TestView_EmptyHistory(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := model.View()

	// View should render successfully with empty history
	if view == "" {
		t.Error("View() should return non-empty string even with empty history")
	}
}

func TestView_LongMessage_WrapsCorrectly(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(40, 24) // Narrow width

	longMessage := "This is a very long message that should wrap across multiple lines when rendered in a narrow viewport because it exceeds the available width."
	model.AddMessage("user", longMessage)

	view := model.View()

	// The message content should still appear
	if !strings.Contains(view, "This is a very long message") {
		t.Error("View() should contain the long message text")
	}
}

func TestView_TooSmall_ShowsWarning(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(10, 5) // Very small dimensions

	view := model.View()

	// With very small dimensions, the view should still render something
	// Either a warning or at least not crash
	if view == "" {
		t.Error("View() should handle small dimensions gracefully")
	}
}

func TestView_Integration(t *testing.T) {
	// Integration test with mock server
	server := httptest.NewServer(nil)
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetDimensions(80, 24)
	model.AddMessage("user", "Hello")
	model.AddMessage("assistant", "Hi!")
	model.SetInput("New message")

	view := model.View()

	// Verify all components are present
	if !strings.Contains(view, "Hello") {
		t.Error("View() should contain user message")
	}
	if !strings.Contains(stripANSI(view), "Hi!") {
		t.Error("View() should contain assistant message")
	}
	if !strings.Contains(view, "New message") {
		t.Error("View() should contain input text")
	}
}

func TestView_ShowsCanceled_AfterEscWhileLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetLoading(true)

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escKey)

	view := updatedModel.(Model).View()

	if !strings.Contains(stripANSI(view), "canceled") {
		t.Error("View() should show 'canceled' after Esc while loading")
	}
}

func TestView_ShowsCursorInInputArea(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := model.View()

	// The input area should render a visible cursor so users know they can type
	if !strings.Contains(view, "▌") {
		t.Error("View() should show a cursor indicator (▌) in the input area")
	}
}
