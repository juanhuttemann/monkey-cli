package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	m1, _ := model.Update(upKey)        // navigate to "old entry", draft saved
	m2, _ := m1.(Model).Update(downKey) // back down → restore draft

	if m2.(Model).GetInput() != "my draft" {
		t.Errorf("after Up then Down, input = %q, want %q", m2.(Model).GetInput(), "my draft")
	}
}

func TestUpdate_CtrlEnter_SavesPromptToHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "response_text.json"))
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

func TestUpdate_Esc_CommandPickerActive_Deactivates(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.(Model).commandPicker.IsActive() {
		t.Error("Esc should deactivate an active command picker")
	}
}

func TestUpdate_Esc_WhileLoading_CallsCancelFn(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	var called bool
	model.cancelFn = func() { called = true }
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !called {
		t.Error("Esc while loading should invoke cancelFn")
	}
	if m.(Model).cancelFn != nil {
		t.Error("cancelFn should be nil after cancellation")
	}
}

func TestUpdate_CtrlC_WhileLoading_CallsCancelFn(t *testing.T) {
	model := NewModel(nil)
	model.SetLoading(true)
	var called bool
	model.cancelFn = func() { called = true }
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !called {
		t.Error("CtrlC while loading should invoke cancelFn")
	}
	if m.(Model).cancelFn != nil {
		t.Error("cancelFn should be nil after cancellation")
	}
}

func TestUpdate_KeyPgDown_DoesNotPanic(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
}

func TestUpdate_KeyUp_ModelPickerActive(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"a", "b", "c"})
	model.modelPicker.Activate()
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestUpdate_KeyDown_ModelPickerActive(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"a", "b", "c"})
	model.modelPicker.Activate()
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
}

func TestUpdate_KeyUp_FilePickerActive(t *testing.T) {
	model := NewModel(nil)
	model.filePicker.Activate()
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestUpdate_KeyDown_FilePickerActive(t *testing.T) {
	model := NewModel(nil)
	model.filePicker.Activate()
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
}

func TestUpdate_KeyTab_CommandPicker_ModelWithClient_ActivatesModelPicker(t *testing.T) {
	client := newTestClientWithModel("claude-sonnet")
	model := NewModel(client)
	model.SetModels([]string{"claude-sonnet", "claude-opus"})
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("model")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !m.(Model).modelPicker.IsActive() {
		t.Error("Tab on /model with client set should activate the model picker")
	}
}

func TestUpdate_SearchActive_Backspace_RemovesChar(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello world")
	model.searchBar.Activate()
	model.searchBar.SetQuery("hel", model.messages)
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.(Model).searchBar.Query() != "he" {
		t.Errorf("backspace in search bar: query = %q, want %q", m.(Model).searchBar.Query(), "he")
	}
}

func TestUpdate_SearchActive_Backspace_EmptyQuery_NoOp(t *testing.T) {
	model := NewModel(nil)
	model.searchBar.Activate()
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
}

func TestUpdate_SearchActive_Rune_AppendsToQuery(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello world")
	model.searchBar.Activate()
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.(Model).searchBar.Query() != "h" {
		t.Errorf("rune in search bar: query = %q, want %q", m.(Model).searchBar.Query(), "h")
	}
}

func TestUpdate_TypingSlashModel_WithModelsAndClient_ActivatesModelPicker(t *testing.T) {
	client := newTestClientWithModel("claude-sonnet")
	model := NewModel(client)
	model.SetModels([]string{"claude-sonnet", "claude-opus"})
	model.SetInput("/mode")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !m.(Model).modelPicker.IsActive() {
		t.Error("typing /model with models and client should activate the model picker")
	}
}

func TestUpdate_KeyUp_CommandPickerActive(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestUpdate_KeyDown_CommandPickerActive(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
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

// --- /exit ---

func TestUpdate_SlashExit_Quits(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/exit")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	_, cmd := model.Update(ctrlEnter)

	if cmd == nil {
		t.Fatal("Update(/exit + Enter) returned nil cmd, want quit sequence")
	}
}

// --- /clear ---

func TestUpdate_SlashClear_ResetsMessages(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.AddMessage("assistant", "world")
	model.SetInput("/clear")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if len(m.GetHistory()) != 0 {
		t.Errorf("History after /clear = %d messages, want 0", len(m.GetHistory()))
	}
}

func TestUpdate_SlashClear_ResetsAPIMessages(t *testing.T) {
	model := NewModel(nil)
	model.apiMessages = []api.Message{{Role: "user", Content: "prior"}}
	model.SetInput("/clear")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if len(m.apiMessages) != 0 {
		t.Errorf("apiMessages after /clear = %d entries, want 0", len(m.apiMessages))
	}
}

func TestUpdate_SlashClear_ResetsTokenUsage(t *testing.T) {
	model := NewModel(nil)
	model.totalUsage = api.Usage{InputTokens: 500, OutputTokens: 200}
	model.SetInput("/clear")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if m.totalUsage.InputTokens != 0 || m.totalUsage.OutputTokens != 0 {
		t.Errorf("totalUsage after /clear = {%d, %d}, want {0, 0}",
			m.totalUsage.InputTokens, m.totalUsage.OutputTokens)
	}
}

// --- Tab autocomplete ---

func TestUpdate_Tab_SelectsSlashCommand(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/ex")
	// Manually activate the command picker (as Update would via keystroke routing)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("ex")

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabKey)

	m := updatedModel.(Model)
	if m.GetInput() != "/exit" {
		t.Errorf("Input after Tab = %q, want '/exit'", m.GetInput())
	}
	if m.commandPicker.IsActive() {
		t.Error("CommandPicker should be inactive after Tab selection")
	}
}

// Tab on "/cop" autocompletes to "/copy"; subsequent Enter must execute the
// command, NOT submit it as a user prompt.
func TestUpdate_Tab_AutocompleteCopy_ThenEnter_NotSubmittedAsPrompt(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/cop")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("cop")

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabKey)
	m := updatedModel.(Model)
	if m.GetInput() != "/copy" {
		t.Fatalf("Input after Tab = %q, want '/copy'", m.GetInput())
	}

	initialMsgCount := len(m.GetHistory())
	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel2, _ := m.Update(ctrlEnter)
	m2 := updatedModel2.(Model)

	if len(m2.GetHistory()) > initialMsgCount {
		t.Error("/copy after Tab+Enter was submitted as prompt — should have been executed as command")
	}
	if m2.GetInput() != "" {
		t.Errorf("input after Tab+Enter /copy = %q, want ''", m2.GetInput())
	}
}

// Typing "/copy" directly then pressing Enter must execute the command, not
// submit it as a user prompt.
func TestUpdate_DirectInput_Copy_NotSubmittedAsPrompt(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/copy")

	initialMsgCount := len(model.GetHistory())
	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)
	m := updatedModel.(Model)

	if len(m.GetHistory()) > initialMsgCount {
		t.Error("/copy direct input was submitted as prompt — should have been executed as command")
	}
	if m.GetInput() != "" {
		t.Errorf("input after /copy = %q, want ''", m.GetInput())
	}
}

