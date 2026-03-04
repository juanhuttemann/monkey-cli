package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelPicker_Inactive(t *testing.T) {
	mp := NewModelPicker(80)
	if mp.IsActive() {
		t.Error("NewModelPicker should be inactive by default")
	}
}

func TestModelPicker_ActivateDeactivate(t *testing.T) {
	mp := NewModelPicker(80)
	mp.Activate()
	if !mp.IsActive() {
		t.Error("After Activate, IsActive = false, want true")
	}
	mp.Deactivate()
	if mp.IsActive() {
		t.Error("After Deactivate, IsActive = true, want false")
	}
}

func TestModelPicker_NoSelectedWhenInactive(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet"})
	if got := mp.SelectedModel(); got != "" {
		t.Errorf("SelectedModel when inactive = %q, want ''", got)
	}
}

func TestModelPicker_SelectedModel_FirstByDefault(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet", "haiku"})
	mp.Activate()
	if got := mp.SelectedModel(); got != "opus" {
		t.Errorf("SelectedModel = %q, want %q", got, "opus")
	}
}

func TestModelPicker_Navigate_Down(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet", "haiku"})
	mp.Activate()

	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := mp.SelectedModel(); got != "sonnet" {
		t.Errorf("after Down, SelectedModel = %q, want %q", got, "sonnet")
	}
}

func TestModelPicker_Navigate_Up_AtTop_Noop(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet"})
	mp.Activate()

	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := mp.SelectedModel(); got != "opus" {
		t.Errorf("Up at top: SelectedModel = %q, want %q (noop)", got, "opus")
	}
}

func TestModelPicker_Navigate_DownAtBottom_Noop(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet"})
	mp.Activate()
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown}) // already at bottom
	if got := mp.SelectedModel(); got != "sonnet" {
		t.Errorf("Down at bottom: SelectedModel = %q, want %q (noop)", got, "sonnet")
	}
}

func TestModelPicker_SetCursor_ExistingModel(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet", "haiku"})
	mp.Activate()
	mp.SetCursor("sonnet")
	if got := mp.SelectedModel(); got != "sonnet" {
		t.Errorf("SetCursor('sonnet'): SelectedModel = %q, want %q", got, "sonnet")
	}
}

func TestModelPicker_SetCursor_UnknownModel_DefaultsToFirst(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus", "sonnet"})
	mp.Activate()
	mp.SetCursor("unknown")
	if got := mp.SelectedModel(); got != "opus" {
		t.Errorf("SetCursor('unknown'): SelectedModel = %q, want %q", got, "opus")
	}
}

func TestModelPicker_NoModels_SelectedEmpty(t *testing.T) {
	mp := NewModelPicker(80)
	mp.Activate()
	if got := mp.SelectedModel(); got != "" {
		t.Errorf("SelectedModel with no models = %q, want ''", got)
	}
}

func TestModelPicker_View_InactiveReturnsEmpty(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"opus"})
	if got := mp.View(); got != "" {
		t.Errorf("View when inactive = %q, want ''", got)
	}
}

func TestModelPicker_View_ActiveContainsModels(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"claude-opus-4", "claude-sonnet-4"})
	mp.Activate()
	view := mp.View()
	if view == "" {
		t.Error("View when active = '', want non-empty")
	}
	if !containsSubstring(view, "claude-opus-4") {
		t.Errorf("View does not contain 'claude-opus-4': %q", view)
	}
	if !containsSubstring(view, "claude-sonnet-4") {
		t.Errorf("View does not contain 'claude-sonnet-4': %q", view)
	}
}

func TestModelPicker_SetModels_ResetsCursorWhenOutOfBounds(t *testing.T) {
	mp := NewModelPicker(80)
	mp.SetModels([]string{"a", "b", "c"})
	mp.Activate()
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	// cursor at index 2

	mp.SetModels([]string{"x"}) // only 1 item now
	if got := mp.SelectedModel(); got != "x" {
		t.Errorf("after SetModels with fewer items, SelectedModel = %q, want %q", got, "x")
	}
}
