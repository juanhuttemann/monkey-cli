package tui

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	glamourstyles "github.com/charmbracelet/glamour/styles"
)

var (
	mdCacheMu    sync.Mutex
	mdCacheWidth int
	mdCacheEntry *glamour.TermRenderer
)

// getMarkdownRenderer returns a cached TermRenderer for the given width,
// creating one on first use or when the width changes. Only the most
// recent width is retained so the cache cannot grow unboundedly as the
// terminal is resized.
func getMarkdownRenderer(width int) (*glamour.TermRenderer, error) {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()

	if mdCacheEntry != nil && mdCacheWidth == width {
		return mdCacheEntry, nil
	}

	style := glamourstyles.DarkStyleConfig
	zero := uint(0)
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	mdCacheWidth = width
	mdCacheEntry = r
	return r, nil
}

func markdownCacheLen() int {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	if mdCacheEntry == nil {
		return 0
	}
	return 1
}

func clearMarkdownCache() {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	mdCacheWidth = 0
	mdCacheEntry = nil
}

// ansiSGR matches ANSI Select Graphic Rendition escape sequences.
var ansiSGR = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// RenderMarkdown renders markdown-formatted content as ANSI-styled terminal text.
// Falls back to the original content if rendering fails or width is zero.
func RenderMarkdown(content string, width int) string {
	if width <= 0 {
		return content
	}

	r, err := getMarkdownRenderer(width)
	if err != nil {
		return content
	}

	rendered, err := r.Render(content)
	if err != nil {
		return content
	}

	return stripANSIBackground(rendered)
}

// stripANSIBackground removes background colour parameters from ANSI SGR
// sequences while preserving foreground colours, bold, italic and other
// text attributes. Correctly handles combined sequences such as
// "38;5;228;48;5;63;1" and true-colour foregrounds like "38;2;255;107;107"
// where the RGB components must not be confused with background codes.
func stripANSIBackground(s string) string {
	return ansiSGR.ReplaceAllStringFunc(s, func(seq string) string {
		inner := seq[2 : len(seq)-1] // strip leading \x1b[ and trailing m
		if inner == "" {
			return seq
		}

		params := strings.Split(inner, ";")
		filtered := make([]string, 0, len(params))
		i := 0
		for i < len(params) {
			n, _ := strconv.Atoi(params[i])
			switch {
			case n == 38:
				// Extended foreground colour — consume and keep all related params
				// so that RGB components (which may be in 100-107) are not
				// misidentified as background codes.
				filtered = append(filtered, params[i])
				i++
				if i < len(params) {
					sub, _ := strconv.Atoi(params[i])
					filtered = append(filtered, params[i])
					i++
					switch sub {
					case 5: // 38;5;N
						if i < len(params) {
							filtered = append(filtered, params[i])
							i++
						}
					case 2: // 38;2;R;G;B
						for j := 0; j < 3 && i < len(params); j++ {
							filtered = append(filtered, params[i])
							i++
						}
					}
				}
			case n == 48:
				// Extended background colour: skip 48;5;N or 48;2;R;G;B.
				if i+1 < len(params) {
					sub, _ := strconv.Atoi(params[i+1])
					switch sub {
					case 5:
						i += 3 // skip 48;5;N
					case 2:
						i += 5 // skip 48;2;R;G;B
					default:
						i += 2
					}
				} else {
					i++
				}
			case (n >= 40 && n <= 47) || n == 49 || (n >= 100 && n <= 107):
				// Standalone standard / bright background colour, or default-bg reset.
				i++
			default:
				filtered = append(filtered, params[i])
				i++
			}
		}

		if len(filtered) == 0 {
			return "" // sequence was background-only; drop it entirely
		}
		return "\x1b[" + strings.Join(filtered, ";") + "m"
	})
}
