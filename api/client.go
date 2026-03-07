// Package api handles LLM API communication
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Constants for API configuration
const (
	DefaultMaxTokens = 8192
	MessagesEndpoint = "/v1/messages"
	AnthropicVersion = "2023-06-01"
)

// StatusError is returned when the API responds with a non-200 status code.
type StatusError struct {
	StatusCode int
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("API returned status %d: %s", e.StatusCode, e.Body)
}

// retryNotifierKey is the context key for the per-request retry callback.
type retryNotifierKey struct{}

// perAttemptTimeoutKey is the context key for the per-attempt timeout duration.
type perAttemptTimeoutKey struct{}

// WithPerAttemptTimeout returns a context that will apply the given timeout to each
// individual request attempt. This allows retrying after a timeout, as each retry
// gets a fresh timeout rather than sharing an already-expired one.
func WithPerAttemptTimeout(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, perAttemptTimeoutKey{}, d)
}

// WithRetryNotifier returns a context that carries a callback invoked before each retry attempt.
// attempt is 1-based; err is the error that triggered the retry.
func WithRetryNotifier(ctx context.Context, fn func(attempt int, err error)) context.Context {
	return context.WithValue(ctx, retryNotifierKey{}, fn)
}

// Client handles communication with the LLM API
type Client struct {
	baseURL      string
	apiKey       string
	model        string
	maxTokens    int
	maxRetries   int
	retryDelay   time.Duration
	httpClient   *http.Client
	systemPrompt string
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// NewClient creates a new API client with the given base URL and API key
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithHTTPClient sets a custom HTTP client for the API client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithModel sets the model to use for API requests
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithMaxTokens sets the maximum number of tokens in API responses.
// If not set, DefaultMaxTokens is used.
func WithMaxTokens(n int) ClientOption {
	return func(c *Client) {
		c.maxTokens = n
	}
}

// WithMaxRetries sets the maximum number of retry attempts after a retryable error.
// Default is 0 (no retries).
func WithMaxRetries(n int) ClientOption {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithSystemPrompt sets the system prompt sent with every request.
func WithSystemPrompt(s string) ClientOption {
	return func(c *Client) {
		c.systemPrompt = s
	}
}

// WithRetryDelay sets the base delay between retry attempts.
// The actual delay doubles with each attempt (exponential backoff).
// Default is 0 (no delay).
func WithRetryDelay(d time.Duration) ClientOption {
	return func(c *Client) {
		c.retryDelay = d
	}
}

// SetModel changes the model used for subsequent API requests.
func (c *Client) SetModel(model string) {
	c.model = model
}

// GetModel returns the model currently configured on the client.
func (c *Client) GetModel() string {
	return c.model
}

// effectiveMaxTokens returns the configured max tokens, falling back to DefaultMaxTokens.
func (c *Client) effectiveMaxTokens() int {
	if c.maxTokens > 0 {
		return c.maxTokens
	}
	return DefaultMaxTokens
}

// sendRawRequest creates and sends one HTTP POST to the messages endpoint with
// standard headers. The caller is responsible for closing resp.Body.
func (c *Client) sendRawRequest(ctx context.Context, jsonBody []byte) (*http.Response, error) {
	reqURL := c.baseURL + MessagesEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", AnthropicVersion)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	return resp, nil
}

// doSingleAttempt sends one HTTP request and returns the full API response.
func (c *Client) doSingleAttempt(ctx context.Context, jsonBody []byte) (apiResponse, error) {
	resp, err := c.sendRawRequest(ctx, jsonBody)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return apiResponse{}, &StatusError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return apiResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return apiResponse{}, errors.New("no content in response")
	}

	return apiResp, nil
}

// applyPerAttemptTimeout wraps ctx in a fresh timeout if perAttemptTimeoutKey is set.
// The returned CancelFunc must always be called (safe to call on a no-op cancel).
func applyPerAttemptTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if d, ok := ctx.Value(perAttemptTimeoutKey{}).(time.Duration); ok && d > 0 {
		return context.WithTimeout(ctx, d)
	}
	return ctx, func() {}
}

// doAttempt wraps doSingleAttempt with a per-attempt timeout if one is configured
// in the context via WithPerAttemptTimeout. This lets each retry get a fresh timeout
// even after the previous attempt's deadline expired.
func (c *Client) doAttempt(parentCtx context.Context, jsonBody []byte) (apiResponse, error) {
	ctx, cancel := applyPerAttemptTimeout(parentCtx)
	defer cancel()
	return c.doSingleAttempt(ctx, jsonBody)
}

// doRequest sends an HTTP request with automatic retries on retryable errors.
// ctx should be a cancellation-only context (no deadline); use WithPerAttemptTimeout
// to apply a per-attempt deadline that resets on each retry.
func (c *Client) doRequest(ctx context.Context, reqBody apiRequest) (apiResponse, error) {
	reqBody.System = c.systemPrompt
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

		resp, err := c.doAttempt(ctx, jsonBody)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryableError(ctx, err) {
			break
		}
	}
	return apiResponse{}, lastErr
}

