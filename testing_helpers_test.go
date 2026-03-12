package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture reads a file from testdata/ and returns its bytes.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %q: %v", name, err)
	}
	return data
}

func TestSetupTestEnv_SetsAllVars(t *testing.T) {
	cleanup := setupTestEnv("test-api-key", "http://test-url", "test-model")
	defer cleanup()

	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "test-api-key" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", got, "test-api-key")
	}
	if got := os.Getenv("ANTHROPIC_BASE_URL"); got != "http://test-url" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", got, "http://test-url")
	}
	if got := os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL"); got != "test-model" {
		t.Errorf("ANTHROPIC_DEFAULT_OPUS_MODEL = %q, want %q", got, "test-model")
	}
}

func TestSetupTestEnv_CleanupRemovesVars(t *testing.T) {
	// Ensure vars are not set before test
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	_ = os.Unsetenv("ANTHROPIC_BASE_URL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")

	cleanup := setupTestEnv("test-api-key", "http://test-url", "test-model")
	cleanup()

	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "" {
		t.Errorf("ANTHROPIC_API_KEY should be unset after cleanup, got %q", got)
	}
	if got := os.Getenv("ANTHROPIC_BASE_URL"); got != "" {
		t.Errorf("ANTHROPIC_BASE_URL should be unset after cleanup, got %q", got)
	}
	if got := os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL"); got != "" {
		t.Errorf("ANTHROPIC_DEFAULT_OPUS_MODEL should be unset after cleanup, got %q", got)
	}
}

func TestSetupTestEnv_PartialVars(t *testing.T) {
	// Ensure clean state
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	_ = os.Unsetenv("ANTHROPIC_BASE_URL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")

	// Set only some vars
	cleanup := setupTestEnv("", "http://test-url", "")
	defer cleanup()

	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "" {
		t.Errorf("ANTHROPIC_API_KEY should be empty, got %q", got)
	}
	if got := os.Getenv("ANTHROPIC_BASE_URL"); got != "http://test-url" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", got, "http://test-url")
	}
	if got := os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL"); got != "" {
		t.Errorf("ANTHROPIC_DEFAULT_OPUS_MODEL should be empty, got %q", got)
	}
}

func TestCreateMockServer_ReturnsCorrectResponse(t *testing.T) {
	expectedBody := `{"content": [{"type": "text", "text": "Hello!"}]}`
	server, cleanup := createMockServer(expectedBody, http.StatusOK)
	defer cleanup()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestCreateMockServer_CleanupClosesServer(t *testing.T) {
	expectedBody := `{"test": "data"}`
	server, cleanup := createMockServer(expectedBody, http.StatusOK)

	// Server should be accessible before cleanup
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request before cleanup: %v", err)
	}
	_ = resp.Body.Close()

	// Cleanup should close the server
	cleanup()

	// After cleanup, request should fail
	_, err = http.Get(server.URL)
	if err == nil {
		t.Error("Expected error after server closed, got nil")
	}
}

func TestCreateMockServerWithValidator_ReceivesCorrectRequest(t *testing.T) {
	var receivedRequest *http.Request
	validator := func(r *http.Request) bool {
		receivedRequest = r
		return true
	}

	expectedBody := `{"content": [{"type": "text", "text": "Validated!"}]}`
	server, cleanup := createMockServerWithValidator(validator, expectedBody, http.StatusOK)
	defer cleanup()

	// Make a POST request
	resp, err := http.Post(server.URL+"/v1/messages", "application/json", strings.NewReader(`{"test":"data"}`))
	if err == nil {
		_ = resp.Body.Close()
	}

	if receivedRequest == nil {
		t.Fatal("Validator did not receive any request")
	}

	if receivedRequest.Method != "POST" {
		t.Errorf("Request method = %q, want %q", receivedRequest.Method, "POST")
	}

	if receivedRequest.URL.Path != "/v1/messages" {
		t.Errorf("Request path = %q, want %q", receivedRequest.URL.Path, "/v1/messages")
	}
}

// Additional helper tests

func TestSuccessResponse_BuildsCorrectJSON(t *testing.T) {
	got := successResponse("Hello, World!")
	want := `{"content": [{"type": "text", "text": "Hello, World!"}]}`

	if got != want {
		t.Errorf("successResponse() = %q, want %q", got, want)
	}
}

func TestCreateMockServer_CustomStatusCode(t *testing.T) {
	server, cleanup := createMockServer(`{"error": "not found"}`, http.StatusNotFound)
	defer cleanup()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestSetupTestEnv_PreservesExistingVars(t *testing.T) {
	// Set up initial values
	_ = os.Setenv("ANTHROPIC_API_KEY", "original-key")
	_ = os.Setenv("ANTHROPIC_BASE_URL", "original-url")
	_ = os.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "original-model")

	cleanup := setupTestEnv("new-key", "new-url", "new-model")
	cleanup()

	// After cleanup, should restore original values
	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "original-key" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", got, "original-key")
	}
	if got := os.Getenv("ANTHROPIC_BASE_URL"); got != "original-url" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", got, "original-url")
	}
	if got := os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL"); got != "original-model" {
		t.Errorf("ANTHROPIC_DEFAULT_OPUS_MODEL = %q, want %q", got, "original-model")
	}

	// Clean up
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	_ = os.Unsetenv("ANTHROPIC_BASE_URL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")
}
