package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
	"github.com/juanhuttemann/monkey-cli/tools"
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

// renderMessageEntry returns the fully-formatted string for messages[i],
// including a search-match prefix (when active) and a timestamp footer.
func (m Model) renderMessageEntry(sw, i int) string {
	msg := m.messages[i]
	rendered := m.renderSingleMessage(sw, msg)
	if m.searchBar.IsActive() && m.searchBar.IsMatch(i) {
		matchLabel := SearchMatchStyle().Render("▶ match")
		rendered = matchLabel + "\n" + rendered
	}
	var sb strings.Builder
	sb.WriteString(rendered)
	sb.WriteString("\n")
	sb.WriteString(MessageTimestampStyle(sw).Render(msg.Timestamp.Format("15:04")))
	sb.WriteString("\n")
	return sb.String()
}

// renderMessages returns the styled content string for all messages.
func (m Model) renderMessages() string {
	sw := m.messageStyleWidth()
	var sb strings.Builder
	for i := m.printedCount; i < len(m.messages); i++ {
		sb.WriteString(m.renderMessageEntry(sw, i))
	}
	return sb.String()
}

// commitUpTo prints messages[printedCount:n] to the terminal scrollback via
// tea.Println and advances printedCount to n. Returns nil if nothing to print.
func (m *Model) commitUpTo(n int) tea.Cmd {
	if n <= m.printedCount {
		return nil
	}
	sw := m.messageStyleWidth()
	var cmds []tea.Cmd
	for i := m.printedCount; i < n; i++ {
		msg := m.messages[i]
		rendered := m.renderSingleMessage(sw, msg)
		ts := msg.Timestamp.Format("15:04")
		text := rendered + "\n" + MessageTimestampStyle(sw).Render(ts)
		cmds = append(cmds, tea.Println(text))
	}
	m.printedCount = n
	return tea.Batch(cmds...)
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
				if m.searchBar.IsActive() {
					m.searchBar.Deactivate()
					m.viewport.SetContent(m.renderMessages())
					return m, nil
				}
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
					m.syncViewportHeight()
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
				m.syncViewportHeight()
				return m, m.timer.Stop()
			}
			return m, tea.Sequence(tea.ClearScreen, tea.Quit)
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
				if m.input.Line() == 0 {
					m.input.SetValue(m.promptHistory.Up(m.input.Value()))
					m.input.CursorEnd()
				} else {
					m.input.CursorUp()
				}
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
				if m.input.Line() == m.input.LineCount()-1 {
					m.input.SetValue(m.promptHistory.Down())
					m.input.CursorEnd()
				} else {
					m.input.CursorDown()
				}
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
		case tea.KeyCtrlT:
			// ctrl+t expands the most recently collapsed tool message.
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "tool" && m.messages[i].Collapsed {
					m.messages[i].Collapsed = false
					m.viewport.SetContent(m.renderMessages())
					break
				}
			}
			return m, nil

		case tea.KeyCtrlF:
			// Ctrl+F toggles the search bar.
			if m.searchBar.IsActive() {
				m.searchBar.Deactivate()
			} else {
				m.searchBar.Activate()
			}
			m.viewport.SetContent(m.renderMessages())
			return m, nil

		case tea.KeyCtrlN:
			// Ctrl+N advances to next search match.
			if m.searchBar.IsActive() {
				m.searchBar.NextMatch()
				m.scrollToMatch()
				return m, nil
			}

		case tea.KeyCtrlP:
			// Ctrl+P retreats to previous search match.
			if m.searchBar.IsActive() {
				m.searchBar.PrevMatch()
				m.scrollToMatch()
				return m, nil
			}

		case tea.KeyCtrlJ:
			// Ctrl+J inserts a newline into the input (multiline support).
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(tea.KeyMsg{Type: tea.KeyEnter})
			cmds = append(cmds, inputCmd)
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
					m.syncViewportHeight()
					return m, m.timer.Stop()
				}
				// Approved: dialog gone, viewport grows back.
				m.syncViewportHeight()
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
			// If command picker is active, Enter executes the highlighted command directly.
			if m.commandPicker.IsActive() {
				selected := m.commandPicker.SelectedCommand()
				m.commandPicker.Deactivate()
				switch selected {
				case "/exit":
					return m, tea.Sequence(tea.ClearScreen, tea.Quit)
				case "/clear":
					m.messages = nil
					m.apiMessages = nil
					m.totalUsage = api.Usage{}
					m.printedCount = 0
					m.input.SetValue("")
					return m, nil
				case "/model":
					m.input.SetValue("")
					m.modelPicker.SetModels(m.models)
					if m.client != nil {
						m.modelPicker.SetCursor(m.client.GetModel())
					}
					m.modelPicker.Activate()
					return m, nil
				case "/ape":
					m.apeMode = !m.apeMode
					m.input.SetValue("")
					return m, nil
				case "/copy":
					if text := m.lastAssistantContent(); text != "" {
						if err := clipboard.WriteAll(text); err != nil {
							m.messages = append(m.messages, Message{Role: "error", Content: "clipboard: " + err.Error(), Timestamp: time.Now()})
						}
					}
					m.input.SetValue("")
					return m, nil
				case "/compact":
					m.input.SetValue("")
					if cmd := m.startCompact(); cmd != nil {
						return m, cmd
					}
					return m, nil
				}
				return m, nil
			}
			inputVal := strings.TrimSpace(m.input.Value())
			if cmd, ok := parseSlashCommand(inputVal); ok {
				switch cmd {
				case "/exit":
					return m, tea.Sequence(tea.ClearScreen, tea.Quit)
				case "/clear":
					m.messages = nil
					m.apiMessages = nil
					m.totalUsage = api.Usage{}
					m.printedCount = 0
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
				case "/ape":
					m.apeMode = !m.apeMode
					m.input.SetValue("")
					m.commandPicker.Deactivate()
					return m, nil
				case "/copy":
					if text := m.lastAssistantContent(); text != "" {
						if err := clipboard.WriteAll(text); err != nil {
							m.messages = append(m.messages, Message{Role: "error", Content: "clipboard: " + err.Error(), Timestamp: time.Now()})
						}
					}
					m.input.SetValue("")
					m.commandPicker.Deactivate()
					return m, nil
				case "/compact":
					m.input.SetValue("")
					m.commandPicker.Deactivate()
					if cmd := m.startCompact(); cmd != nil {
						return m, cmd
					}
					return m, nil
				}
			}
			if m.CanSubmit() {
				rawInput := m.input.Value()
				expandedInput := expandMentions(rawInput)
				m.promptHistory.Add(rawInput)
				// Show the original message in the UI (preserves @mentions)
				m.messages = append(m.messages, Message{Role: "user", Content: rawInput, Timestamp: time.Now()})
				m.pendingPrompt = expandedInput
				if cmd := m.commitUpTo(len(m.messages)); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.input.SetValue("")
				m.filePicker.Deactivate()
				m.state = StateLoading
				m.scrollToBottom = true
				m.syncViewportHeight()
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
				var approvalCh chan ToolApprovalRequestMsg
				if !m.apeMode {
					approvalCh = make(chan ToolApprovalRequestMsg, 1)
				}
				m.approvalCh = approvalCh
				tokenCh := make(chan PartialResponseMsg, 64)
				m.tokenCh = tokenCh
				m.streaming = true
				cmd, cancel := SendPromptCmdWithTimeout(m.client, m.apiMessages, expandedInput, APITimeout, toolCallCh, approvalCh, tokenCh, retryCh)
				m.cancelFn = cancel
				cmds = append(cmds, cmd, m.spinner.Tick, m.timer.Init(), waitForRetry(retryCh), waitForToolCall(toolCallCh), waitForToken(tokenCh))
				if approvalCh != nil {
					cmds = append(cmds, waitForApproval(approvalCh))
				}
			}
		default:
			// When search is active, rune keys update the search query.
			if m.searchBar.IsActive() {
				if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
					switch msg.Type {
					case tea.KeyBackspace, tea.KeyDelete:
						q := m.searchBar.Query()
						if len(q) > 0 {
							m.searchBar.SetQuery(q[:len(q)-1], m.messages)
						}
					default:
						m.searchBar.SetQuery(m.searchBar.Query()+string(msg.Runes), m.messages)
					}
					m.viewport.SetContent(m.renderMessages())
					m.scrollToMatch()
					return m, nil
				}
			}
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
						wasActive := m.filePicker.IsActive()
						m.filePicker.Activate()
						m.filePicker.SetQuery(query)
						if !wasActive {
							// Rescan filesystem so files created after boot appear.
							cmds = append(cmds, LoadFilesCmd())
						}
					} else {
						m.filePicker.Deactivate()
					}
				}
			}
			m.syncViewportHeight()
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
		m.streaming = false
		m.tokenCh = nil
		m.messages = append(m.messages, Message{Role: "error", Content: msg.Err.Error(), Timestamp: time.Now()})
		m.state = StateReady
		m.lastElapsed = time.Since(m.startTime)
		m.timerActive = false
		m.wasCancelled = false
		m.retryAttempt = 0
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
			m.syncViewportHeight()
		}

	case RetryingMsg:
		m.retryAttempt = msg.Attempt
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
		preview := ""
		switch msg.ToolName {
		case "edit":
			path, _ := msg.Input["path"].(string)
			oldStr, _ := msg.Input["old_string"].(string)
			newStr, _ := msg.Input["new_string"].(string)
			if diff, err := tools.DiffEdit(path, oldStr, newStr); err == nil {
				preview = diff
			}
		case "bash":
			preview, _ = msg.Input["command"].(string)
		case "read":
			preview, _ = msg.Input["path"].(string)
		case "write":
			path, _ := msg.Input["path"].(string)
			content, _ := msg.Input["content"].(string)
			lines := strings.SplitN(content, "\n", 6)
			if len(lines) > 5 {
				lines = append(lines[:5], "...")
			}
			preview = path + "\n" + strings.Join(lines, "\n")
		case "glob":
			path, _ := msg.Input["path"].(string)
			pattern, _ := msg.Input["pattern"].(string)
			if path == "" {
				path = "."
			}
			preview = pattern + " in " + path
		case "grep":
			pattern, _ := msg.Input["pattern"].(string)
			path, _ := msg.Input["path"].(string)
			if path == "" {
				path = "."
			}
			preview = pattern + " in " + path
		}
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
			line += " " + TimerStyle().Render(fmt.Sprintf("retrying (%d)", m.retryAttempt))
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

