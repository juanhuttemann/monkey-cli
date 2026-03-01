package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPrintHello(t *testing.T) {
	// Setup mock server for LLM API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "Hello from LLM!"}]}`))
	}))
	defer server.Close()

	// Set environment variables
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	got := printHello()
	want := "Hello from LLM!"

	if got != want {
		t.Errorf("printHello() = %q, want %q", got, want)
	}
}

func TestGetGreeting_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages path, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "Hello from Claude!"}]}`))
	}))
	defer server.Close()

	// Set environment variables
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	// Call getGreeting
	got, err := getGreeting()
	if err != nil {
		t.Fatalf("getGreeting() returned error: %v", err)
	}

	want := "Hello from Claude!"
	if got != want {
		t.Errorf("getGreeting() = %q, want %q", got, want)
	}
}

func TestGetGreeting_MissingAPIKey(t *testing.T) {
	// Ensure API key is not set
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_BASE_URL", "http://localhost")
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error when ANTHROPIC_API_KEY is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error should mention ANTHROPIC_API_KEY, got: %v", err)
	}
}

func TestGetGreeting_MissingBaseURL(t *testing.T) {
	// Ensure base URL is not set
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error when ANTHROPIC_BASE_URL is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_BASE_URL") {
		t.Errorf("error should mention ANTHROPIC_BASE_URL, got: %v", err)
	}
}

func TestGetGreeting_MissingModel(t *testing.T) {
	// Ensure model is not set
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("ANTHROPIC_BASE_URL", "http://localhost")
	os.Unsetenv("ANTHROPIC_MODEL")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error when ANTHROPIC_MODEL is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_MODEL") {
		t.Errorf("error should mention ANTHROPIC_MODEL, got: %v", err)
	}
}

func TestGetGreeting_HTTPError(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately
		w.WriteHeader(http.StatusInternalServerError)
	}))
	// Close the server before making request to force error
	server.Close()

	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error on network failure")
	}

	// Error should indicate request failure
	errMsg := err.Error()
	if !strings.Contains(errMsg, "request") && !strings.Contains(errMsg, "connection") {
		t.Errorf("error should indicate request/connection failure, got: %v", err)
	}
}

func TestGetGreeting_Non200Status(t *testing.T) {
	// Setup mock server returning 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error on non-200 status")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
	if !strings.Contains(errMsg, "internal server error") {
		t.Errorf("error should contain response body, got: %v", err)
	}
}

func TestGetGreeting_InvalidJSON(t *testing.T) {
	// Setup mock server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [}`)) // Invalid JSON
	}))
	defer server.Close()

	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error on invalid JSON")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") {
		t.Errorf("error should indicate parse/unmarshal failure, got: %v", err)
	}
}

func TestGetGreeting_EmptyContent(t *testing.T) {
	// Setup mock server returning empty content array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": []}`))
	}))
	defer server.Close()

	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := getGreeting()
	if err == nil {
		t.Fatal("getGreeting() should return error on empty content")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}
