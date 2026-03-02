package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	// Force TrueColor so lipgloss always emits ANSI codes in tests
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
