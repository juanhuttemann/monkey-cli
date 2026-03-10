package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

// --- Prompt history (Up/Down arrow) ---

func TestUpdate_KeyUp_NavigatesHistoryBackward(t *testing.T) {
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("first", "second", "third")

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := model.Update(upKey)

	m := updated.(Model)
	if m.GetInput() != "third" {
		t.Errorf("after Up, input = %q, want %q", m.GetInput(), "third")
	}
}

func TestUpdate_KeyUp_ContinuesToOlderEntries(t *testing.T) {
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("first", "second", "third")

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	m1, _ := model.Update(upKey)
	m2, _ := m1.(Model).Update(upKey)

	if m2.(Model).GetInput() != "second" {
		t.Errorf("after two Up presses, input = %q, want %q", m2.(Model).GetInput(), "second")
	}
}

func TestUpdate_KeyDown_RestoresDraft(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("my draft")
	model.promptHistory = historyWithEntries("old entry")

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	downKey := tea.KeyMsg{Type: tea.KeyDown}

	m1, _ := model.Update(upKey)   // navigate to "old entry", draft saved
	m2, _ := m1.(Model).Update(downKey) // back down → restore draft

	if m2.(Model).GetInput() != "my draft" {
		t.Errorf("after Up then Down, input = %q, want %q", m2.(Model).GetInput(), "my draft")
	}
}

func TestUpdate_CtrlEnter_SavesPromptToHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("my prompt")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updated, _ := model.Update(ctrlEnter)

	m := updated.(Model)
	if len(m.promptHistory.entries) == 0 {
		t.Error("prompt should be saved to history after submit")
	}
	if m.promptHistory.entries[len(m.promptHistory.entries)-1] != "my prompt" {
		t.Errorf("last history entry = %q, want %q",
			m.promptHistory.entries[len(m.promptHistory.entries)-1], "my prompt")
	}
}

func TestUpdate_KeyUp_NoopWhenPickerActive(t *testing.T) {
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("old")
	model.filePicker.Activate()
	model.SetInput("")

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := model.Update(upKey)

	// input should NOT be changed to history entry while picker is open
	if updated.(Model).GetInput() == "old" {
		t.Error("Up should not navigate history when file picker is active")
	}
}

func TestUpdate_CtrlEnter_SubmitsPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("test prompt")

	// Simulate Ctrl+Enter key (using KeyCtrlM as it closest equivalent)
	// Note: In actual implementation, this may be handled differently
	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, cmd := model.Update(ctrlEnter)

	if cmd == nil {
		t.Error("Update(CtrlEnter) should return a non-nil command when input has content")
	}

	// Model should be in loading state
	m := updatedModel.(Model)
	if !m.IsLoading() {
		t.Error("Model should be in loading state after submitting prompt")
	}
}

func TestUpdate_CtrlEnter_IgnoresEmptyInput(t *testing.T) {
	model := NewModel(nil)
	// Input is empty by default

	// Simulate Ctrl+Enter key
	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, cmd := model.Update(ctrlEnter)

	if cmd != nil {
		t.Error("Update(CtrlEnter) should return nil command when input is empty")
	}

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("Model should not be in loading state with empty input")
	}
}

func TestUpdate_CtrlEnter_DisabledWhileLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("test prompt")
	model.SetLoading(true)

	// Simulate Ctrl+Enter key while already loading
	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, cmd := model.Update(ctrlEnter)

	if cmd != nil {
		t.Error("Update(CtrlEnter) should return nil command when already loading")
	}

	m := updatedModel.(Model)
	// History should not have been modified
	if len(m.GetHistory()) != 0 {
		t.Error("History should be empty when CtrlEnter is pressed during loading")
	}
}

func TestUpdate_MouseWheel_ScrollsViewport(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	// Add several messages to create scrollable content
	for i := 0; i < 20; i++ {
		model.AddMessage("user", "message "+string(rune('0'+i%10)))
	}

	// Simulate mouse wheel down
	mouseDown := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	updatedModel, _ := model.Update(mouseDown)

	m := updatedModel.(Model)
	// After scroll, the model should still have the same content
	if len(m.GetHistory()) != 20 {
		t.Errorf("History should still have 20 messages, got %d", len(m.GetHistory()))
	}

	// Simulate mouse wheel up
	mouseUp := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	updatedModel2, _ := m.Update(mouseUp)

	m2 := updatedModel2.(Model)
	if len(m2.GetHistory()) != 20 {
		t.Errorf("History should still have 20 messages after scroll up, got %d", len(m2.GetHistory()))
	}
}

