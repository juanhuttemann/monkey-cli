package tui

import (
	"os"
	"regexp"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from s, useful for asserting text
// content in views that may contain ANSI styling from glamour or lipgloss.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func TestMain(m *testing.M) {
	// Force TrueColor so lipgloss always emits ANSI codes in tests
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
