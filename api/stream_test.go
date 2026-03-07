package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// sseLines builds a minimal SSE stream for a plain text response.
func sseTextResponse(text string, inputTokens, outputTokens int) string {
	var b strings.Builder
	inputJSON, _ := json.Marshal(struct {
		InputTokens int `json:"input_tokens"`
	}{inputTokens})
	fmt.Fprintf(&b, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":%s}}\n\n", inputJSON)
	fmt.Fprintf(&b, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	textJSON, _ := json.Marshal(text)
	fmt.Fprintf(&b, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", textJSON)
	fmt.Fprintf(&b, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	fmt.Fprintf(&b, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":%d}}\n\n", outputTokens)
	fmt.Fprintf(&b, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	return b.String()
}

// sseToolUseResponse builds an SSE stream with a single tool_use block.
func sseToolUseResponse(toolID, toolName string, input map[string]any) string {
	var b strings.Builder
	b.WriteString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n")
	fmt.Fprintf(&b, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":%q,\"name\":%q,\"input\":{}}}\n\n", toolID, toolName)
	inputJSON, _ := json.Marshal(input)
	inputJSONStr, _ := json.Marshal(string(inputJSON))
	fmt.Fprintf(&b, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":%s}}\n\n", inputJSONStr)
	b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":5}}\n\n")
	b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	return b.String()
}

// --- parseStream tests ---

func TestParseStream_TextOnly(t *testing.T) {
	sse := sseTextResponse("hello world", 10, 3)
	var tokens []string
	resp, err := parseStream(strings.NewReader(sse), func(tok string) { tokens = append(tokens, tok) })
	if err != nil {
		t.Fatalf("parseStream() returned error: %v", err)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Type != "text" {
		t.Errorf("block type = %q, want text", resp.Content[0].Type)
	}
	if resp.Content[0].Text != "hello world" {
		t.Errorf("text = %q, want %q", resp.Content[0].Text, "hello world")
	}
	if len(tokens) == 0 {
		t.Error("onToken was never called")
	}
	if joined := strings.Join(tokens, ""); joined != "hello world" {
		t.Errorf("joined tokens = %q, want %q", joined, "hello world")
	}
}

func TestParseStream_UsageAccumulated(t *testing.T) {
	sse := sseTextResponse("hi", 15, 7)
	resp, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if resp.Usage.InputTokens != 15 {
		t.Errorf("InputTokens = %d, want 15", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 7 {
		t.Errorf("OutputTokens = %d, want 7", resp.Usage.OutputTokens)
	}
}

func TestParseStream_StopReason(t *testing.T) {
	sse := sseTextResponse("done", 5, 2)
	resp, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
}

func TestParseStream_OnTokenNilNoSigsegv(t *testing.T) {
	sse := sseTextResponse("safe", 1, 1)
	_, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() with nil onToken returned error: %v", err)
	}
}

func TestParseStream_ToolUse(t *testing.T) {
	input := map[string]any{"command": "echo hi"}
	sse := sseToolUseResponse("toolu_1", "bash", input)
	resp, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(resp.Content))
	}
	b := resp.Content[0]
	if b.Type != "tool_use" {
		t.Errorf("type = %q, want tool_use", b.Type)
	}
	if b.ID != "toolu_1" {
		t.Errorf("ID = %q, want toolu_1", b.ID)
	}
	if b.Name != "bash" {
		t.Errorf("Name = %q, want bash", b.Name)
	}
	if cmd, _ := b.Input["command"].(string); cmd != "echo hi" {
		t.Errorf("Input[command] = %q, want %q", cmd, "echo hi")
	}
}

func TestParseStream_ToolUseStopReason(t *testing.T) {
	sse := sseToolUseResponse("t1", "bash", map[string]any{"command": "ls"})
	resp, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("StopReason = %q, want tool_use", resp.StopReason)
	}
}

func TestParseStream_EmptyResponse(t *testing.T) {
	// A stream with no content blocks should return an error.
	sse := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	_, err := parseStream(strings.NewReader(sse), nil)
	if err == nil {
		t.Fatal("parseStream() with no content blocks should return error")
	}
}

func TestParseStream_SkipsMalformedDataLines(t *testing.T) {
	// Malformed JSON mixed with valid events should not crash; good events win.
	sse := "data: not-json\n\n" + sseTextResponse("ok", 1, 1)
	resp, err := parseStream(strings.NewReader(sse), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if resp.Content[0].Text != "ok" {
		t.Errorf("text = %q, want ok", resp.Content[0].Text)
	}
}

func TestParseStream_MultipleTextDeltas(t *testing.T) {
	// Verify multiple deltas are all collected.
	var b strings.Builder
	b.WriteString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n")
	b.WriteString("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	for _, tok := range []string{"Hello", ", ", "world", "!"} {
		tj, _ := json.Marshal(tok)
		fmt.Fprintf(&b, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", tj)
	}
	b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":4}}\n\n")
	b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")

	var tokens []string
	resp, err := parseStream(strings.NewReader(b.String()), func(t string) { tokens = append(tokens, t) })
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if resp.Content[0].Text != "Hello, world!" {
		t.Errorf("text = %q, want %q", resp.Content[0].Text, "Hello, world!")
	}
	if len(tokens) != 4 {
		t.Errorf("token count = %d, want 4", len(tokens))
	}
}

func TestParseStream_NonContiguousBlockIndices(t *testing.T) {
	// The API should never skip block indices, but parseStream must not silently
	// drop blocks that appear after a gap in the index sequence.
	var b strings.Builder
	b.WriteString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n")
	// Block at index 0
	b.WriteString("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	b.WriteString("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"block zero\"}}\n\n")
	b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	// Block at index 2 (index 1 skipped — gap)
	b.WriteString("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":2,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	b.WriteString("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":2,\"delta\":{\"type\":\"text_delta\",\"text\":\"block two\"}}\n\n")
	b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":2}\n\n")
	b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n")
	b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")

	resp, err := parseStream(strings.NewReader(b.String()), nil)
	if err != nil {
		t.Fatalf("parseStream() error: %v", err)
	}
	if len(resp.Content) != 2 {
		t.Fatalf("content blocks = %d, want 2 (both blocks must survive the index gap)", len(resp.Content))
	}
	texts := []string{resp.Content[0].Text, resp.Content[1].Text}
	for _, want := range []string{"block zero", "block two"} {
		found := false
		for _, got := range texts {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("text %q missing from content blocks: %v", want, texts)
		}
	}
}

// --- doStreamRequest / doStreamSingleAttempt tests ---

func TestDoStreamRequest_SetsStreamTrue(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseTextResponse("ok", 1, 1)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	_, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var req apiRequest
	json.Unmarshal(body, &req)
	if !req.Stream {
		t.Error("expected stream:true in request body")
	}
}

func TestDoStreamRequest_SetsCorrectHeaders(t *testing.T) {
	var headers http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseTextResponse("ok", 1, 1)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "my-key", WithModel("m"))
	_, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headers.Get("x-api-key") != "my-key" {
		t.Errorf("x-api-key = %q, want my-key", headers.Get("x-api-key"))
	}
	if headers.Get("anthropic-version") != AnthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", headers.Get("anthropic-version"), AnthropicVersion)
	}
}

func TestDoStreamRequest_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	_, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err == nil {
		t.Fatal("expected error on non-200 status")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention 429, got: %v", err)
	}
}

func TestDoStreamRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewClient(server.URL, "key", WithModel("m"))
	_, _, _, err := client.SendMessageWithToolsStreaming(ctx,
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

// --- SendMessageWithToolsStreaming tests ---

func TestSendMessageWithToolsStreaming_NoToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseTextResponse("streamed answer", 5, 3)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	result, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		[]Tool{stubBashTool()}, mockExecutor{result: "unused"}, nil,
	)
	if err != nil {
		t.Fatalf("SendMessageWithToolsStreaming() error: %v", err)
	}
	if result != "streamed answer" {
		t.Errorf("result = %q, want %q", result, "streamed answer")
	}
}

func TestSendMessageWithToolsStreaming_OnTokenCalled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Stream two separate tokens.
		var b strings.Builder
		b.WriteString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n")
		b.WriteString("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		b.WriteString("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"foo\"}}\n\n")
		b.WriteString("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"bar\"}}\n\n")
		b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
		b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n")
		b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		w.Write([]byte(b.String()))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	var tokens []string
	_, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, func(tok string) { tokens = append(tokens, tok) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(tokens, "") != "foobar" {
		t.Errorf("tokens = %q, want foobar", tokens)
	}
}

func TestSendMessageWithToolsStreaming_SingleToolCall(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			w.Write([]byte(sseToolUseResponse("toolu_1", "bash", map[string]any{"command": "echo hi"})))
		} else {
			w.Write([]byte(sseTextResponse("done", 5, 2)))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	result, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "run echo"}},
		[]Tool{stubBashTool()}, mockExecutor{result: "hi\n"}, nil,
	)
	if err != nil {
		t.Fatalf("SendMessageWithToolsStreaming() error: %v", err)
	}
	if result != "done" {
		t.Errorf("result = %q, want done", result)
	}
	if requestCount.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount.Load())
	}
}

