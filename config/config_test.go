package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_AllVarsPresent(t *testing.T) {
	// Clean up before and after
	os.Unsetenv(EnvAPIKey)
	os.Unsetenv(EnvBaseURL)
	os.Unsetenv(EnvModel)
	defer func() {
		os.Unsetenv(EnvAPIKey)
		os.Unsetenv(EnvBaseURL)
		os.Unsetenv(EnvModel)
	}()

	os.Setenv(EnvAPIKey, "test-api-key-123")
	os.Setenv(EnvBaseURL, "https://api.example.com")
	os.Setenv(EnvModel, "claude-3-test")

	loader := NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.APIKey != "test-api-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-api-key-123")
	}
	if cfg.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.example.com")
	}
	if cfg.Model != "claude-3-test" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-3-test")
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	os.Unsetenv(EnvAPIKey)
	os.Setenv(EnvBaseURL, "https://api.example.com")
	os.Setenv(EnvModel, "test-model")
	defer func() {
		os.Unsetenv(EnvBaseURL)
		os.Unsetenv(EnvModel)
	}()

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_API_KEY is missing")
	}

	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Errorf("error should mention %s, got: %v", EnvAPIKey, err)
	}
}

func TestLoad_MissingBaseURL(t *testing.T) {
	os.Setenv(EnvAPIKey, "test-key")
	os.Unsetenv(EnvBaseURL)
	os.Setenv(EnvModel, "test-model")
	defer func() {
		os.Unsetenv(EnvAPIKey)
		os.Unsetenv(EnvModel)
	}()

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_BASE_URL is missing")
	}

	if !strings.Contains(err.Error(), EnvBaseURL) {
		t.Errorf("error should mention %s, got: %v", EnvBaseURL, err)
	}
}

func TestLoad_MissingModel(t *testing.T) {
	os.Setenv(EnvAPIKey, "test-key")
	os.Setenv(EnvBaseURL, "https://api.example.com")
	os.Unsetenv(EnvModel)
	defer func() {
		os.Unsetenv(EnvAPIKey)
		os.Unsetenv(EnvBaseURL)
	}()

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_MODEL is missing")
	}

	if !strings.Contains(err.Error(), EnvModel) {
		t.Errorf("error should mention %s, got: %v", EnvModel, err)
	}
}

func TestLoad_MissingAllVars(t *testing.T) {
	os.Unsetenv(EnvAPIKey)
	os.Unsetenv(EnvBaseURL)
	os.Unsetenv(EnvModel)

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when all vars are missing")
	}

	// Should report the first missing var (API_KEY)
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Errorf("error should mention %s, got: %v", EnvAPIKey, err)
	}
}

func TestEnvConstants(t *testing.T) {
	// Verify environment variable constants are correct
	if EnvAPIKey != "ANTHROPIC_API_KEY" {
		t.Errorf("EnvAPIKey = %q, want %q", EnvAPIKey, "ANTHROPIC_API_KEY")
	}
	if EnvBaseURL != "ANTHROPIC_BASE_URL" {
		t.Errorf("EnvBaseURL = %q, want %q", EnvBaseURL, "ANTHROPIC_BASE_URL")
	}
	if EnvModel != "ANTHROPIC_MODEL" {
		t.Errorf("EnvModel = %q, want %q", EnvModel, "ANTHROPIC_MODEL")
	}
	if EnvMaxTokens != "CLAUDE_CODE_MAX_OUTPUT_TOKENS" {
		t.Errorf("EnvMaxTokens = %q, want %q", EnvMaxTokens, "CLAUDE_CODE_MAX_OUTPUT_TOKENS")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvModel, "test-model")
}

func TestLoad_MaxTokens_NotSet_DefaultsToZero(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv(EnvMaxTokens, "")

	cfg, err := NewEnvLoader().Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.MaxTokens != 0 {
		t.Errorf("MaxTokens = %d, want 0 (unset)", cfg.MaxTokens)
	}
}

func TestLoad_MaxTokens_Valid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv(EnvMaxTokens, "4096")

	cfg, err := NewEnvLoader().Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", cfg.MaxTokens)
	}
}

func TestLoad_MaxTokens_InvalidString(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv(EnvMaxTokens, "abc")

	_, err := NewEnvLoader().Load()
	if err == nil {
		t.Fatal("Load() should return error for non-integer CLAUDE_CODE_MAX_OUTPUT_TOKENS")
	}
	if !strings.Contains(err.Error(), EnvMaxTokens) {
		t.Errorf("error should mention %s, got: %v", EnvMaxTokens, err)
	}
}

func TestLoad_MaxTokens_Zero(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv(EnvMaxTokens, "0")

	_, err := NewEnvLoader().Load()
	if err == nil {
		t.Fatal("Load() should return error for CLAUDE_CODE_MAX_OUTPUT_TOKENS=0")
	}
	if !strings.Contains(err.Error(), EnvMaxTokens) {
		t.Errorf("error should mention %s, got: %v", EnvMaxTokens, err)
	}
}

func TestLoad_MaxTokens_Negative(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv(EnvMaxTokens, "-1")

	_, err := NewEnvLoader().Load()
	if err == nil {
		t.Fatal("Load() should return error for negative CLAUDE_CODE_MAX_OUTPUT_TOKENS")
	}
	if !strings.Contains(err.Error(), EnvMaxTokens) {
		t.Errorf("error should mention %s, got: %v", EnvMaxTokens, err)
	}
}

func TestLoadSystemPromptFile_FileExists(t *testing.T) {
	f, err := os.CreateTemp("", "system*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("# System Prompt\n\nYou are a helpful coding assistant.")
	f.Close()

	got, err := LoadSystemPromptFile(f.Name())
	if err != nil {
		t.Fatalf("LoadSystemPromptFile() returned error: %v", err)
	}
	want := "# System Prompt\n\nYou are a helpful coding assistant."
	if got != want {
		t.Errorf("LoadSystemPromptFile() = %q, want %q", got, want)
	}
}

func TestLoadSystemPromptFile_FileNotExist_ReturnsEmpty(t *testing.T) {
	got, err := LoadSystemPromptFile("/nonexistent/path/system.md")
	if err != nil {
		t.Fatalf("LoadSystemPromptFile() returned error for non-existent file: %v", err)
	}
	if got != "" {
		t.Errorf("LoadSystemPromptFile() = %q, want empty string", got)
	}
}

func TestLoadSystemPromptFile_EmptyFile_ReturnsEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "system*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	got, err := LoadSystemPromptFile(f.Name())
	if err != nil {
		t.Fatalf("LoadSystemPromptFile() returned error: %v", err)
	}
	if got != "" {
		t.Errorf("LoadSystemPromptFile() = %q, want empty string for empty file", got)
	}
}

func TestConfig_EmptyValues(t *testing.T) {
	// Setting empty string should be treated as missing
	os.Setenv(EnvAPIKey, "")
	os.Setenv(EnvBaseURL, "https://api.example.com")
	os.Setenv(EnvModel, "test-model")
	defer func() {
		os.Unsetenv(EnvAPIKey)
		os.Unsetenv(EnvBaseURL)
		os.Unsetenv(EnvModel)
	}()

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_API_KEY is empty string")
	}
}
