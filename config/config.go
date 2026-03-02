// Package config handles application configuration from environment variables
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Environment variable names
const (
	EnvAPIKey    = "ANTHROPIC_API_KEY"
	EnvBaseURL   = "ANTHROPIC_BASE_URL"
	EnvModel     = "ANTHROPIC_MODEL"
	EnvMaxTokens = "CLAUDE_CODE_MAX_OUTPUT_TOKENS"
)

// Config holds the application configuration
type Config struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int // 0 means use the API client's default
}

// Loader defines the interface for loading configuration
type Loader interface {
	Load() (Config, error)
}

// envLoader implements Loader using environment variables
type envLoader struct{}

// NewEnvLoader creates a new configuration loader that reads from environment variables
func NewEnvLoader() Loader {
	return &envLoader{}
}

// Load reads configuration from environment variables
func (l *envLoader) Load() (Config, error) {
	apiKey := os.Getenv(EnvAPIKey)
	if apiKey == "" {
		return Config{}, errors.New("missing required environment variable: " + EnvAPIKey)
	}

	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		return Config{}, errors.New("missing required environment variable: " + EnvBaseURL)
	}

	model := os.Getenv(EnvModel)
	if model == "" {
		return Config{}, errors.New("missing required environment variable: " + EnvModel)
	}

	cfg := Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}

	if s := os.Getenv(EnvMaxTokens); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("%s must be a positive integer, got %q", EnvMaxTokens, s)
		}
		cfg.MaxTokens = n
	}

	return cfg, nil
}

// Validate checks that all required fields are set
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}
	if c.Model == "" {
		return fmt.Errorf("Model is required")
	}
	return nil
}
