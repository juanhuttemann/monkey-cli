package tui

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

// handleKeyMsg dispatches keyboard events and returns the updated model and command.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m.handleEscCtrlC(msg)

	case tea.KeyPgUp:
		m.scrollToBottom = false

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
					m.activateModelPicker()
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
			if m.messages[i].Role == roleTool && m.messages[i].Collapsed {
				m.messages[i].Collapsed = false
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
		return m, nil

	case tea.KeyCtrlN:
		// Ctrl+N advances to next search match.
		if m.searchBar.IsActive() {
			m.searchBar.NextMatch()
			return m, nil
		}

	case tea.KeyCtrlP:
		// Ctrl+P retreats to previous search match.
		if m.searchBar.IsActive() {
			m.searchBar.PrevMatch()
			return m, nil
		}

	case tea.KeyCtrlJ:
		// Ctrl+J inserts a newline into the input (multiline support).
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(tea.KeyMsg{Type: tea.KeyEnter})
		cmds = append(cmds, inputCmd)

	case tea.KeyCtrlM:
		return m.handleEnter()

	default:
		return m.handleDefaultKey(msg)
	}

	return m, tea.Batch(cmds...)
}

// handleEscCtrlC handles Esc (dismiss pickers / cancel loading) and
// Ctrl+C (cancel loading or quit).
func (m Model) handleEscCtrlC(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		if m.searchBar.IsActive() {
			m.searchBar.Deactivate()
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
			m.cancelLoading()
			return m, m.timer.Stop()
		}
		// Esc no longer quits; use /exit instead.
		return m, nil
	}
	// Ctrl+C: cancel loading or quit.
	if m.state == StateLoading {
		m.cancelLoading()
		return m, m.timer.Stop()
	}
	return m, tea.Sequence(tea.ClearScreen, tea.Quit)
}

// cancelLoading cancels the in-flight request and resets loading state.
func (m *Model) cancelLoading() {
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
}

// handleEnter handles the Enter key (Ctrl+M): confirmation dialog, model
// picker selection, command picker execution, slash commands, and prompt submission.
func (m Model) handleEnter() (Model, tea.Cmd) {
	var cmds []tea.Cmd

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

	// Command picker: Enter executes the highlighted command directly.
	if m.commandPicker.IsActive() {
		selected := m.commandPicker.SelectedCommand()
		m.commandPicker.Deactivate()
		if model, cmd, done := m.execSlashCommand(selected, false); done {
			return model, cmd
		}
		return m, nil
	}

	// Try inline slash command from the input field.
	inputVal := strings.TrimSpace(m.input.Value())
	if cmd, ok := parseSlashCommand(inputVal); ok {
		if model, teaCmd, done := m.execSlashCommand(cmd, true); done {
			return model, teaCmd
		}
	}

	// Regular prompt submission.
	if m.CanSubmit() {
		cmd := m.submitPrompt()
		return m, tea.Batch(append(cmds, cmd)...)
	}

	return m, tea.Batch(cmds...)
}

// activateModelPicker loads available models into the picker and activates it.
func (m *Model) activateModelPicker() {
	m.modelPicker.SetModels(m.models)
	if m.client != nil {
		m.modelPicker.SetCursor(m.client.GetModel())
	}
	m.modelPicker.Activate()
}

// execSlashCommand executes a recognised slash command.
// When cleanup is true (inline input), the command/file pickers are also dismissed.
// Returns (model, cmd, true) when the caller should return immediately.
func (m Model) execSlashCommand(cmd string, cleanup bool) (Model, tea.Cmd, bool) {
	var teaCmd tea.Cmd
	switch cmd {
	case "/exit":
		return m, tea.Sequence(tea.ClearScreen, tea.Quit), true
	case "/clear":
		m.messages = nil
		m.apiMessages = nil
		m.totalUsage = api.Usage{}
		m.printedCount = 0
		if cleanup {
			m.filePicker.Deactivate()
		}
	case "/model":
		m.activateModelPicker()
	case "/ape":
		m.autoApprove = !m.autoApprove
	case "/copy":
		if text := m.lastAssistantContent(); text != "" {
			if err := clipboard.WriteAll(text); err != nil {
				m.messages = append(m.messages, Message{Role: roleError, Content: "clipboard: " + err.Error(), Timestamp: time.Now()})
			}
		}
	case "/compact":
		teaCmd = m.startCompact()
	default:
		return m.handleDevSlashCommand(cmd, cleanup)
	}
	m.input.SetValue("")
	if cleanup {
		m.commandPicker.Deactivate()
	}
	return m, teaCmd, true
}

// submitPrompt builds the submission command for the current input value.
func (m *Model) submitPrompt() tea.Cmd {
	rawInput := m.input.Value()
	expandedInput := expandMentions(rawInput)
	m.promptHistory.Add(rawInput)
	// Show the original message in the UI (preserves @mentions).
	m.messages = append(m.messages, Message{Role: roleUser, Content: rawInput, Timestamp: time.Now()})
	m.pendingPrompt = expandedInput
	var cmds []tea.Cmd
	if cmd := m.commitUpTo(len(m.messages)); cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.input.SetValue("")
	m.filePicker.Deactivate()
	m.state = StateLoading
	m.scrollToBottom = true
	m.syncViewportHeight()
	// Start elapsed timer.
	m.wasCancelled = false
	m.timer = timer.NewWithInterval(24*time.Hour, time.Second)
	m.timerActive = true
	m.startTime = time.Now()
	retryCh := make(chan RetryingMsg, 10)
	m.retryCh = retryCh
	toolCallCh := make(chan ToolCallMsg, 10)
	m.toolCallCh = toolCallCh
	var approvalCh chan ToolApprovalRequestMsg
	if !m.autoApprove {
		approvalCh = make(chan ToolApprovalRequestMsg, 1)
	}
	m.approvalCh = approvalCh
	tokenCh := make(chan PartialResponseMsg, 64)
	m.tokenCh = tokenCh
	m.streaming = true
	cmd, cancel := SendPromptCmdWithTimeout(m.client, m.apiMessages, expandedInput, APITimeout, SendPromptOpts{ToolCallCh: toolCallCh, ApprovalCh: approvalCh, TokenCh: tokenCh, RetryCh: retryCh})
	m.cancelFn = cancel
	cmds = append(cmds, cmd, m.spinner.Tick, m.timer.Init(), waitForRetry(retryCh), waitForToolCall(toolCallCh), waitForToken(tokenCh))
	if approvalCh != nil {
		cmds = append(cmds, waitForApproval(approvalCh))
	}
	return tea.Batch(cmds...)
}

// handleDefaultKey processes regular character input, syncing pickers as needed.
func (m Model) handleDefaultKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

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
			m.activateModelPicker()
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
	return m, tea.Batch(cmds...)
}