// isRetryableError reports whether err warrants a retry attempt.
// ctx should be the parent (non-per-attempt) context; if it is cancelled the
// function returns false regardless of the error.
func isRetryableError(ctx context.Context, err error) bool {
	// Explicit user cancellation — do not retry.
	if errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusTooManyRequests || statusErr.StatusCode >= 500
	}
	// Per-attempt timeout expired (parent ctx is still alive) — retry with fresh timeout.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	// Network-level transport errors (connection reset, EOF, etc.) are retryable.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	return false
}

// extractText returns the concatenated text from all text blocks in a response.
func extractText(resp apiResponse) (string, error) {
	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("no text content in response")
	}
	return strings.Join(parts, "\n"), nil
}

// collectToolUseBlocks returns all tool_use content blocks from resp.Content.
func collectToolUseBlocks(content []ContentBlock) []ContentBlock {
	var blocks []ContentBlock
	for _, b := range content {
		if b.Type == "tool_use" {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

// runToolCalls appends the assistant message, executes every tool call, fires
// onCall callbacks, and appends the tool_result user message. Returns updated msgs.
func runToolCalls(msgs []Message, respContent []ContentBlock, toolUseBlocks []ContentBlock, executor ToolExecutor, onCall []func(ToolCallResult)) []Message {
	msgs = append(msgs, Message{Role: "assistant", Content: respContent})
	toolResults := make([]ContentBlock, 0, len(toolUseBlocks))
	for _, tu := range toolUseBlocks {
		output, execErr := executor.ExecuteTool(tu.Name, tu.Input)
		content := output
		if execErr != nil && content == "" {
			content = fmt.Sprintf("error: %v", execErr)
		}
		toolResults = append(toolResults, ContentBlock{
			Type:      "tool_result",
			ToolUseID: tu.ID,
			Content:   content,
		})
		for _, fn := range onCall {
			fn(ToolCallResult{Name: tu.Name, Input: tu.Input, Output: output, Err: execErr})
		}
	}
	return append(msgs, Message{Role: "user", Content: toolResults})
}

// runToolLoop drives the agentic tool-calling loop. fetch is called for each turn;
// tool results are fed back until the model returns a final text-only response.
func runToolLoop(
	msgs []Message,
	tools []Tool,
	executor ToolExecutor,
	onCall []func(ToolCallResult),
	fetch func(msgs []Message, tools []Tool) (apiResponse, error),
) (string, []Message, Usage, error) {
	var totalUsage Usage
	for {
		resp, err := fetch(msgs, tools)
		if err != nil {
			return "", nil, Usage{}, err
		}
		totalUsage = totalUsage.Add(Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		})

		toolUseBlocks := collectToolUseBlocks(resp.Content)
		if len(toolUseBlocks) == 0 {
			text, err := extractText(resp)
			if err != nil {
				return "", nil, Usage{}, err
			}
			return text, append(msgs, Message{Role: "assistant", Content: text}), totalUsage, nil
		}

		msgs = runToolCalls(msgs, resp.Content, toolUseBlocks, executor, onCall)
	}
}

// SendMessage sends a single user message and returns the response text.
func (c *Client) SendMessage(ctx context.Context, prompt string) (string, error) {
	resp, err := c.doRequest(ctx, apiRequest{
		Model:     c.model,
		MaxTokens: c.effectiveMaxTokens(),
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}
	return extractText(resp)
}

// SendMessageWithHistory sends a conversation history and returns the response text.
func (c *Client) SendMessageWithHistory(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages provided")
	}

	resp, err := c.doRequest(ctx, apiRequest{
		Model:     c.model,
		MaxTokens: c.effectiveMaxTokens(),
		Messages:  messages,
	})
	if err != nil {
		return "", err
	}
	return extractText(resp)
}

// SendMessageWithTools sends a conversation with tool definitions, executing any tool calls
// the model makes and continuing the loop until the model returns a final text response.
// It returns the final text response, the full accumulated message history (including
// tool_use/tool_result exchanges and the final assistant message), the total token usage
// across all API calls in the loop, and any error.
// The optional onCall callback is invoked after each tool execution with the result.
func (c *Client) SendMessageWithTools(ctx context.Context, messages []Message, tools []Tool, executor ToolExecutor, onCall ...func(ToolCallResult)) (string, []Message, Usage, error) {
	if len(messages) == 0 {
		return "", nil, Usage{}, errors.New("no messages provided")
	}
	msgs := make([]Message, len(messages))
	copy(msgs, messages)
	return runToolLoop(msgs, tools, executor, onCall, func(msgs []Message, tools []Tool) (apiResponse, error) {
		return c.doRequest(ctx, apiRequest{
			Model:     c.model,
			MaxTokens: c.effectiveMaxTokens(),
			Messages:  msgs,
			Tools:     tools,
		})
	})
}
