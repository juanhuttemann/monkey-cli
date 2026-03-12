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
	RetryAfter time.Duration // parsed from Retry-After header; 0 if not present
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("API returned status %d: %s", e.StatusCode, e.Body)
}

// FriendlyMessage returns a short, human-readable error message by parsing
// the structured JSON body. Falls back to a generic status-code message.
func (e *StatusError) FriendlyMessage() string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(e.Body), &parsed) == nil && parsed.Error.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, parsed.Error.Message)
	}
	return fmt.Sprintf("API error (%d)", e.StatusCode)
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
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return apiResponse{}, &StatusError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
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
			delay := computeRetryDelay(c.retryDelay, attempt, lastErr)
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
	return runToolLoop(ctx, msgs, tools, executor, onCall, func(msgs []Message, tools []Tool) (apiResponse, error) {
		return c.doRequest(ctx, apiRequest{
			Model:     c.model,
			MaxTokens: c.effectiveMaxTokens(),
			Messages:  msgs,
			Tools:     tools,
		})
	})
}
