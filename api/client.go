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
	DefaultMaxTokens = 100
	MessagesEndpoint = "/v1/messages"
	AnthropicVersion = "2023-06-01"
)

// Client handles communication with the LLM API
type Client struct {
	baseURL    string
	apiKey     string
	model      string
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

// doRequest sends an apiRequest to the LLM API and returns the response text
func (c *Client) doRequest(ctx context.Context, reqBody apiRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + MessagesEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", AnthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", errors.New("no content in response")
	}

	return apiResp.Content[0].Text, nil
}

// SendMessage sends a message to the LLM API and returns the response
func (c *Client) SendMessage(ctx context.Context, prompt string) (string, error) {
	return c.doRequest(ctx, apiRequest{
		Model:     c.model,
		MaxTokens: DefaultMaxTokens,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	})
}

// SendMessageWithHistory sends a conversation history to the LLM API and returns the response
func (c *Client) SendMessageWithHistory(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages provided")
	}

	return c.doRequest(ctx, apiRequest{
		Model:     c.model,
		MaxTokens: DefaultMaxTokens,
		Messages:  messages,
	})
}
