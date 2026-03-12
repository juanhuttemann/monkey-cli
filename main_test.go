package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSendPrompt_Success(t *testing.T) {
	// Setup mock server for LLM API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_paris.json"))
	}))
	defer server.Close()

	// Set environment variables
	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	got, err := sendPrompt("What is the capital of France?")
	if err != nil {
		t.Fatalf("sendPrompt() returned error: %v", err)
	}

	want := "The capital of France is Paris."
	if got != want {
		t.Errorf("sendPrompt() = %q, want %q", got, want)
	}
}

func TestSendPrompt_SendsCorrectPrompt(t *testing.T) {
	var requestBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &requestBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("custom test prompt")
	if err != nil {
		t.Fatalf("sendPrompt() returned error: %v", err)
	}

	// Verify the prompt was sent correctly
	messages := requestBody["messages"].([]interface{})
	msg := messages[0].(map[string]interface{})
	content := msg["content"].(string)
	if content != "custom test prompt" {
		t.Errorf("Request content = %q, want %q", content, "custom test prompt")
	}
}

func TestSendPrompt_MissingAPIKey(t *testing.T) {
	// Ensure API key is not set
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_BASE_URL", "http://localhost")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error when ANTHROPIC_API_KEY is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error should mention ANTHROPIC_API_KEY, got: %v", err)
	}
}

func TestSendPrompt_DefaultModelUsedWhenEnvVarsUnset(t *testing.T) {
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if m, ok := body["model"].(string); ok {
			capturedModel = m
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_SONNET_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")

	_, err := sendPrompt("test prompt")
	if err != nil {
		t.Fatalf("sendPrompt() should succeed with default model, got error: %v", err)
	}
	if capturedModel == "" {
		t.Fatal("expected a model to be sent in the request")
	}
}

func TestSendPrompt_HTTPError(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error on network failure")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "request") && !strings.Contains(errMsg, "connection") {
		t.Errorf("error should indicate request/connection failure, got: %v", err)
	}
}

func TestSendPrompt_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(fixture(t, "error_internal.json"))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error on non-200 status")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
	if !strings.Contains(errMsg, "internal server error") {
		t.Errorf("error should contain response body, got: %v", err)
	}
}

func TestSendPrompt_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_invalid.json"))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error on invalid JSON")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") {
		t.Errorf("error should indicate parse/unmarshal failure, got: %v", err)
	}
}

func TestSendPrompt_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_empty_content.json"))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error on empty content")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}

func TestPrintVersion(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printVersion()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, Version) {
		t.Errorf("printVersion() should contain version %q, got: %q", Version, output)
	}
	if !strings.Contains(output, AppTitle) {
		t.Errorf("printVersion() should contain app title %q, got: %q", AppTitle, output)
	}
}

func TestPrintUsage(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsage()

	_ = w.Close()
	os.Stderr = oldStderr

	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected strings
	if !strings.Contains(output, "Usage: monkey -p") {
		t.Errorf("printUsage() should contain 'Usage: monkey -p', got: %q", output)
	}
	if !strings.Contains(output, "-p, --prompt") {
		t.Errorf("printUsage() should contain '-p, --prompt', got: %q", output)
	}
	if !strings.Contains(output, "required") {
		t.Errorf("printUsage() should contain 'required', got: %q", output)
	}
}

func TestShouldLaunchTUI_NoFlags_ReturnsTrue(t *testing.T) {
	// When prompt is empty, TUI should launch
	prompt := ""

	result := shouldLaunchTUI(prompt)
	if !result {
		t.Error("shouldLaunchTUI('') = false, want true")
	}
}

func TestShouldLaunchTUI_WithPromptFlag_ReturnsFalse(t *testing.T) {
	// When prompt is provided, CLI mode should be used
	prompt := "Hello, world!"

	result := shouldLaunchTUI(prompt)
	if result {
		t.Error("shouldLaunchTUI('Hello, world!') = true, want false")
	}
}

func TestShouldLaunchTUI_WithArgs_ReturnsFalse(t *testing.T) {
	// When prompt is built from positional args, CLI mode should be used
	prompt := "hello world from args"

	result := shouldLaunchTUI(prompt)
	if result {
		t.Error("shouldLaunchTUI('hello world from args') = true, want false")
	}
}

func TestShouldLaunchTUI_WhitespaceOnly_ReturnsTrue(t *testing.T) {
	// Whitespace-only prompt should still launch TUI
	prompt := "   \t\n  "

	result := shouldLaunchTUI(prompt)
	if !result {
		t.Error("shouldLaunchTUI('   ') = false, want true (whitespace should launch TUI)")
	}
}

func TestRun_EmptyPrompt_LaunchesTUI(t *testing.T) {
	tuiLaunched := false
	run("", func() { tuiLaunched = true })

	if !tuiLaunched {
		t.Error("run('') should launch TUI when prompt is empty")
	}
}

func TestRun_WhitespacePrompt_LaunchesTUI(t *testing.T) {
	tuiLaunched := false
	run("   \t\n  ", func() { tuiLaunched = true })

	if !tuiLaunched {
		t.Error("run('   ') should launch TUI when prompt is whitespace-only")
	}
}

func TestRun_WithPrompt_DoesNotLaunchTUI(t *testing.T) {
	server, cleanup := createMockServer(successResponse("pong"), 200)
	defer cleanup()
	defer setupTestEnv("test-key", server.URL, "test-model")()

	tuiLaunched := false
	run("ping", func() { tuiLaunched = true })

	if tuiLaunched {
		t.Error("run('ping') should not launch TUI when prompt is provided")
	}
}

func TestRun_WithPrompt_PrintsResponse(t *testing.T) {
	server, cleanup := createMockServer(successResponse("hello back"), 200)
	defer cleanup()
	defer setupTestEnv("test-key", server.URL, "test-model")()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	run("hello", func() {})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	_, _ = io.Copy(&buf, r)

	if !strings.Contains(buf.String(), "hello back") {
		t.Errorf("run() output = %q, want to contain %q", buf.String(), "hello back")
	}
}
