package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"mogger/api"
)

// APITimeout is the default timeout for API requests
const APITimeout = 60 * time.Second

// SendPromptCmd creates a tea.Cmd that sends the conversation history plus
// the new prompt to the API using the default timeout.
// The returned CancelFunc can be called to cancel the in-flight request.
func SendPromptCmd(client *api.Client, messages []Message, prompt string) (tea.Cmd, context.CancelFunc) {
	return SendPromptCmdWithTimeout(client, messages, prompt, APITimeout, nil)
}

// SendPromptCmdWithTimeout creates a tea.Cmd that sends the prompt with a per-attempt timeout.
// The returned CancelFunc can be called to cancel the in-flight request.
// toolCallCh, if non-nil, receives a ToolCallMsg for each tool call as it completes and is closed when done.
// An optional retryCh receives a RetryingMsg before each retry attempt and is closed when done.
func SendPromptCmdWithTimeout(client *api.Client, messages []Message, prompt string, timeout time.Duration, toolCallCh chan<- ToolCallMsg, retryChs ...chan<- RetryingMsg) (tea.Cmd, context.CancelFunc) {
	// Use a cancel-only parent so that each retry gets a fresh per-attempt timeout
	// rather than sharing an already-expired deadline.
	parentCtx, parentCancel := context.WithCancel(context.Background())
	ctx := api.WithPerAttemptTimeout(parentCtx, timeout)

	var retryCh chan<- RetryingMsg
	if len(retryChs) > 0 {
		retryCh = retryChs[0]
	}

	cmd := func() tea.Msg {
		defer parentCancel()
		if retryCh != nil {
			defer close(retryCh)
		}
		if toolCallCh != nil {
			defer close(toolCallCh)
		}

		// Build full message list: history (user/assistant only) + new user prompt
		var apiMessages []api.Message
		for _, m := range messages {
			if m.Role == "user" || m.Role == "assistant" {
				apiMessages = append(apiMessages, api.Message{Role: m.Role, Content: m.Content})
			}
		}
		apiMessages = append(apiMessages, api.Message{Role: "user", Content: prompt})

		if retryCh != nil {
			ctx = api.WithRetryNotifier(ctx, func(attempt int, err error) {
				select {
				case retryCh <- RetryingMsg{Attempt: attempt, Err: err}:
				default:
				}
			})
		}

		response, err := client.SendMessageWithTools(ctx, apiMessages, []api.Tool{api.BashTool()}, api.BashExecutor{},
			func(tc api.ToolCallResult) {
				if toolCallCh != nil {
					toolCallCh <- ToolCallMsg{ToolCall: tc}
				}
			},
		)
		if err != nil {
			if parentCtx.Err() == context.Canceled {
				return PromptCancelledMsg{}
			}
			return PromptErrorMsg{Err: err}
		}
		return PromptResponseMsg{Response: response}
	}

	return cmd, parentCancel
}

// waitForToolCall returns a tea.Cmd that blocks until it receives a ToolCallMsg from ch,
// or returns toolCallDoneMsg when ch is closed.
func waitForToolCall(ch <-chan ToolCallMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return toolCallDoneMsg{}
		}
		return msg
	}
}

// waitForRetry returns a tea.Cmd that blocks until it receives a RetryingMsg from ch,
// or returns retryDoneMsg when ch is closed.
func waitForRetry(ch <-chan RetryingMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return retryDoneMsg{}
		}
		return msg
	}
}
