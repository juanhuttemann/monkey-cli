package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

// toolCollapseLines is the line count above which tool output is auto-collapsed.
const toolCollapseLines = 20

// State represents the current UI state
type State int

const (
	StateReady State = iota
	StateLoading
)

// Model is the main bubbletea model for the TUI
type Model struct {
	messages       []Message
	apiMessages    []api.Message // full API-layer history (includes tool_use/tool_result)
	totalUsage     api.Usage     // cumulative token counts for the session
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
	retryReason    string
	width          int
	height         int
	scrollToBottom bool
	retryCh        chan RetryingMsg
	toolCallCh     chan ToolCallMsg
	approvalCh     chan ToolApprovalRequestMsg
	tokenCh        chan PartialResponseMsg
	streaming      bool
	intro          string
	introTitle     string
	introVersion   string
	filePicker     FilePicker
	commandPicker  CommandPicker
	modelPicker    ModelPicker
	helpPanel      HelpPanel
	approvalDialog ToolApprovalDialog
	models         []string
	apeMode        bool
	pendingPrompt  string // the expanded prompt of the in-flight request; preserved into apiMessages on cancel
	promptHistory  History
	searchBar      SearchBar
	printedCount   int // number of messages already committed to terminal scrollback

	// Streaming render cache: renderedPrior holds the pre-rendered output for
	// all messages except the in-flight last one, computed once per streaming
	// session. streamBuf accumulates tokens without O(N²) string reallocations.
	// []byte is used instead of strings.Builder: Builder panics when copied
	// after first write, and Model is copied on every value-receiver Update call.
	streamBuf          []byte
	renderedPrior      string
	renderedPriorValid bool
}

// IsApeMode reports whether tool approval is disabled.
func (m Model) IsApeMode() bool { return m.apeMode }

