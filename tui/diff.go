package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var hunkHeaderRe = regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// RenderSplitDiff renders a unified diff as two side-by-side panels:
// "Before" (left, red border) shows context + deleted lines,
// "After" (right, green border) shows context + added lines.
// Paired deletion/addition chunks are aligned on the same row.
func RenderSplitDiff(diff string, width int) string {
	if diff == "" {
		return ""
	}

	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDiffDel))
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDiffAdd))
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDiffHunk)).Bold(true)
	gutterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDiffGutter))

	blankGutter := gutterStyle.Render("     │")

	type entry struct{ left, right string }
	var entries []entry

	type pending struct {
		gutter  string
		content string
	}
	var pendingDels []pending

	flushDels := func() {
		for _, p := range pendingDels {
			entries = append(entries, entry{
				left:  p.gutter + delStyle.Render(p.content),
				right: blankGutter,
			})
		}
		pendingDels = nil
	}

	oldLine, newLine := 1, 1

	lines := strings.Split(diff, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			// skip file headers — not useful in the split view

		case strings.HasPrefix(line, "@@"):
			flushDels()
			if m := hunkHeaderRe.FindStringSubmatch(line); m != nil {
				if n, err := strconv.Atoi(m[1]); err == nil {
					oldLine = n
				}
				if n, err := strconv.Atoi(m[2]); err == nil {
					newLine = n
				}
			}
			h := hunkStyle.Render(line)
			entries = append(entries, entry{left: h, right: h})

		case strings.HasPrefix(line, "-"):
			g := gutterStyle.Render(fmt.Sprintf("%4d │", oldLine))
			oldLine++
			pendingDels = append(pendingDels, pending{gutter: g, content: line[1:]})

		case strings.HasPrefix(line, "+"):
			g := gutterStyle.Render(fmt.Sprintf("%4d │", newLine))
			newLine++
			if len(pendingDels) > 0 {
				// pair with the earliest pending deletion → same row
				p := pendingDels[0]
				pendingDels = pendingDels[1:]
				entries = append(entries, entry{
					left:  p.gutter + delStyle.Render(p.content),
					right: g + addStyle.Render(line[1:]),
				})
			} else {
				entries = append(entries, entry{
					left:  blankGutter,
					right: g + addStyle.Render(line[1:]),
				})
			}

		default: // context line — starts with a space in unified diff
			flushDels()
			content := ""
			if len(line) > 0 {
				content = line[1:]
			}
			lg := gutterStyle.Render(fmt.Sprintf("%4d │", oldLine))
			rg := gutterStyle.Render(fmt.Sprintf("%4d │", newLine))
			oldLine++
			newLine++
			entries = append(entries, entry{left: lg + content, right: rg + content})
		}
	}
	flushDels()

	leftLines := make([]string, len(entries))
	rightLines := make([]string, len(entries))
	for i, e := range entries {
		leftLines[i] = e.left
		rightLines[i] = e.right
	}

	panelW := max(10, (width-10)/2)

	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorDiffDel)).
		Width(panelW).
		Render(strings.Join(leftLines, "\n"))

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorDiffAdd)).
		Width(panelW).
		Render(strings.Join(rightLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}