func TestUpdate_WindowResize_AdjustsDimensions(t *testing.T) {
	model := NewModel(nil)

	// Simulate window resize
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(resizeMsg)

	m := updatedModel.(Model)
	width, height := m.GetDimensions()

	if width != 120 {
		t.Errorf("Width after resize = %d, want 120", width)
	}
	if height != 40 {
		t.Errorf("Height after resize = %d, want 40", height)
	}
}

func TestUpdate_CtrlEnter_AddsUserMessageImmediately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("hello there")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	history := m.GetHistory()

	// User message should already be in history — no need to wait for the response
	if len(history) != 1 {
		t.Fatalf("History length after submit = %d, want 1", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "user")
	}
	if history[0].Content != "hello there" {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, "hello there")
	}
}

func TestUpdate_CtrlEnter_ClearsInputImmediately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("hello there")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	// Input must be clear immediately — don't make the user wait for the response
	if m.GetInput() != "" {
		t.Errorf("GetInput() after submit = %q, want empty string", m.GetInput())
	}
}

func TestUpdate_PromptResponse_AddsAssistantMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "assistant response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("user prompt")

	// Submit: user message added immediately
	submitted, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := submitted.(Model)

	// Response arrives: assistant message added
	responseMsg := PromptResponseMsg{Response: "assistant response"}
	updatedModel, _ := m.Update(responseMsg)

	result := updatedModel.(Model)
	history := result.GetHistory()

	if len(history) != 2 {
		t.Fatalf("History length = %d, want 2 (user + assistant)", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "user prompt" {
		t.Errorf("history[0] = {%q, %q}, want {user, user prompt}", history[0].Role, history[0].Content)
	}
	if history[1].Role != "assistant" || history[1].Content != "assistant response" {
		t.Errorf("history[1] = {%q, %q}, want {assistant, assistant response}", history[1].Role, history[1].Content)
	}
}

func TestUpdate_PromptResponse_ClearsInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("user prompt")

	// Input should be empty immediately after submit (before any response)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := updatedModel.(Model)
	if m.GetInput() != "" {
		t.Errorf("Input after submit = %q, want empty string", m.GetInput())
	}
}

func TestUpdate_PromptResponse_SetsLoadingFalse(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("user prompt")
	model.SetLoading(true)

	// Simulate receiving a response
	responseMsg := PromptResponseMsg{Response: "assistant response"}
	updatedModel, _ := model.Update(responseMsg)

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("IsLoading() = true after receiving response, want false")
	}
}

func TestUpdate_PromptErrorMsg_AddsErrorMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("user prompt")

	// Simulate receiving an error
	testErr := &testError{msg: "API error occurred"}
	errorMsg := PromptErrorMsg{Err: testErr}
	updatedModel, _ := model.Update(errorMsg)

	m := updatedModel.(Model)
	history := m.GetHistory()

	// Should have user message and error message
	if len(history) < 1 {
		t.Fatal("History should contain at least one message after error")
	}

	// Find the error message
	var foundError bool
	for _, msg := range history {
		if msg.Role == "error" {
			foundError = true
			if msg.Content == "" {
				t.Error("Error message content should not be empty")
			}
		}
	}
	if !foundError {
		t.Error("History should contain an error message after PromptErrorMsg")
	}
}

func TestUpdate_PromptErrorMsg_SetsLoadingFalse(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)

	// Simulate receiving an error
	testErr := &testError{msg: "API error occurred"}
	errorMsg := PromptErrorMsg{Err: testErr}
	updatedModel, _ := model.Update(errorMsg)

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("IsLoading() = true after receiving error, want false")
	}
}

func TestUpdate_Esc_WhenReady_IsNoop(t *testing.T) {
	model := NewModel(nil)

	// Esc when StateReady is now a no-op (use /exit to quit)
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := model.Update(escKey)

	if cmd != nil {
		result := cmd()
		if _, isQuit := result.(tea.QuitMsg); isQuit {
			t.Error("Esc when StateReady should not quit — use /exit instead")
		}
	}
}

func TestUpdate_Esc_WhenLoading_GoesReady(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := model.Update(escKey)

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("Model should not be loading after Esc while loading")
	}

	// Should not quit — cmd may be non-nil (timer.Stop) but must not produce QuitMsg
	if cmd != nil {
		result := cmd()
		if _, isQuit := result.(tea.QuitMsg); isQuit {
			t.Error("Esc while loading should not quit the app")
		}
	}
}

