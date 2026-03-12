package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel_InitializesEmptyHistory(t *testing.T) {
	model := NewModel(nil)

	if len(model.GetHistory()) != 0 {
		t.Errorf("NewModel().GetHistory() length = %d, want 0", len(model.GetHistory()))
	}
}

func TestNewModel_InitializesEmptyInput(t *testing.T) {
	model := NewModel(nil)

	if model.GetInput() != "" {
		t.Errorf("NewModel().GetInput() = %q, want empty string", model.GetInput())
	}
}

func TestNewModel_NotLoading(t *testing.T) {
	model := NewModel(nil)

	if model.IsLoading() {
		t.Error("NewModel().IsLoading() = true, want false")
	}
}

func TestModel_AddUserMessage(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "Hello, world!")

	history := model.GetHistory()
	if len(history) != 1 {
		t.Fatalf("GetHistory() length = %d, want 1", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "user")
	}
	if history[0].Content != "Hello, world!" {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, "Hello, world!")
	}
}

func TestModel_AddAssistantMessage(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("assistant", "Hello! How can I help you?")

	history := model.GetHistory()
	if len(history) != 1 {
		t.Fatalf("GetHistory() length = %d, want 1", len(history))
	}
	if history[0].Role != "assistant" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "assistant")
	}
	if history[0].Content != "Hello! How can I help you?" {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, "Hello! How can I help you?")
	}
}

func TestModel_AddErrorMessage(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("error", "API connection failed")

	history := model.GetHistory()
	if len(history) != 1 {
		t.Fatalf("GetHistory() length = %d, want 1", len(history))
	}
	if history[0].Role != "error" {
		t.Errorf("history[0].Role = %q, want %q", history[0].Role, "error")
	}
	if history[0].Content != "API connection failed" {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, "API connection failed")
	}
}

func TestModel_GetHistory(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "first")
	model.AddMessage("assistant", "second")
	model.AddMessage("user", "third")

	history := model.GetHistory()
	if len(history) != 3 {
		t.Fatalf("GetHistory() length = %d, want 3", len(history))
	}

	// Verify order is preserved
	if history[0].Content != "first" {
		t.Errorf("history[0].Content = %q, want %q", history[0].Content, "first")
	}
	if history[1].Content != "second" {
		t.Errorf("history[1].Content = %q, want %q", history[1].Content, "second")
	}
	if history[2].Content != "third" {
		t.Errorf("history[2].Content = %q, want %q", history[2].Content, "third")
	}
}

func TestModel_SetInput(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("test input")

	if model.GetInput() != "test input" {
		t.Errorf("GetInput() = %q, want %q", model.GetInput(), "test input")
	}
}

func TestModel_ClearInput(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("test input")
	model.ClearInput()

	if model.GetInput() != "" {
		t.Errorf("GetInput() after ClearInput() = %q, want empty string", model.GetInput())
	}
}

func TestModel_IsLoading(t *testing.T) {
	model := NewModel(nil)

	if model.IsLoading() {
		t.Error("IsLoading() = true initially, want false")
	}

	model.SetLoading(true)
	if !model.IsLoading() {
		t.Error("IsLoading() = false after SetLoading(true), want true")
	}

	model.SetLoading(false)
	if model.IsLoading() {
		t.Error("IsLoading() = true after SetLoading(false), want false")
	}
}

func TestModel_SetLoading(t *testing.T) {
	model := NewModel(nil)

	model.SetLoading(true)
	if !model.IsLoading() {
		t.Error("SetLoading(true) did not set loading state")
	}

	model.SetLoading(false)
	if model.IsLoading() {
		t.Error("SetLoading(false) did not unset loading state")
	}
}

func TestModel_SetDimensions(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(120, 40)

	width, height := model.GetDimensions()
	if width != 120 {
		t.Errorf("GetDimensions() width = %d, want 120", width)
	}
	if height != 40 {
		t.Errorf("GetDimensions() height = %d, want 40", height)
	}
}

func TestModel_CanSubmit_ReturnsFalseWhenEmpty(t *testing.T) {
	model := NewModel(nil)

	// Empty input should not be submittable
	if model.CanSubmit() {
		t.Error("CanSubmit() = true with empty input, want false")
	}
}

func TestModel_CanSubmit_ReturnsFalseWhenLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("test input")
	model.SetLoading(true)

	// Should not be able to submit while loading
	if model.CanSubmit() {
		t.Error("CanSubmit() = true while loading, want false")
	}
}

func TestModel_CanSubmit_ReturnsFalseWhenWhitespaceOnly(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("   \t\n  ")

	// Whitespace-only input should not be submittable
	if model.CanSubmit() {
		t.Error("CanSubmit() = true with whitespace-only input, want false")
	}
}

func TestModel_CanSubmit_ReturnsTrue(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("valid input")

	// Valid input with content should be submittable
	if !model.CanSubmit() {
		t.Error("CanSubmit() = false with valid input, want true")
	}
}

func TestNewModel_InputIsFocused(t *testing.T) {
	model := NewModel(nil)

	if !model.input.Focused() {
		t.Error("NewModel() input should be focused so it can receive keystrokes")
	}
}

func TestUpdate_CharKey_UpdatesInput(t *testing.T) {
	model := NewModel(nil)

	// Simulate typing 'h'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updatedModel, _ := model.Update(keyMsg)

	m := updatedModel.(Model)
	if m.GetInput() != "h" {
		t.Errorf("After typing 'h', GetInput() = %q, want %q", m.GetInput(), "h")
	}
}

func TestUpdate_MultipleKeys_BuildsInput(t *testing.T) {
	model := NewModel(nil)

	for _, r := range "hi" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		updated, _ := model.Update(keyMsg)
		model = updated.(Model)
	}

	if model.GetInput() != "hi" {
		t.Errorf("After typing 'hi', GetInput() = %q, want %q", model.GetInput(), "hi")
	}
}

func TestModel_Init_ReturnsCmd(t *testing.T) {
	model := NewModel(nil)
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init() should return a non-nil Cmd")
	}
}

func TestModel_LastAssistantContent_NoMessages(t *testing.T) {
	model := NewModel(nil)
	if got := model.lastAssistantContent(); got != "" {
		t.Errorf("lastAssistantContent() = %q, want empty", got)
	}
}

func TestModel_LastAssistantContent_NoAssistant(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.AddMessage("user", "world")
	if got := model.lastAssistantContent(); got != "" {
		t.Errorf("lastAssistantContent() with no assistant messages = %q, want empty", got)
	}
}

func TestModel_LastAssistantContent_ReturnsLast(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hi")
	model.AddMessage("assistant", "first")
	model.AddMessage("user", "again")
	model.AddMessage("assistant", "second")
	if got := model.lastAssistantContent(); got != "second" {
		t.Errorf("lastAssistantContent() = %q, want %q", got, "second")
	}
}
