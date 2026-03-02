package tui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mogger/api"
)

func TestSendPromptCmd_ReturnsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{{Role: "user", Content: "test"}}

	cmd, _ := SendPromptCmd(client, messages, "test prompt")
	if cmd == nil {
		t.Fatal("SendPromptCmd() returned nil, want non-nil tea.Cmd")
	}
}

func TestSendPromptCmd_SendsWithContext(t *testing.T) {
	requestReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{{Role: "user", Content: "test"}}

	cmd, _ := SendPromptCmd(client, messages, "test prompt")
	result := cmd()

	// Verify request was made
	if !requestReceived {
		t.Error("SendPromptCmd did not send request to server")
	}

	// Verify response type
	_, ok := result.(PromptResponseMsg)
	if !ok {
		t.Errorf("SendPromptCmd() returned %T, want PromptResponseMsg", result)
	}
}

func TestSendPromptCmd_UsesConversationHistory(t *testing.T) {
	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{
		{Role: "user", Content: "first message"},
		{Role: "assistant", Content: "first response"},
		{Role: "user", Content: "second message"},
	}

	cmd, _ := SendPromptCmd(client, messages, "new prompt")
	_ = cmd()

	// Verify request body contains conversation history
	bodyStr := string(requestBody)
	if !strings.Contains(bodyStr, "first message") {
		t.Error("Request body should contain 'first message' from history")
	}
	if !strings.Contains(bodyStr, "first response") {
		t.Error("Request body should contain 'first response' from history")
	}
	if !strings.Contains(bodyStr, "new prompt") {
		t.Error("Request body should contain 'new prompt'")
	}
}

func TestSendPromptCmd_SuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "Hello from API!"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	cmd, _ := SendPromptCmd(client, messages, "test prompt")
	result := cmd()

	response, ok := result.(PromptResponseMsg)
	if !ok {
		t.Fatalf("SendPromptCmd() returned %T, want PromptResponseMsg", result)
	}
	if response.Err != nil {
		t.Errorf("PromptResponseMsg.Err = %v, want nil", response.Err)
	}
	if response.Response != "Hello from API!" {
		t.Errorf("PromptResponseMsg.Response = %q, want %q", response.Response, "Hello from API!")
	}
}

func TestSendPromptCmd_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	cmd, _ := SendPromptCmd(client, messages, "test prompt")
	result := cmd()

	errMsg, ok := result.(PromptErrorMsg)
	if !ok {
		t.Fatalf("SendPromptCmd() returned %T, want PromptErrorMsg", result)
	}
	if errMsg.Err == nil {
		t.Error("PromptErrorMsg.Err = nil, want error")
	}
}

func TestSendPromptCmd_TimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "too late"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	// Use a short timeout for testing
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test prompt", 100*time.Millisecond, nil)
	result := cmd()

	errMsg, ok := result.(PromptErrorMsg)
	if !ok {
		t.Fatalf("SendPromptCmdWithTimeout() returned %T, want PromptErrorMsg", result)
	}
	if errMsg.Err == nil {
		t.Error("PromptErrorMsg.Err = nil, want timeout error")
	}
	if !strings.Contains(errMsg.Err.Error(), "context") && !strings.Contains(errMsg.Err.Error(), "deadline") {
		t.Errorf("Error should indicate timeout/context issue, got: %v", errMsg.Err)
	}
}

func TestSendPromptCmdWithTimeout_RespectsTimeout(t *testing.T) {
	startTime := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	timeout := 100 * time.Millisecond
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test", timeout, nil)
	_ = cmd()

	elapsed := time.Since(startTime)

	// Should timeout within reasonable margin of the specified timeout
	if elapsed > 500*time.Millisecond {
		t.Errorf("Request took %v, should have timed out around %v", elapsed, timeout)
	}
}

func TestSendPromptCmd_Cancel_ReturnsCancelledMsg(t *testing.T) {
	// Server that blocks until the request context is done
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	cmd, cancel := SendPromptCmd(client, messages, "test prompt")

	// Cancel immediately before executing the cmd
	cancel()

	result := cmd()

	if _, ok := result.(PromptCancelledMsg); !ok {
		t.Errorf("SendPromptCmd() after cancel returned %T, want PromptCancelledMsg", result)
	}
}

func TestSendPromptCmdWithTimeout_StreamsToolCallsToChannel(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		requestCount++
		if requestCount == 1 {
			w.Write([]byte(`{"content":[{"type":"tool_use","id":"t1","name":"bash","input":{"command":"ls"}}],"stop_reason":"tool_use"}`))
		} else {
			w.Write([]byte(`{"content":[{"type":"text","text":"done"}]}`))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	toolCallCh := make(chan ToolCallMsg, 10)
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test", 5*time.Second, toolCallCh)
	result := cmd()

	// Channel should have received the tool call
	select {
	case tc := <-toolCallCh:
		if tc.ToolCall.Name != "bash" {
			t.Errorf("tool call name = %q, want bash", tc.ToolCall.Name)
		}
	default:
		t.Error("expected tool call in channel, but it was empty")
	}

	// Final result is PromptResponseMsg (tool calls are not embedded in it)
	resp, ok := result.(PromptResponseMsg)
	if !ok {
		t.Fatalf("result = %T, want PromptResponseMsg", result)
	}
	if resp.Response != "done" {
		t.Errorf("Response = %q, want %q", resp.Response, "done")
	}
}

// SendPromptCmdWithTimeout is tested above - this tests the exported version
func TestSendPromptCmdWithTimeout_ReturnsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []Message{}

	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test prompt", 5*time.Second, nil)
	if cmd == nil {
		t.Fatal("SendPromptCmdWithTimeout() returned nil, want non-nil tea.Cmd")
	}
}
