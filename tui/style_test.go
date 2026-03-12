package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestUserMessageStyle_HasBorder(t *testing.T) {
	style := UserMessageStyle(80)

	// Get the rendered string to check for border
	rendered := style.Render("test")

	// A bordered style should produce output longer than the content
	if len(rendered) <= len("test") {
		t.Errorf("UserMessageStyle should add border, got rendered length %d, content length %d", len(rendered), len("test"))
	}
}

func TestUserMessageStyle_HasBackground(t *testing.T) {
	style := UserMessageStyle(80)

	// The style should have a background color set
	rendered := style.Render("test")

	// Rendered output should contain ANSI escape codes for styling
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("UserMessageStyle should produce ANSI escape codes for background color")
	}
}

func TestUserMessageStyle_NoLabel(t *testing.T) {
	style := UserMessageStyle(80)
	rendered := style.Render("test message")

	// Should not contain "User:" label
	if strings.Contains(rendered, "User:") {
		t.Error("UserMessageStyle should not include 'User:' label")
	}
	if strings.Contains(rendered, "user:") {
		t.Error("UserMessageStyle should not include 'user:' label")
	}
}

func TestAssistantMessageStyle_HasBorder(t *testing.T) {
	style := AssistantMessageStyle(80)

	rendered := style.Render("test")

	// A bordered style should produce output longer than the content
	if len(rendered) <= len("test") {
		t.Errorf("AssistantMessageStyle should add border, got rendered length %d, content length %d", len(rendered), len("test"))
	}
}

func TestAssistantMessageStyle_HasBackground(t *testing.T) {
	style := AssistantMessageStyle(80)

	rendered := style.Render("test")

	// Rendered output should contain ANSI escape codes for styling
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("AssistantMessageStyle should produce ANSI escape codes for background color")
	}
}

func TestAssistantMessageStyle_NoLabel(t *testing.T) {
	style := AssistantMessageStyle(80)
	rendered := style.Render("test message")

	// Should not contain "Assistant:" label
	if strings.Contains(rendered, "Assistant:") {
		t.Error("AssistantMessageStyle should not include 'Assistant:' label")
	}
	if strings.Contains(rendered, "assistant:") {
		t.Error("AssistantMessageStyle should not include 'assistant:' label")
	}
}

func TestErrorMessageStyle_HasRedColor(t *testing.T) {
	style := ErrorMessageStyle(80)

	rendered := style.Render("error occurred")

	// Rendered output should contain ANSI escape codes
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("ErrorMessageStyle should produce ANSI escape codes for red color")
	}
}

func TestInputAreaStyle_HasBorder(t *testing.T) {
	style := InputStyle(80, 5)

	rendered := style.Render("input area")

	// A bordered style should produce output longer than the content
	if len(rendered) <= len("input area") {
		t.Errorf("InputStyle should add border, got rendered length %d, content length %d", len(rendered), len("input area"))
	}
}

func TestMessageStyles_NoBackgroundColors(t *testing.T) {
	// Message box backgrounds bleed through the border and clash with the
	// terminal's native background. All three message styles must be background-free.
	cases := []struct {
		name  string
		style func(int) lipgloss.Style
	}{
		{"UserMessageStyle", UserMessageStyle},
		{"AssistantMessageStyle", AssistantMessageStyle},
		{"ErrorMessageStyle", ErrorMessageStyle},
	}
	for _, tc := range cases {
		rendered := tc.style(80).Render("test content")
		if stripped := stripANSIBackground(rendered); stripped != rendered {
			t.Errorf("%s: rendered output contains background ANSI codes", tc.name)
		}
	}
}

func TestStyles_AreDistinct(t *testing.T) {
	userStyle := UserMessageStyle(80)
	assistantStyle := AssistantMessageStyle(80)

	userRendered := userStyle.Render("test")
	assistantRendered := assistantStyle.Render("test")

	// The rendered outputs should be different (different colors/borders)
	if userRendered == assistantRendered {
		t.Error("UserMessageStyle and AssistantMessageStyle should produce different output")
	}
}

func TestSpinnerStyle(t *testing.T) {
	style := SpinnerStyle()

	// SpinnerStyle should return a valid style
	rendered := style.Render("⠋")
	if rendered == "" {
		t.Error("SpinnerStyle should produce non-empty output")
	}
}

func TestUserMessageStyle_RespectsWidth(t *testing.T) {
	// Test with different widths
	styleNarrow := UserMessageStyle(40)
	styleWide := UserMessageStyle(120)

	narrowRendered := styleNarrow.Render("test content")
	wideRendered := styleWide.Render("test content")

	// Both should render successfully
	if narrowRendered == "" {
		t.Error("UserMessageStyle(40) should produce non-empty output")
	}
	if wideRendered == "" {
		t.Error("UserMessageStyle(120) should produce non-empty output")
	}
}

func TestToolMessageStyle_HasBorder(t *testing.T) {
	style := ToolMessageStyle(80)
	rendered := style.Render("$ ls")
	if len(rendered) <= len("$ ls") {
		t.Error("ToolMessageStyle should add a border")
	}
}

func TestToolMessageStyle_HasANSI(t *testing.T) {
	style := ToolMessageStyle(80)
	rendered := style.Render("test")
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("ToolMessageStyle should produce ANSI escape codes")
	}
}