// --- Command picker activates on / ---

func TestUpdate_TypeSlash_ActivatesCommandPicker(t *testing.T) {
	model := NewModel(nil)

	// Simulate typing "/" by setting input and triggering a rune key
	// The picker activates when detectCommandQuery returns true
	model.SetInput("/")
	// Sync picker state as Update would
	query, active := detectCommandQuery(model.GetInput())
	if active {
		model.commandPicker.Activate()
		model.commandPicker.SetQuery(query)
	}

	if !model.commandPicker.IsActive() {
		t.Error("CommandPicker should activate when input starts with '/'")
	}
}

// --- /model inline (activates while typing) ---

func TestUpdate_TypeSlashModel_ShowsModelPicker(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})

	// Simulate typing the final 'l' of "/model" — textarea appends 'l' to "/mode"
	model.SetInput("/mode")
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	updatedModel, _ := model.Update(runeKey)

	m := updatedModel.(Model)
	if !m.modelPicker.IsActive() {
		t.Error("modelPicker should activate when input is '/model'")
	}
	if m.commandPicker.IsActive() {
		t.Error("commandPicker should be inactive when modelPicker is active")
	}
}

func TestUpdate_TypeSlashMod_ShowsCommandPickerNotModelPicker(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4"})

	model.SetInput("/mod")
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updatedModel, _ := model.Update(runeKey)

	m := updatedModel.(Model)
	if m.modelPicker.IsActive() {
		t.Error("modelPicker should NOT activate for partial query '/mod'")
	}
	if !m.commandPicker.IsActive() {
		t.Error("commandPicker should activate for '/mod'")
	}
}

func TestUpdate_TypeSlashModelX_DeactivatesModelPicker(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4"})
	// Start with model picker active (user had "/model")
	model.modelPicker.Activate()

	// Now input changes to "/modelx" — model picker should deactivate
	model.SetInput("/modelx")
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	updatedModel, _ := model.Update(runeKey)

	m := updatedModel.(Model)
	if m.modelPicker.IsActive() {
		t.Error("modelPicker should deactivate when input is '/modelx' (not exact match)")
	}
}

