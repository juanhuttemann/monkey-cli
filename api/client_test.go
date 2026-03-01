package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_SetsFields(t *testing.T) {
	client := NewClient("https://api.example.com", "test-api-key")

	if client.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.example.com")
	}
	if client.apiKey != "test-api-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-api-key")
	}
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := NewClient("https://api.example.com", "test-key", WithHTTPClient(customClient))

	if client.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}

	// Verify it's our custom client
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, 30*time.Second)
	}
}

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	client := NewClient("https://api.example.com", "test-key")

	if client.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}
}

func TestSendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "Hello from API!"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.SendMessage(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if result != "Hello from API!" {
		t.Errorf("SendMessage() = %q, want %q", result, "Hello from API!")
	}
}

func TestSendMessage_SetsCorrectHeaders(t *testing.T) {
	var receivedRequest *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "my-secret-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if receivedRequest == nil {
		t.Fatal("Did not receive request")
	}

	// Check Content-Type header
	if ct := receivedRequest.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	// Check x-api-key header
	if key := receivedRequest.Header.Get("x-api-key"); key != "my-secret-key" {
		t.Errorf("x-api-key = %q, want %q", key, "my-secret-key")
	}

	// Check anthropic-version header
	if version := receivedRequest.Header.Get("anthropic-version"); version != "2023-06-01" {
		t.Errorf("anthropic-version = %q, want %q", version, "2023-06-01")
	}
}

func TestSendMessage_SetsCorrectPath(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if requestPath != MessagesEndpoint {
		t.Errorf("Request path = %q, want %q", requestPath, MessagesEndpoint)
	}
}

func TestSendMessage_SetsCorrectMethod(t *testing.T) {
	var method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if method != http.MethodPost {
		t.Errorf("Request method = %q, want %q", method, http.MethodPost)
	}
}

func TestSendMessage_HTTPError(t *testing.T) {
	// Create a server that we'll close to force an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close() // Close immediately

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("SendMessage() should return error on network failure")
	}
}

func TestSendMessage_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("SendMessage() should return error on non-200 status")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
	if !strings.Contains(errMsg, "internal server error") {
		t.Errorf("error should contain response body, got: %v", err)
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [}`)) // Invalid JSON
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("SendMessage() should return error on invalid JSON")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") && !strings.Contains(errMsg, "decode") {
		t.Errorf("error should indicate parse/unmarshal failure, got: %v", err)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("SendMessage() should return error on empty content")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}

func TestSendMessage_WithContext(t *testing.T) {
	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	// Cancel context before request completes
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.SendMessage(ctx, "test")
	if err == nil {
		t.Fatal("SendMessage() should return error when context is cancelled")
	}
}

func TestSendMessage_SendsCorrectRequestBody(t *testing.T) {
	var requestBody apiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithModel("test-model-123"))
	_, err := client.SendMessage(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if requestBody.Model != "test-model-123" {
		t.Errorf("Model = %q, want %q", requestBody.Model, "test-model-123")
	}
	if requestBody.MaxTokens != DefaultMaxTokens {
		t.Errorf("MaxTokens = %d, want %d", requestBody.MaxTokens, DefaultMaxTokens)
	}
	if len(requestBody.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(requestBody.Messages))
	}
	if requestBody.Messages[0].Role != "user" {
		t.Errorf("Message role = %q, want %q", requestBody.Messages[0].Role, "user")
	}
	if requestBody.Messages[0].Content != "Hello, world!" {
		t.Errorf("Message content = %q, want %q", requestBody.Messages[0].Content, "Hello, world!")
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are correct
	if DefaultMaxTokens != 100 {
		t.Errorf("DefaultMaxTokens = %d, want %d", DefaultMaxTokens, 100)
	}
	if MessagesEndpoint != "/v1/messages" {
		t.Errorf("MessagesEndpoint = %q, want %q", MessagesEndpoint, "/v1/messages")
	}
}