func TestToolMessageStyle_IsDistinctFromOtherStyles(t *testing.T) {
	tool := ToolMessageStyle(80).Render("test")
	user := UserMessageStyle(80).Render("test")
	assistant := AssistantMessageStyle(80).Render("test")
	if tool == user {
		t.Error("ToolMessageStyle should differ from UserMessageStyle")
	}
	if tool == assistant {
		t.Error("ToolMessageStyle should differ from AssistantMessageStyle")
	}
}

func TestToolMessageStyle_NoBackground(t *testing.T) {
	rendered := ToolMessageStyle(80).Render("test content")
	if stripped := stripANSIBackground(rendered); stripped != rendered {
		t.Error("ToolMessageStyle should not set a background color")
	}
}

func TestAssistantMessageStyle_RespectsWidth(t *testing.T) {
	// Test with different widths
	styleNarrow := AssistantMessageStyle(40)
	styleWide := AssistantMessageStyle(120)

	narrowRendered := styleNarrow.Render("test content")
	wideRendered := styleWide.Render("test content")

	// Both should render successfully
	if narrowRendered == "" {
		t.Error("AssistantMessageStyle(40) should produce non-empty output")
	}
	if wideRendered == "" {
		t.Error("AssistantMessageStyle(120) should produce non-empty output")
	}
}

// ColorErrorBorder = #BA3F28 = rgb(186,63,40)
func TestErrorMessageStyle_UsesBurntOrangeBorder(t *testing.T) {
	rendered := ErrorMessageStyle(80).Render("error")
	if !strings.Contains(rendered, "186;63;40") {
		t.Errorf("ErrorMessageStyle border should use Burnt Orange (%s = 186;63;40), got ANSI: %q", ColorErrorBorder, rendered)
	}
}

// InputStyle border uses ColorAccent (Antique Gold) = #A89228 → emits 168;146;40
func TestInputStyle_UsesAccentBorderColor(t *testing.T) {
	rendered := InputStyle(80, 3).Render("text")
	if !strings.Contains(rendered, "168;146;40") {
		t.Errorf("InputStyle border should use ColorAccent (%s → 168;146;40), got ANSI: %q", ColorAccent, rendered)
	}
}

// ColorAccent = #A89228 = rgb(168,146,40)
func TestFilePickerCursorStyle_UsesAccentColor(t *testing.T) {
	rendered := FilePickerCursorStyle().Render("item")
	if !strings.Contains(rendered, "168;146;40") {
		t.Errorf("FilePickerCursorStyle should use ColorAccent (%s = 168;146;40), got ANSI: %q", ColorAccent, rendered)
	}
}

func TestWaitingStyle_UsesAccentColor(t *testing.T) {
	rendered := WaitingStyle().Render("What should monkey do?")
	if !strings.Contains(rendered, "168;146;40") {
		t.Errorf("WaitingStyle should use ColorAccent (%s = 168;146;40), got ANSI: %q", ColorAccent, rendered)
	}
}

// --- Palette-driven color tests ---
// These tests verify that style functions use the correct palette constants,
// catching accidental hardcoded hex drift. TrueColor is forced in TestMain,
// so lipgloss emits 38;2;R;G;B ANSI codes we can assert on.

func TestFilePickerCursorStyle_IsDistinctFromAssistantBorder(t *testing.T) {
	cursor := FilePickerCursorStyle().Render("item")
	assistant := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAssistantBorder)).Render("item")
	if cursor == assistant {
		t.Error("FilePickerCursorStyle should not use ColorAssistantBorder — green means 'assistant message', not 'selected'")
	}
}

func TestWaitingStyle_DoesNotUseOldGold(t *testing.T) {
	rendered := WaitingStyle().Render("What should monkey do?")
	// #D4A017 = rgb(212, 160, 23) — the old hardcoded value to eliminate
	if strings.Contains(rendered, "212;160;23") {
		t.Errorf("WaitingStyle should not use old #D4A017 gold (212;160;23); use ColorAccent instead")
	}
}

// ColorAssistantBorder = #729B2F → emits 113;155;47
func TestToolApprovalModelStyle_UsesAssistantBorderColor(t *testing.T) {
	rendered := ToolApprovalModelStyle().Render("claude-sonnet")
	if !strings.Contains(rendered, "113;155;47") {
		t.Errorf("ToolApprovalModelStyle should use ColorAssistantBorder (%s → 113;155;47), got ANSI: %q", ColorAssistantBorder, rendered)
	}
}

// ColorToolBorder = #225057 → emits 34;80;87
func TestToolApprovalToolStyle_UsesToolBorderColor(t *testing.T) {
	rendered := ToolApprovalToolStyle().Render("bash")
	if !strings.Contains(rendered, "34;80;87") {
		t.Errorf("ToolApprovalToolStyle should use ColorToolBorder (%s → 34;80;87), got ANSI: %q", ColorToolBorder, rendered)
	}
}

// ColorGrayMid = #888888 = rgb(136, 136, 136)
func TestToolApprovalPreviewStyle_HasExplicitForeground(t *testing.T) {
	rendered := ToolApprovalPreviewStyle().Render("rm -rf /")
	if !strings.Contains(rendered, "136;136;136") {
		t.Errorf("ToolApprovalPreviewStyle should have explicit foreground ColorGrayMid (%s = 136;136;136), got ANSI: %q", ColorGrayMid, rendered)
	}
}
