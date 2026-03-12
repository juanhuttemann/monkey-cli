package tui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/juanhuttemann/monkey-cli/api"
)

func TestSendPromptCmd_ReturnsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_text.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []api.Message{{Role: "user", Content: "test"}}

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
		_, _ = w.Write(fixture(t, "response_text.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []api.Message{{Role: "user", Content: "test"}}

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
		_, _ = w.Write(fixture(t, "response_text.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	messages := []api.Message{
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
		_, _ = w.Write(fixture(t, "response_hello.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))

	cmd, _ := SendPromptCmd(client, nil, "test prompt")
	result := cmd()

	response, ok := result.(PromptResponseMsg)
	if !ok {
		t.Fatalf("SendPromptCmd() returned %T, want PromptResponseMsg", result)
	}
	if response.Response != "Hello from API!" {
		t.Errorf("PromptResponseMsg.Response = %q, want %q", response.Response, "Hello from API!")
	}
}

func TestSendPromptCmd_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(fixture(t, "error_internal.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))

	cmd, _ := SendPromptCmd(client, nil, "test prompt")
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
		_, _ = w.Write(fixture(t, "response_too_late.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))

	// Use a short timeout for testing
	cmd, _ := SendPromptCmdWithTimeout(client, nil, "test prompt", 100*time.Millisecond, nil, nil, nil)
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

	timeout := 100 * time.Millisecond
	cmd, _ := SendPromptCmdWithTimeout(client, nil, "test", timeout, nil, nil, nil)
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

	cmd, cancel := SendPromptCmd(client, nil, "test prompt")

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
			_, _ = w.Write(fixture(t, "tool_use_ls.json"))
		} else {
			_, _ = w.Write(fixture(t, "response_done.json"))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))

	toolCallCh := make(chan ToolCallMsg, 10)
	cmd, _ := SendPromptCmdWithTimeout(client, nil, "test", 5*time.Second, toolCallCh, nil, nil)
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

func TestSendPromptCmd_SendsAllTools(t *testing.T) {
	var firstBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if firstBody == nil {
			firstBody = body
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	cmd, _ := SendPromptCmd(client, nil, "hi")
	cmd()

	body := string(firstBody)
	for _, name := range []string{"bash", "read", "write", "edit", "glob", "grep"} {
		if !strings.Contains(body, `"`+name+`"`) {
			t.Errorf("expected tool %q in request body", name)
		}
	}
}

func TestSendPromptCmd_PromptResponseContainsAPIMessages(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write(fixture(t, "tool_use_ls.json"))
		} else {
			_, _ = w.Write(fixture(t, "response_files_listed.json"))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	cmd, _ := SendPromptCmd(client, nil, "list my files")
	result := cmd()

	resp, ok := result.(PromptResponseMsg)
	if !ok {
		t.Fatalf("result = %T, want PromptResponseMsg", result)
	}
	// APIMessages should contain: [user, assistant(tool_use), user(tool_result), assistant(final)]
	if len(resp.APIMessages) != 4 {
		t.Fatalf("APIMessages length = %d, want 4", len(resp.APIMessages))
	}
	if resp.APIMessages[0].Role != "user" {
		t.Errorf("APIMessages[0].Role = %q, want user", resp.APIMessages[0].Role)
	}
	if resp.APIMessages[3].Role != "assistant" {
		t.Errorf("APIMessages[3].Role = %q, want assistant", resp.APIMessages[3].Role)
	}
}

func TestSendPromptCmd_APIMessagesPassedAsPriorHistory(t *testing.T) {
	var bodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	// Simulate prior turn: accumulated messages that include a tool_use/tool_result exchange
	prior := []api.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
	}

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	cmd, _ := SendPromptCmd(client, prior, "follow-up")
	cmd()

	if len(bodies) == 0 {
		t.Fatal("no requests received")
	}
	body := string(bodies[0])
	if !strings.Contains(body, "first question") {
		t.Error("request should contain prior history 'first question'")
	}
	if !strings.Contains(body, "first answer") {
		t.Error("request should contain prior history 'first answer'")
	}
	if !strings.Contains(body, "follow-up") {
		t.Error("request should contain new prompt 'follow-up'")
	}
}

// SendPromptCmdWithTimeout is tested above - this tests the exported version
func TestSendPromptCmdWithTimeout_ReturnsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_text.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))

	cmd, _ := SendPromptCmdWithTimeout(client, nil, "test prompt", 5*time.Second, nil, nil, nil)
	if cmd == nil {
		t.Fatal("SendPromptCmdWithTimeout() returned nil, want non-nil tea.Cmd")
	}
}

func TestSendPromptCmdWithTimeout_StreamingPath_DeliversTokens(t *testing.T) {
	// Build a minimal SSE response for the streaming path.
	sseBody := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":5}}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"streamed\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseBody))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	tokenCh := make(chan PartialResponseMsg, 10)

	cmd, _ := SendPromptCmdWithTimeout(client, nil, "test", 5*time.Second, nil, nil, tokenCh)

	var tokens []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for tok := range tokenCh {
			tokens = append(tokens, tok.Token)
		}
	}()

	result := cmd()
	<-done

	if _, ok := result.(PromptResponseMsg); !ok {
		t.Fatalf("streaming path result = %T, want PromptResponseMsg", result)
	}
	if len(tokens) == 0 {
		t.Error("expected streaming tokens, got none")
	}
}

func TestApprovingExecutor_ContextCancellationUnblocksWait(t *testing.T) {
	// Simulate: context is cancelled while the approval dialog is waiting for user input.
	// ExecuteTool must return promptly instead of blocking forever.
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	inner := &stubInnerExecutor{}
	exec := ApprovingExecutor{
		inner:      inner,
		modelName:  "test-model",
		approvalCh: approvalCh,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before ExecuteTool is called

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = exec.ExecuteTool(ctx, "bash", map[string]any{"command": "echo hi"})
	}()

	select {
	case <-done:
		// good: returned promptly
	case <-time.After(2 * time.Second):
		t.Error("ExecuteTool blocked despite cancelled context")
	}
}

// stubInnerExecutor is a no-op ToolExecutor for testing.
type stubInnerExecutor struct{}

func (s *stubInnerExecutor) ExecuteTool(_ context.Context, _ string, _ map[string]any) (string, error) {
	return "ok", nil
}

func TestWaitForToolCall_ReceivesMsg(t *testing.T) {
	ch := make(chan ToolCallMsg, 1)
	tc := ToolCallMsg{ToolCall: api.ToolCallResult{Name: "bash"}}
	ch <- tc

	cmd := waitForToolCall(ch)
	msg := cmd()

	got, ok := msg.(ToolCallMsg)
	if !ok {
		t.Fatalf("waitForToolCall() = %T, want ToolCallMsg", msg)
	}
	if got.ToolCall.Name != "bash" {
		t.Errorf("ToolCallMsg.Name = %q, want %q", got.ToolCall.Name, "bash")
	}
}

func TestWaitForToolCall_ClosedChannel(t *testing.T) {
	ch := make(chan ToolCallMsg)
	close(ch)

	cmd := waitForToolCall(ch)
	msg := cmd()

	if _, ok := msg.(toolCallDoneMsg); !ok {
		t.Fatalf("waitForToolCall on closed ch = %T, want toolCallDoneMsg", msg)
	}
}

func TestWaitForToken_ReceivesMsg(t *testing.T) {
	ch := make(chan PartialResponseMsg, 1)
	ch <- PartialResponseMsg{Token: "hello"}

	cmd := waitForToken(ch)
	msg := cmd()

	got, ok := msg.(PartialResponseMsg)
	if !ok {
		t.Fatalf("waitForToken() = %T, want PartialResponseMsg", msg)
	}
	if got.Token != "hello" {
		t.Errorf("PartialResponseMsg.Token = %q, want %q", got.Token, "hello")
	}
}

func TestWaitForToken_ClosedChannel(t *testing.T) {
	ch := make(chan PartialResponseMsg)
	close(ch)

	cmd := waitForToken(ch)
	msg := cmd()

	if _, ok := msg.(tokenDoneMsg); !ok {
		t.Fatalf("waitForToken on closed ch = %T, want tokenDoneMsg", msg)
	}
}

func TestWaitForApproval_ReceivesMsg(t *testing.T) {
	ch := make(chan ToolApprovalRequestMsg, 1)
	resp := make(chan bool, 1)
	ch <- ToolApprovalRequestMsg{ToolName: "bash", ResponseCh: resp}

	cmd := waitForApproval(ch)
	msg := cmd()

	got, ok := msg.(ToolApprovalRequestMsg)
	if !ok {
		t.Fatalf("waitForApproval() = %T, want ToolApprovalRequestMsg", msg)
	}
	if got.ToolName != "bash" {
		t.Errorf("ToolApprovalRequestMsg.ToolName = %q, want %q", got.ToolName, "bash")
	}
}

func TestWaitForApproval_ClosedChannel(t *testing.T) {
	ch := make(chan ToolApprovalRequestMsg)
	close(ch)

	cmd := waitForApproval(ch)
	msg := cmd()

	if _, ok := msg.(toolApprovalDoneMsg); !ok {
		t.Fatalf("waitForApproval on closed ch = %T, want toolApprovalDoneMsg", msg)
	}
}

func TestSendCompactCmd_NilClient(t *testing.T) {
	cmd := SendCompactCmd(nil, nil, 5*time.Second)
	msg := cmd()

	errMsg, ok := msg.(PromptErrorMsg)
	if !ok {
		t.Fatalf("SendCompactCmd(nil) = %T, want PromptErrorMsg", msg)
	}
	if errMsg.Err == nil {
		t.Error("PromptErrorMsg.Err = nil, want error")
	}
}

func TestSendCompactCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_summary.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	history := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	cmd := SendCompactCmd(client, history, 5*time.Second)
	msg := cmd()

	resp, ok := msg.(CompactResponseMsg)
	if !ok {
		t.Fatalf("SendCompactCmd() = %T, want CompactResponseMsg", msg)
	}
	if resp.Summary == "" {
		t.Error("CompactResponseMsg.Summary is empty")
	}
}

