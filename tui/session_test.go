package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/juanhuttemann/monkey-cli/api"
)

func TestSaveAndLoadSession_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")

	apiMsgs := []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	displayMsgs := []Message{
		{Role: "user", Content: "hello", Timestamp: time.Now().Truncate(time.Second)},
		{Role: "assistant", Content: "hi there", Timestamp: time.Now().Truncate(time.Second)},
	}

	if err := SaveSession(path, "claude-sonnet-4-6", apiMsgs, displayMsgs); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	if sess.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q", sess.Model, "claude-sonnet-4-6")
	}
	if len(sess.APIMessages) != 2 {
		t.Fatalf("APIMessages len = %d, want 2", len(sess.APIMessages))
	}
	if sess.APIMessages[0].Role != "user" {
		t.Errorf("APIMessages[0].Role = %q, want %q", sess.APIMessages[0].Role, "user")
	}
	if c, ok := sess.APIMessages[1].Content.(string); !ok || c != "hi there" {
		t.Errorf("APIMessages[1].Content = %v, want %q", sess.APIMessages[1].Content, "hi there")
	}
	if len(sess.Messages) != 2 {
		t.Fatalf("Messages len = %d, want 2", len(sess.Messages))
	}
	if sess.Messages[0].Content != "hello" {
		t.Errorf("Messages[0].Content = %q, want %q", sess.Messages[0].Content, "hello")
	}
}

func TestLoadSession_FileNotFound_ReturnsNil(t *testing.T) {
	sess, err := LoadSession("/nonexistent/session.json")
	if err != nil {
		t.Fatalf("LoadSession non-existent file returned error: %v", err)
	}
	if sess != nil {
		t.Errorf("LoadSession non-existent = %v, want nil", sess)
	}
}

func TestSaveSession_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "session.json")

	if err := SaveSession(path, "model", nil, nil); err != nil {
		t.Fatalf("SaveSession with nested path returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("session file not created at %s: %v", path, err)
	}
}

func TestSessionPath_ContainsMonkey(t *testing.T) {
	p := SessionPath()
	if !strings.Contains(p, "monkey") {
		t.Errorf("SessionPath() = %q, want path containing 'monkey'", p)
	}
}

func TestRestoreSession_LoadsMessagesAndAPIMessages(t *testing.T) {
	m := NewModel(nil)
	sess := &SessionData{
		Model: "claude-sonnet-4-6",
		APIMessages: []api.Message{
			{Role: "user", Content: "hi"},
		},
		Messages: []Message{
			{Role: "user", Content: "hi", Timestamp: time.Now()},
		},
	}
	m.RestoreSession(sess)

	if len(m.GetHistory()) != 1 {
		t.Errorf("GetHistory() len = %d, want 1", len(m.GetHistory()))
	}
	if m.GetHistory()[0].Content != "hi" {
		t.Errorf("GetHistory()[0].Content = %q, want %q", m.GetHistory()[0].Content, "hi")
	}
	if len(m.GetAPIMessages()) != 1 {
		t.Errorf("GetAPIMessages() len = %d, want 1", len(m.GetAPIMessages()))
	}
}

func TestRestoreSession_NilIsNoop(t *testing.T) {
	m := NewModel(nil)
	m.RestoreSession(nil) // should not panic
	if len(m.GetHistory()) != 0 {
		t.Errorf("GetHistory() after nil restore = %d, want 0", len(m.GetHistory()))
	}
}

func TestLoadSession_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	if err := os.WriteFile(path, []byte(`{not valid json`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSession(path)
	if err == nil {
		t.Fatal("LoadSession with invalid JSON should return error")
	}
}

func TestLoadSession_ReadError(t *testing.T) {
	// Create a directory at the session path — os.ReadFile on a dir returns an
	// error that is NOT os.IsNotExist, exercising the non-nil, non-NotExist branch.
	dir := t.TempDir()
	path := filepath.Join(dir, "session_dir")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSession(path)
	if err == nil {
		t.Fatal("LoadSession on a directory should return error")
	}
}

func TestSaveSession_WriteError(t *testing.T) {
	// Make the target path a directory so os.WriteFile fails.
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}

	err := SaveSession(target, "model", nil, nil)
	if err == nil {
		t.Fatal("SaveSession should return error when write fails")
	}
}

func TestSaveSession_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where SaveSession would try to create a directory.
	parentPath := filepath.Join(dir, "notadir")
	if err := os.WriteFile(parentPath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Session path is inside "notadir" (which is a file, not a dir).
	path := filepath.Join(parentPath, "session.json")

	err := SaveSession(path, "model", nil, nil)
	if err == nil {
		t.Fatal("SaveSession should return error when MkdirAll fails")
	}
}

func TestSessionPath_XDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	p := SessionPath()
	if !strings.Contains(p, "/custom/config") {
		t.Errorf("SessionPath() = %q, want path under XDG_CONFIG_HOME", p)
	}
}

func TestGetAPIMessages_InitiallyEmpty(t *testing.T) {
	m := NewModel(nil)
	if len(m.GetAPIMessages()) != 0 {
		t.Errorf("GetAPIMessages() on new model = %d, want 0", len(m.GetAPIMessages()))
	}
}

func TestRestoreSession_WithModel_SetsClientModel(t *testing.T) {
	client := newTestClientWithModel("claude-sonnet")
	m := NewModel(client)
	sess := &SessionData{
		Model:       "claude-opus",
		APIMessages: []api.Message{{Role: "user", Content: "hi"}},
		Messages:    []Message{{Role: "user", Content: "hi", Timestamp: time.Now()}},
	}
	m.RestoreSession(sess)
	if client.GetModel() != "claude-opus" {
		t.Errorf("RestoreSession should set client model to %q, got %q", "claude-opus", client.GetModel())
	}
}
