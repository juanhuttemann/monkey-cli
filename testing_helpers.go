package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
)

// setupTestEnv sets environment variables and returns a cleanup function
// that restores the original values (or unsets if they were not set).
// model is set as ANTHROPIC_DEFAULT_OPUS_MODEL.
func setupTestEnv(apiKey, baseURL, model string) func() {
	// Save original values
	originalAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	originalBaseURL := os.Getenv("ANTHROPIC_BASE_URL")
	originalModel := os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL")

	// Set new values
	os.Setenv("ANTHROPIC_API_KEY", apiKey)
	os.Setenv("ANTHROPIC_BASE_URL", baseURL)
	os.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", model)

	// Return cleanup function
	return func() {
		if originalAPIKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalAPIKey)
		}
		if originalBaseURL == "" {
			os.Unsetenv("ANTHROPIC_BASE_URL")
		} else {
			os.Setenv("ANTHROPIC_BASE_URL", originalBaseURL)
		}
		if originalModel == "" {
			os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")
		} else {
			os.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", originalModel)
		}
	}
}

// createMockServer creates a test HTTP server that responds with the given body and status code
func createMockServer(responseBody string, statusCode int) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(responseBody))
	}))
	return server, server.Close
}

// createMockServerWithValidator creates a test HTTP server that validates requests
// and responds with the given body and status code
func createMockServerWithValidator(validator func(*http.Request) bool, responseBody string, statusCode int) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validator(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(responseBody))
	}))
	return server, server.Close
}

// successResponse builds a standard success response JSON string
func successResponse(text string) string {
	return fmt.Sprintf(`{"content": [{"type": "text", "text": %q}]}`, text)
}
