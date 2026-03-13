// Package api handles LLM API communication via the Anthropic Go SDK.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Constants for API configuration.
const (
	DefaultMaxTokens = 8192
	MessagesEndpoint = "/v1/messages"
	AnthropicVersion = "2023-06-01"
)

// StatusError is returned when the API responds with a non-2xx status code.
// It wraps the SDK's *anthropic.Error to preserve the FriendlyMessage API
// used by tui/api.go without requiring changes there.
type StatusError struct {
	StatusCode int
	Body       string
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

// convertSDKError maps *anthropic.Error → *StatusError so existing callers
// (tui/api.go) can continue using errors.As(*StatusError) without change.
// Other error types are returned as-is.
func convertSDKError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return &StatusError{
			StatusCode: apiErr.StatusCode,
			Body:       apiErr.RawJSON(),
		}
	}
	return err
}

// Client handles communication with the Anthropic API using the official SDK.
type Client struct {
	model              string
	maxTokens          int
	maxRetries         int
	systemPrompt       string
	httpClientOverride *http.Client
	sdkClient          anthropic.Client
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// NewClient creates a new API client backed by the Anthropic Go SDK.
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	sdkOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(strings.TrimRight(baseURL, "/")),
		option.WithMaxRetries(c.maxRetries),
	}
	if c.httpClientOverride != nil {
		sdkOpts = append(sdkOpts, option.WithHTTPClient(c.httpClientOverride))
	}
	c.sdkClient = anthropic.NewClient(sdkOpts...)
	return c
}

// WithModel sets the model to use for API requests.
func WithModel(model string) ClientOption { return func(c *Client) { c.model = model } }

// WithMaxTokens sets the maximum number of tokens in API responses.
func WithMaxTokens(n int) ClientOption { return func(c *Client) { c.maxTokens = n } }

// WithMaxRetries sets the maximum number of retry attempts after a retryable error.
func WithMaxRetries(n int) ClientOption { return func(c *Client) { c.maxRetries = n } }

// WithSystemPrompt sets the system prompt sent with every request.
func WithSystemPrompt(s string) ClientOption { return func(c *Client) { c.systemPrompt = s } }

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(h *http.Client) ClientOption { return func(c *Client) { c.httpClientOverride = h } }

// SetModel changes the model used for subsequent API requests.
func (c *Client) SetModel(model string) { c.model = model }

// GetModel returns the model currently configured on the client.
func (c *Client) GetModel() string { return c.model }

func (c *Client) effectiveMaxTokens() int {
	if c.maxTokens > 0 {
		return c.maxTokens
	}
	return DefaultMaxTokens
}

// requestOpts extracts context-injected per-attempt timeout and retry notifier
// and translates them into SDK request options.
func (c *Client) requestOpts(ctx context.Context) []option.RequestOption {
	var opts []option.RequestOption
	if d, ok := ctx.Value(perAttemptTimeoutKey{}).(time.Duration); ok && d > 0 {
		opts = append(opts, option.WithRequestTimeout(d))
	}
	if fn, ok := ctx.Value(retryNotifierKey{}).(func(int, error)); ok {
		opts = append(opts, retryNotifierMiddleware(fn))
	}
	return opts
}

// retryNotifierMiddleware returns an SDK middleware option that calls fn
// before each retry. fn receives (attempt, lastErr) where attempt is 1-based.
func retryNotifierMiddleware(fn func(int, error)) option.RequestOption {
	var count atomic.Int32
	var lastStatus atomic.Int32
	return option.WithMiddleware(func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		n := int(count.Add(1))
		if n > 1 {
			code := int(lastStatus.Load())
			var notifyErr error
			if code != 0 {
				notifyErr = &StatusError{StatusCode: code}
			}
			fn(n-1, notifyErr)
		}
		resp, err := next(req)
		if resp != nil {
			lastStatus.Store(int32(resp.StatusCode))
		}
		return resp, err
	})
}

// newParams builds MessageNewParams from messages and tools with the client's
// configured model, max tokens, and system prompt.
func (c *Client) newParams(messages []anthropic.MessageParam, tools []anthropic.ToolUnionParam) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(c.effectiveMaxTokens()),
		Messages:  messages,
	}
	if c.systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: c.systemPrompt}}
	}
	if len(tools) > 0 {
		params.Tools = tools
	}
	return params
}

// SendMessage sends a single user message and returns the response text.
func (c *Client) SendMessage(ctx context.Context, prompt string) (string, error) {
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}
	msg, err := c.sdkClient.Messages.New(ctx, c.newParams(msgs, nil), c.requestOpts(ctx)...)
	if err != nil {
		return "", convertSDKError(err)
	}
	return extractText(fromSDKMessage(msg))
}

// SendMessageWithHistory sends a conversation history and returns the response text.
func (c *Client) SendMessageWithHistory(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages provided")
	}
	sdkMsgs, err := toSDKMessages(messages)
	if err != nil {
		return "", err
	}
	msg, err := c.sdkClient.Messages.New(ctx, c.newParams(sdkMsgs, nil), c.requestOpts(ctx)...)
	if err != nil {
		return "", convertSDKError(err)
	}
	return extractText(fromSDKMessage(msg))
}

// SendMessageWithTools sends a conversation with tool definitions, executing any tool calls
// the model makes and continuing the loop until the model returns a final text response.
// It returns the final text response, the full accumulated message history, the total token
// usage across all API calls in the loop, and any error.
// The optional onCall callback is invoked after each tool execution with the result.
func (c *Client) SendMessageWithTools(ctx context.Context, messages []Message, tools []Tool, executor ToolExecutor, onCall ...func(ToolCallResult)) (string, []Message, Usage, error) {
	if len(messages) == 0 {
		return "", nil, Usage{}, errors.New("no messages provided")
	}
	sdkTools := toSDKTools(tools)
	msgs := make([]Message, len(messages))
	copy(msgs, messages)
	return runToolLoop(ctx, msgs, tools, executor, onCall, func(loopMsgs []Message, _ []Tool) (apiResponse, error) {
		sdkMsgs, err := toSDKMessages(loopMsgs)
		if err != nil {
			return apiResponse{}, err
		}
		msg, err := c.sdkClient.Messages.New(ctx, c.newParams(sdkMsgs, sdkTools), c.requestOpts(ctx)...)
		if err != nil {
			return apiResponse{}, convertSDKError(err)
		}
		return fromSDKMessage(msg), nil
	})
}
