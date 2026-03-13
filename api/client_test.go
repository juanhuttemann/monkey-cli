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
		w.Header().Set("Content-Type", "application/json")
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

	if ct := receivedRequest.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if key := receivedRequest.Header.Get("x-api-key"); key != "my-secret-key" {
		t.Errorf("x-api-key = %q, want %q", key, "my-secret-key")
	}
	if version := receivedRequest.Header.Get("anthropic-version"); version != AnthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", version, AnthropicVersion)
	}
}

func TestSendMessage_SetsCorrectPath(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close() // Close immediately to force a connection error.

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

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
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

	if !strings.Contains(err.Error(), "content") {
		t.Errorf("error should mention content, got: %v", err)
	}
}

func TestSendMessage_WithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

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
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithModel("test-model-123"))
	_, err := client.SendMessage(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	// Unmarshal into a generic map to be format-agnostic about the SDK's request shape.
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}

	if body["model"] != "test-model-123" {
		t.Errorf("model = %q, want %q", body["model"], "test-model-123")
	}
	if mt, _ := body["max_tokens"].(float64); int(mt) != DefaultMaxTokens {
		t.Errorf("max_tokens = %v, want %d", body["max_tokens"], DefaultMaxTokens)
	}

	msgs, _ := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	firstMsg, _ := msgs[0].(map[string]any)
	if firstMsg["role"] != "user" {
		t.Errorf("message role = %q, want user", firstMsg["role"])
	}
	// The SDK sends content as an array of content blocks.
	content, _ := firstMsg["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected non-empty content array")
	}
	textBlock, _ := content[0].(map[string]any)
	if textBlock["text"] != "Hello, world!" {
		t.Errorf("text block text = %q, want %q", textBlock["text"], "Hello, world!")
	}
}

func TestSendMessage_BaseURLWithTrailingSlash(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
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
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithModel("m"), WithMaxTokens(4096))
	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	var body map[string]any
	_ = json.Unmarshal(rawBody, &body)
	if mt, _ := body["max_tokens"].(float64); int(mt) != 4096 {
		t.Errorf("max_tokens = %v, want 4096", body["max_tokens"])
	}
}

func TestDefaultMaxTokens_IsLargeEnoughForRealResponses(t *testing.T) {
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
		var body map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		lastModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
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
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
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

	var body map[string]any
	_ = json.Unmarshal(rawBody, &body)
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages in request, got %d", len(msgs))
	}

	textOf := func(i int) string {
		msg, _ := msgs[i].(map[string]any)
		content, _ := msg["content"].([]any)
		if len(content) == 0 {
			return ""
		}
		block, _ := content[0].(map[string]any)
		s, _ := block["text"].(string)
		return s
	}

	if got := textOf(0); got != "Message one" {
		t.Errorf("messages[0] text = %q, want %q", got, "Message one")
	}
	if got := textOf(1); got != "Response one" {
		t.Errorf("messages[1] text = %q, want %q", got, "Response one")
	}
	if got := textOf(2); got != "Message two" {
		t.Errorf("messages[2] text = %q, want %q", got, "Message two")
	}
}

func TestSendMessageWithHistory_EmptyMessages(t *testing.T) {
	client := NewClient("http://localhost", "test-key", WithModel("test-model"))
	_, err := client.SendMessageWithHistory(context.Background(), []Message{})
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
		w.Header().Set("Content-Type", "application/json")
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

	if ct := receivedRequest.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if key := receivedRequest.Header.Get("x-api-key"); key != "my-secret-key" {
		t.Errorf("x-api-key = %q, want %q", key, "my-secret-key")
	}
	if version := receivedRequest.Header.Get("anthropic-version"); version != AnthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", version, AnthropicVersion)
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
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
}

func TestSendMessageWithHistory_WithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	messages := []Message{{Role: "user", Content: "test"}}
	client := NewClient(server.URL, "test-key")

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.SendMessageWithHistory(ctx, messages)
	if err == nil {
		t.Fatal("SendMessageWithHistory() should return error when context is cancelled")
	}
}

func TestWithSystemPrompt_IncludedInRequest(t *testing.T) {
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", WithSystemPrompt("You are a helpful coding assistant."))
	_, err := client.SendMessage(context.Background(), "hi")
	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	// The SDK sends system as an array of text blocks; check that the text is present.
	if !strings.Contains(string(rawBody), "You are a helpful coding assistant.") {
		t.Errorf("system prompt not found in request body: %s", rawBody)
	}
}

func TestWithSystemPrompt_Empty_OmittedFromRequest(t *testing.T) {
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
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
