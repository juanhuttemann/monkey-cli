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
		_, _ = w.Write(fixture(t, "response_hello.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "error_internal.json"))
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
		_, _ = w.Write(fixture(t, "response_invalid.json"))
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
		_, _ = w.Write(fixture(t, "response_empty_content.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_ = json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
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

func TestNewClient_StripsTrailingSlash(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"https://api.example.com/", "https://api.example.com"},
		{"https://api.example.com//", "https://api.example.com"},
		{"https://api.example.com", "https://api.example.com"},
	}
	for _, tc := range cases {
		client := NewClient(tc.input, "key")
		if client.baseURL != tc.want {
			t.Errorf("NewClient(%q).baseURL = %q, want %q", tc.input, client.baseURL, tc.want)
		}
	}
}

func TestSendMessage_BaseURLWithTrailingSlash(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", "test-key")
	_, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if requestPath != MessagesEndpoint {
		t.Errorf("Request path = %q, want %q (double-slash from trailing slash bug)", requestPath, MessagesEndpoint)
	}
}

func TestWithMaxTokens_OverridesDefaultInRequest(t *testing.T) {
	var requestBody apiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithModel("m"), WithMaxTokens(4096))
	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}
	if requestBody.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", requestBody.MaxTokens)
	}
}

func TestDefaultMaxTokens_IsLargeEnoughForRealResponses(t *testing.T) {
	// 100 tokens ≈ 75 words ≈ 7 lines — too small for useful LLM responses.
	const minReasonable = 8192
	if DefaultMaxTokens < minReasonable {
		t.Errorf("DefaultMaxTokens = %d, want >= %d; small value cuts responses short", DefaultMaxTokens, minReasonable)
	}
}

func TestConstants(t *testing.T) {
	if MessagesEndpoint != "/v1/messages" {
		t.Errorf("MessagesEndpoint = %q, want %q", MessagesEndpoint, "/v1/messages")
	}
}

func TestClient_SetModel_ChangesModel(t *testing.T) {
	var lastModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body apiRequest
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		lastModel = body.Model
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("original"))
	client.SetModel("updated")

	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}
	if lastModel != "updated" {
		t.Errorf("model in request = %q, want %q", lastModel, "updated")
	}
}

func TestClient_GetModel(t *testing.T) {
	client := NewClient("https://api.example.com", "key", WithModel("claude-opus-4"))
	if got := client.GetModel(); got != "claude-opus-4" {
		t.Errorf("GetModel() = %q, want %q", got, "claude-opus-4")
	}

	client.SetModel("claude-sonnet-4")
	if got := client.GetModel(); got != "claude-sonnet-4" {
		t.Errorf("GetModel() after SetModel = %q, want %q", got, "claude-sonnet-4")
	}
}

func TestSendMessageWithHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_with_history.json"))
	}))
	defer server.Close()

	messages := []Message{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "First response"},
		{Role: "user", Content: "Second message"},
	}

	client := NewClient(server.URL, "test-key", WithModel("test-model"))
	result, err := client.SendMessageWithHistory(context.Background(), messages)
	if err != nil {
		t.Fatalf("SendMessageWithHistory() returned error: %v", err)
	}

	if result != "Response with history!" {
		t.Errorf("SendMessageWithHistory() = %q, want %q", result, "Response with history!")
	}
}

func TestSendMessageWithHistory_SendsAllMessages(t *testing.T) {
	var requestBody apiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	messages := []Message{
		{Role: "user", Content: "Message one"},
		{Role: "assistant", Content: "Response one"},
		{Role: "user", Content: "Message two"},
	}

	client := NewClient(server.URL, "test-key", WithModel("test-model"))
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err != nil {
		t.Fatalf("SendMessageWithHistory() returned error: %v", err)
	}

	// Verify all messages were sent
	if len(requestBody.Messages) != 3 {
		t.Fatalf("Expected 3 messages in request, got %d", len(requestBody.Messages))
	}
	if requestBody.Messages[0].Content != "Message one" {
		t.Errorf("Messages[0].Content = %q, want %q", requestBody.Messages[0].Content, "Message one")
	}
	if requestBody.Messages[1].Content != "Response one" {
		t.Errorf("Messages[1].Content = %q, want %q", requestBody.Messages[1].Content, "Response one")
	}
	if requestBody.Messages[2].Content != "Message two" {
		t.Errorf("Messages[2].Content = %q, want %q", requestBody.Messages[2].Content, "Message two")
	}
}

