package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/juanhuttemann/monkey-cli/config"
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

	// Verify the prompt was sent correctly.
	// The SDK sends content as an array of typed content blocks.
	messages := requestBody["messages"].([]interface{})
	msg := messages[0].(map[string]interface{})
	contentArr, _ := msg["content"].([]interface{})
	if len(contentArr) == 0 {
		t.Fatal("expected non-empty content array in request")
	}
	textBlock, _ := contentArr[0].(map[string]interface{})
	text, _ := textBlock["text"].(string)
	if text != "custom test prompt" {
		t.Errorf("Request content = %q, want %q", text, "custom test prompt")
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
	if !strings.Contains(errMsg, "pars") && !strings.Contains(errMsg, "unmarshal") && !strings.Contains(errMsg, "invalid") {
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

func TestIntroContent_NonEmpty(t *testing.T) {
	content := introContent()
	if content == "" {
		t.Error("introContent() should return non-empty ASCII art string")
	}
}

func TestBuildClientOpts_WithMaxTokens(t *testing.T) {
	server, cleanup := createMockServer(successResponse("ok"), 200)
	defer cleanup()
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")
	t.Setenv("ANTHROPIC_MAX_TOKENS", "1024")

	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("loader.Load() error: %v", err)
	}
	opts, err := buildClientOpts(cfg)
	if err != nil {
		t.Fatalf("buildClientOpts() error: %v", err)
	}
	if len(opts) == 0 {
		t.Error("buildClientOpts should return non-empty options")
	}
}

func TestBuildClientOpts_SystemPromptAndClaudeMD(t *testing.T) {
	// Test the concat path: both MONKEY.md and CLAUDE.md are found.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MONKEY.md"), []byte("monkey system prompt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("claude context"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.example.com")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	opts, err := buildClientOpts(cfg)
	if err != nil {
		t.Fatalf("buildClientOpts() error: %v", err)
	}
	if len(opts) == 0 {
		t.Error("buildClientOpts should return non-empty options")
	}
}

func TestBuildClientOpts_SystemPromptReadError(t *testing.T) {
	// Create a directory at MONKEY.md path so LoadSystemPromptFile returns a non-IsNotExist error.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "MONKEY.md"), 0o700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.example.com")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err = buildClientOpts(cfg)
	if err == nil {
		t.Fatal("buildClientOpts should return error when MONKEY.md is unreadable")
	}
}

func TestSendPrompt_BuildOptsError(t *testing.T) {
	// Create a directory at MONKEY.md path so buildClientOpts returns an error.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "MONKEY.md"), 0o700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.example.com")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err := sendPrompt("test")
	if err == nil {
		t.Fatal("sendPrompt should return error when buildClientOpts fails")
	}
}

func TestBuildClientOpts_LoadsSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	promptFile := dir + "/MONKEY.md"
	if err := os.WriteFile(promptFile, []byte("You are a helpful monkey."), 0o600); err != nil {
		t.Fatal(err)
	}

	server, cleanup := createMockServer(successResponse("ok"), 200)
	defer cleanup()
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "test-model")

	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("loader.Load() error: %v", err)
	}

	// Temporarily chdir to dir so MONKEY.md is found
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	opts, err := buildClientOpts(cfg)
	if err != nil {
		t.Fatalf("buildClientOpts() error: %v", err)
	}
	if len(opts) == 0 {
		t.Error("buildClientOpts should return non-empty options")
	}
}

func TestBuildDynamicContext_ContainsDate(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	got := buildDynamicContext(now, t.TempDir())
	want := "Today's date: 2025-06-15"
	if !strings.Contains(got, want) {
		t.Errorf("buildDynamicContext() = %q, want it to contain %q", got, want)
	}
}

func TestBuildDynamicContext_ContainsCwd(t *testing.T) {
	dir := t.TempDir()
	got := buildDynamicContext(time.Now(), dir)
	if !strings.Contains(got, dir) {
		t.Errorf("buildDynamicContext() = %q, want it to contain cwd %q", got, dir)
	}
}

func TestBuildDynamicContext_NoGitBranch_WhenNotARepo(t *testing.T) {
	dir := t.TempDir()
	got := buildDynamicContext(time.Now(), dir)
	if strings.Contains(got, "Git branch:") {
		t.Errorf("buildDynamicContext() = %q, should not contain git branch outside a repo", got)
	}
}

func TestBuildDynamicContext_IncludesGitBranch_WhenInRepo(t *testing.T) {
	dir := t.TempDir()
	// Initialize a git repo with a known branch name.
	if err := exec.Command("git", "-C", dir, "init", "-b", "test-branch").Run(); err != nil {
		t.Skip("git not available:", err)
	}
	got := buildDynamicContext(time.Now(), dir)
	if !strings.Contains(got, "Git branch: test-branch") {
		t.Errorf("buildDynamicContext() = %q, want it to contain \"Git branch: test-branch\"", got)
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