func TestUpdate_PromptCancelled_WhenLoading_GoesReady(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	initialHistory := len(model.GetHistory())

	updatedModel, _ := model.Update(PromptCancelledMsg{})

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("Model should not be loading after PromptCancelledMsg")
	}
	if len(m.GetHistory()) != initialHistory {
		t.Errorf("PromptCancelledMsg should not add history entries: got %d, want %d",
			len(m.GetHistory()), initialHistory)
	}
}

func TestUpdate_PromptCancelled_WhenReady_IsNoOp(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	initialHistory := len(model.GetHistory())

	updatedModel, _ := model.Update(PromptCancelledMsg{})

	m := updatedModel.(Model)
	if m.IsLoading() {
		t.Error("Model should remain in StateReady after PromptCancelledMsg when already ready")
	}
	if len(m.GetHistory()) != initialHistory {
		t.Errorf("PromptCancelledMsg when ready should not change history: got %d, want %d",
			len(m.GetHistory()), initialHistory)
	}
}

func TestUpdate_CtrlC_Exits(t *testing.T) {
	model := NewModel(nil)

	// Simulate Ctrl+C key
	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(ctrlC)

	// Should return a quit command
	if cmd == nil {
		t.Error("Update(CtrlC) should return a non-nil command")
	}

	// cmd is non-nil: the quit sequence (ClearScreen + Quit) will be executed by the runtime.
}

func TestUpdate_SpinnerTick_StopsWhenNotLoading(t *testing.T) {
	model := NewModel(nil)
	// StateReady by default — spinner should not keep ticking

	_, cmd := model.Update(spinner.TickMsg{})

	if cmd != nil {
		t.Error("spinner.TickMsg in StateReady should return nil cmd (no more ticks)")
	}
}

func TestUpdate_PromptResponse_NoToolCallsWhenEmpty(t *testing.T) {
	model := NewModel(nil)

	responseMsg := PromptResponseMsg{Response: "plain answer"}
	updatedModel, _ := model.Update(responseMsg)

	m := updatedModel.(Model)
	history := m.GetHistory()

	if len(history) != 1 {
		t.Fatalf("expected 1 message (assistant only), got %d", len(history))
	}
	if history[0].Role != "assistant" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "assistant")
	}
}

func TestUpdate_ToolCallMsg_AddsToolMessage(t *testing.T) {
	model := NewModel(nil)

	tc := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "ls"}, Output: "file.txt\n"}
	updatedModel, _ := model.Update(ToolCallMsg{ToolCall: tc})

	m := updatedModel.(Model)
	history := m.GetHistory()

	if len(history) != 1 {
		t.Fatalf("expected 1 message after ToolCallMsg, got %d", len(history))
	}
	if history[0].Role != "tool" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "tool")
	}
}

func TestUpdate_ToolCallMsg_MessageContent(t *testing.T) {
	model := NewModel(nil)

	tc := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "echo hello"}, Output: "hello\n"}
	updatedModel, _ := model.Update(ToolCallMsg{ToolCall: tc})

	m := updatedModel.(Model)
	toolMsg := m.GetHistory()[0]

	if !strings.Contains(toolMsg.Content, "echo hello") {
		t.Errorf("tool message should contain command, got: %q", toolMsg.Content)
	}
	if !strings.Contains(toolMsg.Content, "hello") {
		t.Errorf("tool message should contain output, got: %q", toolMsg.Content)
	}
}

func TestUpdate_ToolCallMsg_AppearsBeforeAssistant(t *testing.T) {
	model := NewModel(nil)

	tc := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "ls"}, Output: "file.txt\n"}
	m1, _ := model.Update(ToolCallMsg{ToolCall: tc})
	m2, _ := m1.(Model).Update(PromptResponseMsg{Response: "Here is the result"})

	history := m2.(Model).GetHistory()
	if len(history) != 2 {
		t.Fatalf("expected 2 messages (tool + assistant), got %d", len(history))
	}
	if history[0].Role != "tool" {
		t.Errorf("history[0].Role = %q, want tool", history[0].Role)
	}
	if history[1].Role != "assistant" {
		t.Errorf("history[1].Role = %q, want assistant", history[1].Role)
	}
}

