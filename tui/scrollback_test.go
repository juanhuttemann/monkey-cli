package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

var errTest = errors.New("test error")

// TestScrollback_UserSubmit_CommitsUserMessage verifies that when the user submits
// a prompt, the user message is immediately committed to scrollback (printedCount advances).
func TestScrollback_UserSubmit_CommitsUserMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("hello")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := updated.(Model)

	if m.printedCount != 1 {
		t.Errorf("printedCount = %d, want 1 after user submit", m.printedCount)
	}
}

// TestScrollback_DuringStreaming_OnlyUserCommitted verifies that during streaming
// the in-progress assistant message is NOT committed (only the user message is).
func TestScrollback_DuringStreaming_OnlyUserCommitted(t *testing.T) {
	model := NewModel(nil)
	model.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	model.printedCount = 1
	model.streaming = true

	updated, _ := model.Update(PartialResponseMsg{Token: "hi"})
	m := updated.(Model)

	if m.printedCount != 1 {
		t.Errorf("during streaming, printedCount = %d, want 1 (assistant not yet committed)", m.printedCount)
	}
}

// TestScrollback_AfterResponse_CommitsAssistantMessage verifies that when
// PromptResponseMsg arrives, the assistant message is committed to scrollback.
func TestScrollback_AfterResponse_CommitsAssistantMessage(t *testing.T) {
	model := NewModel(nil)
	model.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	model.printedCount = 1
	model.streaming = true

	updated, _ := model.Update(PromptResponseMsg{Response: "world", APIMessages: nil})
	m := updated.(Model)

	if m.printedCount != 2 {
		t.Errorf("after response, printedCount = %d, want 2", m.printedCount)
	}
}

// TestScrollback_AfterToolCall_CommitsToolMessage verifies that each ToolCallMsg
// commits its message to scrollback immediately.
func TestScrollback_AfterToolCall_CommitsToolMessage(t *testing.T) {
	model := NewModel(nil)
	model.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	model.printedCount = 1

	toolMsg := ToolCallMsg{ToolCall: api.ToolCallResult{
		Name:   "bash",
		Input:  map[string]any{"command": "ls"},
		Output: "file.txt",
	}}
	updated, _ := model.Update(toolMsg)
	m := updated.(Model)

	if m.printedCount != 2 {
		t.Errorf("after tool call, printedCount = %d, want 2", m.printedCount)
	}
}

// TestScrollback_AfterError_CommitsErrorMessage verifies that error messages
// are committed to scrollback.
func TestScrollback_AfterError_CommitsErrorMessage(t *testing.T) {
	model := NewModel(nil)
	model.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	model.printedCount = 1
	model.streaming = true

	updated, _ := model.Update(PromptErrorMsg{Err: errTest})
	m := updated.(Model)

	if m.printedCount != 2 {
		t.Errorf("after error, printedCount = %d, want 2", m.printedCount)
	}
}

// TestScrollback_View_ExcludesCommittedMessages verifies that messages already
// committed to scrollback do not appear in View().
func TestScrollback_View_ExcludesCommittedMessages(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.messages = []Message{
		{Role: "user", Content: "committed user msg", Timestamp: time.Now()},
		{Role: "assistant", Content: "committed assistant msg", Timestamp: time.Now()},
	}
	model.printedCount = 2

	view := model.View()

	if containsSubstring(view, "committed user msg") {
		t.Error("View() should not include committed user message")
	}
	if containsSubstring(view, "committed assistant msg") {
		t.Error("View() should not include committed assistant message")
	}
}

// TestScrollback_View_IncludesStreamingMessage verifies that the in-progress
// assistant message (not yet committed) is visible in View().
func TestScrollback_View_IncludesStreamingMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.messages = []Message{
		{Role: "user", Content: "committed user", Timestamp: time.Now()},
		{Role: "assistant", Content: "streaming content", Timestamp: time.Now()},
	}
	model.printedCount = 1
	model.streaming = true

	view := model.View()

	if !containsSubstring(view, "streaming content") {
		t.Error("View() should include the streaming assistant message")
	}
	if containsSubstring(view, "committed user") {
		t.Error("View() should not include the committed user message")
	}
}

// TestScrollback_RestoreSession_CommitsAllMessages verifies that messages
// loaded from a session are immediately considered committed (they go to
// scrollback on next render, not shown in the active view).
func TestScrollback_RestoreSession_CommitsAllMessages(t *testing.T) {
	model := NewModel(nil)
	sess := &SessionData{
		Messages: []Message{
			{Role: "user", Content: "old msg"},
			{Role: "assistant", Content: "old response"},
		},
	}
	model.RestoreSession(sess)

	if model.printedCount != 2 {
		t.Errorf("after RestoreSession, printedCount = %d, want 2", model.printedCount)
	}
}
