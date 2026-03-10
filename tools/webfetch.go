package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/juanhuttemann/monkey-cli/api"
)

const (
	defaultFetchTimeout = 15 * time.Second
	maxFetchBytes       = 20 * 1024 // 20 KB
	fetchUserAgent      = "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0"
)

// WebFetchTool returns the Tool definition for fetching web page content.
func WebFetchTool() api.Tool {
	return api.Tool{
		Name:        "web_fetch",
		Description: "Fetch the text content of a URL. HTML is converted to plain text. Output is capped at 20 KB.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"url": {
					Type:        "string",
					Description: "The URL to fetch.",
				},
			},
			Required: []string{"url"},
		},
	}
}

// WebFetchExecutor implements api.ToolExecutor for the web_fetch tool.
// Client overrides the HTTP client (nil uses a default).
type WebFetchExecutor struct {
	Client *http.Client
}

func (w *WebFetchExecutor) client() *http.Client {
	if w.Client != nil {
		return w.Client
	}
	return http.DefaultClient
}

// ExecuteTool fetches the URL from input["url"] and returns its text content.
func (w *WebFetchExecutor) ExecuteTool(_ string, input map[string]any) (string, error) {
	rawURL, ok := input["url"].(string)
	if !ok || rawURL == "" {
		return "", fmt.Errorf("web_fetch: missing or empty url")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("web_fetch: invalid url: %w", err)
	}
	req.Header.Set("User-Agent", fetchUserAgent)

	resp, err := w.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("web_fetch: server returned %s", resp.Status)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		text, err := htmlToText(io.LimitReader(resp.Body, maxFetchBytes+1))
		if err != nil {
			return "", fmt.Errorf("web_fetch: %w", err)
		}
		return truncateFetch(text), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	return truncateFetch(string(body)), nil
}

// htmlToText extracts visible text from HTML, skipping script/style/head elements.
func htmlToText(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "head", "noscript":
				return
			}
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteByte('\n')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return sb.String(), nil
}

func truncateFetch(s string) string {
	if len(s) <= maxFetchBytes {
		return s
	}
	return s[:maxFetchBytes] + fmt.Sprintf("\n[content truncated: showing first %d bytes]", maxFetchBytes)
}
