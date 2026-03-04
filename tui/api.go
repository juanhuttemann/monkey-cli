package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"monkey/api"
)

// ApprovingExecutor wraps a ToolExecutor and requests user approval before each tool call.
// It sends a ToolApprovalRequestMsg on approvalCh and blocks until the TUI responds.
type ApprovingExecutor struct {
	inner      api.ToolExecutor
	modelName  string
	approvalCh chan<- ToolApprovalRequestMsg
}

// ExecuteTool requests approval via approvalCh, then delegates to inner on approval.
func (a ApprovingExecutor) ExecuteTool(name string, input map[string]any) (string, error) {
	responseCh := make(chan bool, 1)
	a.approvalCh <- ToolApprovalRequestMsg{
		ModelName:  a.modelName,
		ToolName:   name,
		Input:      input,
		ResponseCh: responseCh,
	}
	approved := <-responseCh
	if !approved {
		return "", fmt.Errorf("tool call declined by user")
	}
	return a.inner.ExecuteTool(name, input)
}

// APITimeout is the default timeout for API requests
const APITimeout = 60 * time.Second

// SendPromptCmd creates a tea.Cmd that sends the conversation history plus
// the new prompt to the API using the default timeout.
// The returned CancelFunc can be called to cancel the in-flight request.
func SendPromptCmd(client *api.Client, messages []Message, prompt string) (tea.Cmd, context.CancelFunc) {
	return SendPromptCmdWithTimeout(client, messages, prompt, APITimeout, nil, nil)
}

// SendPromptCmdWithTimeout creates a tea.Cmd that sends the prompt with a per-attempt timeout.
// The returned CancelFunc can be called to cancel the in-flight request.
// toolCallCh, if non-nil, receives a ToolCallMsg for each tool call as it completes and is closed when done.
// approvalCh, if non-nil, enables the approval harness: each tool call sends a ToolApprovalRequestMsg
// and blocks until the TUI responds. When nil, tools execute without approval.
// An optional retryCh receives a RetryingMsg before each retry attempt and is closed when done.
func SendPromptCmdWithTimeout(client *api.Client, messages []Message, prompt string, timeout time.Duration, toolCallCh chan<- ToolCallMsg, approvalCh chan<- ToolApprovalRequestMsg, retryChs ...chan<- RetryingMsg) (tea.Cmd, context.CancelFunc) {
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
		if approvalCh != nil {
			defer close(approvalCh)
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

		// Use ApprovingExecutor when approvalCh is set, otherwise run tools directly.
		var executor api.ToolExecutor = api.BashExecutor{}
		if approvalCh != nil {
			modelName := ""
			if client != nil {
				modelName = client.GetModel()
			}
			executor = ApprovingExecutor{inner: api.BashExecutor{}, modelName: modelName, approvalCh: approvalCh}
		}

		response, err := client.SendMessageWithTools(ctx, apiMessages, []api.Tool{api.BashTool()}, executor,
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

// waitForApproval returns a tea.Cmd that blocks until it receives a ToolApprovalRequestMsg from ch,
// or returns toolApprovalDoneMsg when ch is closed.
func waitForApproval(ch <-chan ToolApprovalRequestMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return toolApprovalDoneMsg{}
		}
		return msg
	}
}
