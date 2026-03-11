package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- WebFetchTool definition tests ---

func TestWebFetchTool_Name(t *testing.T) {
	tool := WebFetchTool()
	if tool.Name != "web_fetch" {
		t.Errorf("WebFetchTool().Name = %q, want %q", tool.Name, "web_fetch")
	}
}

func TestWebFetchTool_HasDescription(t *testing.T) {
	tool := WebFetchTool()
	if tool.Description == "" {
		t.Error("WebFetchTool().Description should not be empty")
	}
}

func TestWebFetchTool_InputSchemaType(t *testing.T) {
	tool := WebFetchTool()
	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
}

func TestWebFetchTool_HasURLProperty(t *testing.T) {
	tool := WebFetchTool()
	prop, ok := tool.InputSchema.Properties["url"]
	if !ok {
		t.Fatal("InputSchema.Properties should have 'url' key")
	}
	if prop.Type != "string" {
		t.Errorf("url property Type = %q, want %q", prop.Type, "string")
	}
	if prop.Description == "" {
		t.Error("url property Description should not be empty")
	}
}

func TestWebFetchTool_URLIsRequired(t *testing.T) {
	tool := WebFetchTool()
	for _, r := range tool.InputSchema.Required {
		if r == "url" {
			return
		}
	}
	t.Error("'url' should be in InputSchema.Required")
}

// --- WebFetchExecutor tests ---

func TestWebFetchExecutor_MissingURL(t *testing.T) {
	exec := &WebFetchExecutor{}
	_, err := exec.ExecuteTool("web_fetch", map[string]any{})
	if err == nil {
		t.Error("expected error for missing url")
	}
}

func TestWebFetchExecutor_EmptyURL(t *testing.T) {
	exec := &WebFetchExecutor{}
	_, err := exec.ExecuteTool("web_fetch", map[string]any{"url": ""})
	if err == nil {
		t.Error("expected error for empty url")
	}
}

func TestWebFetchExecutor_FetchesHTMLAsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Test</title></head><body><p>Hello world</p></body></html>`))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected 'Hello world' in result, got: %q", result)
	}
}

func TestWebFetchExecutor_SkipsScriptAndStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><script>alert('x')</script><style>body{color:red}</style><p>Visible</p></body></html>`))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if strings.Contains(result, "alert") || strings.Contains(result, "color:red") {
		t.Errorf("result should not contain script/style content: %q", result)
	}
	if !strings.Contains(result, "Visible") {
		t.Errorf("expected 'Visible' in result, got: %q", result)
	}
}

func TestWebFetchExecutor_SkipsHeadContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><meta charset="utf-8"><title>PageTitle</title></head><body><p>Body text</p></body></html>`))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if strings.Contains(result, "utf-8") {
		t.Errorf("result should not contain head meta content: %q", result)
	}
	if !strings.Contains(result, "Body text") {
		t.Errorf("expected 'Body text' in result, got: %q", result)
	}
}

func TestWebFetchExecutor_FetchesPlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text response"))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if result != "plain text response" {
		t.Errorf("got %q, want %q", result, "plain text response")
	}
}

func TestWebFetchExecutor_TruncatesLargeContent(t *testing.T) {
	bigContent := strings.Repeat("x", maxFetchBytes+1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(bigContent))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if !strings.Contains(result, "[content truncated") {
		t.Errorf("expected truncation notice in result, got: %q", result[:200])
	}
}

func TestWebFetchExecutor_SkipsNoscript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><noscript>Please enable JavaScript</noscript><p>Main content</p></body></html>`))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if strings.Contains(result, "Please enable JavaScript") {
		t.Errorf("result should not contain noscript content: %q", result)
	}
	if !strings.Contains(result, "Main content") {
		t.Errorf("expected 'Main content' in result, got: %q", result)
	}
}

func TestWebFetchExecutor_LargeHTMLIsBounded(t *testing.T) {
	// Serve an HTML page significantly larger than maxFetchBytes.
	bigHTML := "<html><body>" + strings.Repeat("<p>"+strings.Repeat("x", 100)+"</p>", 500) + "</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(bigHTML))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if len(result) > 2*maxFetchBytes {
		t.Errorf("result length %d exceeds 2×maxFetchBytes (%d)", len(result), 2*maxFetchBytes)
	}
}

func TestWebFetchExecutor_NonOKStatusReturnsError(t *testing.T) {
	for _, code := range []int{404, 429, 500, 503} {
		code := code
		t.Run(fmt.Sprintf("HTTP%d", code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer srv.Close()

			exec := &WebFetchExecutor{}
			_, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
			if err == nil {
				t.Errorf("expected error for HTTP %d, got nil", code)
			}
		})
	}
}

func TestWebFetchExecutor_InvalidURL(t *testing.T) {
	exec := &WebFetchExecutor{}
	_, err := exec.ExecuteTool("web_fetch", map[string]any{"url": "://bad-url"})
	if err == nil {
		t.Error("expected error for invalid url")
	}
}

func TestWebFetchExecutor_LargeHeadDoesNotMaskBodyContent(t *testing.T) {
	// Simulate a page where <head> with large scripts exceeds maxFetchBytes,
	// but body has visible text that must still be returned.
	largeScript := "<script>" + strings.Repeat("var x=1;", 3000) + "</script>"
	page := "<html><head>" + largeScript + largeScript + "</head><body><p>Visible body content</p></body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	exec := &WebFetchExecutor{}
	result, err := exec.ExecuteTool("web_fetch", map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("ExecuteTool() error: %v", err)
	}
	if !strings.Contains(result, "Visible body content") {
		t.Errorf("body content lost due to large head; got: %q", result)
	}
}

// --- normalizeURL tests ---

func TestNormalizeURL_PrependsSchemeToBareHost(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"example.com", "https://example.com"},
		{"www.example.com/path", "https://www.example.com/path"},
		{"news.org", "https://news.org"},
	}
	for _, c := range cases {
		got := normalizeURL(c.input)
		if got != c.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestNormalizeURL_LeavesSchemeUnchanged(t *testing.T) {
	cases := []string{
		"https://example.com",
		"http://example.com",
		"https://abc.com.py/page?q=1",
	}
	for _, c := range cases {
		got := normalizeURL(c)
		if got != c {
			t.Errorf("normalizeURL(%q) = %q, want unchanged", c, got)
		}
	}
}
