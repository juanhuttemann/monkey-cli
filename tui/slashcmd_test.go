package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- detectCommandQuery ---

func TestDetectCommandQuery_SlashOnly(t *testing.T) {
	query, active := detectCommandQuery("/")
	if !active {
		t.Error("active = false for '/', want true")
	}
	if query != "" {
		t.Errorf("query = %q, want ''", query)
	}
}

func TestDetectCommandQuery_SlashWithPartial(t *testing.T) {
	query, active := detectCommandQuery("/ex")
	if !active {
		t.Error("active = false for '/ex', want true")
	}
	if query != "ex" {
		t.Errorf("query = %q, want 'ex'", query)
	}
}

func TestDetectCommandQuery_SlashWithFullCommand(t *testing.T) {
	query, active := detectCommandQuery("/exit")
	if !active {
		t.Error("active = false for '/exit', want true")
	}
	if query != "exit" {
		t.Errorf("query = %q, want 'exit'", query)
	}
}

func TestDetectCommandQuery_NoSlash(t *testing.T) {
	_, active := detectCommandQuery("hello world")
	if active {
		t.Error("active = true for non-slash input, want false")
	}
}

func TestDetectCommandQuery_Empty(t *testing.T) {
	_, active := detectCommandQuery("")
	if active {
		t.Error("active = true for empty input, want false")
	}
}

func TestDetectCommandQuery_WithSpace_Inactive(t *testing.T) {
	// Once a space is typed after the command, picker deactivates
	_, active := detectCommandQuery("/exit ")
	if active {
		t.Error("active = true after space, want false")
	}
}

func TestDetectCommandQuery_Multiline_Inactive(t *testing.T) {
	_, active := detectCommandQuery("/exit\nsome text")
	if active {
		t.Error("active = true for multiline input, want false")
	}
}

func TestDetectCommandQuery_NotAtBeginning_Inactive(t *testing.T) {
	// slash command only works at the beginning of input
	_, active := detectCommandQuery("hello /exit")
	if active {
		t.Error("active = true when / is not at start, want false")
	}
}

// --- parseSlashCommand ---

func TestParseSlashCommand_Exit(t *testing.T) {
	cmd, ok := parseSlashCommand("/exit")
	if !ok {
		t.Error("ok = false for '/exit', want true")
	}
	if cmd != "/exit" {
		t.Errorf("cmd = %q, want '/exit'", cmd)
	}
}

func TestParseSlashCommand_Clear(t *testing.T) {
	cmd, ok := parseSlashCommand("/clear")
	if !ok {
		t.Error("ok = false for '/clear', want true")
	}
	if cmd != "/clear" {
		t.Errorf("cmd = %q, want '/clear'", cmd)
	}
}

func TestParseSlashCommand_NoSlash(t *testing.T) {
	_, ok := parseSlashCommand("exit")
	if ok {
		t.Error("ok = true for 'exit' (no slash), want false")
	}
}

func TestParseSlashCommand_Multiline_NotOk(t *testing.T) {
	_, ok := parseSlashCommand("/exit\nmore text")
	if ok {
		t.Error("ok = true for multiline slash command, want false")
	}
}

func TestParseSlashCommand_Unknown(t *testing.T) {
	cmd, ok := parseSlashCommand("/unknown")
	if !ok {
		t.Error("ok = false for unknown slash command, want true (parsed but not executed)")
	}
	if cmd != "/unknown" {
		t.Errorf("cmd = %q, want '/unknown'", cmd)
	}
}

// --- CommandPicker ---

func TestNewCommandPicker_Inactive(t *testing.T) {
	cp := NewCommandPicker(80)
	if cp.IsActive() {
		t.Error("NewCommandPicker should be inactive by default")
	}
}

func TestCommandPicker_ActivateDeactivate(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	if !cp.IsActive() {
		t.Error("After Activate, IsActive = false, want true")
	}
	cp.Deactivate()
	if cp.IsActive() {
		t.Error("After Deactivate, IsActive = true, want false")
	}
}

func TestCommandPicker_NoSelectedWhenInactive(t *testing.T) {
	cp := NewCommandPicker(80)
	if got := cp.SelectedCommand(); got != "" {
		t.Errorf("SelectedCommand when inactive = %q, want ''", got)
	}
}

func TestCommandPicker_SetQuery_FiltersCommands(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("ex")
	got := cp.SelectedCommand()
	if got != "/exit" {
		t.Errorf("SelectedCommand after SetQuery('ex') = %q, want '/exit'", got)
	}
}

func TestCommandPicker_SetQuery_EmptyShowsAll(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("")
	// Should have a selected command (first one)
	if cp.SelectedCommand() == "" {
		t.Error("SelectedCommand after empty query = '', want a command")
	}
}

func TestCommandPicker_SetQuery_NoMatch(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("zzzzz")
	if got := cp.SelectedCommand(); got != "" {
		t.Errorf("SelectedCommand with no match = %q, want ''", got)
	}
}

func TestCommandPicker_Navigate(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("") // show all commands (exit, clear)
	first := cp.SelectedCommand()
	cp, _ = cp.Update(tea.KeyMsg{Type: tea.KeyDown})
	second := cp.SelectedCommand()
	if first == second {
		t.Errorf("Down did not change selection: before=%q after=%q", first, second)
	}
}

func TestCommandPicker_NavigateUp_AtTop_Noop(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("")
	cp, _ = cp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cp.SelectedCommand() == "" {
		t.Error("SelectedCommand after Up at top = '', want first command")
	}
}

func TestCommandPicker_ContainsExitAndClear(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()

	cp.SetQuery("exit")
	if cp.SelectedCommand() != "/exit" {
		t.Errorf("expected /exit in commands, SelectedCommand = %q", cp.SelectedCommand())
	}

	cp.SetQuery("clear")
	if cp.SelectedCommand() != "/clear" {
		t.Errorf("expected /clear in commands, SelectedCommand = %q", cp.SelectedCommand())
	}
}

// --- Model integration: /exit ---

func TestUpdate_SlashExit_Quits(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/exit")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	_, cmd := model.Update(ctrlEnter)

	if cmd == nil {
		t.Fatal("Update(/exit + Enter) returned nil cmd, want tea.Quit")
	}
	result := cmd()
	if _, isQuit := result.(tea.QuitMsg); !isQuit {
		t.Errorf("Update(/exit + Enter) produced %T, want tea.QuitMsg", result)
	}
}

// --- Model integration: /clear ---

func TestUpdate_SlashClear_ClearsHistory(t *testing.T) {
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

func TestUpdate_SlashClear_ClearsInput(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/clear")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	updatedModel, _ := model.Update(ctrlEnter)

	m := updatedModel.(Model)
	if m.GetInput() != "" {
		t.Errorf("Input after /clear = %q, want ''", m.GetInput())
	}
}

func TestUpdate_SlashClear_DoesNotQuit(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/clear")

	ctrlEnter := tea.KeyMsg{Type: tea.KeyCtrlM}
	_, cmd := model.Update(ctrlEnter)

	if cmd != nil {
		result := cmd()
		if _, isQuit := result.(tea.QuitMsg); isQuit {
			t.Error("/clear should not quit the app")
		}
	}
}

// --- Model integration: Tab autocomplete ---

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

// --- Model integration: command picker activates on / ---

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
