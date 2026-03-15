//go:build dev

package tui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

//go:embed mockup_scenarios.json
var mockupScenariosJSON []byte

// step is one action in a scenario script.
type step struct {
	Type    string         `json:"type"`    // stream | tool | error | retry
	Text    string         `json:"text"`    // stream / final text
	Name    string         `json:"name"`    // tool name
	Input   map[string]any `json:"input"`   // tool input
	Output  string         `json:"output"`  // tool fake output
	Status  int            `json:"status"`  // error HTTP status (0 = network error)
	Message string         `json:"message"` // error message
	Attempt int            `json:"attempt"` // retry attempt number
}

// mockupScenario is a named sequence of steps loaded from JSON.
type mockupScenario struct {
	Name  string `json:"name"`
	Steps []step `json:"steps"`
}

var mockupScenarios []mockupScenario

func init() {
	if err := json.Unmarshal(mockupScenariosJSON, &mockupScenarios); err != nil {
		panic("mockup_scenarios.json: " + err.Error())
	}
	slashCommands = append(slashCommands, SlashCommand{
		Name: "mockup",
		Desc: "run all dev mode scenarios in sequence (no API calls)",
	})
}

// handleDevSlashCommand handles /mockup in dev builds. A single invocation
// streams all scenarios in sequence so the full TUI can be evaluated at once.
func (m Model) handleDevSlashCommand(cmd string, cleanup bool) (Model, tea.Cmd, bool) {
	if cmd != "/mockup" {
		return m, nil, false
	}

	m.messages = append(m.messages, Message{Role: roleUser, Content: "/mockup", Timestamp: time.Now()})
	m.input.SetValue("")
	if cleanup {
		m.commandPicker.Deactivate()
		m.filePicker.Deactivate()
	}

	// Mirror submitPrompt's channel setup so all TUI machinery works natively.
	m.state = StateLoading
	m.scrollToBottom = true
	m.wasCancelled = false
	m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
	m.timerActive = true
	m.startTime = time.Now()
	m.streaming = true

	retryCh := make(chan RetryingMsg, 10)
	m.retryCh = retryCh
	toolCallCh := make(chan ToolCallMsg, 10)
	m.toolCallCh = toolCallCh
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	m.approvalCh = approvalCh
	tokenCh := make(chan PartialResponseMsg, 64)
	m.tokenCh = tokenCh
	m.syncViewportHeight()

	parentCtx, parentCancel := context.WithCancel(context.Background())
	m.cancelFn = parentCancel

	teaCmd := playAllScenarios(parentCtx, parentCancel, tokenCh, approvalCh, toolCallCh, retryCh)

	return m, tea.Batch(
		teaCmd,
		m.spinner.Tick,
		m.timer.Init(),
		waitForRetry(retryCh),
		waitForToolCall(toolCallCh),
		waitForToken(tokenCh),
		waitForApproval(approvalCh),
	), true
}

// setupMockupModel configures m to play a single scenario at idx and returns
// the ready model with its initial tea.Cmd batch. Used by tests.
func setupMockupModel(m Model, idx int) (Model, tea.Cmd) {
	sc := mockupScenarios[idx]
	m.messages = append(m.messages, Message{Role: roleUser, Content: "/mockup", Timestamp: time.Now()})
	m.state = StateLoading
	m.scrollToBottom = true
	m.wasCancelled = false
	m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
	m.timerActive = true
	m.startTime = time.Now()
	m.streaming = true

	retryCh := make(chan RetryingMsg, 10)
	m.retryCh = retryCh
	toolCallCh := make(chan ToolCallMsg, 10)
	m.toolCallCh = toolCallCh
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	m.approvalCh = approvalCh
	tokenCh := make(chan PartialResponseMsg, 64)
	m.tokenCh = tokenCh
	m.syncViewportHeight()

	parentCtx, parentCancel := context.WithCancel(context.Background())
	m.cancelFn = parentCancel

	teaCmd := playScenario(parentCtx, parentCancel, sc, tokenCh, approvalCh, toolCallCh, retryCh)
	return m, tea.Batch(
		teaCmd,
		m.spinner.Tick,
		m.timer.Init(),
		waitForRetry(retryCh),
		waitForToolCall(toolCallCh),
		waitForToken(tokenCh),
		waitForApproval(approvalCh),
	)
}

