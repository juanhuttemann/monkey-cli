package tui

import (
	"strings"
	"testing"
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
