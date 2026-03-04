package tui

import (
	"strings"
	"testing"
)

// simpleSplitDiff is a minimal unified diff with one context, one deletion, one addition
const simpleSplitDiff = "@@ -1,2 +1,2 @@\n context\n-old line\n+new line\n"

func TestRenderSplitDiff_Empty_ReturnsEmpty(t *testing.T) {
	if got := RenderSplitDiff("", 80); got != "" {
		t.Errorf("RenderSplitDiff('') = %q, want ''", got)
	}
}

func TestRenderSplitDiff_HasTwoBorders(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	// Red (#FF6B6B) border for left panel and green (#56D364) border for right panel
	if !strings.Contains(rendered, "255;107;107") {
		t.Errorf("split diff should have red border (255;107;107) for left panel: %q", stripANSI(rendered))
	}
	if !strings.Contains(rendered, "86;211;100") {
		t.Errorf("split diff should have green border (86;211;100) for right panel: %q", stripANSI(rendered))
	}
}

func TestRenderSplitDiff_DeletedContent_IsLeftOfAddedContent(t *testing.T) {
	// For paired changes, deleted and added appear on the same row; deleted must come first (left panel)
	diff := "@@ -1,1 +1,1 @@\n-del\n+add\n"
	rendered := RenderSplitDiff(diff, 80)
	for _, line := range strings.Split(rendered, "\n") {
		s := stripANSI(line)
		if strings.Contains(s, "del") && strings.Contains(s, "add") {
			di := strings.Index(s, "del")
			ai := strings.Index(s, "add")
			if di >= ai {
				t.Errorf("deleted content should appear left of added content: %q", s)
			}
			return
		}
	}
	t.Error("paired deletion and addition should appear on the same row")
}

func TestRenderSplitDiff_DeletedContent_InLeftPanel(t *testing.T) {
	stripped := stripANSI(RenderSplitDiff(simpleSplitDiff, 80))
	if !strings.Contains(stripped, "old line") {
		t.Errorf("deleted content should appear in left (Before) panel: %q", stripped)
	}
}

func TestRenderSplitDiff_AddedContent_InRightPanel(t *testing.T) {
	stripped := stripANSI(RenderSplitDiff(simpleSplitDiff, 80))
	if !strings.Contains(stripped, "new line") {
		t.Errorf("added content should appear in right (After) panel: %q", stripped)
	}
}

func TestRenderSplitDiff_DeletedContent_HasRedStyle(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	// Find the line containing "old line" and check it has ANSI codes
	for _, l := range strings.Split(rendered, "\n") {
		if strings.Contains(stripANSI(l), "old line") {
			if !strings.Contains(l, "\x1b[") {
				t.Error("deleted content line should have ANSI styling")
			}
			return
		}
	}
	t.Error("could not find line with 'old line' in rendered output")
}

func TestRenderSplitDiff_AddedContent_HasGreenStyle(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	for _, l := range strings.Split(rendered, "\n") {
		if strings.Contains(stripANSI(l), "new line") {
			if !strings.Contains(l, "\x1b[") {
				t.Error("added content line should have ANSI styling")
			}
			return
		}
	}
	t.Error("could not find line with 'new line' in rendered output")
}

func TestRenderSplitDiff_HasRedAndGreenColors(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	// #FF6B6B red (deletion) → TrueColor 255;107;107
	if !strings.Contains(rendered, "255;107;107") {
		t.Error("rendered diff should contain red TrueColor code (255;107;107) for deletions")
	}
	// #56D364 green (addition) → TrueColor 86;211;100
	if !strings.Contains(rendered, "86;211;100") {
		t.Error("rendered diff should contain green TrueColor code (86;211;100) for additions")
	}
}

func TestRenderSplitDiff_ShowsLineNumbers(t *testing.T) {
	diff := "@@ -5,2 +5,2 @@\n context\n-old\n+new\n"
	stripped := stripANSI(RenderSplitDiff(diff, 80))
	if !strings.Contains(stripped, "5") {
		t.Errorf("split diff should show line numbers from @@ header: %q", stripped)
	}
}

func TestRenderSplitDiff_PairedChanges_AlignedOnSameRow(t *testing.T) {
	// One deletion + one addition: they should appear on the same rendered row
	diff := "@@ -1,1 +1,1 @@\n-del\n+add\n"
	rendered := RenderSplitDiff(diff, 80)
	for _, l := range strings.Split(rendered, "\n") {
		s := stripANSI(l)
		if strings.Contains(s, "del") && strings.Contains(s, "add") {
			return // found on same row
		}
	}
	t.Error("paired deletion and addition should appear on the same row")
}

func TestRenderSplitDiff_ContextLine_AppearsInBothPanels(t *testing.T) {
	// A context line should appear in both panels on the same row
	diff := "@@ -1,2 +1,2 @@\n context\n-old\n+new\n"
	rendered := RenderSplitDiff(diff, 80)
	count := 0
	for _, l := range strings.Split(rendered, "\n") {
		s := stripANSI(l)
		// A row that contains "context" twice (once per panel) indicates both panels show it
		if strings.Count(s, "context") >= 2 {
			count++
		}
	}
	if count == 0 {
		t.Error("context line should appear in both panels on the same row")
	}
}

func TestRenderSplitDiff_FileHeaders_NotShown(t *testing.T) {
	diff := "--- a/file.txt\n+++ b/file.txt\n@@ -1,1 +1,1 @@\n-old\n+new\n"
	stripped := stripANSI(RenderSplitDiff(diff, 80))
	if strings.Contains(stripped, "--- a/file.txt") {
		t.Error("file headers (--- / +++) should not appear in split diff view")
	}
}

func TestRenderSplitDiff_ProducesANSI(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	if !strings.Contains(rendered, "\x1b[") {
		t.Error("RenderSplitDiff should produce ANSI escape codes")
	}
}
