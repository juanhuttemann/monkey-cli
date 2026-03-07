package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FilesLoadedMsg is sent when the async file walk completes.
type FilesLoadedMsg struct {
	Files []string
}

// FilePicker is a keyboard-navigable file selection dropdown.
type FilePicker struct {
	files    []string // all available files
	filtered []string // currently matching files
	cursor   int      // highlighted index into filtered
	active   bool
	width    int
}

const filePickerMaxVisible = 8

// NewFilePicker returns an inactive file picker with the given display width.
func NewFilePicker(width int) FilePicker {
	return FilePicker{width: width}
}

// LoadFilesCmd returns a tea.Cmd that walks the current directory tree and
// collects relative file paths, skipping hidden directories and vendor/node_modules.
func LoadFilesCmd() tea.Cmd {
	return func() tea.Msg {
		var files []string
		_ = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			base := filepath.Base(path)
			if d.IsDir() {
				if path != "." && (base == ".git" || base == "node_modules" || base == "vendor" ||
					strings.HasPrefix(base, ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			files = append(files, path)
			return nil
		})
		return FilesLoadedMsg{Files: files}
	}
}

// SetFiles replaces the full file list and resets filtering.
func (fp *FilePicker) SetFiles(files []string) {
	fp.files = files
	fp.filtered = files
	fp.cursor = 0
}

// SetQuery filters the list by fuzzy-matching each file path against query.
// An empty query shows all files.
func (fp *FilePicker) SetQuery(query string) {
	if query == "" {
		fp.filtered = fp.files
		fp.cursor = 0
		return
	}
	fp.filtered = nil
	for _, f := range fp.files {
		if fuzzyMatch(query, f) {
			fp.filtered = append(fp.filtered, f)
		}
	}
	fp.cursor = 0
}

// Activate makes the picker visible.
func (fp *FilePicker) Activate() { fp.active = true }

// Deactivate hides the picker.
func (fp *FilePicker) Deactivate() { fp.active = false }

// IsActive reports whether the picker is currently visible.
func (fp FilePicker) IsActive() bool { return fp.active }

// SelectedFile returns the currently highlighted file path, or "" if none.
func (fp FilePicker) SelectedFile() string {
	if len(fp.filtered) == 0 || fp.cursor < 0 || fp.cursor >= len(fp.filtered) {
		return ""
	}
	return fp.filtered[fp.cursor]
}

// SetWidth updates the display width.
func (fp *FilePicker) SetWidth(width int) { fp.width = width }

// Update handles Up/Down cursor navigation.
func (fp FilePicker) Update(msg tea.Msg) (FilePicker, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return fp, nil
	}
	switch key.Type {
	case tea.KeyDown:
		if fp.cursor < len(fp.filtered)-1 {
			fp.cursor++
		}
	case tea.KeyUp:
		if fp.cursor > 0 {
			fp.cursor--
		}
	}
	return fp, nil
}

// View renders the picker as a bordered dropdown. Returns "" when inactive.
func (fp FilePicker) View() string {
	if !fp.active {
		return ""
	}
	if len(fp.filtered) == 0 {
		return FilePickerStyle(fp.width).Render("  no files found")
	}

	start := 0
	if fp.cursor >= filePickerMaxVisible {
		start = fp.cursor - filePickerMaxVisible + 1
	}
	end := start + filePickerMaxVisible
	if end > len(fp.filtered) {
		end = len(fp.filtered)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		if i == fp.cursor {
			sb.WriteString(FilePickerCursorStyle().Render("> " + fp.filtered[i]))
		} else {
			sb.WriteString("  " + fp.filtered[i])
		}
		if i < end-1 {
			sb.WriteByte('\n')
		}
	}
	return FilePickerStyle(fp.width).Render(sb.String())
}

// fuzzyMatch returns true if every rune of query appears in target in order
// (case-insensitive). An empty query always matches.
func fuzzyMatch(query, target string) bool {
	q := []rune(strings.ToLower(query))
	qi := 0
	for _, c := range strings.ToLower(target) {
		if qi < len(q) && q[qi] == c {
			qi++
		}
	}
	return qi == len(q)
}

// detectMentionQuery inspects input for a trailing @mention.
// Returns (query, true) when the text ends with @<non-whitespace>.
// A mention is inactive once there is whitespace after the last @.
func detectMentionQuery(input string) (query string, active bool) {
	idx := strings.LastIndex(input, "@")
	if idx == -1 {
		return "", false
	}
	after := input[idx+1:]
	if strings.ContainsAny(after, " \t\n") {
		return "", false
	}
	return after, true
}

// replaceCurrentMention swaps the trailing @query in input with @selectedPath + " ".
func replaceCurrentMention(input, selectedPath string) string {
	idx := strings.LastIndex(input, "@")
	if idx == -1 {
		return input + "@" + selectedPath + " "
	}
	return input[:idx] + "@" + selectedPath + " "
}

var mentionRe = regexp.MustCompile(`@(\S+)`)

// expandMentions appends the contents of any readable @path references as
// fenced code blocks. Files that cannot be read are left as-is.
func expandMentions(text string) string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return text
	}

	seen := make(map[string]bool)
	var appendix strings.Builder
	for _, m := range matches {
		path := m[1]
		if seen[path] {
			continue
		}
		seen[path] = true
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		const maxMentionBytes = 100 * 1024
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		content := strings.TrimRight(string(data), "\n")
		note := ""
		if len(data) > maxMentionBytes {
			content = string(data[:maxMentionBytes])
			note = fmt.Sprintf("\n[truncated: file is %d bytes, showing first %d]", len(data), maxMentionBytes)
		}
		appendix.WriteString(fmt.Sprintf(
			"\n\n---\nFile: %s\n```%s\n%s\n```%s",
			path, ext, content, note,
		))
	}

	if appendix.Len() == 0 {
		return text
	}
	return text + appendix.String()
}
