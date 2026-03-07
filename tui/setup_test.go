package tui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/juanhuttemann/monkey-cli/api"
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

// newTestClientWithModel creates a minimal api.Client with the given model for testing.
func newTestClientWithModel(model string) *api.Client {
	return api.NewClient("https://api.example.com", "test-key", api.WithModel(model))
}

// containsSubstring reports whether s contains substr (after stripping ANSI codes).
func containsSubstring(s, substr string) bool {
	return strings.Contains(stripANSI(s), substr)
}
