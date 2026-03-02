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
func SendPromptCmd(client *api.Client, messages []Message, prompt string) tea.Cmd {
	return SendPromptCmdWithTimeout(client, messages, prompt, APITimeout)
}

// SendPromptCmdWithTimeout creates a tea.Cmd that sends the prompt with a custom timeout.
func SendPromptCmdWithTimeout(client *api.Client, messages []Message, prompt string, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Build full message list: history + new user prompt
		apiMessages := make([]api.Message, len(messages))
		for i, m := range messages {
			apiMessages[i] = api.Message{Role: m.Role, Content: m.Content}
		}
		apiMessages = append(apiMessages, api.Message{Role: "user", Content: prompt})

		response, err := client.SendMessageWithHistory(ctx, apiMessages)
		if err != nil {
			return PromptErrorMsg{Err: err}
		}
		return PromptResponseMsg{Response: response}
	}
}
