package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const historyFile = ".monkey_history" // legacy fallback, not used by LoadHistory

// History manages prompt history persisted to a local file.
// Navigation mimics bash: Up moves to older entries, Down moves to newer ones.
type History struct {
	entries []string
	cursor  int    // len(entries) means "at current draft"
	draft   string // saved input when the user first navigates up
	path    string
}

// loadHistoryFromPath loads history from the given file, creating it if missing.
func loadHistoryFromPath(path string) History {
	h := History{path: path}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return h
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			h.entries = append(h.entries, line)
		}
	}
	h.cursor = len(h.entries)
	return h
}

// LoadHistory loads history from ~/.config/monkey/history (respecting
// XDG_DATA_HOME), creating it if it does not exist.
func LoadHistory() History {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return loadHistoryFromPath(historyFile)
		}
		dir = filepath.Join(home, ".config", "monkey")
	} else {
		dir = filepath.Join(dir, "monkey")
	}
	_ = os.MkdirAll(dir, 0o700)
	return loadHistoryFromPath(filepath.Join(dir, "history"))
}

// Add appends an entry to the history and persists it to disk.
// Blank entries and consecutive duplicates are ignored.
func (h *History) Add(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry {
		h.reset()
		return
	}
	h.entries = append(h.entries, entry)
	h.reset()

	if h.path != "" {
		f, err := os.OpenFile(h.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintln(f, entry)
			_ = f.Close()
		}
	}
}

// Up moves to the previous (older) entry and returns the text to place in the
// input box. currentInput is saved as the draft the first time Up is pressed.
func (h *History) Up(currentInput string) string {
	if len(h.entries) == 0 {
		return currentInput
	}
	if h.cursor == len(h.entries) {
		h.draft = currentInput
	}
	if h.cursor > 0 {
		h.cursor--
	}
	return h.entries[h.cursor]
}

// Down moves to the next (newer) entry, returning the saved draft when the end
// is reached.
func (h *History) Down() string {
	if h.cursor < len(h.entries) {
		h.cursor++
	}
	if h.cursor == len(h.entries) {
		return h.draft
	}
	return h.entries[h.cursor]
}

func (h *History) reset() {
	h.cursor = len(h.entries)
	h.draft = ""
}
