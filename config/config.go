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
	EnvAPIKey      = "ANTHROPIC_API_KEY"
	EnvBaseURL     = "ANTHROPIC_BASE_URL"
	EnvOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
	EnvSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
	EnvHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
	EnvMaxTokens   = "CLAUDE_CODE_MAX_OUTPUT_TOKENS"
)

// Config holds the application configuration
type Config struct {
	APIKey      string
	BaseURL     string
	OpusModel   string
	SonnetModel string
	HaikuModel  string
	MaxTokens   int // 0 means use the API client's default
}

// AvailableModels returns configured model values in priority order (opus, sonnet, haiku).
func (c Config) AvailableModels() []string {
	var m []string
	if c.OpusModel != "" {
		m = append(m, c.OpusModel)
	}
	if c.SonnetModel != "" {
		m = append(m, c.SonnetModel)
	}
	if c.HaikuModel != "" {
		m = append(m, c.HaikuModel)
	}
	return m
}

// DefaultModel returns the highest-priority configured model (opus > sonnet > haiku).
func (c Config) DefaultModel() string {
	if c.OpusModel != "" {
		return c.OpusModel
	}
	if c.SonnetModel != "" {
		return c.SonnetModel
	}
	return c.HaikuModel
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

	const defaultBaseURL = "https://api.anthropic.com"
	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		OpusModel:   os.Getenv(EnvOpusModel),
		SonnetModel: os.Getenv(EnvSonnetModel),
		HaikuModel:  os.Getenv(EnvHaikuModel),
	}

	if cfg.DefaultModel() == "" {
		return Config{}, fmt.Errorf("at least one model must be set (%s, %s, or %s)",
			EnvOpusModel, EnvSonnetModel, EnvHaikuModel)
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

// LoadSystemPromptFile reads a markdown file and returns its content as a string.
// If the file does not exist, it returns an empty string and no error.
func LoadSystemPromptFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Validate checks that all required fields are set
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}
	if c.DefaultModel() == "" {
		return fmt.Errorf("at least one model must be set")
	}
	return nil
}