func TestUpdate_ToolCallMsg_MultipleToolCalls(t *testing.T) {
	model := NewModel(nil)

	tc1 := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "echo a"}, Output: "a\n"}
	tc2 := api.ToolCallResult{Name: "bash", Input: map[string]any{"command": "echo b"}, Output: "b\n"}
	m1, _ := model.Update(ToolCallMsg{ToolCall: tc1})
	m2, _ := m1.(Model).Update(ToolCallMsg{ToolCall: tc2})
	m3, _ := m2.(Model).Update(PromptResponseMsg{Response: "all done"})

	history := m3.(Model).GetHistory()
	if len(history) != 3 {
		t.Fatalf("expected 3 messages (tool + tool + assistant), got %d", len(history))
	}
	if history[0].Role != "tool" || history[1].Role != "tool" || history[2].Role != "assistant" {
		t.Errorf("expected [tool, tool, assistant], got [%q, %q, %q]",
			history[0].Role, history[1].Role, history[2].Role)
	}
}

func TestUpdate_SpinnerTick_ContinuesWhenLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)

	_, cmd := model.Update(spinner.TickMsg{})

	if cmd == nil {
		t.Error("spinner.TickMsg in StateLoading should return a non-nil cmd (keep ticking)")
	}
}

// --- Multiline Up/Down navigation ---

func TestUpdate_KeyUp_MultilineInput_MovesWithinTextFirst(t *testing.T) {
	// Cursor is on line 1 (last line) of a 2-line input.
	// Pressing Up should move cursor to line 0, NOT navigate history.
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("old entry")
	model.SetInput("line1\nline2") // cursor ends up on line 1

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := model.Update(upKey)

	m := updated.(Model)
	// Input must be unchanged — history navigation must NOT have triggered.
	if m.GetInput() != "line1\nline2" {
		t.Errorf("Up on line 1 changed input to %q; want %q (cursor should move within text, not navigate history)",
			m.GetInput(), "line1\nline2")
	}
	// Cursor should now be on line 0.
	if m.input.Line() != 0 {
		t.Errorf("cursor line after Up = %d, want 0", m.input.Line())
	}
}

func TestUpdate_KeyUp_MultilineInput_NavigatesHistoryFromFirstLine(t *testing.T) {
	// Cursor is already on line 0 of a 2-line input.
	// Pressing Up should navigate to the previous history entry.
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("old entry")
	model.SetInput("line1\nline2") // cursor on line 1

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	// First Up: move cursor to line 0.
	m1, _ := model.Update(upKey)
	// Second Up: now on line 0, navigate history.
	m2, _ := m1.(Model).Update(upKey)

	if m2.(Model).GetInput() != "old entry" {
		t.Errorf("second Up should navigate history; got %q, want %q",
			m2.(Model).GetInput(), "old entry")
	}
}

func TestUpdate_KeyDown_MultilineInput_MovesWithinTextFirst(t *testing.T) {
	// Cursor is on line 0 of a 2-line input.
	// Pressing Down should move cursor to line 1, NOT navigate history.
	model := NewModel(nil)
	model.promptHistory = historyWithEntries("old entry")
	model.SetInput("line1\nline2") // cursor on line 1 after SetInput

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	downKey := tea.KeyMsg{Type: tea.KeyDown}

	// Move cursor to line 0.
	m1, _ := model.Update(upKey)
	if m1.(Model).input.Line() != 0 {
		t.Skip("precondition failed: cursor not on line 0 after Up")
	}

	// Now press Down: should move to line 1, not navigate history.
	m2, _ := m1.(Model).Update(downKey)

	m := m2.(Model)
	if m.GetInput() != "line1\nline2" {
		t.Errorf("Down on line 0 of multiline changed input to %q; want %q",
			m.GetInput(), "line1\nline2")
	}
	if m.input.Line() != 1 {
		t.Errorf("cursor line after Down = %d, want 1", m.input.Line())
	}
}

// --- Ctrl+J inserts newline ---

func TestUpdate_CtrlJ_InsertsNewline(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("hello")

	ctrlJ := tea.KeyMsg{Type: tea.KeyCtrlJ}
	updated, _ := model.Update(ctrlJ)

	m := updated.(Model)
	if !strings.Contains(m.GetInput(), "\n") {
		t.Errorf("Ctrl+J should insert a newline, got input: %q", m.GetInput())
	}
}

func TestUpdate_CtrlJ_DoesNotSubmit(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("hello")

	ctrlJ := tea.KeyMsg{Type: tea.KeyCtrlJ}
	updated, _ := model.Update(ctrlJ)

	m := updated.(Model)
	if m.IsLoading() {
		t.Error("Ctrl+J should not submit the message")
	}
	if len(m.GetHistory()) != 0 {
		t.Error("Ctrl+J should not add to history")
	}
}

