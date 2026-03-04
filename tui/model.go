package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"monkey/api"
)

// State represents the current UI state
type State int

const (
	StateReady State = iota
	StateLoading
)

// Model is the main bubbletea model for the TUI
type Model struct {
	messages       []Message
	input          textarea.Model
	viewport       viewport.Model
	state          State
	spinner        spinner.Model
	timer          timer.Model
	startTime      time.Time
	timerActive    bool
	lastElapsed    time.Duration
	client         *api.Client
	cancelFn       context.CancelFunc
	wasCancelled   bool
	retryAttempt   int
	width          int
	height         int
	err            error
	scrollToBottom bool
	retryCh        chan RetryingMsg
	toolCallCh     chan ToolCallMsg
	approvalCh     chan ToolApprovalRequestMsg
	intro          string
	introTitle     string
	introVersion   string
	filePicker     FilePicker
	commandPicker  CommandPicker
	modelPicker    ModelPicker
	helpPanel      HelpPanel
	approvalDialog ToolApprovalDialog
	models         []string
}

// NewModel creates a new TUI model with initialized components
func NewModel(client *api.Client) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+Enter to send, /exit to quit)"
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Focus()

	vp := viewport.New(80, 20)

	sp := spinner.New()
	sp.Spinner = spinner.Monkey
	t := timer.NewWithInterval(24*time.Hour, time.Second)

	return Model{
		client:         client,
		messages:       []Message{},
		input:          ta,
		viewport:       vp,
		spinner:        sp,
		timer:          t,
		state:          StateReady,
		width:          80,
		height:         24,
		scrollToBottom: true,
		filePicker:     NewFilePicker(80),
		commandPicker:  NewCommandPicker(80),
		modelPicker:    NewModelPicker(80),
		helpPanel:      NewHelpPanel(80),
		approvalDialog: NewToolApprovalDialog(80),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, LoadFilesCmd())
}

// messageStyleWidth returns the style parameter used for message bubbles.
// It caps the terminal width so text stays within ~75 chars for readability.
// Width(p-4) + Padding(0,1) = text area of p-6; at p=81 that is 75 chars.
func (m Model) messageStyleWidth() int {
	const maxStyleWidth = 126
	if m.width > maxStyleWidth {
		return maxStyleWidth
	}
	return m.width
}

