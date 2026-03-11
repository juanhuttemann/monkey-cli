package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// makeDDGPage generates a mock DDG HTML page with n search results.
func makeDDGPage(n int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><body><div class="results">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb,
			`<div class="web-result"><h2 class="result__title"><a class="result__a" href="https://example%d.com">Result %d</a></h2><a class="result__snippet" href="#">Snippet for result %d.</a></div>`,
			i, i, i)
	}
	sb.WriteString(`</div></body></html>`)
	return sb.String()
}

const mockDDGPage = `<!DOCTYPE html>
<html><body>
<div class="results">
  <div class="web-result">
    <h2 class="result__title">
      <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2F&rut=abc">Example Domain</a>
    </h2>
    <a class="result__snippet" href="#">This domain is for illustrative examples.</a>
  </div>
  <div class="web-result">
    <h2 class="result__title">
      <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Ffoo.org%2F&rut=xyz">Foo Organization</a>
    </h2>
    <a class="result__snippet" href="#">The foo organization website.</a>
  </div>
</div>
</body></html>`

func newDDGTestServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
}

// --- WebSearchTool definition tests ---

func TestWebSearchTool_Name(t *testing.T) {
	tool := WebSearchTool()
	if tool.Name != "web_search" {
		t.Errorf("WebSearchTool().Name = %q, want %q", tool.Name, "web_search")
	}
}

func TestWebSearchTool_HasDescription(t *testing.T) {
	tool := WebSearchTool()
	if tool.Description == "" {
		t.Error("WebSearchTool().Description should not be empty")
	}
}

func TestWebSearchTool_InputSchemaType(t *testing.T) {
	tool := WebSearchTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestWebSearchTool_HasQueryProperty(t *testing.T) {
	tool := WebSearchTool()
	prop, ok := tool.InputSchema.Properties["query"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'query' key")
	}
	if prop.Type != "string" {
		t.Errorf("query property Type = %q, want %q", prop.Type, "string")
	}
	if prop.Description == "" {
		t.Error("query property Description should not be empty")
	}
}

func TestWebSearchTool_QueryIsRequired(t *testing.T) {
	tool := WebSearchTool()
	for _, r := range tool.InputSchema.Required {
		if r == "query" {
			return
		}
	}
	t.Error("'query' should be in InputSchema.Required")
}

// --- WebSearchExecutor tests ---

func TestWebSearchExecutor_MissingQuery(t *testing.T) {
	exec := &WebSearchExecutor{}
	_, err := exec.ExecuteTool("web_search", map[string]any{})
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestWebSearchExecutor_EmptyQuery(t *testing.T) {
	exec := &WebSearchExecutor{}
	_, err := exec.ExecuteTool("web_search", map[string]any{"query": ""})
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestWebSearchExecutor_ReturnsResults(t *testing.T) {
	srv := newDDGTestServer(mockDDGPage)
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if !strings.Contains(result, "Example Domain") {
		t.Errorf("expected 'Example Domain' in result, got: %q", result)
	}
	if !strings.Contains(result, "https://example.com/") {
		t.Errorf("expected decoded URL in result, got: %q", result)
	}
	if !strings.Contains(result, "This domain is for illustrative examples.") {
		t.Errorf("expected snippet in result, got: %q", result)
	}
}

func TestWebSearchExecutor_RespectsMaxResults(t *testing.T) {
	srv := newDDGTestServer(mockDDGPage)
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test", "max_results": 1})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if strings.Contains(result, "Foo Organization") {
		t.Error("result should not contain second result when max_results=1")
	}
}

func TestWebSearchExecutor_EmptyPage(t *testing.T) {
	srv := newDDGTestServer("<html><body></body></html>")
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if result != "No results found." {
		t.Errorf("expected 'No results found.', got: %q", result)
	}
}

func TestWebSearchExecutor_MaxResultsCappedAt10(t *testing.T) {
	srv := newDDGTestServer(makeDDGPage(15))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test", "max_results": 99})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	// Results 11–15 must not appear when capped at 10.
	for i := 11; i <= 15; i++ {
		if strings.Contains(result, fmt.Sprintf("Result %d", i)) {
			t.Errorf("result should not contain 'Result %d' when capped at 10", i)
		}
	}
	if !strings.Contains(result, "Result 1") {
		t.Error("expected at least one result")
	}
}

func TestWebSearchExecutor_UsesPostMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body></body></html>"))
	}))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	_, _ = exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q, want %q", gotMethod, http.MethodPost)
	}
}