// renderStatusBar renders a 1-line footer: model | ape | tokens.
func (m Model) renderStatusBar() string {
	sep := StatusBarSepStyle().Render(" | ")

	model := ""
	if m.client != nil {
		model = m.client.GetModel()
	}
	modelSeg := StatusBarModelStyle().Render(model)

	var apeSeg string
	if m.apeMode {
		apeSeg = ApeModeActiveStyle().Render("ape mode: on")
	} else {
		apeSeg = ApeModeInactiveStyle().Render("ape mode: off")
	}

	total := m.totalUsage.InputTokens + m.totalUsage.OutputTokens
	if total > 0 {
		tokenStr := fmt.Sprintf("%s tokens", formatTokenCount(total))
		if cost := formatCost(estimateCost(model, m.totalUsage)); cost != "" {
			tokenStr += "  " + cost
		}
		tokenSeg := StatusBarTokenStyle().Render(tokenStr)
		return modelSeg + sep + apeSeg + sep + tokenSeg
	}
	return modelSeg + sep + apeSeg
}

// formatTokenCount formats an integer token count with commas (e.g. 8341 → "8,341").
func formatTokenCount(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	// Insert commas every 3 digits from the right.
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
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

// formatToolCall formats an api.ToolCallResult for display in the conversation.
// For bash: "$ <command>\n<output>". For other tools: shows the key input + output.
func formatToolCall(tc api.ToolCallResult) string {
	if cmd, ok := tc.Input["command"].(string); ok {
		content := "$ " + cmd
		if tc.Output != "" {
			content += "\n" + strings.TrimRight(tc.Output, "\n")
		}
		return content
	}
	// For read/write/edit/glob/grep: prefix with the primary input parameter.
	var header string
	switch tc.Name {
	case "read", "write", "edit":
		if path, ok := tc.Input["path"].(string); ok && path != "" {
			header = path
		}
	case "glob":
		if pat, ok := tc.Input["pattern"].(string); ok && pat != "" {
			header = pat
			if p, ok := tc.Input["path"].(string); ok && p != "" {
				header += " in " + p
			}
		}
	case "grep":
		if pat, ok := tc.Input["pattern"].(string); ok && pat != "" {
			header = pat
			if p, ok := tc.Input["path"].(string); ok && p != "" {
				header += " in " + p
			}
		}
	}
	if header == "" {
		return tc.Output
	}
	if tc.Output == "" {
		return header
	}
	return header + "\n" + strings.TrimRight(tc.Output, "\n")
}

// inputWithCursor returns the input value with the ▌ cursor character inserted
// at the actual textarea cursor position (row/col), so the visual cursor tracks
// correctly when CursorUp/CursorDown moves within multiline input.
func (m Model) inputWithCursor() string {
	row := m.input.Line()
	info := m.input.LineInfo()
	col := info.StartColumn + info.ColumnOffset // raw rune index within row

	lines := strings.Split(m.input.Value(), "\n")
	if row < len(lines) {
		line := []rune(lines[row])
		if col > len(line) {
			col = len(line)
		}
		lines[row] = string(line[:col]) + "▌" + string(line[col:])
	}
	return strings.Join(lines, "\n")
}

// scrollToMatch sets the viewport Y offset so the current search match is visible.
// It estimates line positions by rendering each message in sequence.
func (m *Model) scrollToMatch() {
	idx := m.searchBar.CurrentMatchIndex()
	if idx < 0 {
		return
	}
	sw := m.messageStyleWidth()
	line := 0
	for i, msg := range m.messages {
		if i == idx {
			break
		}
		rendered := m.renderSingleMessage(sw, msg)
		line += strings.Count(rendered, "\n") + 2 // +1 timestamp line, +1 gap
	}
	m.viewport.SetYOffset(line)
}

// renderSingleMessage returns the styled string for one message (without timestamp).
func (m Model) renderSingleMessage(sw int, msg Message) string {
	switch msg.Role {
	case "user":
		return RenderUserBlock(sw, msg.Content)
	case "assistant":
		md := strings.TrimRight(RenderMarkdown(msg.Content, sw-8), "\n")
		modelName := ""
		if m.client != nil {
			modelName = m.client.GetModel()
		}
		if modelName != "" {
			return RenderAssistantBlock(sw, modelName, md)
		}
		return AssistantMessageStyle(sw).Render(md)
	case "tool":
		content := msg.Content
		if msg.Collapsed {
			lines := strings.Split(content, "\n")
			content = fmt.Sprintf("%s\n[%d lines hidden — ctrl+t to expand]", lines[0], len(lines)-1)
		}
		return RenderToolBlock(sw, msg.ToolName, content)
	case "system":
		return SystemMessageStyle(sw).Render(msg.Content)
	default:
		return ErrorMessageStyle(sw).Render(msg.Content)
	}
}