// playScenario returns a tea.Cmd that executes each step in the scenario.
func playScenario(
	ctx context.Context,
	cancel context.CancelFunc,
	sc mockupScenario,
	tokenCh chan PartialResponseMsg,
	approvalCh chan ToolApprovalRequestMsg,
	toolCallCh chan ToolCallMsg,
	retryCh chan RetryingMsg,
) tea.Cmd {
	return func() tea.Msg {
		defer cancel()
		defer close(tokenCh)
		defer close(approvalCh)
		defer close(toolCallCh)
		defer close(retryCh)

		var finalText string

		for _, s := range sc.Steps {
			select {
			case <-ctx.Done():
				return PromptCancelledMsg{}
			default:
			}

			switch s.Type {

			case "stream":
				for _, word := range strings.Fields(s.Text) {
					select {
					case tokenCh <- PartialResponseMsg{Token: word + " "}:
						time.Sleep(30 * time.Millisecond)
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
				}
				finalText = s.Text

			case "tool":
				responseCh := make(chan bool, 1)
				select {
				case approvalCh <- ToolApprovalRequestMsg{
					ModelName:  "dev-mockup",
					ToolName:   s.Name,
					Input:      s.Input,
					ResponseCh: responseCh,
				}:
				case <-ctx.Done():
					return PromptCancelledMsg{}
				}
				var approved bool
				select {
				case approved = <-responseCh:
				case <-ctx.Done():
					return PromptCancelledMsg{}
				}
				if !approved {
					return PromptCancelledMsg{}
				}
				select {
				case toolCallCh <- ToolCallMsg{ToolCall: api.ToolCallResult{
					Name:   s.Name,
					Input:  s.Input,
					Output: s.Output,
				}}:
				case <-ctx.Done():
					return PromptCancelledMsg{}
				}

			case "error":
				if s.Status == 0 {
					return PromptErrorMsg{Err: fmt.Errorf("%s", s.Message)}
				}
				body := fmt.Sprintf(`{"error":{"message":%q}}`, s.Message)
				return PromptErrorMsg{Err: &api.StatusError{StatusCode: s.Status, Body: body}}

			case "retry":
				select {
				case retryCh <- RetryingMsg{Attempt: s.Attempt, Err: &api.StatusError{StatusCode: s.Status}}:
				case <-ctx.Done():
					return PromptCancelledMsg{}
				}
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
					return PromptCancelledMsg{}
				}
			}
		}

		return PromptResponseMsg{
			Response: finalText,
			Usage:    api.Usage{InputTokens: 128, OutputTokens: len(strings.Fields(finalText))},
		}
	}
}

// playAllScenarios runs every scenario in sequence as one continuous streaming
// response. Error scenarios are rendered inline so the run continues uninterrupted;
// retry and tool scenarios use the authentic TUI channels.
func playAllScenarios(
	ctx context.Context,
	cancel context.CancelFunc,
	tokenCh chan PartialResponseMsg,
	approvalCh chan ToolApprovalRequestMsg,
	toolCallCh chan ToolCallMsg,
	retryCh chan RetryingMsg,
) tea.Cmd {
	return func() tea.Msg {
		defer cancel()
		defer close(tokenCh)
		defer close(approvalCh)
		defer close(toolCallCh)
		defer close(retryCh)

		send := func(token string) bool {
			select {
			case tokenCh <- PartialResponseMsg{Token: token}:
				return true
			case <-ctx.Done():
				return false
			}
		}

		// Simulate context growth: input tokens accumulate across scenarios
		// just like a real multi-turn conversation.
		var lastText string
		contextTokens := 128 // base system prompt
		var totalOutputTokens int
		for i, sc := range mockupScenarios {
			select {
			case <-ctx.Done():
				return PromptCancelledMsg{}
			default:
			}

			// Scenario header — sent as a single token to preserve newlines.
			header := fmt.Sprintf("**[%d/%d] %s**\n\n", i+1, len(mockupScenarios), sc.Name)
			if i > 0 {
				header = "\n\n---\n\n" + header
			}
			if !send(header) {
				return PromptCancelledMsg{}
			}
			time.Sleep(50 * time.Millisecond)

			for _, s := range sc.Steps {
				select {
				case <-ctx.Done():
					return PromptCancelledMsg{}
				default:
				}

				switch s.Type {
				case "stream":
					words := strings.Fields(s.Text)
					for _, word := range words {
						select {
						case tokenCh <- PartialResponseMsg{Token: word + " "}:
							time.Sleep(30 * time.Millisecond)
						case <-ctx.Done():
							return PromptCancelledMsg{}
						}
					}
					totalOutputTokens += len(words)
					lastText = s.Text

				case "tool":
					responseCh := make(chan bool, 1)
					select {
					case approvalCh <- ToolApprovalRequestMsg{
						ModelName:  "dev-mockup",
						ToolName:   s.Name,
						Input:      s.Input,
						ResponseCh: responseCh,
					}:
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
					var approved bool
					select {
					case approved = <-responseCh:
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
					if !approved {
						if !send("[tool denied]\n\n") {
							return PromptCancelledMsg{}
						}
						continue
					}
					select {
					case toolCallCh <- ToolCallMsg{ToolCall: api.ToolCallResult{
						Name:   s.Name,
						Input:  s.Input,
						Output: s.Output,
					}}:
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}

				case "error":
					// Show retry spinner for 1 second before displaying the inline error.
					var retryErr error
					if s.Status != 0 {
						retryErr = &api.StatusError{StatusCode: s.Status}
					}
					select {
					case retryCh <- RetryingMsg{Attempt: 1, Err: retryErr}:
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
					select {
					case <-time.After(1 * time.Second):
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
					// Inline error text keeps all scenarios running.
					var errText string
					if s.Status == 0 {
						errText = fmt.Sprintf("Network Error: %s\n\n", s.Message)
					} else {
						errText = fmt.Sprintf("HTTP %d Error: %s\n\n", s.Status, s.Message)
					}
					if !send(errText) {
						return PromptCancelledMsg{}
					}
					time.Sleep(50 * time.Millisecond)

				case "retry":
					select {
					case retryCh <- RetryingMsg{Attempt: s.Attempt, Err: &api.StatusError{StatusCode: s.Status}}:
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
					select {
					case <-time.After(2 * time.Second):
					case <-ctx.Done():
						return PromptCancelledMsg{}
					}
				}
			}
			// Each scenario adds ~150 tokens to the simulated context window,
			// mirroring how real input tokens grow with conversation history.
			contextTokens += 150
		}

		return PromptResponseMsg{
			Response: lastText,
			Usage: api.Usage{
				InputTokens:  contextTokens,
				OutputTokens: totalOutputTokens,
			},
		}
	}
}
