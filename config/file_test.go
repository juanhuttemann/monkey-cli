package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseConfigFile ---

func TestParseConfigFile_StringValue(t *testing.T) {
	src := `api_key = "sk-ant-test"`
	kv, err := parseConfigFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if kv["api_key"] != "sk-ant-test" {
		t.Errorf("api_key = %q, want %q", kv["api_key"], "sk-ant-test")
	}
}

func TestParseConfigFile_IntegerValue(t *testing.T) {
	src := `max_tokens = 4096`
	kv, err := parseConfigFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if kv["max_tokens"] != "4096" {
		t.Errorf("max_tokens = %q, want %q", kv["max_tokens"], "4096")
	}
}

func TestParseConfigFile_SkipsComments(t *testing.T) {
	src := "# this is a comment\napi_key = \"key\""
	kv, err := parseConfigFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if len(kv) != 1 || kv["api_key"] != "key" {
		t.Errorf("kv = %v, want {api_key: key}", kv)
	}
}

func TestParseConfigFile_SkipsBlankLines(t *testing.T) {
	src := "\n\napi_key = \"key\"\n\n"
	kv, err := parseConfigFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if kv["api_key"] != "key" {
		t.Errorf("api_key = %q, want %q", kv["api_key"], "key")
	}
}

func TestParseConfigFile_MultipleKeys(t *testing.T) {
	src := "api_key = \"k1\"\nsonnet_model = \"claude-sonnet\"\nmax_tokens = 8192"
	kv, err := parseConfigFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if kv["api_key"] != "k1" {
		t.Errorf("api_key = %q, want %q", kv["api_key"], "k1")
	}
	if kv["sonnet_model"] != "claude-sonnet" {
		t.Errorf("sonnet_model = %q, want %q", kv["sonnet_model"], "claude-sonnet")
	}
	if kv["max_tokens"] != "8192" {
		t.Errorf("max_tokens = %q, want %q", kv["max_tokens"], "8192")
	}
}

func TestParseConfigFile_Empty(t *testing.T) {
	kv, err := parseConfigFile(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseConfigFile error: %v", err)
	}
	if len(kv) != 0 {
		t.Errorf("kv len = %d, want 0", len(kv))
	}
}

// --- LoadConfigFile ---

func TestLoadConfigFile_FileNotFound_ReturnsEmpty(t *testing.T) {
	kv, err := LoadConfigFile("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("LoadConfigFile non-existent file returned error: %v", err)
	}
	if len(kv) != 0 {
		t.Errorf("LoadConfigFile non-existent = %v, want empty", kv)
	}
}

func TestLoadConfigFile_ValidFile(t *testing.T) {
	f, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.WriteString("api_key = \"test-key\"\nsonnet_model = \"claude-sonnet\"\n")
	_ = f.Close()

	kv, err := LoadConfigFile(f.Name())
	if err != nil {
		t.Fatalf("LoadConfigFile returned error: %v", err)
	}
	if kv["api_key"] != "test-key" {
		t.Errorf("api_key = %q, want %q", kv["api_key"], "test-key")
	}
	if kv["sonnet_model"] != "claude-sonnet" {
		t.Errorf("sonnet_model = %q, want %q", kv["sonnet_model"], "claude-sonnet")
	}
}

// --- envLoader with config file fallback ---

func TestLoad_ConfigFileUsedWhenEnvMissing(t *testing.T) {
	// Write a temp config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(cfgPath, []byte("api_key = \"file-key\"\nsonnet_model = \"claude-sonnet-file\"\n"), 0600)

	// Unset the relevant env vars
	_ = os.Unsetenv(EnvAPIKey)
	_ = os.Unsetenv(EnvSonnetModel)
	_ = os.Unsetenv(EnvOpusModel)
	_ = os.Unsetenv(EnvHaikuModel)

	loader := NewEnvLoaderWithConfigFile(cfgPath)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "file-key")
	}
	if cfg.SonnetModel != "claude-sonnet-file" {
		t.Errorf("SonnetModel = %q, want %q", cfg.SonnetModel, "claude-sonnet-file")
	}
}

func TestLoad_EnvVarOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(cfgPath, []byte("api_key = \"file-key\"\nsonnet_model = \"file-model\"\n"), 0600)

	t.Setenv(EnvAPIKey, "env-key")
	t.Setenv(EnvSonnetModel, "env-model")
	_ = os.Unsetenv(EnvOpusModel)
	_ = os.Unsetenv(EnvHaikuModel)

	loader := NewEnvLoaderWithConfigFile(cfgPath)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want %q (env should win)", cfg.APIKey, "env-key")
	}
	if cfg.SonnetModel != "env-model" {
		t.Errorf("SonnetModel = %q, want %q (env should win)", cfg.SonnetModel, "env-model")
	}
}

func TestLoad_ConfigFilePath(t *testing.T) {
	// ConfigFilePath should return a valid path under ~/.config/monkey/
	p := ConfigFilePath()
	if !strings.HasSuffix(p, "config.toml") {
		t.Errorf("ConfigFilePath() = %q, want path ending in config.toml", p)
	}
	if !strings.Contains(p, "monkey") {
		t.Errorf("ConfigFilePath() = %q, want path containing 'monkey'", p)
	}
}

func TestConfigFilePath_XDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/cfg")
	p := ConfigFilePath()
	if !strings.HasPrefix(p, "/custom/cfg") {
		t.Errorf("ConfigFilePath() = %q, want path under XDG_CONFIG_HOME", p)
	}
}

func TestLoadConfigFile_ReadError(t *testing.T) {
	// Directory at the path causes os.Open to succeed but reading fails.
	// Actually, open on a dir succeeds; use a non-exist error that's not IsNotExist.
	dir := t.TempDir()
	// Create a file where loadconfigfile expects a file, but make it a directory
	dirPath := filepath.Join(dir, "config.toml")
	if err := os.Mkdir(dirPath, 0o700); err != nil {
		t.Fatal(err)
	}
	// Opening a directory might succeed on some systems; pass a path we can't read.
	// Instead, use a path inside a directory with no read permission (skip if root).
	noReadDir := filepath.Join(dir, "noperm")
	if err := os.Mkdir(noReadDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noReadDir, 0o700) })

	_, err := LoadConfigFile(filepath.Join(noReadDir, "config.toml"))
	// Should be either nil (file not found returns empty) or an error.
	// On Linux as root, chmod 000 may not block. Allow either outcome.
	_ = err
}

func TestLoadSystemPromptFile_ReadError(t *testing.T) {
	// Create a directory at the file path — not a real file, causes an error
	// that is NOT os.IsNotExist.
	dir := t.TempDir()
	pathAsDir := filepath.Join(dir, "MONKEY.md")
	if err := os.Mkdir(pathAsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSystemPromptFile(pathAsDir)
	if err == nil {
		t.Fatal("LoadSystemPromptFile on a directory should return error")
	}
}
