package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
)

func TestTimer_NotRunningInitially(t *testing.T) {
	model := NewModel(nil)
	if model.IsTimerRunning() {
		t.Error("IsTimerRunning() = true on fresh model, want false")
	}
}

func TestTimer_StartsOnSubmit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "response"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-key", api.WithModel("test-model"))
	model := NewModel(client)
	model.SetInput("hello")

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlM})
	m := updatedModel.(Model)

	if !m.IsTimerRunning() {
		t.Error("IsTimerRunning() = false after prompt submit, want true")
	}
}

func TestTimer_StopsOnResponse(t *testing.T) {
	model := NewModel(nil)
	model.SetTimerActive(true)

	updatedModel, _ := model.Update(PromptResponseMsg{Response: "hi"})
	m := updatedModel.(Model)

	if m.IsTimerRunning() {
		t.Error("IsTimerRunning() = true after PromptResponseMsg, want false")
	}
}

func TestTimer_StopsOnError(t *testing.T) {
	model := NewModel(nil)
	model.SetTimerActive(true)

	updatedModel, _ := model.Update(PromptErrorMsg{Err: &testError{msg: "api failed"}})
	m := updatedModel.(Model)

	if m.IsTimerRunning() {
		t.Error("IsTimerRunning() = true after PromptErrorMsg, want false")
	}
}

func TestTimer_TickMsg_UpdatesTimerWhenActive(t *testing.T) {
	model := NewModel(nil)
	model.SetTimerActive(true)

	_, cmd := model.Update(timer.TickMsg{})

	if cmd == nil {
		t.Error("timer.TickMsg when timer is active should return non-nil cmd (continue ticking)")
	}
}

func TestTimer_TickMsg_IgnoredWhenNotActive(t *testing.T) {
	model := NewModel(nil)
	// timerActive is false by default

	_, cmd := model.Update(timer.TickMsg{})

	if cmd != nil {
		t.Error("timer.TickMsg when timer is not active should return nil cmd (no ticking)")
	}
}

func TestView_ShowsElapsedTimeWhenLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetLoading(true)
	model.SetTimerActive(true)

	view := model.View()

	// Should show a duration string (e.g. "0s") — no ⏱ emoji
	if !strings.Contains(view, "0s") {
		t.Error("View() should show elapsed duration (e.g. '0s') when loading with active timer")
	}
	if strings.Contains(view, "⏱") {
		t.Error("View() should not contain the ⏱ emoji (renders poorly in many terminals)")
	}
}

func TestView_HidesElapsedTimeWhenNotLoading(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetLoading(false)

	view := model.View()

	if strings.Contains(view, "⏱") {
		t.Error("View() should not show elapsed time indicator (⏱) when not loading")
	}
}

func TestView_ShowsLastElapsedAfterResponse(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetTimerActive(true)

	updatedModel, _ := model.Update(PromptResponseMsg{Response: "hi"})
	m := updatedModel.(Model)

	view := m.View()

	if !strings.Contains(view, "took ") {
		t.Error("View() should show 'took <duration>' after response arrives")
	}
}

func TestView_ShowsLastElapsedAfterError(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetTimerActive(true)

	updatedModel, _ := model.Update(PromptErrorMsg{Err: &testError{msg: "oops"}})
	m := updatedModel.(Model)

	view := m.View()

	if !strings.Contains(view, "took ") {
		t.Error("View() should show 'took <duration>' after error arrives")
	}
}

func TestView_NoLastElapsedBeforeFirstSubmit(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	view := model.View()

	if strings.Contains(view, "took ") {
		t.Error("View() should not show 'took' before any prompt has been submitted")
	}
}
