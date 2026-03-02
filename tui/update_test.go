package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"mogger/api"
)

func TestUpdate_CtrlEnter_SubmitsPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
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
	mouseDown := tea.MouseMsg{Type: tea.MouseWheelDown}
	updatedModel, _ := model.Update(mouseDown)

	m := updatedModel.(Model)
	// After scroll, the model should still have the same content
	if len(m.GetHistory()) != 20 {
		t.Errorf("History should still have 20 messages, got %d", len(m.GetHistory()))
	}

	// Simulate mouse wheel up
	mouseUp := tea.MouseMsg{Type: tea.MouseWheelUp}
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
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
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
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
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
		w.Write([]byte(`{"content": [{"type": "text", "text": "assistant response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("user prompt")

	// Submit: user message added immediately
	submitted, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := submitted.(Model)

	// Response arrives: assistant message added
	responseMsg := PromptResponseMsg{Response: "assistant response", Err: nil}
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
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
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
	responseMsg := PromptResponseMsg{Response: "assistant response", Err: nil}
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

func TestUpdate_Esc_Exits(t *testing.T) {
	model := NewModel(nil)

	// Simulate Escape key
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := model.Update(escKey)

	// Should return a quit command
	if cmd == nil {
		t.Error("Update(Esc) should return a non-nil command")
	}

	// The quit command should produce a tea.QuitMsg
	quitMsg := cmd()
	if quitMsg != tea.Quit() {
		t.Error("Esc key should trigger a quit command")
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

	// The quit command should produce a tea.QuitMsg
	quitMsg := cmd()
	if quitMsg != tea.Quit() {
		t.Error("CtrlC key should trigger a quit command")
	}
}

func TestUpdate_SpinnerTick_StopsWhenNotLoading(t *testing.T) {
	model := NewModel(nil)
	// StateReady by default — spinner should not keep ticking

	_, cmd := model.Update(spinner.TickMsg{})

	if cmd != nil {
		t.Error("spinner.TickMsg in StateReady should return nil cmd (no more ticks)")
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
