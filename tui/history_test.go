package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// historyWithEntries builds a History pre-loaded with entries (no file I/O).
func historyWithEntries(entries ...string) History {
	h := History{entries: make([]string, len(entries))}
	copy(h.entries, entries)
	h.cursor = len(h.entries)
	return h
}

// TestLoadHistory_CreatesFileIfMissing verifies that LoadHistory creates the
// history file when it does not exist yet.
func TestLoadHistory_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h := loadHistoryFromPath(path)

	if h.path != path {
		t.Errorf("path = %q, want %q", h.path, path)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("history file should have been created")
	}
}

// TestLoadHistory_LoadsExistingEntries verifies that existing lines are loaded.
func TestLoadHistory_LoadsExistingEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	if err := os.WriteFile(path, []byte("first\nsecond\nthird\n"), 0600); err != nil {
		t.Fatal(err)
	}

	h := loadHistoryFromPath(path)

	if len(h.entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(h.entries))
	}
	if h.entries[0] != "first" || h.entries[1] != "second" || h.entries[2] != "third" {
		t.Errorf("entries = %v, want [first second third]", h.entries)
	}
}

// TestLoadHistory_CursorStartsAtEnd verifies the cursor is past the last entry.
func TestLoadHistory_CursorStartsAtEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0600); err != nil {
		t.Fatal(err)
	}

	h := loadHistoryFromPath(path)

	if h.cursor != len(h.entries) {
		t.Errorf("cursor = %d, want %d (past last entry)", h.cursor, len(h.entries))
	}
}

// TestHistory_Add_AppendsPersists verifies Add stores the entry in memory and on disk.
func TestHistory_Add_AppendsPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h := loadHistoryFromPath(path)
	h.Add("hello world")

	if len(h.entries) != 1 || h.entries[0] != "hello world" {
		t.Errorf("entries after Add = %v, want [hello world]", h.entries)
	}

	// Reload and confirm persistence.
	h2 := loadHistoryFromPath(path)
	if len(h2.entries) != 1 || h2.entries[0] != "hello world" {
		t.Errorf("reloaded entries = %v, want [hello world]", h2.entries)
	}
}

// TestHistory_Add_IgnoresEmptyEntry verifies blank entries are not stored.
func TestHistory_Add_IgnoresEmptyEntry(t *testing.T) {
	h := historyWithEntries()
	h.Add("")
	h.Add("   ")

	if len(h.entries) != 0 {
		t.Errorf("entries after empty Add = %v, want []", h.entries)
	}
}

// TestHistory_Add_IgnoresConsecutiveDuplicate verifies duplicate consecutive entries are skipped.
func TestHistory_Add_IgnoresConsecutiveDuplicate(t *testing.T) {
	h := historyWithEntries("hello")
	h.Add("hello")

	if len(h.entries) != 1 {
		t.Errorf("entries after duplicate Add = %d, want 1", len(h.entries))
	}
}

// TestHistory_Add_AllowsNonConsecutiveDuplicate verifies non-consecutive duplicates are stored.
func TestHistory_Add_AllowsNonConsecutiveDuplicate(t *testing.T) {
	h := historyWithEntries("hello", "world")
	h.Add("hello")

	if len(h.entries) != 3 {
		t.Errorf("entries = %d, want 3", len(h.entries))
	}
}

// TestHistory_Add_ResetsCursor verifies the cursor returns to the end after Add.
func TestHistory_Add_ResetsCursor(t *testing.T) {
	h := historyWithEntries("a", "b")
	h.Up("") // move cursor up
	h.Add("c")

	if h.cursor != len(h.entries) {
		t.Errorf("cursor after Add = %d, want %d", h.cursor, len(h.entries))
	}
}

// TestHistory_Up_ReturnsPreviousEntry verifies Up returns the most recent entry.
func TestHistory_Up_ReturnsPreviousEntry(t *testing.T) {
	h := historyWithEntries("first", "second", "third")

	got := h.Up("current draft")

	if got != "third" {
		t.Errorf("first Up() = %q, want %q", got, "third")
	}
}

// TestHistory_Up_SavesDraft verifies the current input is saved when Up is first pressed.
func TestHistory_Up_SavesDraft(t *testing.T) {
	h := historyWithEntries("first")

	h.Up("my draft")

	if h.draft != "my draft" {
		t.Errorf("draft = %q, want %q", h.draft, "my draft")
	}
}

// TestHistory_Up_ContinuesToOlderEntries verifies repeated Up calls go further back.
func TestHistory_Up_ContinuesToOlderEntries(t *testing.T) {
	h := historyWithEntries("first", "second", "third")
	h.Up("")        // third
	got := h.Up("") // second

	if got != "second" {
		t.Errorf("second Up() = %q, want %q", got, "second")
	}
}

// TestHistory_Up_StopsAtOldest verifies Up stays at the oldest entry when at the top.
func TestHistory_Up_StopsAtOldest(t *testing.T) {
	h := historyWithEntries("only")
	h.Up("")        // "only"
	got := h.Up("") // still "only"

	if got != "only" {
		t.Errorf("Up() past oldest = %q, want %q", got, "only")
	}
}

// TestHistory_Up_EmptyHistory returns the current input unchanged.
func TestHistory_Up_EmptyHistory(t *testing.T) {
	h := historyWithEntries()
	got := h.Up("current")

	if got != "current" {
		t.Errorf("Up() on empty history = %q, want %q", got, "current")
	}
}

// TestHistory_Down_ReturnsNewerEntry verifies Down moves toward the draft.
func TestHistory_Down_ReturnsNewerEntry(t *testing.T) {
	h := historyWithEntries("first", "second", "third")
	h.Up("draft") // third
	h.Up("")      // second
	got := h.Down()

	if got != "third" {
		t.Errorf("Down() = %q, want %q", got, "third")
	}
}

// TestHistory_Down_ReturnsDraftAtEnd verifies Down restores the saved draft.
func TestHistory_Down_ReturnsDraftAtEnd(t *testing.T) {
	h := historyWithEntries("first")
	h.Up("my draft")
	got := h.Down()

	if got != "my draft" {
		t.Errorf("Down() at end = %q, want %q", got, "my draft")
	}
}

// TestHistory_Down_EmptyHistoryReturnsDraft verifies Down on empty history returns draft.
func TestHistory_Down_EmptyHistoryReturnsDraft(t *testing.T) {
	h := historyWithEntries()
	h.draft = "saved"
	got := h.Down()

	if got != "saved" {
		t.Errorf("Down() on empty history = %q, want %q", got, "saved")
	}
}
