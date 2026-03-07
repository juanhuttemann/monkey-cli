package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_AllVarsPresent(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-api-key-123")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvOpusModel, "claude-opus-test")
	t.Setenv(EnvSonnetModel, "claude-sonnet-test")
	t.Setenv(EnvHaikuModel, "claude-haiku-test")

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
	if cfg.OpusModel != "claude-opus-test" {
		t.Errorf("OpusModel = %q, want %q", cfg.OpusModel, "claude-opus-test")
	}
	if cfg.SonnetModel != "claude-sonnet-test" {
		t.Errorf("SonnetModel = %q, want %q", cfg.SonnetModel, "claude-sonnet-test")
	}
	if cfg.HaikuModel != "claude-haiku-test" {
		t.Errorf("HaikuModel = %q, want %q", cfg.HaikuModel, "claude-haiku-test")
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	os.Unsetenv(EnvAPIKey)
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvOpusModel, "test-model")

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_API_KEY is missing")
	}
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Errorf("error should mention %s, got: %v", EnvAPIKey, err)
	}
}

func TestLoad_MissingBaseURL_DefaultsToAnthropicAPI(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-key")
	os.Unsetenv(EnvBaseURL)
	t.Setenv(EnvOpusModel, "test-model")

	loader := NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() should succeed when ANTHROPIC_BASE_URL is unset (use default): %v", err)
	}
	if cfg.BaseURL != "https://api.anthropic.com" {
		t.Errorf("BaseURL = %q, want default %q", cfg.BaseURL, "https://api.anthropic.com")
	}
}

func TestLoad_NoModels(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	os.Unsetenv(EnvOpusModel)
	os.Unsetenv(EnvSonnetModel)
	os.Unsetenv(EnvHaikuModel)

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when no model env vars are set")
	}
	if !strings.Contains(err.Error(), EnvOpusModel) {
		t.Errorf("error should mention %s, got: %v", EnvOpusModel, err)
	}
}

func TestLoad_OnlyOpusModel(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvOpusModel, "claude-opus-4")
	os.Unsetenv(EnvSonnetModel)
	os.Unsetenv(EnvHaikuModel)

	cfg, err := NewEnvLoader().Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.DefaultModel() != "claude-opus-4" {
		t.Errorf("DefaultModel() = %q, want %q", cfg.DefaultModel(), "claude-opus-4")
	}
}

func TestLoad_OnlySonnetModel(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	os.Unsetenv(EnvOpusModel)
	t.Setenv(EnvSonnetModel, "claude-sonnet-4")
	os.Unsetenv(EnvHaikuModel)

	cfg, err := NewEnvLoader().Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.DefaultModel() != "claude-sonnet-4" {
		t.Errorf("DefaultModel() = %q, want %q", cfg.DefaultModel(), "claude-sonnet-4")
	}
}

func TestLoad_OnlyHaikuModel(t *testing.T) {
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	os.Unsetenv(EnvOpusModel)
	os.Unsetenv(EnvSonnetModel)
	t.Setenv(EnvHaikuModel, "claude-haiku-4")

	cfg, err := NewEnvLoader().Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.DefaultModel() != "claude-haiku-4" {
		t.Errorf("DefaultModel() = %q, want %q", cfg.DefaultModel(), "claude-haiku-4")
	}
}

