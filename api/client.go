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
)

// Constants for API configuration
const (
	DefaultMaxTokens = 8192
	MessagesEndpoint = "/v1/messages"
	AnthropicVersion = "2023-06-01"
)

// Client handles communication with the LLM API
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
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

// effectiveMaxTokens returns the configured max tokens, falling back to DefaultMaxTokens.
func (c *Client) effectiveMaxTokens() int {
	if c.maxTokens > 0 {
		return c.maxTokens
	}
	return DefaultMaxTokens
}

// doRequest sends one HTTP request and returns the full API response.
func (c *Client) doRequest(ctx context.Context, reqBody apiRequest) (apiResponse, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + MessagesEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", AnthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return apiResponse{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
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
// The optional onCall callback is invoked after each tool execution with the result.
func (c *Client) SendMessageWithTools(ctx context.Context, messages []Message, tools []Tool, executor ToolExecutor, onCall ...func(ToolCallResult)) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages provided")
	}

	msgs := make([]Message, len(messages))
	copy(msgs, messages)

	for {
		resp, err := c.doRequest(ctx, apiRequest{
			Model:     c.model,
			MaxTokens: c.effectiveMaxTokens(),
			Messages:  msgs,
			Tools:     tools,
		})
		if err != nil {
			return "", err
		}

		// Collect any tool_use blocks.
		var toolUseBlocks []ContentBlock
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				toolUseBlocks = append(toolUseBlocks, block)
			}
		}

		// No tool calls → return the final text.
		if len(toolUseBlocks) == 0 {
			return extractText(resp)
		}

		// Append the assistant's tool_use message to history.
		msgs = append(msgs, Message{Role: "assistant", Content: resp.Content})

		// Execute each tool and collect results.
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

		// Append tool results as a user message.
		msgs = append(msgs, Message{Role: "user", Content: toolResults})
	}
}
