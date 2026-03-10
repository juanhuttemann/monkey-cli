// Package config handles application configuration from environment variables
package config

import (
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

	defaultBaseURL     = "https://api.anthropic.com"
	defaultOpusModel   = "claude-opus-4-6"
	defaultSonnetModel = "claude-sonnet-4-6"
	defaultHaikuModel  = "claude-haiku-4-5-20251001"
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

// envLoader implements Loader using environment variables with an optional config file fallback.
type envLoader struct {
	configFile string // path to config.toml; empty means use ConfigFilePath()
}

// NewEnvLoader creates a loader that reads from environment variables, falling back
// to ~/.config/monkey/config.toml (or $XDG_CONFIG_HOME/monkey/config.toml).
func NewEnvLoader() Loader {
	return &envLoader{}
}

// NewEnvLoaderWithConfigFile creates a loader that reads from environment variables,
// falling back to the specified config file path.
func NewEnvLoaderWithConfigFile(path string) Loader {
	return &envLoader{configFile: path}
}

// fileKey maps an environment variable name to its config-file key equivalent.
var fileKey = map[string]string{
	EnvAPIKey:      "api_key",
	EnvBaseURL:     "base_url",
	EnvOpusModel:   "opus_model",
	EnvSonnetModel: "sonnet_model",
	EnvHaikuModel:  "haiku_model",
	EnvMaxTokens:   "max_tokens",
}

// getEnvOrFile returns the value of the environment variable named by envKey;
// if empty, falls back to the corresponding key in the config file map.
func getEnvOrFile(envKey string, file map[string]string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return file[fileKey[envKey]]
}

// Load reads configuration from environment variables, falling back to the config file.
func (l *envLoader) Load() (Config, error) {
	cfgPath := l.configFile
	if cfgPath == "" {
		cfgPath = ConfigFilePath()
	}
	file, err := LoadConfigFile(cfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", cfgPath, err)
	}

	apiKey := getEnvOrFile(EnvAPIKey, file)
	if apiKey == "" {
		return Config{}, fmt.Errorf("missing required configuration: set %s env var or api_key in %s", EnvAPIKey, cfgPath)
	}

	baseURL := getEnvOrFile(EnvBaseURL, file)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	opusModel := getEnvOrFile(EnvOpusModel, file)
	if opusModel == "" {
		opusModel = defaultOpusModel
	}
	sonnetModel := getEnvOrFile(EnvSonnetModel, file)
	if sonnetModel == "" {
		sonnetModel = defaultSonnetModel
	}
	haikuModel := getEnvOrFile(EnvHaikuModel, file)
	if haikuModel == "" {
		haikuModel = defaultHaikuModel
	}

	cfg := Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		OpusModel:   opusModel,
		SonnetModel: sonnetModel,
		HaikuModel:  haikuModel,
	}

	if s := getEnvOrFile(EnvMaxTokens, file); s != "" {
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
