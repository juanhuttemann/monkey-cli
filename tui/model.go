package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"mogger/api"
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
	width          int
	height         int
	err            error
	scrollToBottom bool
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
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// renderMessages returns the styled content string for all messages.
func (m Model) renderMessages() string {
	var sb strings.Builder
	for _, msg := range m.messages {
		var rendered string
		switch msg.Role {
		case "user":
			rendered = UserMessageStyle(m.width).Render(msg.Content)
		case "assistant":
			rendered = AssistantMessageStyle(m.width).Render(msg.Content)
		default:
			rendered = ErrorMessageStyle(m.width).Render(msg.Content)
		}
		sb.WriteString(rendered)
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
			return m, tea.Quit
		case tea.KeyCtrlM:
			if m.CanSubmit() {
				input := m.input.Value()
				// Show the user message and clear the textarea immediately
				m.messages = append(m.messages, Message{Role: "user", Content: input, Timestamp: time.Now()})
				m.input.SetValue("")
				m.state = StateLoading
				m.scrollToBottom = true
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				// Start elapsed timer
				m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
				m.timerActive = true
				m.startTime = time.Now()
				cmd := SendPromptCmd(m.client, m.messages, input)
				cmds = append(cmds, cmd, m.spinner.Tick, m.timer.Init())
			}
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
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
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case PromptErrorMsg:
		m.messages = append(m.messages, Message{Role: "error", Content: msg.Err.Error(), Timestamp: time.Now()})
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

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
	m.viewport.SetContent(m.renderMessages())
	if m.scrollToBottom {
		m.viewport.GotoBottom()
	}

	var view strings.Builder
	view.WriteString(m.viewport.View())
	view.WriteString("\n")

	if m.state == StateLoading {
		line := SpinnerStyle().Render(m.spinner.View())
		if m.timerActive {
			elapsed := time.Since(m.startTime).Round(time.Second)
			line += " " + TimerStyle().Render(elapsed.String())
		}
		view.WriteString(line)
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

// AddMessage appends a message to the conversation history (pointer receiver to mutate in place)
func (m *Model) AddMessage(role, content string) {
	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}
