package tui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRenderMarkdown_NoLeadingBlankLine(t *testing.T) {
	result := stripANSI(RenderMarkdown("hello world", 80))
	if strings.HasPrefix(result, "\n") {
		t.Errorf("RenderMarkdown should not start with a blank line, got: %q", result[:min(len(result), 20)])
	}
}

func TestRenderMarkdown_NoLeftIndent(t *testing.T) {
	result := stripANSI(RenderMarkdown("hello world", 80))
	for _, line := range strings.Split(result, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") {
			t.Errorf("RenderMarkdown should not indent content, got line: %q", line)
		}
		break // only check first non-empty line
	}
}

func TestRenderMarkdown_PreservesPlainText(t *testing.T) {
	result := RenderMarkdown("hello world", 80)
	if !strings.Contains(stripANSI(result), "hello world") {
		t.Errorf("RenderMarkdown should preserve plain text, got: %q", result)
	}
}

func TestRenderMarkdown_BoldText_ContainsContent(t *testing.T) {
	result := RenderMarkdown("**bold text**", 80)
	if !strings.Contains(result, "bold text") {
		t.Errorf("RenderMarkdown should contain bold text content, got: %q", result)
	}
}

func TestRenderMarkdown_BoldText_HasANSI(t *testing.T) {
	result := RenderMarkdown("**bold text**", 80)
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("RenderMarkdown should produce ANSI codes for bold text, got: %q", result)
	}
}

func TestRenderMarkdown_InlineCode_ContainsContent(t *testing.T) {
	result := RenderMarkdown("`code snippet`", 80)
	if !strings.Contains(result, "code snippet") {
		t.Errorf("RenderMarkdown should contain inline code content, got: %q", result)
	}
}

func TestRenderMarkdown_CodeBlock_ContainsContent(t *testing.T) {
	result := RenderMarkdown("```\ncode here\n```", 80)
	if !strings.Contains(result, "code here") {
		t.Errorf("RenderMarkdown should contain code block content, got: %q", result)
	}
}

func TestRenderMarkdown_Header_ContainsContent(t *testing.T) {
	result := RenderMarkdown("# Section Header", 80)
	if !strings.Contains(stripANSI(result), "Section Header") {
		t.Errorf("RenderMarkdown should contain heading content, got: %q", result)
	}
}

func TestRenderMarkdown_ZeroWidth_FallsBack(t *testing.T) {
	result := RenderMarkdown("some text", 0)
	if !strings.Contains(result, "some text") {
		t.Errorf("RenderMarkdown with zero width should fall back to plain text, got: %q", result)
	}
}

func TestRenderMarkdown_EmptyContent(t *testing.T) {
	// Should not panic
	_ = RenderMarkdown("", 80)
}

func TestRenderMarkdown_NoBackgroundColors(t *testing.T) {
	// Headings in glamour's dark style carry a background_color. It must be
	// stripped so it doesn't bleed through the lipgloss border background.
	result := RenderMarkdown("# Heading\n\ntext\n\n```go\ncode\n```", 80)

	ansiSeqRe := regexp.MustCompile(`\x1b\[([0-9;]*)m`)
	for _, match := range ansiSeqRe.FindAllStringSubmatch(result, -1) {
		for _, param := range strings.Split(match[1], ";") {
			n, _ := strconv.Atoi(param)
			if n == 48 || (n >= 40 && n <= 47) || (n >= 100 && n <= 107) {
				t.Errorf("RenderMarkdown output contains background ANSI code %d in sequence %q", n, match[0])
				return
			}
		}
	}
}

func TestView_AssistantMessage_UsesMarkdown(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.AddMessage("assistant", "**important message**")

	view := model.View()

	if !strings.Contains(stripANSI(view), "important message") {
		t.Error("View() should contain assistant markdown content")
	}
	if !strings.Contains(view, "\x1b[") {
		t.Error("View() should contain ANSI codes from markdown rendering")
	}
}