func TestUpdate_PromptResponse_AccumulatesTokens(t *testing.T) {
	model := NewModel(nil)
	model.totalUsage = api.Usage{InputTokens: 100, OutputTokens: 50}

	msg := PromptResponseMsg{Response: "hi", Usage: api.Usage{InputTokens: 200, OutputTokens: 30}}
	updated, _ := model.Update(msg)

	m := updated.(Model)
	if m.totalUsage.InputTokens != 300 {
		t.Errorf("totalUsage.InputTokens = %d, want 300", m.totalUsage.InputTokens)
	}
	if m.totalUsage.OutputTokens != 80 {
		t.Errorf("totalUsage.OutputTokens = %d, want 80", m.totalUsage.OutputTokens)
	}
}

func TestUpdate_Clear_ResetsTokens(t *testing.T) {
	model := NewModel(nil)
	model.totalUsage = api.Usage{InputTokens: 500, OutputTokens: 200}

	model.SetInput("/clear")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})

	m := updated.(Model)
	if m.totalUsage.InputTokens != 0 || m.totalUsage.OutputTokens != 0 {
		t.Errorf("totalUsage after /clear = {%d, %d}, want {0, 0}",
			m.totalUsage.InputTokens, m.totalUsage.OutputTokens)
	}
}

func TestUpdate_PromptResponse_StoresAPIMessages(t *testing.T) {
	model := NewModel(nil)
	prior := []api.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", Content: "a"},
	}
	responseMsg := PromptResponseMsg{Response: "reply", APIMessages: prior}
	updated, _ := model.Update(responseMsg)

	m := updated.(Model)
	if len(m.apiMessages) != 2 {
		t.Fatalf("apiMessages length = %d, want 2", len(m.apiMessages))
	}
	if m.apiMessages[1].Content != "a" {
		t.Errorf("apiMessages[1].Content = %v, want %q", m.apiMessages[1].Content, "a")
	}
}

func TestUpdate_Clear_ResetsAPIMessages(t *testing.T) {
	model := NewModel(nil)
	model.apiMessages = []api.Message{{Role: "user", Content: "prior"}}

	model.SetInput("/clear")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})

	m := updated.(Model)
	if len(m.apiMessages) != 0 {
		t.Errorf("apiMessages after /clear = %d entries, want 0", len(m.apiMessages))
	}
}

func TestNewModel_UsesMonkeySpinner(t *testing.T) {
	model := NewModel(nil)

	want := spinner.Monkey
	got := model.spinner.Spinner

	if got.FPS != want.FPS {
		t.Errorf("spinner FPS = %v, want %v (Monkey)", got.FPS, want.FPS)
	}
	if len(got.Frames) != len(want.Frames) {
		t.Errorf("spinner frames = %v, want %v (Monkey)", got.Frames, want.Frames)
	}
	for i, frame := range want.Frames {
		if got.Frames[i] != frame {
			t.Errorf("spinner.Frames[%d] = %q, want %q", i, got.Frames[i], frame)
		}
	}
}

// --- /compact command ---

func TestUpdate_SlashCompact_NoMessages_Noop(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/compact")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, cmd := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	// No messages to compact — should not start loading
	if m.IsLoading() {
		t.Error("IsLoading() = true after /compact with no messages, want false")
	}
	if cmd != nil {
		t.Error("cmd should be nil when there is nothing to compact")
	}
}

func TestUpdate_SlashCompact_WithMessages_StartsLoading(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content": [{"type": "text", "text": "summary"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.AddMessage("user", "hello")
	model.AddMessage("assistant", "hi there")
	model.SetInput("/compact")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, cmd := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if !m.IsLoading() {
		t.Error("IsLoading() = false after /compact with messages, want true")
	}
	if cmd == nil {
		t.Error("cmd should not be nil when compacting non-empty history")
	}
	if m.GetInput() != "" {
		t.Errorf("GetInput() = %q after /compact, want ''", m.GetInput())
	}
}

