package tools

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juanhuttemann/monkey-cli/api"
)

// GlobTool returns the Tool definition for finding files by glob pattern.
func GlobTool() api.Tool {
	return api.Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern. Supports *, **, ?, and character classes [abc]. Returns matching paths sorted by modification time (newest first), one per line.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to match files against (e.g. **/*.go, src/**/*.ts).",
				},
				"path": {
					Type:        "string",
					Description: "Directory to search in. Defaults to current directory.",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

// GlobExecutor implements api.ToolExecutor for the glob tool.
type GlobExecutor struct{}

// ExecuteTool walks the filesystem from input["path"] (default ".") and returns all
// files matching input["pattern"], sorted by modification time (newest first).
func (g GlobExecutor) ExecuteTool(_ context.Context, _ string, input map[string]any) (string, error) {
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("glob: missing or empty pattern")
	}

	root := "."
	if p, ok := input["path"].(string); ok && p != "" {
		root = p
	}

	type entry struct {
		path    string
		modTime time.Time
	}

	var entries []entry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if globMatch(pattern, rel) {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			entries = append(entries, entry{path: rel, modTime: info.ModTime()})
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("glob: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})

	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.path
	}
	return strings.Join(paths, "\n"), nil
}

// globMatch reports whether path matches pattern using glob syntax.
// In addition to filepath.Match syntax (*, ?, [abc]), it supports ** which
// matches zero or more path segments.
func globMatch(pattern, path string) bool {
	return matchSegments(
		strings.Split(filepath.ToSlash(pattern), "/"),
		strings.Split(path, "/"),
	)
}

// matchSegments recursively matches pattern segments against path segments.
func matchSegments(patSegs, pathSegs []string) bool {
	for len(patSegs) > 0 {
		if patSegs[0] == "**" {
			patSegs = patSegs[1:]
			if len(patSegs) == 0 {
				// ** at end matches everything remaining
				return true
			}
			// Try matching the remaining pattern starting at each position
			for i := 0; i <= len(pathSegs); i++ {
				if matchSegments(patSegs, pathSegs[i:]) {
					return true
				}
			}
			return false
		}

		if len(pathSegs) == 0 {
			return false
		}
		matched, err := filepath.Match(patSegs[0], pathSegs[0])
		if err != nil || !matched {
			return false
		}
		patSegs = patSegs[1:]
		pathSegs = pathSegs[1:]
	}
	return len(pathSegs) == 0
}
