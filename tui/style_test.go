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

func TestRenderIntroBlock_HasBorder(t *testing.T) {
	rendered := RenderIntroBlock(80, "Monkey", "", "ascii art")
	if len(rendered) <= len("ascii art") {
		t.Error("RenderIntroBlock should add a border")
	}
}

func TestRenderIntroBlock_TitleAppearsInBorder(t *testing.T) {
	rendered := RenderIntroBlock(80, "Monkey", "", "ascii art")
	if !strings.Contains(stripANSI(rendered), "Monkey") {
		t.Error("RenderIntroBlock should include the title in the border")
	}
}

func TestRenderIntroBlock_ShowsHelpText(t *testing.T) {
	rendered := RenderIntroBlock(80, "Monkey", "", "ascii art")
	if !strings.Contains(stripANSI(rendered), "Type ? for help") {
		t.Error("RenderIntroBlock should show 'Type ? for help' in the right panel")
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

func TestRenderAssistantBlock_ShowsModelInBorder(t *testing.T) {
	rendered := RenderAssistantBlock(80, "claude-sonnet-4-5", "response text")
	if !strings.Contains(stripANSI(rendered), "claude-sonnet-4-5") {
		t.Error("RenderAssistantBlock should show the model name in the top border")
	}
}

func TestRenderAssistantBlock_ShowsContent(t *testing.T) {
	rendered := RenderAssistantBlock(80, "claude-sonnet-4-5", "hello world")
	if !strings.Contains(stripANSI(rendered), "hello world") {
		t.Error("RenderAssistantBlock should include the message content")
	}
}

func TestRenderAssistantBlock_HasBorder(t *testing.T) {
	rendered := RenderAssistantBlock(80, "claude-sonnet-4-5", "text")
	if len(rendered) <= len("text") {
		t.Error("RenderAssistantBlock should add a border around the content")
	}
}

func TestRenderAssistantBlock_HasANSI(t *testing.T) {
	rendered := RenderAssistantBlock(80, "claude-sonnet-4-5", "text")
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("RenderAssistantBlock should produce ANSI escape codes for styling")
	}
}
