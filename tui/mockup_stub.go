//go:build !dev

package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleDevSlashCommand(cmd string, cleanup bool) (Model, tea.Cmd, bool) {
	return m, nil, false
}
