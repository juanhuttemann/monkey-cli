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
	// ColorDiffDel = ColorErrorBorder = #BA3F28 → emits 186;63;40
	if !strings.Contains(rendered, "186;63;40") {
		t.Errorf("split diff should have burnt-orange border (186;63;40) for left panel: %q", stripANSI(rendered))
	}
	// ColorDiffAdd = ColorAssistantBorder = #729B2F → emits 113;155;47
	if !strings.Contains(rendered, "113;155;47") {
		t.Errorf("split diff should have leaf-green border (113;155;47) for right panel: %q", stripANSI(rendered))
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
	// ColorDiffDel = ColorErrorBorder = #BA3F28 → emits 186;63;40
	if !strings.Contains(rendered, "186;63;40") {
		t.Error("rendered diff should contain burnt-orange TrueColor code (186;63;40) for deletions")
	}
	// ColorDiffAdd = ColorAssistantBorder = #729B2F → emits 113;155;47
	if !strings.Contains(rendered, "113;155;47") {
		t.Error("rendered diff should contain leaf-green TrueColor code (113;155;47) for additions")
	}
}

func TestRenderSplitDiff_HunkHeader_UsesToolBorderColor(t *testing.T) {
	rendered := RenderSplitDiff(simpleSplitDiff, 80)
	// ColorDiffHunk = ColorToolBorder = #225057 → emits 34;80;87
	if !strings.Contains(rendered, "34;80;87") {
		t.Errorf("hunk header should use ColorToolBorder (34;80;87), got ANSI: %q", rendered)
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
