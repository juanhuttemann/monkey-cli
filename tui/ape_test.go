package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- /ape command: command picker ---

func TestCommandPicker_ContainsApe(t *testing.T) {
	cp := NewCommandPicker(80)
	cp.Activate()
	cp.SetQuery("ape")
	if cp.SelectedCommand() != "/ape" {
		t.Errorf("expected /ape in commands, SelectedCommand = %q", cp.SelectedCommand())
	}
}

// --- /ape command: typed input ---

func TestUpdate_SlashApe_EnablesApeMode(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/ape")

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})

	m := updatedModel.(Model)
	if !m.IsApeMode() {
		t.Error("IsApeMode = false after /ape, want true")
	}
}

func TestUpdate_SlashApe_ClearsInput(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/ape")

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})

	m := updatedModel.(Model)
	if m.GetInput() != "" {
		t.Errorf("Input after /ape = %q, want ''", m.GetInput())
	}
}

func TestUpdate_SlashApe_Twice_DisablesApeMode(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/ape")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := updated.(Model)

	m.SetInput("/ape")
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m2 := updated2.(Model)

	if m2.IsApeMode() {
		t.Error("IsApeMode = true after /ape twice, want false")
	}
}

// --- /ape command: via command picker ---

func TestUpdate_Enter_CommandPicker_Ape_TogglesApeMode(t *testing.T) {
	model := NewModel(nil)
	model.SetInput("/")
	model.commandPicker.Activate()
	model.commandPicker.SetQuery("")
	for model.commandPicker.SelectedCommand() != "/ape" {
		model.commandPicker, _ = model.commandPicker.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})

	m := updatedModel.(Model)
	if !m.IsApeMode() {
		t.Error("IsApeMode = false after Enter+/ape in picker, want true")
	}
	if m.commandPicker.IsActive() {
		t.Error("commandPicker should be inactive after /ape")
	}
}

// --- View indicator ---

func TestView_ApeModeIndicator_ShowsDisabledByDefault(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := stripANSI(model.View())

	if !strings.Contains(view, "Ape Mode: Disabled") {
		t.Errorf("View should show 'Ape Mode: Disabled' by default, got:\n%s", view)
	}
}

func TestView_ApeModeIndicator_HasBottomPadding(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := stripANSI(model.View())

	if !strings.Contains(view, "Ape Mode: Disabled\n\n") {
		t.Errorf("Ape mode indicator should have a blank line after it, got:\n%q", view)
	}
}

func TestView_ApeModeIndicator_DoesNotOverflowTerminalHeight(t *testing.T) {
	height := 24
	model := NewModel(nil)
	model.SetDimensions(80, height)

	view := stripANSI(model.View())
	lines := strings.Split(view, "\n")

	// The view must not exceed terminal height (overflow pushes viewport top off screen).
	if len(lines) > height {
		t.Errorf("View has %d lines but terminal height is %d — indicator overflows and clips top content", len(lines), height)
	}
}

func TestView_ApeModeIndicator_ShowsEnabledWhenActive(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.apeMode = true

	view := stripANSI(model.View())

	if !strings.Contains(view, "🍌 Ape Mode: Enabled") {
		t.Errorf("View should show '🍌 Ape Mode: Enabled' when active, got:\n%s", view)
	}
}

func TestView_ApeModeIndicator_EnabledUsesYellowColor(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.apeMode = true

	view := model.View() // raw with ANSI

	// ColorAccent = #B5B737 (Olive Yellow) → emits 181;183;55
	if !strings.Contains(view, "181;183;55") {
		t.Errorf("Enabled ape mode should use ColorAccent (181;183;55 in ANSI), got:\n%s", stripANSI(view))
	}
}
