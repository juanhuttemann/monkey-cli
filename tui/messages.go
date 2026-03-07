// Package tui provides the Terminal User Interface for monkey
package tui

import (
	"time"

	"monkey/api"
)

// Message represents a single message in the conversation
type Message struct {
	Role      string
	Content   string
	ToolName  string // set for Role=="tool" messages
	Timestamp time.Time
	Collapsed bool // for tool messages: true when long output is folded
}

// PromptResponseMsg is returned when the API responds successfully
type PromptResponseMsg struct {
	Response    string
	APIMessages []api.Message // full accumulated history including tool_use/tool_result
	Usage       api.Usage     // token counts for this turn
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

// ToolApprovalRequestMsg is sent when the model wants to execute a tool and needs user approval.
type ToolApprovalRequestMsg struct {
	ModelName  string
	ToolName   string
	Input      map[string]any
	ResponseCh chan<- bool
}

// toolApprovalDoneMsg is sent when the approval channel is closed (API goroutine finished).
type toolApprovalDoneMsg struct{}

// CompactResponseMsg is returned when a /compact summarization request succeeds.
// Summary holds the condensed conversation text that replaces the message history.
type CompactResponseMsg struct {
	Summary string
}

// PartialResponseMsg carries a single streamed text token from the assistant.
type PartialResponseMsg struct {
	Token string
}

// tokenDoneMsg is sent when the token channel is closed (streaming finished).
type tokenDoneMsg struct{}
