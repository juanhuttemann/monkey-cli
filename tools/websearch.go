package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/juanhuttemann/monkey-cli/api"
)

const (
	defaultSearchTimeout = 15 * time.Second
	defaultMaxResults    = 5
	maxSnippetChars      = 200
	ddgSearchURL         = "https://html.duckduckgo.com/html/"
)

// ddgUserAgents are tried in order; if one triggers a bot challenge the next is used.
var ddgUserAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
}

// SearchResult holds a single DuckDuckGo search result.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// WebSearchTool returns the Tool definition for DuckDuckGo web search.
func WebSearchTool() api.Tool {
	return api.Tool{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo. Returns titles, URLs, and snippets for the top results.",
		InputSchema: api.InputSchema{
			Type: "object",
			Properties: map[string]api.PropertyDef{
				"query": {
					Type:        "string",
					Description: "The search query.",
				},
				"max_results": {
					Type:        "integer",
					Description: "Maximum number of results to return (default: 5, max: 10).",
				},
			},
			Required: []string{"query"},
		},
	}
}

// WebSearchExecutor implements api.ToolExecutor for the web_search tool.
// Client overrides the HTTP client (nil uses a default with timeout via context).
// BaseURL overrides the DuckDuckGo endpoint (for testing).
type WebSearchExecutor struct {
	Client  *http.Client
	BaseURL string
}

func (w *WebSearchExecutor) client() *http.Client {
	if w.Client != nil {
		return w.Client
	}
	return http.DefaultClient
}

func (w *WebSearchExecutor) baseURL() string {
	if w.BaseURL != "" {
		return w.BaseURL
	}
	return ddgSearchURL
}

// ExecuteTool performs a DuckDuckGo search and returns formatted results.
func (w *WebSearchExecutor) ExecuteTool(ctx context.Context, _ string, input map[string]any) (string, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("web_search: missing or empty query")
	}

	max := defaultMaxResults
	if n := toInt(input["max_results"]); n > 0 {
		max = n
		if max > 10 {
			max = 10
		}
	}

	results, err := w.search(ctx, query, max)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}

	return formatSearchResults(results), nil
}

func (w *WebSearchExecutor) search(ctx context.Context, query string, max int) ([]SearchResult, error) {
	for _, ua := range ddgUserAgents {
		results, blocked, err := w.searchWithAgent(ctx, query, max, ua)
		if err != nil {
			return nil, err
		}
		if !blocked {
			return results, nil
		}
	}
	return nil, fmt.Errorf("bot protection triggered for all user agents")
}

func (w *WebSearchExecutor) searchWithAgent(ctx context.Context, query string, max int, ua string) (results []SearchResult, blocked bool, err error) {
	formBody := url.Values{
		"q":  {query},
		"kl": {"wt-wt"},
		"df": {""},
		"b":  {""},
	}.Encode()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultSearchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL(), strings.NewReader(formBody))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://html.duckduckgo.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("User-Agent", ua)

	resp, err := w.client().Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("server returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	if isBotChallenge(body) {
		return nil, true, nil
	}
	results, err = parseSearchResults(bytes.NewReader(body), max)
	return results, false, err
}

// isBotChallenge reports whether the response body is a DDG bot-detection page.
func isBotChallenge(body []byte) bool {
	return bytes.Contains(body, []byte("anomaly-modal"))
}

// parseSearchResults extracts up to max search results from DDG HTML.
func parseSearchResults(r io.Reader, max int) ([]SearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	var results []SearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= max {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && nodeHasClass(n, "web-result") {
			if r := extractSearchResult(n); r.Title != "" {
				results = append(results, r)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results, nil
}

func extractSearchResult(n *html.Node) SearchResult {
	var res SearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && nodeHasClass(n, "result__a") {
			res.Title = nodeTextContent(n)
			res.URL = decodeDDGHref(nodeAttr(n, "href"))
		}
		if n.Type == html.ElementNode && nodeHasClass(n, "result__snippet") {
			res.Snippet = truncateAtWord(nodeTextContent(n), maxSnippetChars)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return res
}

// decodeDDGHref unwraps DuckDuckGo's redirect href to the real URL.
func decodeDDGHref(href string) string {
	const prefix = "//duckduckgo.com/l/?uddg="
	if !strings.HasPrefix(href, prefix) {
		return href
	}
	encoded := href[len(prefix):]
	if i := strings.Index(encoded, "&"); i >= 0 {
		encoded = encoded[:i]
	}
	u, err := url.QueryUnescape(encoded)
	if err != nil {
		return href
	}
	return u
}

func formatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}
	var sb strings.Builder
	for i, r := range results {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%d. %s\n%s\n%s", i+1, r.Title, r.URL, r.Snippet)
	}
	return sb.String()
}

// nodeHasClass reports whether an HTML element node has the given CSS class.
func nodeHasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// nodeAttr returns the value of the named attribute, or "".
func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// truncateAtWord shortens s to at most max bytes, cutting at the last space
// before the limit so words are not split. A trailing ellipsis is appended
// when truncation occurs.
func truncateAtWord(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := strings.LastIndex(s[:max], " ")
	if cut <= 0 {
		cut = max
	}
	return s[:cut] + "…"
}

// nodeTextContent returns all text content within n, whitespace-trimmed.
func nodeTextContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
