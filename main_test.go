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
		w.Write([]byte(`{"content": [{"type": "text", "text": "The capital of France is Paris."}]}`))
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
		json.Unmarshal(body, &requestBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
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
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_BASE_URL", "http://localhost")
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error when ANTHROPIC_API_KEY is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error should mention ANTHROPIC_API_KEY, got: %v", err)
	}
}

func TestSendPrompt_MissingBaseURL(t *testing.T) {
	// Ensure base URL is not set
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error when ANTHROPIC_BASE_URL is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_BASE_URL") {
		t.Errorf("error should mention ANTHROPIC_BASE_URL, got: %v", err)
	}
}

func TestSendPrompt_MissingModel(t *testing.T) {
	// Ensure model is not set
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("ANTHROPIC_BASE_URL", "http://localhost")
	os.Unsetenv("ANTHROPIC_MODEL")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error when ANTHROPIC_MODEL is missing")
	}

	if !strings.Contains(err.Error(), "ANTHROPIC_MODEL") {
		t.Errorf("error should mention ANTHROPIC_MODEL, got: %v", err)
	}
}

func TestSendPrompt_HTTPError(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close()

	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", server.URL)
	os.Setenv("ANTHROPIC_MODEL", "test-model")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
		os.Unsetenv("ANTHROPIC_MODEL")
	}()

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

	_, err := sendPrompt("test prompt")
	if err == nil {
		t.Fatal("sendPrompt() should return error on empty content")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}

func TestPrintUsage(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsage()

	w.Close()
	os.Stderr = oldStderr

	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected strings
	if !strings.Contains(output, "Usage: mogger -p") {
		t.Errorf("printUsage() should contain 'Usage: mogger -p', got: %q", output)
	}
	if !strings.Contains(output, "-p, --prompt") {
		t.Errorf("printUsage() should contain '-p, --prompt', got: %q", output)
	}
	if !strings.Contains(output, "required") {
		t.Errorf("printUsage() should contain 'required', got: %q", output)
	}
}
