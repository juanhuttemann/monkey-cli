// Package tui provides the Terminal User Interface for mogger
package tui

import (
	"time"

	"mogger/api"
)

// Message represents a single message in the conversation
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// NewMessage creates a new conversation message with the current timestamp
func NewMessage(role, content string) Message {
	return Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// SendPromptMsg is an internal bubbletea message type
type SendPromptMsg struct {
	Prompt string
}

// PromptResponseMsg is returned when the API responds successfully
type PromptResponseMsg struct {
	Response string
	Err      error
}

// PromptErrorMsg is returned when the API returns an error
type PromptErrorMsg struct {
	Err error
}

// PromptCancelledMsg is returned when the in-flight request is cancelled by the user
type PromptCancelledMsg struct{}

// RetryingMsg is sent to the model when the API client is about to retry a failed request.
type RetryingMsg struct {
	Attempt int
	Err     error
}

// retryDoneMsg is sent when the retry notification channel has been closed.
type retryDoneMsg struct{}

// ToolCallMsg is sent to the model as each tool call completes during the API loop.
type ToolCallMsg struct {
	ToolCall api.ToolCallResult
}

// toolCallDoneMsg is sent when the tool call channel has been closed.
type toolCallDoneMsg struct{}
