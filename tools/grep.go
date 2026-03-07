package tools

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/juanhuttemann/monkey-cli/api"
)

const grepMaxResults = 200

// GrepTool returns the Tool definition for searching file contents by regex.
func GrepTool() api.Tool {
	return api.Tool{
		Name:        "grep",
		Description: "Search file contents using a regular expression. Returns matching lines in file:line:content format, capped at 200 results.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"pattern": {
					Type:        "string",
					Description: "Regular expression to search for.",
				},
				"path": {
					Type:        "string",
					Description: "Directory to search in. Defaults to current directory.",
				},
				"glob": {
					Type:        "string",
					Description: "File filter glob pattern (e.g. *.go). Searches all files when omitted.",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

// GrepExecutor implements api.ToolExecutor for the grep tool.
type GrepExecutor struct{}

// ExecuteTool searches files under input["path"] (default ".") for lines matching
// input["pattern"] (regex). An optional input["glob"] filters which files are
// searched. Returns up to 200 matches in file:line:content format.
func (g GrepExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("grep: missing or empty pattern")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("grep: invalid pattern: %w", err)
	}

	root := "."
	if p, ok := input["path"].(string); ok && p != "" {
		root = p
	}

	globPat, _ := input["glob"].(string)

	var sb strings.Builder
	count := 0
	truncated := false

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if globPat != "" {
			matched, err := filepath.Match(globPat, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = f.Close() }()

		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}

		lineNum := 0
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				if count >= grepMaxResults {
					truncated = true
					return fmt.Errorf("stop") // sentinel to stop walking
				}
				fmt.Fprintf(&sb, "%s:%d:%s\n", rel, lineNum, line)
				count++
			}
		}
		return nil
	})
	// swallow the sentinel "stop" error used to break out of WalkDir
	if err != nil && err.Error() != "stop" {
		return "", fmt.Errorf("grep: %w", err)
	}

	result := sb.String()
	if truncated {
		result += fmt.Sprintf("[results truncated: showing first %d matches]\n", grepMaxResults)
	}
	return strings.TrimRight(result, "\n"), nil
}
