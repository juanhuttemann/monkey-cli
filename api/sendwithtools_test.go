package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// fixture reads a file from testdata/, falling back to ../testdata/ for shared fixtures.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	for _, dir := range []string{"testdata", "../testdata"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return data
		}
	}
	t.Fatalf("fixture %q: not found in testdata/ or ../testdata/", name)
	return nil
}

// stubBashTool returns a minimal Tool for use in SendMessageWithTools tests.
func stubBashTool() Tool {
	return Tool{
		Name:        "bash",
		Description: "Execute a bash command.",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]PropertyDef{"command": {Type: "string", Description: "command"}},
			Required:   []string{"command"},
		},
	}
}

// mockExecutor is a ToolExecutor that returns a preset result.
type mockExecutor struct {
	result string
	err    error
}

func (m mockExecutor) ExecuteTool(_ context.Context, _ string, _ map[string]any) (string, error) {
	return m.result, m.err
}

// toolUseResponse builds a JSON response containing a tool_use block.
func toolUseResponse(toolID, toolName string, input map[string]any) string {
	inputJSON, _ := json.Marshal(input)
	return `{"content":[{"type":"tool_use","id":"` + toolID + `","name":"` + toolName + `","input":` + string(inputJSON) + `}],"stop_reason":"tool_use"}`
}

// textResponse builds a JSON response containing a text block.
func textResponse(text string) string {
	b, _ := json.Marshal(text)
	return `{"content":[{"type":"text","text":` + string(b) + `}],"stop_reason":"end_turn"}`
}

func TestSendMessageWithTools_NoToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("plain answer")))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, _, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "unused"},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}
	if result != "plain answer" {
		t.Errorf("result = %q, want %q", result, "plain answer")
	}
}

func TestSendMessageWithTools_SingleToolCall(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write([]byte(toolUseResponse("toolu_1", "bash", map[string]any{"command": "echo hi"})))
		} else {
			_, _ = w.Write([]byte(textResponse("done")))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, _, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "run echo"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "hi\n"},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}
	if result != "done" {
		t.Errorf("result = %q, want %q", result, "done")
	}
	if requestCount.Load() != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", requestCount.Load())
	}
}

func TestSendMessageWithTools_SendsToolsInFirstRequest(t *testing.T) {
	var firstBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if firstBody == nil {
			firstBody = body
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("ok")))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, _, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: ""},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}

	var req apiRequest
	_ = json.Unmarshal(firstBody, &req)
	if len(req.Tools) == 0 {
		t.Fatal("expected tools to be sent in request, got none")
	}
	if req.Tools[0].Name != "bash" {
		t.Errorf("Tools[0].Name = %q, want %q", req.Tools[0].Name, "bash")
	}
}

func TestSendMessageWithTools_SendsToolResultInSecondRequest(t *testing.T) {
	var secondBody []byte
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		if n == 2 {
			secondBody = body
		}
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write([]byte(toolUseResponse("toolu_42", "bash", map[string]any{"command": "pwd"})))
		} else {
			_, _ = w.Write([]byte(textResponse("ok")))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, _, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "where am I?"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "/home/user\n"},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}

	var req apiRequest
	_ = json.Unmarshal(secondBody, &req)

	// Last message should be user with tool_result content
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" {
		t.Errorf("last message role = %q, want %q", last.Role, "user")
	}

	// Content should be a JSON array with a tool_result block
	contentJSON, _ := json.Marshal(last.Content)
	if !strings.Contains(string(contentJSON), "tool_result") {
		t.Errorf("last message content should contain tool_result, got: %s", contentJSON)
	}
	if !strings.Contains(string(contentJSON), "toolu_42") {
		t.Errorf("last message content should reference tool_use_id toolu_42, got: %s", contentJSON)
	}
	if !strings.Contains(string(contentJSON), "/home/user") {
		t.Errorf("last message content should include tool output, got: %s", contentJSON)
	}
}

func TestSendMessageWithTools_AssistantToolUseAddedToHistory(t *testing.T) {
	var secondBody []byte
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		if n == 2 {
			secondBody = body
		}
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write([]byte(toolUseResponse("toolu_7", "bash", map[string]any{"command": "ls"})))
		} else {
			_, _ = w.Write([]byte(textResponse("ok")))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, _, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "list files"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "file.txt\n"},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}

	var req apiRequest
	_ = json.Unmarshal(secondBody, &req)

	// Messages in second request: [user, assistant(tool_use), user(tool_result)]
	if len(req.Messages) != 3 {
		t.Fatalf("expected 3 messages in second request, got %d", len(req.Messages))
	}
	if req.Messages[1].Role != "assistant" {
		t.Errorf("messages[1].Role = %q, want %q", req.Messages[1].Role, "assistant")
	}
	contentJSON, _ := json.Marshal(req.Messages[1].Content)
	if !strings.Contains(string(contentJSON), "tool_use") {
		t.Errorf("assistant message content should contain tool_use, got: %s", contentJSON)
	}
}

func TestSendMessageWithTools_EmptyMessages(t *testing.T) {
	client := NewClient("http://localhost", "key")
	_, _, _, err := client.SendMessageWithTools(context.Background(), nil, []Tool{stubBashTool()}, mockExecutor{})
	if err == nil {
		t.Error("SendMessageWithTools() should return error for nil messages")
	}
}

func TestSendMessageWithTools_AccumulatesUsage(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write(fixture(t, "tool_use_date_with_usage.json"))
		} else {
			_, _ = w.Write(fixture(t, "response_done_with_usage.json"))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, _, usage, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "output"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.InputTokens != 30 {
		t.Errorf("InputTokens = %d, want 30 (10+20)", usage.InputTokens)
	}
	if usage.OutputTokens != 8 {
		t.Errorf("OutputTokens = %d, want 8 (5+3)", usage.OutputTokens)
	}
}

func TestSendMessageWithTools_ReturnsAccumulatedMessages(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write([]byte(toolUseResponse("toolu_99", "bash", map[string]any{"command": "date"})))
		} else {
			_, _ = w.Write([]byte(textResponse("today")))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, msgs, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "what day is it?"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: "Monday\n"},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}

	// Expect: [user, assistant(tool_use), user(tool_result), assistant(final)]
	if len(msgs) != 4 {
		t.Fatalf("expected 4 accumulated messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("msgs[0].Role = %q, want user", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("msgs[1].Role = %q, want assistant", msgs[1].Role)
	}
	if msgs[2].Role != "user" {
		t.Errorf("msgs[2].Role = %q, want user (tool_result)", msgs[2].Role)
	}
	// Final assistant message contains the text response
	if msgs[3].Role != "assistant" {
		t.Errorf("msgs[3].Role = %q, want assistant", msgs[3].Role)
	}
	if msgs[3].Content != "today" {
		t.Errorf("msgs[3].Content = %v, want %q", msgs[3].Content, "today")
	}
}

func TestSendMessageWithTools_NoToolUse_ReturnsMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("simple answer")))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	text, msgs, _, err := client.SendMessageWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		[]Tool{stubBashTool()},
		mockExecutor{result: ""},
	)
	if err != nil {
		t.Fatalf("SendMessageWithTools() returned error: %v", err)
	}
	if text != "simple answer" {
		t.Errorf("text = %q, want %q", text, "simple answer")
	}
	// Expect: [user, assistant(final)]
	if len(msgs) != 2 {
		t.Fatalf("expected 2 accumulated messages, got %d", len(msgs))
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "simple answer" {
		t.Errorf("msgs[1] = {%q, %v}, want {assistant, simple answer}", msgs[1].Role, msgs[1].Content)
	}
}
