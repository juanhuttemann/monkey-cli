package tui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/juanhuttemann/monkey-cli/api"
)

// SessionData is the persisted state for --continue.
type SessionData struct {
	Model       string        `json:"model"`
	APIMessages []api.Message `json:"api_messages"`
	Messages    []Message     `json:"messages"`
}

// SessionPath returns the path to the session file, respecting XDG_CONFIG_HOME.
func SessionPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "session.json"
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "monkey", "session.json")
}

// SaveSession persists the current session to path, creating parent directories as needed.
func SaveSession(path, model string, apiMessages []api.Message, messages []Message) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(SessionData{
		Model:       model,
		APIMessages: apiMessages,
		Messages:    messages,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSession reads the session file at path and returns the decoded SessionData.
// Returns (nil, nil) when the file does not exist.
func LoadSession(path string) (*SessionData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sess SessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}
