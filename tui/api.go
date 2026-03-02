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
	return SendPromptCmdWithTimeout(client, messages, prompt, APITimeout)
}

// SendPromptCmdWithTimeout creates a tea.Cmd that sends the prompt with a custom timeout.
// The returned CancelFunc can be called to cancel the in-flight request.
func SendPromptCmdWithTimeout(client *api.Client, messages []Message, prompt string, timeout time.Duration) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	cmd := func() tea.Msg {
		defer cancel()

		// Build full message list: history (user/assistant only) + new user prompt
		var apiMessages []api.Message
		for _, m := range messages {
			if m.Role == "user" || m.Role == "assistant" {
				apiMessages = append(apiMessages, api.Message{Role: m.Role, Content: m.Content})
			}
		}
		apiMessages = append(apiMessages, api.Message{Role: "user", Content: prompt})

		var toolCalls []api.ToolCallResult
		response, err := client.SendMessageWithTools(ctx, apiMessages, []api.Tool{api.BashTool()}, api.BashExecutor{},
			func(tc api.ToolCallResult) {
				toolCalls = append(toolCalls, tc)
			},
		)
		if err != nil {
			if ctx.Err() == context.Canceled {
				return PromptCancelledMsg{}
			}
			return PromptErrorMsg{Err: err}
		}
		return PromptResponseMsg{Response: response, ToolCalls: toolCalls}
	}

	return cmd, cancel
}
