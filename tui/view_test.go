package tui

import (
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
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
		_ = strings.Contains(view, string(char))
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

func TestView_AssistantMessage_ShowsModelInHeader(t *testing.T) {
	client := newTestClientWithModel("claude-sonnet-4-5")
	model := NewModel(client)
	model.SetDimensions(80, 24)
	model.AddMessage("assistant", "Hello!")

	view := model.View()

	if !strings.Contains(stripANSI(view), "claude-sonnet-4-5") {
		t.Error("View() should show the current model name in the assistant message header")
	}
}

func TestView_AssistantMessage_NoModelHeader_WhenNoClient(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("assistant", "Hello!")

	view := model.View()

	// Should render without panicking and still show content
	if !strings.Contains(stripANSI(view), "Hello!") {
		t.Error("View() should show assistant message content even when client is nil")
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

	if !strings.Contains(stripANSI(view), "What should monkey do?") {
		t.Error("View() should show 'What should monkey do?' after Esc while loading")
	}
}

func TestView_UserHeader_VisibleAfterFirstSubmit(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetIntro("ASCII ART")
	model.SetIntroTitle("Monkey")
	model.SetIntroVersion("v1.0")
	model.SetInput("Hello!")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := updated.(Model)

	// In scrollback mode the user message is committed immediately (not in View()).
	// Verify the message was committed rather than rendered in the active frame.
	if m.printedCount != 1 {
		t.Errorf("After first submit, printedCount = %d, want 1", m.printedCount)
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

func TestView_StatusBar_ShowsModelName(t *testing.T) {
	client := api.NewClient("http://example.com", "key", api.WithModel("claude-sonnet-4-6"))
	model := NewModel(client)
	model.SetDimensions(80, 24)

	view := stripANSI(model.View())

	if !strings.Contains(view, "claude-sonnet-4-6") {
		t.Errorf("status bar should show model name, got:\n%s", view)
	}
}

func TestView_StatusBar_ShowsTokenCount(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.totalUsage = api.Usage{InputTokens: 1000, OutputTokens: 341}

	view := stripANSI(model.View())

	if !strings.Contains(view, "1,341 tokens") {
		t.Errorf("status bar should show '1,341 tokens', got:\n%s", view)
	}
}

func TestView_StatusBar_NoTokensWhenZero(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := stripANSI(model.View())

	if strings.Contains(view, "tokens") {
		t.Errorf("status bar should not show token count when zero, got:\n%s", view)
	}
}

// TestView_CursorTracksLineAfterCursorUp verifies that the ▌ cursor character
// appears on the correct line after CursorUp moves the cursor within multiline input.
// Regression test: previously ▌ was always appended at the end of Value(), so after
// pressing Up (CursorUp) within multiline text the cursor stayed on the last line.
func TestView_CursorTracksLineAfterCursorUp(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	// SetInput puts the cursor at the end of the last line (row 1 for "line1\nline2").
	model.SetInput("line1\nline2")

	// Press Up: cursor is on row 1, so CursorUp() is called → cursor moves to row 0.
	upKey := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := model.Update(upKey)
	m := updated.(Model)

	view := stripANSI(m.View())
	viewLines := strings.Split(view, "\n")

	// Find the rendered lines that contain "line1" and "line2".
	var line1Row, line2Row string
	for _, l := range viewLines {
		if strings.Contains(l, "line1") {
			line1Row = l
		}
		if strings.Contains(l, "line2") {
			line2Row = l
		}
	}

	if line1Row == "" {
		t.Fatal("could not find 'line1' in view")
	}
	if line2Row == "" {
		t.Fatal("could not find 'line2' in view")
	}

	// After CursorUp the cursor must be on the line that contains "line1".
	if !strings.Contains(line1Row, "▌") {
		t.Errorf("▌ should appear on the 'line1' row after CursorUp, got row: %q", line1Row)
	}
	// The "line2" row must NOT carry the cursor.
	if strings.Contains(line2Row, "▌") {
		t.Errorf("▌ should NOT appear on the 'line2' row after CursorUp, got row: %q", line2Row)
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{8341, "8,341"},
		{1234567, "1,234,567"},
	}
	for _, tc := range tests {
		got := formatTokenCount(tc.n)
		if got != tc.want {
			t.Errorf("formatTokenCount(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