func TestLoad_MissingAllVars(t *testing.T) {
	os.Unsetenv(EnvAPIKey)
	os.Unsetenv(EnvBaseURL)
	os.Unsetenv(EnvOpusModel)
	os.Unsetenv(EnvSonnetModel)
	os.Unsetenv(EnvHaikuModel)

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
	if EnvAPIKey != "ANTHROPIC_API_KEY" {
		t.Errorf("EnvAPIKey = %q, want %q", EnvAPIKey, "ANTHROPIC_API_KEY")
	}
	if EnvBaseURL != "ANTHROPIC_BASE_URL" {
		t.Errorf("EnvBaseURL = %q, want %q", EnvBaseURL, "ANTHROPIC_BASE_URL")
	}
	if EnvOpusModel != "ANTHROPIC_DEFAULT_OPUS_MODEL" {
		t.Errorf("EnvOpusModel = %q, want %q", EnvOpusModel, "ANTHROPIC_DEFAULT_OPUS_MODEL")
	}
	if EnvSonnetModel != "ANTHROPIC_DEFAULT_SONNET_MODEL" {
		t.Errorf("EnvSonnetModel = %q, want %q", EnvSonnetModel, "ANTHROPIC_DEFAULT_SONNET_MODEL")
	}
	if EnvHaikuModel != "ANTHROPIC_DEFAULT_HAIKU_MODEL" {
		t.Errorf("EnvHaikuModel = %q, want %q", EnvHaikuModel, "ANTHROPIC_DEFAULT_HAIKU_MODEL")
	}
	if EnvMaxTokens != "CLAUDE_CODE_MAX_OUTPUT_TOKENS" {
		t.Errorf("EnvMaxTokens = %q, want %q", EnvMaxTokens, "CLAUDE_CODE_MAX_OUTPUT_TOKENS")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvAPIKey, "test-key")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvOpusModel, "test-model")
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

func TestConfig_DefaultModel_OpusFirst(t *testing.T) {
	cfg := Config{OpusModel: "opus", SonnetModel: "sonnet", HaikuModel: "haiku"}
	if got := cfg.DefaultModel(); got != "opus" {
		t.Errorf("DefaultModel() = %q, want %q", got, "opus")
	}
}

func TestConfig_DefaultModel_SonnetWhenNoOpus(t *testing.T) {
	cfg := Config{SonnetModel: "sonnet", HaikuModel: "haiku"}
	if got := cfg.DefaultModel(); got != "sonnet" {
		t.Errorf("DefaultModel() = %q, want %q", got, "sonnet")
	}
}

func TestConfig_DefaultModel_HaikuWhenOnlyHaiku(t *testing.T) {
	cfg := Config{HaikuModel: "haiku"}
	if got := cfg.DefaultModel(); got != "haiku" {
		t.Errorf("DefaultModel() = %q, want %q", got, "haiku")
	}
}

func TestConfig_DefaultModel_EmptyWhenNone(t *testing.T) {
	cfg := Config{}
	if got := cfg.DefaultModel(); got != "" {
		t.Errorf("DefaultModel() = %q, want ''", got)
	}
}

func TestConfig_AvailableModels_All(t *testing.T) {
	cfg := Config{OpusModel: "opus", SonnetModel: "sonnet", HaikuModel: "haiku"}
	got := cfg.AvailableModels()
	if len(got) != 3 {
		t.Fatalf("AvailableModels() len = %d, want 3", len(got))
	}
	if got[0] != "opus" || got[1] != "sonnet" || got[2] != "haiku" {
		t.Errorf("AvailableModels() = %v, want [opus sonnet haiku]", got)
	}
}

func TestConfig_AvailableModels_OnlyOpus(t *testing.T) {
	cfg := Config{OpusModel: "opus"}
	got := cfg.AvailableModels()
	if len(got) != 1 || got[0] != "opus" {
		t.Errorf("AvailableModels() = %v, want [opus]", got)
	}
}

func TestConfig_AvailableModels_Empty(t *testing.T) {
	cfg := Config{}
	got := cfg.AvailableModels()
	if len(got) != 0 {
		t.Errorf("AvailableModels() = %v, want []", got)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{APIKey: "key", BaseURL: "url", OpusModel: "model"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_NoModel(t *testing.T) {
	cfg := Config{APIKey: "key", BaseURL: "url"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error when no model set")
	}
}

func TestConfig_EmptyValues(t *testing.T) {
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "https://api.example.com")
	t.Setenv(EnvOpusModel, "test-model")

	loader := NewEnvLoader()
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() should return error when ANTHROPIC_API_KEY is empty string")
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