func TestSendCompactCmd_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(fixture(t, "../testdata/error_internal.json"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	cmd := SendCompactCmd(client, nil, 5*time.Second)
	msg := cmd()

	errMsg, ok := msg.(PromptErrorMsg)
	if !ok {
		t.Fatalf("SendCompactCmd error = %T, want PromptErrorMsg", msg)
	}
	if errMsg.Err == nil {
		t.Error("PromptErrorMsg.Err = nil, want error")
	}
}

func TestApprovingExecutor_Approved_CallsInner(t *testing.T) {
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	inner := &stubInnerExecutor{}
	exec := ApprovingExecutor{
		inner:      inner,
		modelName:  "test-model",
		approvalCh: approvalCh,
	}

	ctx := context.Background()
	done := make(chan string, 1)
	go func() {
		result, _ := exec.ExecuteTool(ctx, "bash", map[string]any{"command": "echo hi"})
		done <- result
	}()

	// Receive the approval request and approve it.
	req := <-approvalCh
	req.ResponseCh <- true

	select {
	case result := <-done:
		if result != "ok" {
			t.Errorf("ExecuteTool approved = %q, want %q", result, "ok")
		}
	case <-time.After(2 * time.Second):
		t.Error("ExecuteTool blocked after approval")
	}
}

func TestApprovingExecutor_Declined_ReturnsError(t *testing.T) {
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	inner := &stubInnerExecutor{}
	exec := ApprovingExecutor{
		inner:      inner,
		modelName:  "test-model",
		approvalCh: approvalCh,
	}

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, err := exec.ExecuteTool(ctx, "bash", map[string]any{"command": "rm -rf /"})
		done <- err
	}()

	req := <-approvalCh
	req.ResponseCh <- false

	select {
	case err := <-done:
		if err != errToolDeclined {
			t.Errorf("ExecuteTool declined error = %v, want errToolDeclined", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("ExecuteTool blocked after decline")
	}
}

func TestApprovingExecutor_ContextCancelledWhileWaitingForResponse(t *testing.T) {
	approvalCh := make(chan ToolApprovalRequestMsg, 1)
	inner := &stubInnerExecutor{}
	exec := ApprovingExecutor{
		inner:      inner,
		modelName:  "test-model",
		approvalCh: approvalCh,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := exec.ExecuteTool(ctx, "bash", map[string]any{"command": "echo hi"})
		done <- err
	}()

	// Wait for the approval request, then cancel the context before responding.
	<-approvalCh
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("ExecuteTool should return error when context cancelled while waiting for response")
		}
	case <-time.After(2 * time.Second):
		t.Error("ExecuteTool blocked despite cancelled context")
	}
}
