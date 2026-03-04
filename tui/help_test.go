package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- detectHelpQuery ---

func TestDetectHelpQuery_QuestionMark(t *testing.T) {
	if !detectHelpQuery("?") {
		t.Error("detectHelpQuery('?') = false, want true")
	}
}

func TestDetectHelpQuery_Empty(t *testing.T) {
	if detectHelpQuery("") {
		t.Error("detectHelpQuery('') = true, want false")
	}
}

func TestDetectHelpQuery_QuestionMarkWithText(t *testing.T) {
	if detectHelpQuery("?foo") {
		t.Error("detectHelpQuery('?foo') = true, want false (? not alone)")
	}
}

func TestDetectHelpQuery_TextThenQuestionMark(t *testing.T) {
	if detectHelpQuery("foo?") {
		t.Error("detectHelpQuery('foo?') = true, want false (? not at start)")
	}
}

func TestDetectHelpQuery_Multiline(t *testing.T) {
	if detectHelpQuery("?\nmore") {
		t.Error("detectHelpQuery with newline = true, want false")
	}
}

func TestDetectHelpQuery_SlashNotHelp(t *testing.T) {
	if detectHelpQuery("/") {
		t.Error("detectHelpQuery('/') = true, want false")
	}
}

// --- HelpPanel ---

func TestNewHelpPanel_Inactive(t *testing.T) {
	hp := NewHelpPanel(80)
	if hp.IsActive() {
		t.Error("NewHelpPanel should be inactive by default")
	}
}

func TestHelpPanel_ActivateDeactivate(t *testing.T) {
	hp := NewHelpPanel(80)
	hp.Activate()
	if !hp.IsActive() {
		t.Error("After Activate, IsActive = false, want true")
	}
	hp.Deactivate()
	if hp.IsActive() {
		t.Error("After Deactivate, IsActive = true, want false")
	}
}

func TestHelpPanel_View_WhenInactive_Empty(t *testing.T) {
	hp := NewHelpPanel(80)
	if v := hp.View(); v != "" {
		t.Errorf("HelpPanel.View() when inactive = %q, want ''", v)
	}
}

func TestHelpPanel_View_WhenActive_ContainsSlash(t *testing.T) {
	hp := NewHelpPanel(80)
	hp.Activate()
	v := stripANSI(hp.View())
	if !strings.Contains(v, "/") {
		t.Error("HelpPanel.View() should mention '/' for slash commands")
	}
}

func TestHelpPanel_View_WhenActive_ContainsAt(t *testing.T) {
	hp := NewHelpPanel(80)
	hp.Activate()
	v := stripANSI(hp.View())
	if !strings.Contains(v, "@") {
		t.Error("HelpPanel.View() should mention '@' for file mentions")
	}
}

func TestHelpPanel_View_WhenActive_ContainsQuestionMark(t *testing.T) {
	hp := NewHelpPanel(80)
	hp.Activate()
	v := stripANSI(hp.View())
	if !strings.Contains(v, "?") {
		t.Error("HelpPanel.View() should mention '?' itself")
	}
}

// --- Model integration ---

func TestUpdate_TypeQuestionMark_ActivatesHelpPanel(t *testing.T) {
	m := NewModel(nil)

	// Simulate textarea having "" before the keystroke, then "?" after.
	m.SetInput("")
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if !got.helpPanel.IsActive() {
		t.Error("helpPanel should activate when '?' is typed alone")
	}
}

func TestUpdate_TypeQuestionMark_ClearsInput(t *testing.T) {
	m := NewModel(nil)

	m.SetInput("")
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if got.GetInput() != "" {
		t.Errorf("input after '?' = %q, want '' (? should be consumed)", got.GetInput())
	}
}

func TestUpdate_HelpPanel_Esc_Dismisses(t *testing.T) {
	m := NewModel(nil)
	m.helpPanel.Activate()

	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := m.Update(escKey)

	got := updatedModel.(Model)
	if got.helpPanel.IsActive() {
		t.Error("helpPanel should be dismissed by Esc")
	}
}

func TestUpdate_TypeTextAfterHelp_DismissesHelpPanel(t *testing.T) {
	m := NewModel(nil)
	m.helpPanel.Activate()

	// Typing any character (not ?) while help is active should dismiss it
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if got.helpPanel.IsActive() {
		t.Error("helpPanel should deactivate when non-'?' content is typed")
	}
}

func TestUpdate_TypeQuestionMarkWithPriorText_NoHelp(t *testing.T) {
	m := NewModel(nil)
	m.SetInput("hello")

	// Simulate typing '?' when input already has text → result is "hello?"
	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if got.helpPanel.IsActive() {
		t.Error("helpPanel should NOT activate when '?' is not the only character")
	}
}

func TestUpdate_QuestionMark_DoesNotActivateCommandPicker(t *testing.T) {
	m := NewModel(nil)
	m.SetInput("")

	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if got.commandPicker.IsActive() {
		t.Error("commandPicker should not activate when '?' is typed")
	}
}

func TestUpdate_QuestionMark_DoesNotActivateFilePicker(t *testing.T) {
	m := NewModel(nil)
	m.SetInput("")

	runeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(runeKey)

	got := updatedModel.(Model)
	if got.filePicker.IsActive() {
		t.Error("filePicker should not activate when '?' is typed")
	}
}