func TestUpdate_CompactResponseMsg_ReplacesHistory(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.AddMessage("assistant", "hi there")
	model.AddMessage("user", "how are you")
	model.SetLoading(true)

	compact := CompactResponseMsg{Summary: "User asked hello; assistant said hi; user asked how are you."}
	updatedModel, _ := model.Update(compact)

	m := updatedModel.(Model)
	history := m.GetHistory()
	if len(history) != 1 {
		t.Fatalf("History length after compact = %d, want 1", len(history))
	}
	if history[0].Role != "system" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "system")
	}
	if history[0].Content != compact.Summary {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, compact.Summary)
	}
	if m.IsLoading() {
		t.Error("IsLoading() = true after CompactResponseMsg, want false")
	}
}

// --- Streaming (PartialResponseMsg) ---

func TestUpdate_PartialResponseMsg_CreatesAssistantMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	model.streaming = true

	m1, _ := model.Update(PartialResponseMsg{Token: "Hello"})
	msgs := m1.(Model).GetHistory()
	if len(msgs) != 1 {
		t.Fatalf("history length = %d, want 1", len(msgs))
	}
	if msgs[0].Role != "assistant" {
		t.Errorf("role = %q, want assistant", msgs[0].Role)
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("content = %q, want Hello", msgs[0].Content)
	}
}

func TestUpdate_PartialResponseMsg_AppendsToLastAssistantMessage(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	model.streaming = true

	m1, _ := model.Update(PartialResponseMsg{Token: "foo"})
	m2, _ := m1.(Model).Update(PartialResponseMsg{Token: "bar"})
	msgs := m2.(Model).GetHistory()
	if len(msgs) != 1 {
		t.Fatalf("history length = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "foobar" {
		t.Errorf("content = %q, want foobar", msgs[0].Content)
	}
}

func TestUpdate_PartialResponseMsg_IgnoredWhenNotStreaming(t *testing.T) {
	// Late/stale tokens after streaming ends should be silently dropped.
	model := NewModel(nil)
	model.streaming = false

	m1, _ := model.Update(PartialResponseMsg{Token: "stale"})
	if len(m1.(Model).GetHistory()) != 0 {
		t.Error("late token should not add a message when streaming=false")
	}
}

func TestUpdate_PromptResponseMsg_ReplacesLastAssistantContent(t *testing.T) {
	// Simulates streaming: some tokens arrived, then PromptResponseMsg confirms the final text.
	model := NewModel(nil)
	model.SetLoading(true)
	model.streaming = true

	m1, _ := model.Update(PartialResponseMsg{Token: "partial"})
	finalMsg := PromptResponseMsg{
		Response:    "complete response",
		APIMessages: []api.Message{{Role: "user", Content: "q"}, {Role: "assistant", Content: "complete response"}},
	}
	m2, _ := m1.(Model).Update(finalMsg)

	m := m2.(Model)
	msgs := m.GetHistory()
	if len(msgs) != 1 {
		t.Fatalf("history length = %d, want 1 (partial replaced, not appended)", len(msgs))
	}
	if msgs[0].Content != "complete response" {
		t.Errorf("content = %q, want complete response", msgs[0].Content)
	}
	if m.streaming {
		t.Error("streaming should be false after PromptResponseMsg")
	}
}

func TestUpdate_PromptResponseMsg_AppendsIfNoStreamingMessage(t *testing.T) {
	// Non-streaming path: no tokens arrived, PromptResponseMsg appends normally.
	model := NewModel(nil)
	model.SetLoading(true)

	finalMsg := PromptResponseMsg{
		Response:    "answer",
		APIMessages: []api.Message{{Role: "user", Content: "q"}, {Role: "assistant", Content: "answer"}},
	}
	m1, _ := model.Update(finalMsg)

	msgs := m1.(Model).GetHistory()
	if len(msgs) != 1 || msgs[0].Content != "answer" {
		t.Errorf("expected one assistant message with 'answer', got %v", msgs)
	}
}

func TestUpdate_CompactResponseMsg_ResetsAPIMessages(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.SetLoading(true)
	// Simulate some API-layer history being present
	model.apiMessages = []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	compact := CompactResponseMsg{Summary: "Brief summary."}
	updatedModel, _ := model.Update(compact)

	m := updatedModel.(Model)
	// After compact, apiMessages should hold the summary as context for the next turn.
	// The first message should be the summary injected as an assistant context message.
	if len(m.apiMessages) == 0 {
		t.Fatal("apiMessages should not be empty after compact — summary must be retained as context")
	}
	found := false
	for _, msg := range m.apiMessages {
		if c, ok := msg.Content.(string); ok && strings.Contains(c, compact.Summary) {
			found = true
		}
	}
	if !found {
		t.Error("summary text should appear in apiMessages after compact")
	}
}
