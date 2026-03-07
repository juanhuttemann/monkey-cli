package tui

import "strings"

type helpItem struct {
	Trigger string
	Desc    string
}

var helpItems = []helpItem{
	{Trigger: "?", Desc: "show this help"},
	{Trigger: "/", Desc: "slash commands  /exit  /clear  /model  /ape  /copy  /compact"},
	{Trigger: "@", Desc: "mention files"},
}

// HelpPanel is a static panel listing available input shortcuts.
type HelpPanel struct {
	active bool
	width  int
}

// NewHelpPanel returns an inactive help panel.
func NewHelpPanel(width int) HelpPanel {
	return HelpPanel{width: width}
}

// Activate makes the panel visible.
func (hp *HelpPanel) Activate() { hp.active = true }

// Deactivate hides the panel.
func (hp *HelpPanel) Deactivate() { hp.active = false }

// IsActive reports whether the panel is visible.
func (hp HelpPanel) IsActive() bool { return hp.active }

// SetWidth updates the display width.
func (hp *HelpPanel) SetWidth(w int) { hp.width = w }

// View renders the help panel. Returns "" when inactive.
func (hp HelpPanel) View() string {
	if !hp.active {
		return ""
	}
	var sb strings.Builder
	for i, item := range helpItems {
		line := item.Trigger + "  " + item.Desc
		if i < len(helpItems)-1 {
			sb.WriteString(line + "\n")
		} else {
			sb.WriteString(line)
		}
	}
	return FilePickerStyle(hp.width).Render(sb.String())
}

// detectHelpQuery returns true when input is exactly "?" — a bare question mark
// with no other text, triggering the help panel and consuming the "?".
func detectHelpQuery(input string) bool {
	return input == "?"
}