func TestSendMessageWithToolsStreaming_AccumulatesUsage(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			w.Write([]byte(sseToolUseResponse("t1", "bash", map[string]any{"command": "date"})))
		} else {
			w.Write([]byte(sseTextResponse("done", 20, 3)))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	_, _, usage, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		[]Tool{stubBashTool()}, mockExecutor{result: "output"}, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Turn 1: 10 input (from sseToolUseResponse hardcoded), 5 output
	// Turn 2: 20 input, 3 output
	if usage.InputTokens != 30 {
		t.Errorf("InputTokens = %d, want 30", usage.InputTokens)
	}
	if usage.OutputTokens != 8 {
		t.Errorf("OutputTokens = %d, want 8 (5+3)", usage.OutputTokens)
	}
}

func TestSendMessageWithToolsStreaming_Retries429BeforeStream(t *testing.T) {
	// A 429 response arrives before any bytes are streamed — safe to retry.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("rate limited"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseTextResponse("ok", 1, 1)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"), WithMaxRetries(1), WithRetryDelay(0))
	result, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want ok", result)
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", calls.Load())
	}
}

func TestSendMessageWithToolsStreaming_NoRetryAfterStreamStart(t *testing.T) {
	// A 200 is received and streaming begins — the result of parseStream is returned
	// directly, not retried (to prevent duplicate tokens).
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Respond with a valid stream — parse succeeds, no retry should occur.
		w.Write([]byte(sseTextResponse("streamed", 1, 1)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"), WithMaxRetries(3), WithRetryDelay(0))
	result, _, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		nil, mockExecutor{}, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "streamed" {
		t.Errorf("result = %q, want streamed", result)
	}
	if calls.Load() != 1 {
		t.Errorf("expected exactly 1 request (no retry after stream start), got %d", calls.Load())
	}
}

func TestSendMessageWithToolsStreaming_EmptyMessages(t *testing.T) {
	client := NewClient("http://localhost", "key")
	_, _, _, err := client.SendMessageWithToolsStreaming(context.Background(), nil, nil, mockExecutor{}, nil)
	if err == nil {
		t.Error("expected error for nil messages")
	}
}

func TestSendMessageWithToolsStreaming_ReturnsAccumulatedMessages(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			w.Write([]byte(sseToolUseResponse("toolu_99", "bash", map[string]any{"command": "date"})))
		} else {
			w.Write([]byte(sseTextResponse("today", 5, 2)))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithModel("m"))
	_, msgs, _, err := client.SendMessageWithToolsStreaming(context.Background(),
		[]Message{{Role: "user", Content: "what day?"}},
		[]Tool{stubBashTool()}, mockExecutor{result: "Monday\n"}, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// [user, assistant(tool_use), user(tool_result), assistant(final)]
	if len(msgs) != 4 {
		t.Fatalf("msgs length = %d, want 4", len(msgs))
	}
	if msgs[3].Content != "today" {
		t.Errorf("final message content = %v, want today", msgs[3].Content)
	}
}
