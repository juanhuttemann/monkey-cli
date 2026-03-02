package tui

import (
	"strings"
	"testing"
	"time"
)

func TestMessage_Role(t *testing.T) {
	msg := Message{
		Role:      "user",
		Content:   "test content",
		Timestamp: time.Now(),
	}

	if msg.Role != "user" {
		t.Errorf("Message.Role = %q, want %q", msg.Role, "user")
	}
}

func TestMessage_Content(t *testing.T) {
	msg := Message{
		Role:      "assistant",
		Content:   "test response content",
		Timestamp: time.Now(),
	}

	if msg.Content != "test response content" {
		t.Errorf("Message.Content = %q, want %q", msg.Content, "test response content")
	}
}

func TestMessage_Timestamp(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:      "user",
		Content:   "test",
		Timestamp: now,
	}

	if msg.Timestamp != now {
		t.Errorf("Message.Timestamp = %v, want %v", msg.Timestamp, now)
	}
}

func TestNewMessage(t *testing.T) {
	role := "user"
	content := "Hello, world!"

	msg := NewMessage(role, content)

	if msg.Role != role {
		t.Errorf("NewMessage().Role = %q, want %q", msg.Role, role)
	}
	if msg.Content != content {
		t.Errorf("NewMessage().Content = %q, want %q", msg.Content, content)
	}
	if msg.Timestamp.IsZero() {
		t.Error("NewMessage().Timestamp should be set, got zero value")
	}
}

func TestMsg_SendPromptMsg(t *testing.T) {
	msg := SendPromptMsg{Prompt: "test prompt"}

	if msg.Prompt != "test prompt" {
		t.Errorf("SendPromptMsg.Prompt = %q, want %q", msg.Prompt, "test prompt")
	}
}

func TestMsg_PromptResponseMsg(t *testing.T) {
	msg := PromptResponseMsg{Response: "test response", Err: nil}

	if msg.Response != "test response" {
		t.Errorf("PromptResponseMsg.Response = %q, want %q", msg.Response, "test response")
	}
	if msg.Err != nil {
		t.Errorf("PromptResponseMsg.Err = %v, want nil", msg.Err)
	}
}

func TestMsg_PromptResponseMsg_WithError(t *testing.T) {
	testErr := error(&testError{msg: "test error"})
	msg := PromptResponseMsg{Response: "", Err: testErr}

	if msg.Err == nil {
		t.Error("PromptResponseMsg.Err should not be nil")
	}
	if !strings.Contains(msg.Err.Error(), "test error") {
		t.Errorf("PromptResponseMsg.Err.Error() = %q, should contain %q", msg.Err.Error(), "test error")
	}
}

func TestMsg_PromptErrorMsg(t *testing.T) {
	testErr := error(&testError{msg: "API error"})
	msg := PromptErrorMsg{Err: testErr}

	if msg.Err == nil {
		t.Error("PromptErrorMsg.Err should not be nil")
	}
	if !strings.Contains(msg.Err.Error(), "API error") {
		t.Errorf("PromptErrorMsg.Err.Error() = %q, should contain %q", msg.Err.Error(), "API error")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