// NewModel creates a new TUI model with initialized components
func NewModel(client *api.Client) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Ctrl+J for newline, /exit to quit)"
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
		promptHistory:  LoadHistory(),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.Sequence(tea.ClearScreen, tea.Println("")), textarea.Blink, LoadFilesCmd())
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

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
		m.viewport.Width = msg.Width
		m.syncViewportHeight()
		m.renderedPriorValid = false // width changed; prior-render cache is stale
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			m.scrollToBottom = false
		}
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)

	case PartialResponseMsg:
		if m.streaming {
			n := len(m.messages)
			if n > 0 && m.messages[n-1].Role == "assistant" {
				// Append to buf (O(1) amortised) instead of O(n) string concat.
				m.streamBuf = append(m.streamBuf, msg.Token...)
				m.messages[n-1].Content = string(m.streamBuf)
			} else {
				// First token: start a new assistant message and cache the rendered
				// output of all prior visible messages so we don't re-render them on
				// every subsequent token.
				m.streamBuf = m.streamBuf[:0]
				m.streamBuf = append(m.streamBuf, msg.Token...)
				m.messages = append(m.messages, Message{Role: "assistant", Content: msg.Token, Timestamp: time.Now()})
				n = len(m.messages)
				sw := m.messageStyleWidth()
				var prior strings.Builder
				for i := m.printedCount; i < n-1; i++ {
					prior.WriteString(m.renderMessageEntry(sw, i))
				}
				m.renderedPrior = prior.String()
				m.renderedPriorValid = true
			}
			m.scrollToBottom = true
			// Re-render only the last (in-flight) message; prior messages are cached.
			// Fall back to full render when search is active (match markers may change).
			if m.renderedPriorValid && !m.searchBar.IsActive() {
				sw := m.messageStyleWidth()
				n = len(m.messages)
				m.viewport.SetContent(m.renderedPrior + m.renderMessageEntry(sw, n-1))
			} else {
				m.viewport.SetContent(m.renderMessages())
			}
			m.viewport.GotoBottom()
		}
		if m.tokenCh != nil {
			cmds = append(cmds, waitForToken(m.tokenCh))
		}

	case tokenDoneMsg:
		// Token channel closed; the PromptResponseMsg will arrive separately.

	case PromptResponseMsg:
		m.streaming = false
		m.tokenCh = nil
		m.pendingPrompt = ""
		// Replace last assistant message with complete response (handles any race with PartialResponseMsg),
		// or append if streaming didn't produce one yet.
		n := len(m.messages)
		if n > 0 && m.messages[n-1].Role == "assistant" {
			m.messages[n-1].Content = msg.Response
		} else {
			m.messages = append(m.messages, Message{Role: "assistant", Content: msg.Response, Timestamp: time.Now()})
		}
		m.apiMessages = msg.APIMessages
		m.totalUsage = m.totalUsage.Add(msg.Usage)
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.retryAttempt = 0
		m.retryReason = ""
		m.streamBuf = m.streamBuf[:0]
		m.renderedPriorValid = false
		if cmd := m.commitUpTo(len(m.messages)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.scrollToBottom = true
		m.syncViewportHeight()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case CompactResponseMsg:
		// Replace entire message history with a single summary context message.
		m.messages = []Message{{Role: "system", Content: msg.Summary, Timestamp: time.Now()}}
		m.printedCount = 0
		// Seed apiMessages so the next turn has the summary as prior context.
		m.apiMessages = []api.Message{
			{Role: "user", Content: "Conversation context (summarized):\n" + msg.Summary},
			{Role: "assistant", Content: "Understood. I have the conversation context."},
		}
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.scrollToBottom = true
		m.syncViewportHeight()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case PromptErrorMsg:
		wasStreaming := m.streaming
		m.streaming = false
		m.tokenCh = nil
		// If we were streaming, remove any partial assistant message before appending the error.
		if wasStreaming {
			if n := len(m.messages); n > 0 && m.messages[n-1].Role == "assistant" {
				m.messages = m.messages[:n-1]
			}
			m.streamBuf = m.streamBuf[:0]
			m.renderedPriorValid = false
		}
		errContent := friendlyError(msg.Err)
		m.messages = append(m.messages, Message{Role: "error", Content: errContent, Timestamp: time.Now()})
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.retryAttempt = 0
		m.retryReason = ""
		if cmd := m.commitUpTo(len(m.messages)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.scrollToBottom = true
		m.syncViewportHeight()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case PromptCancelledMsg:
		m.streaming = false
		m.tokenCh = nil
		m.streamBuf = m.streamBuf[:0]
		m.renderedPriorValid = false
		// Preserve cancelled user message regardless of current state: ESC sets
		// state=Ready immediately, before this message arrives asynchronously.
		if m.pendingPrompt != "" {
			m.apiMessages = append(m.apiMessages, api.Message{Role: "user", Content: m.pendingPrompt})
			m.pendingPrompt = ""
		}
		if m.state == StateLoading {
			m.state = StateReady
			m.timerActive = false
			m.wasCancelled = true
			m.retryAttempt = 0
			m.retryReason = ""
			m.syncViewportHeight()
		}

	case RetryingMsg:
		m.retryAttempt = msg.Attempt
		m.retryReason = retryReasonFor(msg.Err)
		if m.retryCh != nil {
			cmds = append(cmds, waitForRetry(m.retryCh))
		}

	case retryDoneMsg:
		// Retry channel closed; the API result will arrive separately.

	case ToolCallMsg:
		if content := formatToolCall(msg.ToolCall); content != "" {
			collapsed := strings.Count(content, "\n") >= toolCollapseLines
			m.messages = append(m.messages, Message{
				Role:      "tool",
				Content:   content,
				ToolName:  msg.ToolCall.Name,
				Timestamp: time.Now(),
				Collapsed: collapsed,
			})
		}
		if cmd := m.commitUpTo(len(m.messages)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.scrollToBottom = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		if m.toolCallCh != nil {
			cmds = append(cmds, waitForToolCall(m.toolCallCh))
		}

	case toolCallDoneMsg:
		// Tool call channel closed; the API result will arrive separately.

	case ToolApprovalRequestMsg:
		// Ape mode may have been enabled after this request started (during streaming).
		// Auto-approve without showing the dialog.
		if m.apeMode {
			msg.ResponseCh <- true
			if m.approvalCh != nil {
				cmds = append(cmds, waitForApproval(m.approvalCh))
			}
			return m, tea.Batch(cmds...)
		}
		preview := buildApprovalPreview(msg.ToolName, msg.Input)
		m.approvalDialog.Activate(msg.ModelName, msg.ToolName, preview, msg.ResponseCh)
		m.syncViewportHeight()
		if m.approvalCh != nil {
			cmds = append(cmds, waitForApproval(m.approvalCh))
		}

	case toolApprovalDoneMsg:
		// Approval channel closed; the API result will arrive separately.
		// If the dialog is still active (context was cancelled without user
		// interaction), dismiss it now so the stale prompt doesn't linger.
		if m.approvalDialog.IsActive() {
			m.approvalDialog.Deny()
			m.syncViewportHeight()
		}

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
	var view strings.Builder

	// In inline (no alt-screen) mode render content directly so it only takes
	// as much vertical space as needed — no blank gap from a fixed-height viewport.
	if len(m.messages) == 0 && m.intro != "" {
		view.WriteString(RenderIntroBlock(m.width, m.introTitle, m.introVersion, m.intro))
	} else {
		view.WriteString(m.renderMessages())
	}
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
	} else if m.approvalDialog.IsDenied() {
		view.WriteString(m.approvalDialog.DeniedView())
		view.WriteString("\n")
	}

	if m.state == StateLoading {
		line := SpinnerStyle().Render(m.spinner.View())
		if m.timerActive {
			elapsed := time.Since(m.startTime).Round(time.Second)
			line += " " + TimerStyle().Render(elapsed.String())
		}
		if m.retryAttempt > 0 {
			line += " " + TimerStyle().Render(formatRetryLabel(m.retryAttempt, m.retryReason))
		}
		view.WriteString(line)
		view.WriteString("\n")
	} else if m.wasCancelled {
		view.WriteString(WaitingStyle().Render("What should monkey do?"))
		view.WriteString("\n")
	} else if m.lastElapsed > 0 {
		view.WriteString(TimerStyle().Render("took " + m.lastElapsed.Round(time.Second).String()))
		view.WriteString("\n")
	}

	// Render input area with ▌ at the actual cursor position so it tracks
	// correctly during multiline Up/Down navigation.
	view.WriteString(InputStyle(m.width, 3).Render(m.inputWithCursor()))
	view.WriteString("\n")
	view.WriteString(m.renderStatusBar())
	view.WriteString("\n\n")

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

// GetAPIMessages returns the full API-layer message history (includes tool_use/tool_result).
func (m Model) GetAPIMessages() []api.Message {
	return m.apiMessages
}

// RestoreSession loads a saved SessionData into the model.
func (m *Model) RestoreSession(sess *SessionData) {
	if sess == nil {
		return
	}
	m.messages = sess.Messages
	m.apiMessages = sess.APIMessages
	m.printedCount = len(m.messages)
	if sess.Model != "" && m.client != nil {
		m.client.SetModel(sess.Model)
	}
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
	m.syncViewportHeight()
}

// GetDimensions returns the current width and height
func (m Model) GetDimensions() (int, int) {
	return m.width, m.height
}

// syncViewportHeight recomputes and sets the viewport height based on current
// state so the input area stays at a fixed position on screen regardless of
// whether the status line or a dialog is visible.
//
// Reserved non-viewport rows (base, no status, no dialog):
//   - 1  \n separator between viewport and rest
//   - 5  input box (3 content + 2 border)
//   - 1  \n after input
//   - 1  ape-mode line
//   - 1  \n from \n\n trailing  (the second \n is below terminal bottom)
//
// = 9 rows.  Each optional element adds its own row count on top.
func (m *Model) syncViewportHeight() {
	reserved := 9

	// Status line: spinner, "What should monkey do?", or "took N s"
	if m.state == StateLoading || m.wasCancelled || m.lastElapsed > 0 {
		reserved++
	}

	// Pickers (at most one is active at a time)
	if m.commandPicker.IsActive() {
		reserved += m.commandPicker.Height()
	} else if m.filePicker.IsActive() {
		reserved += m.filePicker.Height()
	} else if m.modelPicker.IsActive() {
		reserved += m.modelPicker.Height()
	}

	// Approval / denied dialog
	if m.approvalDialog.IsActive() {
		reserved += m.approvalDialog.Height()
	} else if m.approvalDialog.IsDenied() {
		reserved += m.approvalDialog.DeniedHeight()
	}

	h := m.height - reserved
	if h < 1 {
		h = 1
	}
	m.viewport.Height = h
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
	m.viewport.Width = width
	m.syncViewportHeight()
}

// GetViewportHeight returns the current viewport height (for testing).
func (m Model) GetViewportHeight() int { return m.viewport.Height }

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

// startCompact initiates a /compact summarization request.
// Returns a tea.Cmd to run if there are messages to compact, nil otherwise.
func (m *Model) startCompact() tea.Cmd {
	if len(m.messages) == 0 {
		return nil
	}
	m.state = StateLoading
	m.startTime = time.Now()
	m.wasCancelled = false
	m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
	m.timerActive = true
	m.syncViewportHeight()
	return tea.Batch(SendCompactCmd(m.client, m.messages, APITimeout), m.spinner.Tick, m.timer.Init())
}

// lastAssistantContent returns the content of the most recent assistant message, or "".
func (m Model) lastAssistantContent() string {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "assistant" {
			return m.messages[i].Content
		}
	}
	return ""
}
