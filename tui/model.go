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
	intro          string
	introTitle     string
	introVersion   string
	filePicker     FilePicker
}

// NewModel creates a new TUI model with initialized components
func NewModel(client *api.Client) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+Enter to send, Esc to quit)"
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
			rendered = AssistantMessageStyle(sw).Render(md)
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
			// Esc first dismisses the file picker; Ctrl+C always quits.
			if msg.Type == tea.KeyEsc && m.filePicker.IsActive() {
				m.filePicker.Deactivate()
				return m, nil
			}
			if m.state == StateLoading {
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
			if m.filePicker.IsActive() {
				var fpCmd tea.Cmd
				m.filePicker, fpCmd = m.filePicker.Update(msg)
				cmds = append(cmds, fpCmd)
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyDown:
			if m.filePicker.IsActive() {
				var fpCmd tea.Cmd
				m.filePicker, fpCmd = m.filePicker.Update(msg)
				cmds = append(cmds, fpCmd)
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyTab:
			if m.filePicker.IsActive() {
				if selected := m.filePicker.SelectedFile(); selected != "" {
					m.input.SetValue(replaceCurrentMention(m.input.Value(), selected))
					m.filePicker.Deactivate()
				}
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		case tea.KeyCtrlM:
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
				cmd, cancel := SendPromptCmdWithTimeout(m.client, m.messages, expandedInput, APITimeout, toolCallCh, retryCh)
				m.cancelFn = cancel
				cmds = append(cmds, cmd, m.spinner.Tick, m.timer.Init(), waitForRetry(retryCh), waitForToolCall(toolCallCh))
			}
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
			// Sync picker state with the new input value
			query, fpActive := detectMentionQuery(m.input.Value())
			if fpActive {
				m.filePicker.Activate()
				m.filePicker.SetQuery(query)
			} else {
				m.filePicker.Deactivate()
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

	if m.filePicker.IsActive() {
		view.WriteString(m.filePicker.View())
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
