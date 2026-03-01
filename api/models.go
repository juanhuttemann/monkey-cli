// Package api defines data structures for API communication
package api

// apiRequest represents the request body sent to the LLM API
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

// apiMessage represents a single message in the conversation
type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// apiResponse represents the response from the LLM API
type apiResponse struct {
	Content []contentBlock `json:"content"`
}

// contentBlock represents a single content block in the response
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
