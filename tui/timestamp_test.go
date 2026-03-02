package tui

import (
	"strings"
	"testing"
	"time"
)

// fixedTime returns a deterministic time for timestamp tests: 14:23.
var fixedTime = time.Date(2024, 1, 15, 14, 23, 0, 0, time.UTC)

func modelWithTimestampedMessage(role, content string) Model {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	m.messages = append(m.messages, Message{Role: role, Content: content, Timestamp: fixedTime})
	return m
}

func TestView_ShowsTimestampForUserMessage(t *testing.T) {
	model := modelWithTimestampedMessage("user", "Hello")

	view := stripANSI(model.View())

	if !strings.Contains(view, "14:23") {
		t.Error("View() should contain timestamp '14:23' for user message")
	}
}

func TestView_ShowsTimestampForAssistantMessage(t *testing.T) {
	model := modelWithTimestampedMessage("assistant", "Hi there!")

	view := stripANSI(model.View())

	if !strings.Contains(view, "14:23") {
		t.Error("View() should contain timestamp '14:23' for assistant message")
	}
}

func TestView_ShowsTimestampForErrorMessage(t *testing.T) {
	model := modelWithTimestampedMessage("error", "Something went wrong")

	view := stripANSI(model.View())

	if !strings.Contains(view, "14:23") {
		t.Error("View() should contain timestamp '14:23' for error message")
	}
}

func TestView_TimestampIsRightAligned(t *testing.T) {
	model := modelWithTimestampedMessage("user", "Hello")

	view := stripANSI(model.View())

	// Find the line containing the timestamp and verify it's in the right half.
	for _, line := range strings.Split(view, "\n") {
		idx := strings.Index(line, "14:23")
		if idx == -1 {
			continue
		}
		if idx < len(line)/2 {
			t.Errorf("Timestamp should be right-aligned: found at index %d in %d-char line %q", idx, len(line), line)
		}
		return
	}
	t.Error("No line containing timestamp '14:23' found in view")
}

func TestView_EachMessageHasOwnTimestamp(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 24)
	t1 := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 9, 5, 0, 0, time.UTC)
	m.messages = append(m.messages,
		Message{Role: "user", Content: "First", Timestamp: t1},
		Message{Role: "assistant", Content: "Second", Timestamp: t2},
	)

	view := stripANSI(m.View())

	if !strings.Contains(view, "09:00") {
		t.Error("View() should contain timestamp '09:00' for first message")
	}
	if !strings.Contains(view, "09:05") {
		t.Error("View() should contain timestamp '09:05' for second message")
	}
}
