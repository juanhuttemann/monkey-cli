package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- SearchBar unit tests ---

func TestSearchBar_InactiveByDefault(t *testing.T) {
	sb := NewSearchBar()
	if sb.IsActive() {
		t.Error("NewSearchBar().IsActive() = true, want false")
	}
}

func TestSearchBar_ActivateDeactivate(t *testing.T) {
	sb := NewSearchBar()
	sb.Activate()
	if !sb.IsActive() {
		t.Error("After Activate(), IsActive() = false, want true")
	}
	sb.Deactivate()
	if sb.IsActive() {
		t.Error("After Deactivate(), IsActive() = true, want false")
	}
}

func TestSearchBar_SetQuery_EmptyMatchesNothing(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "hello world", Timestamp: time.Now()},
		{Role: "assistant", Content: "hi there", Timestamp: time.Now()},
	}
	sb.SetQuery("", msgs)
	if sb.MatchCount() != 0 {
		t.Errorf("MatchCount with empty query = %d, want 0", sb.MatchCount())
	}
}

func TestSearchBar_SetQuery_FindsMatches(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "hello world", Timestamp: time.Now()},
		{Role: "assistant", Content: "hi there", Timestamp: time.Now()},
		{Role: "user", Content: "hello again", Timestamp: time.Now()},
	}
	sb.SetQuery("hello", msgs)
	if sb.MatchCount() != 2 {
		t.Errorf("MatchCount('hello') = %d, want 2", sb.MatchCount())
	}
}

func TestSearchBar_SetQuery_CaseInsensitive(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "Hello World", Timestamp: time.Now()},
		{Role: "assistant", Content: "HELLO again", Timestamp: time.Now()},
	}
	sb.SetQuery("hello", msgs)
	if sb.MatchCount() != 2 {
		t.Errorf("MatchCount (case-insensitive) = %d, want 2", sb.MatchCount())
	}
}

func TestSearchBar_NextMatch_CyclesForward(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "foo", Timestamp: time.Now()},
		{Role: "user", Content: "bar", Timestamp: time.Now()},
		{Role: "user", Content: "foo", Timestamp: time.Now()},
	}
	sb.SetQuery("foo", msgs)
	first := sb.CurrentMatchIndex()
	sb.NextMatch()
	second := sb.CurrentMatchIndex()
	if first == second {
		t.Error("NextMatch() did not advance cursor")
	}
}

func TestSearchBar_NextMatch_WrapsAround(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "foo", Timestamp: time.Now()},
		{Role: "user", Content: "foo", Timestamp: time.Now()},
	}
	sb.SetQuery("foo", msgs)
	first := sb.CurrentMatchIndex()
	sb.NextMatch()
	sb.NextMatch() // wrap back to first
	wrapped := sb.CurrentMatchIndex()
	if first != wrapped {
		t.Errorf("NextMatch did not wrap: first=%d, after 2 nexts=%d", first, wrapped)
	}
}

func TestSearchBar_PrevMatch_CyclesBackward(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "foo", Timestamp: time.Now()},
		{Role: "user", Content: "foo", Timestamp: time.Now()},
	}
	sb.SetQuery("foo", msgs)
	first := sb.CurrentMatchIndex()
	sb.PrevMatch()
	prev := sb.CurrentMatchIndex()
	if first == prev {
		t.Errorf("PrevMatch() did not change cursor: %d", first)
	}
}

func TestSearchBar_CurrentMatchIndex_NoMatches_ReturnsNegative(t *testing.T) {
	sb := NewSearchBar()
	sb.SetQuery("zzz", []Message{{Role: "user", Content: "hello"}})
	if sb.CurrentMatchIndex() != -1 {
		t.Errorf("CurrentMatchIndex with no matches = %d, want -1", sb.CurrentMatchIndex())
	}
}

func TestSearchBar_IsMatch_ReturnsTrueForMatchingMessages(t *testing.T) {
	sb := NewSearchBar()
	msgs := []Message{
		{Role: "user", Content: "hello world", Timestamp: time.Now()},
		{Role: "user", Content: "goodbye", Timestamp: time.Now()},
	}
	sb.SetQuery("hello", msgs)
	if !sb.IsMatch(0) {
		t.Error("IsMatch(0) = false, want true for 'hello world'")
	}
	if sb.IsMatch(1) {
		t.Error("IsMatch(1) = true, want false for 'goodbye'")
	}
}

// --- Ctrl+F integration ---

func TestUpdate_CtrlF_ActivatesSearch(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("user", "hello")

	ctrlF := tea.KeyMsg{Type: tea.KeyCtrlF}
	updated, _ := m.Update(ctrlF)

	result := updated.(Model)
	if !result.searchBar.IsActive() {
		t.Error("Ctrl+F should activate search bar")
	}
}

func TestUpdate_CtrlF_WhenSearchActive_Deactivates(t *testing.T) {
	m := NewModel(nil)
	m.searchBar.Activate()

	ctrlF := tea.KeyMsg{Type: tea.KeyCtrlF}
	updated, _ := m.Update(ctrlF)

	result := updated.(Model)
	if result.searchBar.IsActive() {
		t.Error("Ctrl+F when search active should deactivate it")
	}
}

func TestUpdate_Esc_DismissesSearch(t *testing.T) {
	m := NewModel(nil)
	m.searchBar.Activate()
	m.searchBar.SetQuery("foo", m.messages)

	esc := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(esc)

	result := updated.(Model)
	if result.searchBar.IsActive() {
		t.Error("Esc should dismiss search bar")
	}
}

func TestUpdate_CtrlN_AdvancesMatch(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("user", "foo")
	m.AddMessage("user", "bar")
	m.AddMessage("user", "foo")
	m.searchBar.Activate()
	m.searchBar.SetQuery("foo", m.messages)
	first := m.searchBar.CurrentMatchIndex()

	ctrlN := tea.KeyMsg{Type: tea.KeyCtrlN}
	updated, _ := m.Update(ctrlN)

	result := updated.(Model)
	if result.searchBar.CurrentMatchIndex() == first {
		t.Error("Ctrl+N should advance to next match")
	}
}

func TestUpdate_CtrlP_RetreatsMatch(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("user", "foo")
	m.AddMessage("user", "foo")
	m.searchBar.Activate()
	m.searchBar.SetQuery("foo", m.messages)
	first := m.searchBar.CurrentMatchIndex()

	ctrlP := tea.KeyMsg{Type: tea.KeyCtrlP}
	updated, _ := m.Update(ctrlP)

	result := updated.(Model)
	if result.searchBar.CurrentMatchIndex() == first {
		t.Error("Ctrl+P should retreat to previous match")
	}
}

// --- Rendering: search-active messages get visual indicator ---

func TestRenderMessages_SearchActive_MatchHighlighted(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)
	m.AddMessage("user", "hello world")
	m.AddMessage("user", "goodbye")
	m.searchBar.Activate()
	m.searchBar.SetQuery("hello", m.messages)

	rendered := m.renderMessages()
	// The "hello" match should appear in the rendered output
	if !strings.Contains(rendered, "hello") {
		t.Error("rendered output should contain the matched content")
	}
}
