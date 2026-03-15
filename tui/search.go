package tui

import (
	"strings"
)

// SearchBar provides incremental search over conversation messages.
// Ctrl+F activates it; Ctrl+N / Ctrl+P cycle matches; Esc dismisses.
type SearchBar struct {
	active   bool
	query    string
	matches  []int        // message indices that match the current query (ordered)
	matchSet map[int]bool // set view of matches for O(1) IsMatch lookups
	cursor   int          // position within matches
}

// NewSearchBar returns an inactive SearchBar.
func NewSearchBar() SearchBar { return SearchBar{} }

// Activate makes the search bar visible.
func (s *SearchBar) Activate() { s.active = true }

// Deactivate hides the search bar and clears state.
func (s *SearchBar) Deactivate() {
	s.active = false
	s.query = ""
	s.matches = nil
	s.matchSet = nil
	s.cursor = 0
}

// IsActive reports whether the search bar is visible.
func (s SearchBar) IsActive() bool { return s.active }

// Query returns the current search string.
func (s SearchBar) Query() string { return s.query }

// SetQuery updates the search query and recomputes matches from messages.
// An empty query clears all matches.
func (s *SearchBar) SetQuery(q string, messages []Message) {
	s.query = q
	s.matches = nil
	s.matchSet = nil
	s.cursor = 0
	if q == "" {
		return
	}
	lower := strings.ToLower(q)
	s.matchSet = make(map[int]bool)
	for i, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Content), lower) {
			s.matches = append(s.matches, i)
			s.matchSet[i] = true
		}
	}
}

// MatchCount returns the number of matching messages.
func (s SearchBar) MatchCount() int { return len(s.matches) }

// CurrentMatchIndex returns the message index of the current match, or -1 when
// there are no matches.
func (s SearchBar) CurrentMatchIndex() int {
	if len(s.matches) == 0 {
		return -1
	}
	return s.matches[s.cursor]
}

// IsMatch reports whether message index i is one of the current matches.
func (s SearchBar) IsMatch(i int) bool {
	return s.matchSet[i]
}

// NextMatch advances the cursor to the next match, wrapping around.
func (s *SearchBar) NextMatch() {
	if len(s.matches) == 0 {
		return
	}
	s.cursor = (s.cursor + 1) % len(s.matches)
}

// PrevMatch retreats the cursor to the previous match, wrapping around.
func (s *SearchBar) PrevMatch() {
	if len(s.matches) == 0 {
		return
	}
	s.cursor = (s.cursor - 1 + len(s.matches)) % len(s.matches)
}