func TestWebSearchExecutor_QueryEncodedInFormBody(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			gotQuery = r.FormValue("q")
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body></body></html>"))
	}))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	_, _ = exec.ExecuteTool("web_search", map[string]any{"query": "hello world & more"})
	if gotQuery != "hello world & more" {
		t.Errorf("server received form query %q, want %q", gotQuery, "hello world & more")
	}
}

func TestWebSearchExecutor_SetsRequiredHeaders(t *testing.T) {
	var gotReferer, gotSecFetchSite, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("Referer")
		gotSecFetchSite = r.Header.Get("Sec-Fetch-Site")
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body></body></html>"))
	}))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	_, _ = exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if gotReferer != "https://html.duckduckgo.com/" {
		t.Errorf("Referer = %q, want %q", gotReferer, "https://html.duckduckgo.com/")
	}
	if gotSecFetchSite != "same-origin" {
		t.Errorf("Sec-Fetch-Site = %q, want %q", gotSecFetchSite, "same-origin")
	}
	if !strings.HasPrefix(gotContentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
}

func TestWebSearchExecutor_RetriesOnBotChallenge(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "text/html")
		if attempts == 1 {
			_, _ = w.Write([]byte(`<html><body><div class="anomaly-modal">challenge</div></body></html>`))
		} else {
			_, _ = w.Write([]byte(mockDDGPage))
		}
	}))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if !strings.Contains(result, "Example Domain") {
		t.Errorf("expected results after retry, got: %q", result)
	}
	if attempts < 2 {
		t.Errorf("expected at least 2 attempts (1 blocked + 1 success), got %d", attempts)
	}
}

func TestWebSearchExecutor_AllAgentsBlockedReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><div class="anomaly-modal">challenge</div></body></html>`))
	}))
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	_, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if err == nil {
		t.Error("expected error when all user agents are blocked")
	}
}

func TestWebSearchExecutor_NonOKStatusReturnsError(t *testing.T) {
	for _, code := range []int{429, 503, 500} {
		code := code
		t.Run(fmt.Sprintf("HTTP%d", code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer srv.Close()

			exec := &WebSearchExecutor{BaseURL: srv.URL}
			_, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
			if err == nil {
				t.Errorf("expected error for HTTP %d, got nil", code)
			}
		})
	}
}

func TestWebSearchExecutor_SnippetTruncatesAtWordBoundary(t *testing.T) {
	// Build a snippet that exceeds maxSnippetChars, where the last word
	// straddles the boundary so a word-boundary truncation differs from
	// a char-boundary truncation.
	words := strings.Repeat("hello ", 40) // 240 chars, boundary lands mid-word
	snippet := `<a class="result__snippet" href="#">` + words + `</a>`
	page := `<!DOCTYPE html><html><body><div class="web-result">` +
		`<a class="result__a" href="https://example.com">Title</a>` +
		snippet +
		`</div></body></html>`

	srv := newDDGTestServer(page)
	defer srv.Close()

	exec := &WebSearchExecutor{BaseURL: srv.URL}
	result, err := exec.ExecuteTool("web_search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	// Must not end mid-word (no trailing partial "hello")
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	snippetLine := lines[len(lines)-1]
	if strings.HasSuffix(snippetLine, "hel") || strings.HasSuffix(snippetLine, "hell") ||
		strings.HasSuffix(snippetLine, "he") || strings.HasSuffix(snippetLine, "h") {
		t.Errorf("snippet appears to be cut mid-word: %q", snippetLine)
	}
}

// --- decodeDDGHref tests ---

func TestDecodeDDGHref_Standard(t *testing.T) {
	href := "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2F&rut=abc"
	got := decodeDDGHref(href)
	if got != "https://example.com/" {
		t.Errorf("decodeDDGHref() = %q, want %q", got, "https://example.com/")
	}
}

func TestDecodeDDGHref_NonDDG(t *testing.T) {
	href := "https://example.com/"
	got := decodeDDGHref(href)
	if got != href {
		t.Errorf("decodeDDGHref() = %q, want passthrough %q", got, href)
	}
}

func TestDecodeDDGHref_NoRutParam(t *testing.T) {
	href := "//duckduckgo.com/l/?uddg=https%3A%2F%2Fbar.com%2F"
	got := decodeDDGHref(href)
	if got != "https://bar.com/" {
		t.Errorf("decodeDDGHref() = %q, want %q", got, "https://bar.com/")
	}
}
