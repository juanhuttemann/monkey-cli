package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ModelPicker is a keyboard-navigable list for switching the active model.
type ModelPicker struct {
	models []string
	cursor int
	active bool
	width  int
}

// NewModelPicker returns an inactive model picker.
func NewModelPicker(width int) ModelPicker {
	return ModelPicker{width: width}
}

// Activate makes the picker visible.
func (mp *ModelPicker) Activate() { mp.active = true }

// Deactivate hides the picker.
func (mp *ModelPicker) Deactivate() { mp.active = false }

// IsActive reports whether the picker is visible.
func (mp ModelPicker) IsActive() bool { return mp.active }

// SetWidth updates the display width.
func (mp *ModelPicker) SetWidth(w int) { mp.width = w }

// SetModels sets the available model values.
func (mp *ModelPicker) SetModels(models []string) {
	mp.models = models
	if mp.cursor >= len(models) {
		mp.cursor = 0
	}
}

// SetCursor positions the cursor on the given model value (0 if not found).
func (mp *ModelPicker) SetCursor(model string) {
	for i, m := range mp.models {
		if m == model {
			mp.cursor = i
			return
		}
	}
	mp.cursor = 0
}

// SelectedModel returns the model under the cursor, or "" when inactive or empty.
func (mp ModelPicker) SelectedModel() string {
	if !mp.active || len(mp.models) == 0 || mp.cursor < 0 || mp.cursor >= len(mp.models) {
		return ""
	}
	return mp.models[mp.cursor]
}

// Update handles Up/Down cursor navigation.
func (mp ModelPicker) Update(msg tea.Msg) (ModelPicker, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return mp, nil
	}
	switch key.Type {
	case tea.KeyDown:
		if mp.cursor < len(mp.models)-1 {
			mp.cursor++
		}
	case tea.KeyUp:
		if mp.cursor > 0 {
			mp.cursor--
		}
	}
	return mp, nil
}

// View renders the model picker dropdown. Returns "" when inactive.
func (mp ModelPicker) View() string {
	if !mp.active {
		return ""
	}
	if len(mp.models) == 0 {
		return FilePickerStyle(mp.width).Render("  no models configured")
	}
	var sb strings.Builder
	for i, model := range mp.models {
		if i == mp.cursor {
			sb.WriteString(FilePickerCursorStyle().Render("> " + model))
		} else {
			sb.WriteString("  " + model)
		}
		if i < len(mp.models)-1 {
			sb.WriteByte('\n')
		}
	}
	return FilePickerStyle(mp.width).Render(sb.String())
}