func TestSendMessageWithHistory_EmptyMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	messages := []Message{}

	client := NewClient(server.URL, "test-key", WithModel("test-model"))
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error with empty messages")
	}

	if !strings.Contains(err.Error(), "message") {
		t.Errorf("error should mention messages, got: %v", err)
	}
}

func TestSendMessageWithHistory_SetsCorrectHeaders(t *testing.T) {
	var receivedRequest *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}

	client := NewClient(server.URL, "my-secret-key")
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err != nil {
		t.Fatalf("SendMessageWithHistory() returned error: %v", err)
	}

	if receivedRequest == nil {
		t.Fatal("Did not receive request")
	}

	// Check headers
	if ct := receivedRequest.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if key := receivedRequest.Header.Get("x-api-key"); key != "my-secret-key" {
		t.Errorf("x-api-key = %q, want %q", key, "my-secret-key")
	}
	if version := receivedRequest.Header.Get("anthropic-version"); version != "2023-06-01" {
		t.Errorf("anthropic-version = %q, want %q", version, "2023-06-01")
	}
}

func TestSendMessageWithHistory_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(fixture(t, "error_internal.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error on non-200 status")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
}

func TestSendMessageWithHistory_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_invalid.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error on invalid JSON")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") {
		t.Errorf("error should indicate parse/unmarshal failure, got: %v", err)
	}
}

func TestSendMessageWithHistory_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_empty_content.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessageWithHistory(context.Background(), messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error on empty content")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}

func TestWithSystemPrompt_IncludedInRequest(t *testing.T) {
	var requestBody apiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithSystemPrompt("You are a helpful coding assistant."))
	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}
	if requestBody.System != "You are a helpful coding assistant." {
		t.Errorf("System = %q, want %q", requestBody.System, "You are a helpful coding assistant.")
	}
}

func TestWithSystemPrompt_Empty_OmittedFromRequest(t *testing.T) {
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}
	if strings.Contains(string(rawBody), `"system"`) {
		t.Errorf("expected no system field in request when not set, got: %s", rawBody)
	}
}

func TestSendMessageWithHistory_WithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}

	client := NewClient(server.URL, "test-key")

	// Cancel context before request completes
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.SendMessageWithHistory(ctx, messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error when context is cancelled")
	}
}

func TestStatusError_FriendlyMessage_WithStructuredBody(t *testing.T) {
	body := `{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your rate limit."}}`
	e := &StatusError{StatusCode: 429, Body: body}
	got := e.FriendlyMessage()
	if !strings.Contains(got, "429") {
		t.Errorf("FriendlyMessage() = %q, want status code 429", got)
	}
	if !strings.Contains(got, "This request would exceed your rate limit.") {
		t.Errorf("FriendlyMessage() = %q, want API message text", got)
	}
	// Should NOT contain the raw JSON
	if strings.Contains(got, `{"type"`) {
		t.Errorf("FriendlyMessage() = %q, should not contain raw JSON", got)
	}
}

func TestStatusError_FriendlyMessage_WithUnstructuredBody(t *testing.T) {
	e := &StatusError{StatusCode: 500, Body: "internal server error"}
	got := e.FriendlyMessage()
	if !strings.Contains(got, "500") {
		t.Errorf("FriendlyMessage() = %q, want status code 500", got)
	}
}

func TestStatusError_FriendlyMessage_WithEmptyBody(t *testing.T) {
	e := &StatusError{StatusCode: 401, Body: ""}
	got := e.FriendlyMessage()
	if !strings.Contains(got, "401") {
		t.Errorf("FriendlyMessage() = %q, want status code 401", got)
	}
}