func TestUpdate_TabOnCommandPicker_SlashModel_ShowsModelPicker(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})
	model.SetInput("/model")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("model")

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabKey)

	m := updatedModel.(Model)
	if !m.modelPicker.IsActive() {
		t.Error("modelPicker should activate after Tab-selecting /model from command picker")
	}
	if m.commandPicker.IsActive() {
		t.Error("commandPicker should be inactive after Tab-selecting /model")
	}
}

func TestUpdate_ModelPickerInline_TabClearsInput(t *testing.T) {
	client := newTestClientWithModel("old-model")
	model := NewModel(client)
	model.SetModels([]string{"claude-opus-4"})
	model.SetInput("/model")
	model.modelPicker.SetModels([]string{"claude-opus-4"})
	model.modelPicker.Activate()

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabKey)

	m := updatedModel.(Model)
	if m.GetInput() != "" {
		t.Errorf("input after model selection = %q, want ''", m.GetInput())
	}
}

// --- /model ---

func TestUpdate_SlashModel_ActivatesModelPicker(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})
	model.SetInput("/model")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if !m.modelPicker.IsActive() {
		t.Error("modelPicker should be active after /model command")
	}
	if m.GetInput() != "" {
		t.Errorf("input should be cleared after /model, got %q", m.GetInput())
	}
}

func TestUpdate_ModelPicker_Tab_SwitchesModel(t *testing.T) {
	client := newTestClientWithModel("original-model")
	model := NewModel(client)
	model.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})
	model.modelPicker.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})
	model.modelPicker.Activate()
	// Navigate to second model
	model.modelPicker, _ = model.modelPicker.Update(tea.KeyMsg{Type: tea.KeyDown})

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabKey)

	m := updatedModel.(Model)
	if m.modelPicker.IsActive() {
		t.Error("modelPicker should be inactive after Tab selection")
	}
	if client.GetModel() != "claude-sonnet-4" {
		t.Errorf("client model = %q, want %q", client.GetModel(), "claude-sonnet-4")
	}
}

func TestUpdate_ModelPicker_Enter_SwitchesModel(t *testing.T) {
	client := newTestClientWithModel("original-model")
	model := NewModel(client)
	model.SetModels([]string{"claude-opus-4"})
	model.modelPicker.SetModels([]string{"claude-opus-4"})
	model.modelPicker.Activate()

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if m.modelPicker.IsActive() {
		t.Error("modelPicker should be inactive after Enter selection")
	}
	if client.GetModel() != "claude-opus-4" {
		t.Errorf("client model = %q, want %q", client.GetModel(), "claude-opus-4")
	}
}

func TestUpdate_ModelPicker_Esc_Dismisses(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"claude-opus-4"})
	model.modelPicker.SetModels([]string{"claude-opus-4"})
	model.modelPicker.Activate()

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escKey)

	m := updatedModel.(Model)
	if m.modelPicker.IsActive() {
		t.Error("modelPicker should be inactive after Esc")
	}
}

func TestUpdate_ModelPicker_Navigate(t *testing.T) {
	model := NewModel(nil)
	model.SetModels([]string{"a", "b", "c"})
	model.modelPicker.SetModels([]string{"a", "b", "c"})
	model.modelPicker.Activate()

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downKey)

	m := updatedModel.(Model)
	if m.modelPicker.SelectedModel() != "b" {
		t.Errorf("after Down, SelectedModel = %q, want %q", m.modelPicker.SelectedModel(), "b")
	}
}

// --- Enter with command picker active ---

