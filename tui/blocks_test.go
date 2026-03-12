package tui

import (
	"strings"
	"testing"
)

func TestRenderIntroBlock_HasBorder(t *testing.T) {
	rendered := RenderIntroBlock(80, "Monkey", "", "ascii art")
	if len(rendered) <= len("ascii art") {
		t.Error("RenderIntroBlock should add a border")
	}
}

func TestRenderIntroBlock_TitleAppearsInRightPanel(t *testing.T) {
	art := "line1\nline2\nline3\nline4\nline5"
	rendered := RenderIntroBlock(80, "Monkey", "", art)
	if !strings.Contains(stripANSI(rendered), "Monkey") {
		t.Error("RenderIntroBlock should show the title in the right panel")
	}
}

func TestRenderIntroBlock_ShowsHelpText(t *testing.T) {
	art := "line1\nline2\nline3\nline4\nline5"
	rendered := RenderIntroBlock(80, "Monkey", "", art)
	if !strings.Contains(stripANSI(rendered), "Type ? for help") {
		t.Error("RenderIntroBlock should show 'Type ? for help' in the right panel")
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

func TestRenderUserBlock_ShowsYouInBorder(t *testing.T) {
	rendered := RenderUserBlock(80, "hello")
	if !strings.Contains(stripANSI(rendered), "You") {
		t.Error("RenderUserBlock should show 'You' in the border")
	}
}

func TestRenderUserBlock_YouInTopBorder(t *testing.T) {
	rendered := RenderUserBlock(80, "hello")
	stripped := stripANSI(rendered)
	lines := strings.Split(stripped, "\n")
	firstLine := lines[0]
	if !strings.Contains(firstLine, "You") {
		t.Errorf("RenderUserBlock 'You' label should be in the top border, got: %q", firstLine)
	}
}

func TestRenderUserBlock_ShowsContent(t *testing.T) {
	rendered := RenderUserBlock(80, "hello world")
	if !strings.Contains(stripANSI(rendered), "hello world") {
		t.Error("RenderUserBlock should include the message content")
	}
}

func TestRenderUserBlock_HasBorder(t *testing.T) {
	rendered := RenderUserBlock(80, "text")
	if len(rendered) <= len("text") {
		t.Error("RenderUserBlock should add a border around the content")
	}
}

func TestRenderUserBlock_HasANSI(t *testing.T) {
	rendered := RenderUserBlock(80, "text")
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("RenderUserBlock should produce ANSI escape codes for styling")
	}
}

func TestRenderToolBlock_ShowsToolNameInBorder(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "$ ls")
	if !strings.Contains(stripANSI(rendered), "bash") {
		t.Error("RenderToolBlock should show tool name in the top border")
	}
}

func TestRenderToolBlock_NoWrenchEmoji(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "$ ls")
	if strings.Contains(stripANSI(rendered), "🔧") {
		t.Error("RenderToolBlock should not show the wrench emoji")
	}
}

func TestRenderToolBlock_ShowsContent(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "$ ls -la")
	if !strings.Contains(stripANSI(rendered), "$ ls -la") {
		t.Error("RenderToolBlock should include the message content")
	}
}

func TestRenderToolBlock_HasBorder(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "text")
	if len(rendered) <= len("text") {
		t.Error("RenderToolBlock should add a border around the content")
	}
}

func TestRenderToolBlock_HasANSI(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "text")
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("RenderToolBlock should produce ANSI escape codes for styling")
	}
}

// --- Palette: message block border colors ---
// Each block's border ANSI code is asserted against the palette value.

// ColorUserBorder = #8B3A21 → emits 139;58;32
func TestRenderUserBlock_UsesRustBrownBorder(t *testing.T) {
	rendered := RenderUserBlock(80, "hello")
	if !strings.Contains(rendered, "139;58;32") {
		t.Errorf("RenderUserBlock border should use Rust Brown (%s → 139;58;32), got ANSI: %q", ColorUserBorder, rendered)
	}
}

// ColorAssistantBorder = #729B2F → emits 113;155;47
func TestRenderAssistantBlock_UsesLeafGreenBorder(t *testing.T) {
	rendered := RenderAssistantBlock(80, "model", "response")
	if !strings.Contains(rendered, "113;155;47") {
		t.Errorf("RenderAssistantBlock border should use Leaf Green (%s → 113;155;47), got ANSI: %q", ColorAssistantBorder, rendered)
	}
}

// ColorToolBorder = #225057 = rgb(34,80,87)
func TestRenderToolBlock_UsesDarkTealBorder(t *testing.T) {
	rendered := RenderToolBlock(80, "bash", "$ ls")
	if !strings.Contains(rendered, "34;80;87") {
		t.Errorf("RenderToolBlock border should use Dark Slate/Teal (%s = 34;80;87), got ANSI: %q", ColorToolBorder, rendered)
	}
}

func TestColorizeArt_AppliesColorsToBlockChars(t *testing.T) {
	input := "█▒░ text"
	result := colorizeArt(input)

	// ANSI codes should be present (coloring applied)
	if !strings.Contains(result, "\x1b[") {
		t.Error("colorizeArt should apply ANSI colors to block chars")
	}
	// Plain text and spaces should pass through
	if !strings.Contains(stripANSI(result), " text") {
		t.Error("colorizeArt should preserve non-block characters")
	}
	// All three special chars should appear in output
	if !strings.Contains(stripANSI(result), "█") {
		t.Error("colorizeArt should include █ in output")
	}
	if !strings.Contains(stripANSI(result), "▒") {
		t.Error("colorizeArt should include ▒ in output")
	}
	if !strings.Contains(stripANSI(result), "░") {
		t.Error("colorizeArt should include ░ in output")
	}
}

func TestColorizeArt_EmptyString(t *testing.T) {
	result := colorizeArt("")
	if result != "" {
		t.Errorf("colorizeArt('') = %q, want ''", result)
	}
}

func TestRenderIntroBlock_UsesPrimaryDividerColor(t *testing.T) {
	rendered := RenderIntroBlock(80, "Monkey", "", "ascii art")
	if !strings.Contains(rendered, "70;25;20") {
		t.Errorf("RenderIntroBlock divider should use ColorPrimary (%s → 70;25;20), got ANSI: %q", ColorPrimary, rendered)
	}
}