// renderMessages returns the styled content string for all messages.
func (m Model) renderMessages() string {
	sw := m.messageStyleWidth()
	var sb strings.Builder
	for _, msg := range m.messages {
		var rendered string
		switch msg.Role {
		case "user":
			rendered = UserMessageStyle(sw).Render(msg.Content)
		case "assistant":
			md := strings.TrimRight(RenderMarkdown(msg.Content, sw-8), "\n")
			modelName := ""
			if m.client != nil {
				modelName = m.client.GetModel()
			}
			if modelName != "" {
				rendered = RenderAssistantBlock(sw, modelName, md)
			} else {
				rendered = AssistantMessageStyle(sw).Render(md)
			}
		case "tool":
			rendered = ToolMessageStyle(sw).Render(msg.Content)
		case "system":
			rendered = SystemMessageStyle(sw).Render(msg.Content)
		default:
			rendered = ErrorMessageStyle(sw).Render(msg.Content)
		}
		sb.WriteString(rendered)
		sb.WriteString("\n")
		ts := msg.Timestamp.Format("15:04")
		sb.WriteString(MessageTimestampStyle(sw).Render(ts))
		sb.WriteString("\n")
	}
	return sb.String()
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			// Esc dismisses pickers or cancels loading; Ctrl+C always quits.
			if msg.Type == tea.KeyEsc {
				if m.helpPanel.IsActive() {
					m.helpPanel.Deactivate()
					return m, nil
				}
				if m.modelPicker.IsActive() {
					m.modelPicker.Deactivate()
					return m, nil
				}
				if m.filePicker.IsActive() {
					m.filePicker.Deactivate()
					return m, nil
				}
				if m.commandPicker.IsActive() {
					m.commandPicker.Deactivate()
					return m, nil
				}
				if m.state == StateLoading {
					if m.approvalDialog.IsActive() {
						m.approvalDialog.Deny()
					}
					if m.cancelFn != nil {
						m.cancelFn()
						m.cancelFn = nil
					}
					m.state = StateReady
					m.timerActive = false
					m.wasCancelled = true
					return m, m.timer.Stop()
				}
				// Esc no longer quits; use /exit instead.
				return m, nil
			}
			// Ctrl+C: cancel loading or quit.
			if m.state == StateLoading {
				if m.approvalDialog.IsActive() {
					m.approvalDialog.Deny()
				}
				if m.cancelFn != nil {
					m.cancelFn()
					m.cancelFn = nil
				}
				m.state = StateReady
				m.timerActive = false
				m.wasCancelled = true
				return m, m.timer.Stop()
			}
			return m, tea.Quit
		case tea.KeyPgUp:
			m.scrollToBottom = false
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		case tea.KeyPgDown:
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		case tea.KeyUp:
			if m.approvalDialog.IsActive() {
				var adCmd tea.Cmd
				m.approvalDialog, adCmd = m.approvalDialog.Update(msg)
				cmds = append(cmds, adCmd)
			} else if m.modelPicker.IsActive() {
				var mpCmd tea.Cmd
				m.modelPicker, mpCmd = m.modelPicker.Update(msg)
				cmds = append(cmds, mpCmd)
			} else if m.filePicker.IsActive() {
				var fpCmd tea.Cmd
				m.filePicker, fpCmd = m.filePicker.Update(msg)
				cmds = append(cmds, fpCmd)
			} else if m.commandPicker.IsActive() {
				var cpCmd tea.Cmd
				m.commandPicker, cpCmd = m.commandPicker.Update(msg)
				cmds = append(cmds, cpCmd)
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyDown:
			if m.approvalDialog.IsActive() {
				var adCmd tea.Cmd
				m.approvalDialog, adCmd = m.approvalDialog.Update(msg)
				cmds = append(cmds, adCmd)
			} else if m.modelPicker.IsActive() {
				var mpCmd tea.Cmd
				m.modelPicker, mpCmd = m.modelPicker.Update(msg)
				cmds = append(cmds, mpCmd)
			} else if m.filePicker.IsActive() {
				var fpCmd tea.Cmd
				m.filePicker, fpCmd = m.filePicker.Update(msg)
				cmds = append(cmds, fpCmd)
			} else if m.commandPicker.IsActive() {
				var cpCmd tea.Cmd
				m.commandPicker, cpCmd = m.commandPicker.Update(msg)
				cmds = append(cmds, cpCmd)
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyTab:
			if m.modelPicker.IsActive() {
				if selected := m.modelPicker.SelectedModel(); selected != "" {
					m.applyModelSelection(selected)
				}
			} else if m.filePicker.IsActive() {
				if selected := m.filePicker.SelectedFile(); selected != "" {
					m.input.SetValue(replaceCurrentMention(m.input.Value(), selected))
					m.filePicker.Deactivate()
				}
			} else if m.commandPicker.IsActive() {
				if selected := m.commandPicker.SelectedCommand(); selected != "" {
					m.commandPicker.Deactivate()
					if selected == "/model" && len(m.models) > 0 {
						// Transition directly to model picker.
						m.input.SetValue("/model")
						m.modelPicker.SetModels(m.models)
						if m.client != nil {
							m.modelPicker.SetCursor(m.client.GetModel())
						}
						m.modelPicker.Activate()
					} else {
						m.input.SetValue(selected)
					}
				}
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyCtrlM:
			// Approval dialog takes priority when active.
			if m.approvalDialog.IsActive() {
				approved := m.approvalDialog.IsApproved()
				m.approvalDialog.Confirm()
				if !approved {
					// "No" behaves like Esc: cancel the in-flight request entirely.
					if m.cancelFn != nil {
						m.cancelFn()
						m.cancelFn = nil
					}
					m.state = StateReady
					m.timerActive = false
					m.wasCancelled = true
					return m, m.timer.Stop()
				}
				if m.approvalCh != nil {
					cmds = append(cmds, waitForApproval(m.approvalCh))
				}
				return m, tea.Batch(cmds...)
			}
			// Model picker selection takes priority when active.
			if m.modelPicker.IsActive() {
				if selected := m.modelPicker.SelectedModel(); selected != "" {
					m.applyModelSelection(selected)
				}
				return m, nil
			}
			// If command picker is active, Tab should select; Enter executes current input.
			inputVal := strings.TrimSpace(m.input.Value())
			if cmd, ok := parseSlashCommand(inputVal); ok {
				switch cmd {
				case "/exit":
					return m, tea.Quit
				case "/clear":
					m.messages = nil
					m.input.SetValue("")
					m.commandPicker.Deactivate()
					m.filePicker.Deactivate()
					return m, nil
				case "/model":
					m.input.SetValue("")
					m.commandPicker.Deactivate()
					m.modelPicker.SetModels(m.models)
					if m.client != nil {
						m.modelPicker.SetCursor(m.client.GetModel())
					}
					m.modelPicker.Activate()
					return m, nil
				}
			}
			if m.CanSubmit() {
				rawInput := m.input.Value()
				expandedInput := expandMentions(rawInput)
				// Show the original message in the UI (preserves @mentions)
				m.messages = append(m.messages, Message{Role: "user", Content: rawInput, Timestamp: time.Now()})
				m.input.SetValue("")
				m.filePicker.Deactivate()
				m.state = StateLoading
				m.scrollToBottom = true
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				// Start elapsed timer
				m.wasCancelled = false
				m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
				m.timerActive = true
				m.startTime = time.Now()
				retryCh := make(chan RetryingMsg, 10)
				m.retryCh = retryCh
				toolCallCh := make(chan ToolCallMsg, 10)
				m.toolCallCh = toolCallCh
				approvalCh := make(chan ToolApprovalRequestMsg, 1)
				m.approvalCh = approvalCh
				cmd, cancel := SendPromptCmdWithTimeout(m.client, m.messages, expandedInput, APITimeout, toolCallCh, approvalCh, retryCh)
				m.cancelFn = cancel
				cmds = append(cmds, cmd, m.spinner.Tick, m.timer.Init(), waitForRetry(retryCh), waitForToolCall(toolCallCh), waitForApproval(approvalCh))
			}
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
			// Sync command picker state (slash commands take priority over file picker).
			if cmdQuery, cpActive := detectCommandQuery(m.input.Value()); cpActive {
				if cmdQuery == "model" && len(m.models) > 0 {
					// Exact "/model" typed → show model picker inline (like @file picker).
					m.commandPicker.Deactivate()
					m.filePicker.Deactivate()
					m.modelPicker.SetModels(m.models)
					if m.client != nil {
						m.modelPicker.SetCursor(m.client.GetModel())
					}
					m.modelPicker.Activate()
				} else {
					m.commandPicker.Activate()
					m.commandPicker.SetQuery(cmdQuery)
					m.filePicker.Deactivate()
					m.modelPicker.Deactivate()
				}
			} else {
				m.commandPicker.Deactivate()
				m.modelPicker.Deactivate()
				if detectHelpQuery(m.input.Value()) {
					m.helpPanel.Activate()
					m.input.SetValue("")
					m.filePicker.Deactivate()
				} else {
					m.helpPanel.Deactivate()
					// Sync file picker state with the new input value.
					query, fpActive := detectMentionQuery(m.input.Value())
					if fpActive {
						m.filePicker.Activate()
						m.filePicker.SetQuery(query)
					} else {
						m.filePicker.Deactivate()
					}
				}
			}
		}

	case FilesLoadedMsg:
		m.filePicker.SetFiles(msg.Files)
		if m.filePicker.IsActive() {
			query, _ := detectMentionQuery(m.input.Value())
			m.filePicker.SetQuery(query)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
		m.filePicker.SetWidth(msg.Width)
		m.commandPicker.SetWidth(msg.Width)
		m.modelPicker.SetWidth(msg.Width)
		m.helpPanel.SetWidth(msg.Width)
		m.approvalDialog.SetWidth(msg.Width)
		vpHeight := msg.Height - 6
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = vpHeight
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			m.scrollToBottom = false
		}
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)

	case PromptResponseMsg:
		m.messages = append(m.messages, Message{Role: "assistant", Content: msg.Response, Timestamp: time.Now()})
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.retryAttempt = 0
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case PromptErrorMsg:
		m.messages = append(m.messages, Message{Role: "error", Content: msg.Err.Error(), Timestamp: time.Now()})
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.retryAttempt = 0
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case PromptCancelledMsg:
		if m.state == StateLoading {
			m.state = StateReady
			m.timerActive = false
			m.wasCancelled = true
			m.retryAttempt = 0
		}

	case RetryingMsg:
		m.retryAttempt = msg.Attempt
		if m.retryCh != nil {
			cmds = append(cmds, waitForRetry(m.retryCh))
		}

	case retryDoneMsg:
		// Retry channel closed; the API result will arrive separately.

	case ToolCallMsg:
		m.messages = append(m.messages, Message{
			Role:      "tool",
			Content:   formatToolCall(msg.ToolCall),
			Timestamp: time.Now(),
		})
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		if m.toolCallCh != nil {
			cmds = append(cmds, waitForToolCall(m.toolCallCh))
		}

	case toolCallDoneMsg:
		// Tool call channel closed; the API result will arrive separately.

	case ToolApprovalRequestMsg:
		preview := ""
		if msg.ToolName == "edit" {
			path, _ := msg.Input["path"].(string)
			oldStr, _ := msg.Input["old_string"].(string)
			newStr, _ := msg.Input["new_string"].(string)
			if diff, err := api.DiffEdit(path, oldStr, newStr); err == nil {
				preview = diff
			}
		}
		m.approvalDialog.Activate(msg.ModelName, msg.ToolName, preview, msg.ResponseCh)
		if m.approvalCh != nil {
			cmds = append(cmds, waitForApproval(m.approvalCh))
		}

	case toolApprovalDoneMsg:
		// Approval channel closed; the API result will arrive separately.

	case timer.TickMsg:
		if m.timerActive {
			var timerCmd tea.Cmd
			m.timer, timerCmd = m.timer.Update(msg)
			cmds = append(cmds, timerCmd)
		}

	case spinner.TickMsg:
		if m.state == StateLoading {
			var spinCmd tea.Cmd
			m.spinner, spinCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	// Sync viewport content (handles the AddMessage + View() direct path in tests).
	// This does not affect YOffset, preserving the user's scroll position.
	if len(m.messages) == 0 && m.intro != "" {
		m.viewport.SetContent(RenderIntroBlock(m.width, m.introTitle, m.introVersion, m.intro))
	} else {
		m.viewport.SetContent(m.renderMessages())
	}
	if m.scrollToBottom {
		m.viewport.GotoBottom()
	}

	var view strings.Builder
	view.WriteString(m.viewport.View())
	view.WriteString("\n")

	if m.modelPicker.IsActive() {
		view.WriteString(m.modelPicker.View())
		view.WriteString("\n")
	}
	if m.filePicker.IsActive() {
		view.WriteString(m.filePicker.View())
		view.WriteString("\n")
	}
	if m.commandPicker.IsActive() {
		view.WriteString(m.commandPicker.View())
		view.WriteString("\n")
	}

	if m.approvalDialog.IsActive() {
		view.WriteString(m.approvalDialog.View())
		view.WriteString("\n")
	}

	if m.state == StateLoading {
		line := SpinnerStyle().Render(m.spinner.View())
		if m.timerActive {
			elapsed := time.Since(m.startTime).Round(time.Second)
			line += " " + TimerStyle().Render(elapsed.String())
		}
		if m.retryAttempt > 0 {
			line += " " + TimerStyle().Render(fmt.Sprintf("retrying (%d)", m.retryAttempt))
		}
		view.WriteString(line)
		view.WriteString("\n")
	} else if m.wasCancelled {
		view.WriteString(TimerStyle().Render("canceled"))
		view.WriteString("\n")
	} else if m.lastElapsed > 0 {
		view.WriteString(TimerStyle().Render("took " + m.lastElapsed.Round(time.Second).String()))
		view.WriteString("\n")
	}

	// Render input area: use raw value + block cursor so tests can find the
	// text as a contiguous string while still providing a visible cursor.
	view.WriteString(InputStyle(m.width, 3).Render(m.input.Value() + "▌"))

	if m.helpPanel.IsActive() {
		view.WriteString("\n")
		view.WriteString(m.helpPanel.View())
	}

	return view.String()
}

// GetHistory returns the conversation history
func (m Model) GetHistory() []Message {
	return m.messages
}

// GetInput returns the current input text
func (m Model) GetInput() string {
	return m.input.Value()
}

// SetInput sets the input text (pointer receiver to mutate in place)
func (m *Model) SetInput(text string) {
	m.input.SetValue(text)
}

// ClearInput clears the input text (pointer receiver to mutate in place)
func (m *Model) ClearInput() {
	m.input.SetValue("")
}

// IsLoading returns whether the model is in loading state
func (m Model) IsLoading() bool {
	return m.state == StateLoading
}

// SetLoading sets the loading state (pointer receiver to mutate in place)
func (m *Model) SetLoading(loading bool) {
	if loading {
		m.state = StateLoading
	} else {
		m.state = StateReady
	}
}

// GetDimensions returns the current width and height
func (m Model) GetDimensions() (int, int) {
	return m.width, m.height
}

// SetDimensions sets the viewport and textarea dimensions (pointer receiver to mutate in place)
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(width - 4)
	m.filePicker.SetWidth(width)
	m.commandPicker.SetWidth(width)
	m.modelPicker.SetWidth(width)
	m.helpPanel.SetWidth(width)
	m.approvalDialog.SetWidth(width)
	vpHeight := height - 6
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.Width = width
	m.viewport.Height = vpHeight
}

// CanSubmit returns true when input is non-empty (trimmed) and not loading
func (m Model) CanSubmit() bool {
	return strings.TrimSpace(m.input.Value()) != "" && m.state != StateLoading
}

// IsTimerRunning returns whether the elapsed timer is active
func (m Model) IsTimerRunning() bool {
	return m.timerActive
}

// SetTimerActive sets the timer active state and records the start time when activating
func (m *Model) SetTimerActive(v bool) {
	m.timerActive = v
	if v {
		m.startTime = time.Now()
	}
}

// SetIntro sets the intro content (ASCII art + version) shown on startup before any messages.
func (m *Model) SetIntro(intro string) {
	m.intro = intro
}

// SetIntroTitle sets the title displayed in the intro block's border.
func (m *Model) SetIntroTitle(title string) {
	m.introTitle = title
}

// SetIntroVersion sets the version string displayed next to the title in the intro block's border.
func (m *Model) SetIntroVersion(version string) {
	m.introVersion = version
}

// SetModels sets the available models shown in the /model picker.
func (m *Model) SetModels(models []string) {
	m.models = models
	m.modelPicker.SetModels(models)
}

// applyModelSelection switches to the selected model and dismisses the picker.
func (m *Model) applyModelSelection(model string) {
	if m.client != nil {
		m.client.SetModel(model)
	}
	m.modelPicker.Deactivate()
	m.input.SetValue("")
	m.messages = append(m.messages, Message{Role: "system", Content: "model: " + model, Timestamp: time.Now()})
	m.scrollToBottom = true
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

// AddMessage appends a message to the conversation history (pointer receiver to mutate in place)
func (m *Model) AddMessage(role, content string) {
	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// formatToolCall formats an api.ToolCallResult for display in the conversation.
// For bash calls it shows "$ <command>\n<output>"; other tools show "<name>: <output>".
func formatToolCall(tc api.ToolCallResult) string {
	if cmd, ok := tc.Input["command"].(string); ok {
		content := "$ " + cmd
		if tc.Output != "" {
			content += "\n" + strings.TrimRight(tc.Output, "\n")
		}
		return content
	}
	return tc.Output
}
