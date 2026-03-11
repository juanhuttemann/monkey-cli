package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// sseData is the decoded payload from a single SSE "data:" line.
// Each event type uses only the relevant subset of fields.
type sseData struct {
	Type string `json:"type"`

	// message_start
	Message *struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`

	// content_block_start / content_block_delta / content_block_stop
	Index        int              `json:"index"`
	ContentBlock *sseContentBlock `json:"content_block,omitempty"`
	Delta        *sseDelta        `json:"delta,omitempty"`

	// message_delta usage field
	Usage *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

type sseContentBlock struct {
	Type string `json:"type"` // "text" or "tool_use"
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type sseDelta struct {
	// content_block_delta
	Type        string `json:"type"`                   // "text_delta" or "input_json_delta"
	Text        string `json:"text,omitempty"`         // text_delta
	PartialJSON string `json:"partial_json,omitempty"` // input_json_delta

	// message_delta
	StopReason string `json:"stop_reason,omitempty"`
}

// streamBlock accumulates data for one content block during streaming.
type streamBlock struct {
	blockType   string // "text" or "tool_use"
	text        strings.Builder
	id          string
	name        string
	inputBuffer bytes.Buffer
}

// parseStream reads SSE events from r, calling onToken for each text token,
// and returns a reconstructed apiResponse once streaming completes.
// onToken may be nil.
func parseStream(r io.Reader, onToken func(string)) (apiResponse, error) {
	blocks := map[int]*streamBlock{}
	var inputTokens, outputTokens int
	var stopReason string

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var ev sseData
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // skip malformed events
		}

		switch ev.Type {
		case "message_start":
			if ev.Message != nil {
				inputTokens = ev.Message.Usage.InputTokens
			}
		case "content_block_start":
			if ev.ContentBlock != nil {
				bs := &streamBlock{blockType: ev.ContentBlock.Type}
				switch ev.ContentBlock.Type {
				case "text":
					bs.text.WriteString(ev.ContentBlock.Text)
				case "tool_use":
					bs.id = ev.ContentBlock.ID
					bs.name = ev.ContentBlock.Name
				}
				blocks[ev.Index] = bs
			}
		case "content_block_delta":
			bs, ok := blocks[ev.Index]
			if !ok || ev.Delta == nil {
				continue
			}
			switch ev.Delta.Type {
			case "text_delta":
				bs.text.WriteString(ev.Delta.Text)
				if onToken != nil {
					onToken(ev.Delta.Text)
				}
			case "input_json_delta":
				bs.inputBuffer.WriteString(ev.Delta.PartialJSON)
			}
		case "message_delta":
			if ev.Delta != nil && ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
			if ev.Usage != nil {
				outputTokens = ev.Usage.OutputTokens
			}
		}
		// content_block_stop and message_stop require no action
	}

	if err := scanner.Err(); err != nil {
		return apiResponse{}, fmt.Errorf("error reading stream: %w", err)
	}

	// Reconstruct content blocks in index order.
	// Collect and sort keys so gaps in the index sequence don't truncate output.
	indices := make([]int, 0, len(blocks))
	for i := range blocks {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	var content []ContentBlock
	for _, i := range indices {
		bs := blocks[i]
		switch bs.blockType {
		case "text":
			if t := bs.text.String(); t != "" {
				content = append(content, ContentBlock{Type: "text", Text: t})
			}
		case "tool_use":
			var input map[string]any
			if bs.inputBuffer.Len() > 0 {
				if err := json.Unmarshal(bs.inputBuffer.Bytes(), &input); err != nil {
					return apiResponse{}, fmt.Errorf("failed to parse tool input JSON: %w", err)
				}
			}
			content = append(content, ContentBlock{
				Type:  "tool_use",
				ID:    bs.id,
				Name:  bs.name,
				Input: input,
			})
		}
	}

	if len(content) == 0 {
		return apiResponse{}, errors.New("no content in stream response")
	}

	return apiResponse{
		Content:    content,
		StopReason: stopReason,
		Usage: struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}, nil
}

// doStreamRequest serializes reqBody (with stream:true) and sends it, retrying on
// pre-stream errors (non-200 status, network failures) up to c.maxRetries times.
// Retrying is safe only before streaming begins: once we receive a 200 and start
// reading the body via parseStream, the result is returned directly without retry
// to prevent delivering duplicate tokens to the caller.
func (c *Client) doStreamRequest(ctx context.Context, reqBody apiRequest, onToken func(string)) (apiResponse, error) {
	reqBody.System = c.systemPrompt
	reqBody.Stream = true
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if fn, ok := ctx.Value(retryNotifierKey{}).(func(int, error)); ok {
				fn(attempt, lastErr)
			}
			delay := c.retryDelay * time.Duration(1<<(attempt-1))
			if delay > 0 {
				select {
				case <-ctx.Done():
					return apiResponse{}, ctx.Err()
				case <-time.After(delay):
				}
			} else if ctx.Err() != nil {
				return apiResponse{}, ctx.Err()
			}
		}

		attemptCtx, cancel := applyPerAttemptTimeout(ctx)
		resp, err := c.sendRawRequest(attemptCtx, jsonBody)
		if err != nil {
			cancel()
			lastErr = err
			if !isRetryableError(ctx, err) {
				break
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			cancel()
			lastErr = &StatusError{StatusCode: resp.StatusCode, Body: string(body)}
			if !isRetryableError(ctx, lastErr) {
				break
			}
			continue
		}

		// 200 OK — begin streaming. No retry from this point: parseStream reads the
		// body incrementally and may already have delivered tokens via onToken.
		result, parseErr := parseStream(resp.Body, onToken)
		_ = resp.Body.Close()
		cancel()
		return result, parseErr
	}
	return apiResponse{}, lastErr
}

// SendMessageWithToolsStreaming is like SendMessageWithTools but streams text tokens
// via the SSE API. onToken is called for each text token as it arrives; it may be nil.
// Tool-calling turns are handled the same way as in SendMessageWithTools.
func (c *Client) SendMessageWithToolsStreaming(ctx context.Context, messages []Message, tools []Tool, executor ToolExecutor, onToken func(string), onCall ...func(ToolCallResult)) (string, []Message, Usage, error) {
	if len(messages) == 0 {
		return "", nil, Usage{}, errors.New("no messages provided")
	}
	msgs := make([]Message, len(messages))
	copy(msgs, messages)
	return runToolLoop(ctx, msgs, tools, executor, onCall, func(msgs []Message, tools []Tool) (apiResponse, error) {
		return c.doStreamRequest(ctx, apiRequest{
			Model:     c.model,
			MaxTokens: c.effectiveMaxTokens(),
			Messages:  msgs,
			Tools:     tools,
		}, onToken)
	})
}