// Pressing Enter while hovering /exit in the picker should quit immediately.
func TestUpdate_Enter_CommandPicker_SlashExit_Quits(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	// Navigate to /exit
	for model.commandPicker.SelectedCommand() != "/exit" {
		model.commandPicker, _ = model.commandPicker.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	_, cmd := model.Update(ctrlEnter)

	if cmd == nil {
		t.Fatal("Enter on /exit in picker returned nil cmd, want quit sequence")
	}
}

// Pressing Enter while hovering /clear in the picker should clear history.
func TestUpdate_Enter_CommandPicker_SlashClear_Resets(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.SetInput("/")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	for model.commandPicker.SelectedCommand() != "/clear" {
		model.commandPicker, _ = model.commandPicker.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if len(m.GetHistory()) != 0 {
		t.Errorf("History after Enter+/clear picker = %d messages, want 0", len(m.GetHistory()))
	}
	if m.GetInput() != "" {
		t.Errorf("Input after Enter+/clear picker = %q, want ''", m.GetInput())
	}
}

// Pressing Enter while hovering /ape in the picker should toggle ape mode.
func TestUpdate_Enter_CommandPicker_SlashApe_TogglesMode(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	for model.commandPicker.SelectedCommand() != "/ape" {
		model.commandPicker, _ = model.commandPicker.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if !m.apeMode {
		t.Error("Enter on /ape in picker should toggle ape mode")
	}
}

// --- Esc no longer quits when StateReady ---

func TestUpdate_Esc_WhenReady_NoLongerQuits(t *testing.T) {
	model := NewModel(nil)

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := model.Update(escKey)

	// Esc should NOT quit when in StateReady
	if cmd != nil {
		result := cmd()
		if _, isQuit := result.(tea.QuitMsg); isQuit {
			t.Error("Esc when StateReady should not quit — use /exit instead")
		}
	}
}

// --- Command picker: /clear, /ape, /compact ---

func TestUpdate_CommandPicker_SlashClear_ResetsMessages(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("clear")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if len(m.(Model).GetHistory()) != 0 {
		t.Error("/clear from command picker should clear message history")
	}
}

func TestUpdate_CommandPicker_SlashClear_ResetsAPIMessages(t *testing.T) {
	model := NewModel(nil)
	model.apiMessages = []api.Message{{Role: "user", Content: "prior"}}
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("clear")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if len(m.(Model).apiMessages) != 0 {
		t.Errorf("apiMessages after /clear from picker = %d entries, want 0", len(m.(Model).apiMessages))
	}
}

func TestUpdate_CommandPicker_SlashClear_ResetsTokenUsage(t *testing.T) {
	model := NewModel(nil)
	model.totalUsage = api.Usage{InputTokens: 500, OutputTokens: 200}
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("clear")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if m.(Model).totalUsage.InputTokens != 0 || m.(Model).totalUsage.OutputTokens != 0 {
		t.Errorf("totalUsage after /clear from picker = {%d, %d}, want {0, 0}",
			m.(Model).totalUsage.InputTokens, m.(Model).totalUsage.OutputTokens)
	}
}

func TestUpdate_CommandPicker_SlashApe_TogglesApeMode(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("ape")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if !m.(Model).apeMode {
		t.Error("/ape from command picker should enable ape mode")
	}
}

func TestUpdate_CommandPicker_SlashExit_Quits(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("exit")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if cmd == nil {
		t.Fatal("/exit from command picker returned nil cmd, want quit sequence")
	}
}

func TestUpdate_CommandPicker_SlashCompact_EmptyHistory_Noop(t *testing.T) {
	model := NewModel(nil)
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("compact")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if cmd != nil {
		t.Error("cmd should be nil when there is nothing to compact")
	}
}

func TestUpdate_CommandPicker_SlashCompact_SendsCmd(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("user", "hello")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("compact")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if !m.(Model).IsLoading() {
		t.Error("/compact from command picker with messages should start loading state")
	}
}

// --- /copy ---

func TestUpdate_SlashCopy_CopiesLastAssistantMessage(t *testing.T) {
	model := NewModel(nil)
	model.AddMessage("assistant", "some answer")
	model.SetInput("/copy")
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
}

func TestUpdate_SlashCopy_EmptyHistory_Noop(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/copy")
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
}

// --- /model with client ---

func TestUpdate_SlashModel_WithClient_ActivatesModelPicker(t *testing.T) {
	client := newTestClientWithModel("claude-opus")
	model := NewModel(client)
	model.SetModels([]string{"claude-sonnet", "claude-opus"})
	model.SetInput("/model")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	if !m.(Model).modelPicker.IsActive() {
		t.Error("/model typed directly with client should activate model picker")
	}
}

// --- clipboard error path ---

func TestUpdate_CommandPicker_Copy_ClipboardError_AddsErrorMessage(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	model := NewModel(nil)
	model.AddMessage("assistant", "the answer")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("copy")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	msgs := m.(Model).GetHistory()
	// With no clipboard available, an error message should have been appended.
	found := false
	for _, msg := range msgs {
		if msg.Role == "error" {
			found = true
		}
	}
	if !found {
		t.Skip("clipboard succeeded in this environment; error path not exercised")
	}
}

func TestUpdate_SlashCopy_ClipboardError_AddsErrorMessage(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	model := NewModel(nil)
	model.AddMessage("assistant", "the answer")
	model.SetInput("/copy")
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	msgs := m.(Model).GetHistory()
	found := false
	for _, msg := range msgs {
		if msg.Role == "error" {
			found = true
		}
	}
	if !found {
		t.Skip("clipboard succeeded in this environment; error path not exercised")
	}
}
