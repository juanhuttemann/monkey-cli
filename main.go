package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// API request/response structures
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// getConfig reads and validates required environment variables
func getConfig() (apiKey, baseURL, model string, err error) {
	apiKey = os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", "", "", errors.New("missing required environment variable: ANTHROPIC_API_KEY")
	}

	baseURL = os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		return "", "", "", errors.New("missing required environment variable: ANTHROPIC_BASE_URL")
	}

	model = os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		return "", "", "", errors.New("missing required environment variable: ANTHROPIC_MODEL")
	}

	return apiKey, baseURL, model, nil
}

// getGreeting calls the LLM API and returns a greeting message
func getGreeting() (string, error) {
	apiKey, baseURL, model, err := getConfig()
	if err != nil {
		return "", err
	}

	// Build request body
	reqBody := apiRequest{
		Model:     model,
		MaxTokens: 100,
		Messages: []apiMessage{
			{
				Role:    "user",
				Content: "Return a greeting message",
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := baseURL + "/v1/messages"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate content
	if len(apiResp.Content) == 0 {
		return "", errors.New("no content in response")
	}

	return apiResp.Content[0].Text, nil
}

// printHello calls the LLM API and returns a greeting message
// On error, it prints to stderr and exits with code 1
func printHello() string {
	greeting, err := getGreeting()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return greeting
}

func main() {
	fmt.Println(printHello())
}
