// Package tui provides the Terminal User Interface for mogger
package tui

import "time"

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
