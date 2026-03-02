package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mogger/api"
)

func TestWaitForRetry_ReturnsRetryingMsg(t *testing.T) {
	ch := make(chan RetryingMsg, 1)
	ch <- RetryingMsg{Attempt: 2}

	cmd := waitForRetry(ch)
	result := cmd()

	msg, ok := result.(RetryingMsg)
	if !ok {
		t.Fatalf("waitForRetry returned %T, want RetryingMsg", result)
	}
	if msg.Attempt != 2 {
		t.Errorf("Attempt = %d, want 2", msg.Attempt)
	}
}

func TestWaitForRetry_ReturnsRetryDoneMsg_WhenClosed(t *testing.T) {
	ch := make(chan RetryingMsg)
	close(ch)

	cmd := waitForRetry(ch)
	result := cmd()

	if _, ok := result.(retryDoneMsg); !ok {
		t.Fatalf("waitForRetry returned %T, want retryDoneMsg", result)
	}
}

func TestSendPromptCmdWithTimeout_SendsRetryNotifications(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "key", api.WithModel("m"), api.WithMaxRetries(2), api.WithRetryDelay(0))
	messages := []Message{}

	retryCh := make(chan RetryingMsg, 10)
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test", 5*time.Second, nil, retryCh)

	// Drain retryCh in a goroutine while cmd runs
	var retryMsgs []RetryingMsg
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range retryCh {
			retryMsgs = append(retryMsgs, msg)
		}
	}()

	result := cmd()
	<-done

	if _, ok := result.(PromptResponseMsg); !ok {
		t.Fatalf("expected PromptResponseMsg, got %T", result)
	}
	if len(retryMsgs) != 1 {
		t.Errorf("expected 1 retry notification, got %d", len(retryMsgs))
	}
	if len(retryMsgs) > 0 && retryMsgs[0].Attempt != 1 {
		t.Errorf("first retry attempt = %d, want 1", retryMsgs[0].Attempt)
	}
}

func TestSendPromptCmdWithTimeout_ClosesRetryCh_OnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "key", api.WithModel("m"))
	messages := []Message{}

	retryCh := make(chan RetryingMsg, 10)
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test", 5*time.Second, nil, retryCh)
	cmd()

	select {
	case _, ok := <-retryCh:
		if ok {
			t.Error("retryCh should be closed after successful request")
		}
	case <-time.After(time.Second):
		t.Error("retryCh was not closed after request completed")
	}
}

func TestSendPromptCmdWithTimeout_ClosesRetryCh_OnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "key", api.WithModel("m"))
	messages := []Message{}

	retryCh := make(chan RetryingMsg, 10)
	cmd, _ := SendPromptCmdWithTimeout(client, messages, "test", 5*time.Second, nil, retryCh)
	cmd()

	select {
	case _, ok := <-retryCh:
		if ok {
			t.Error("retryCh should be closed after request error")
		}
	case <-time.After(time.Second):
		t.Error("retryCh was not closed after request error")
	}
}

func TestModel_RetryingMsg_DoesNotAddToHistory(t *testing.T) {
	m := NewModel(nil)

	model, _ := m.Update(RetryingMsg{Attempt: 1})

	updated := model.(Model)
	if len(updated.GetHistory()) != 0 {
		t.Errorf("RetryingMsg should not add messages to history, got: %v", updated.GetHistory())
	}
}

func TestModel_RetryingMsg_ShowsInSpinnerLine(t *testing.T) {
	m := NewModel(nil)
	m.SetLoading(true)
	m.SetTimerActive(true)
	m.SetDimensions(80, 24)

	model, _ := m.Update(RetryingMsg{Attempt: 2})
	updated := model.(Model)

	view := updated.View()
	if !strings.Contains(view, "retrying") {
		t.Errorf("View() should contain 'retrying', got:\n%s", view)
	}
	if !strings.Contains(view, "2") {
		t.Errorf("View() should contain attempt number '2', got:\n%s", view)
	}
}

func TestModel_RetryDoneMsg_IsIgnored(t *testing.T) {
	m := NewModel(nil)
	m.AddMessage("user", "hello")

	model, _ := m.Update(retryDoneMsg{})

	updated := model.(Model)
	if len(updated.GetHistory()) != 1 {
		t.Errorf("retryDoneMsg should not add messages, got %d messages", len(updated.GetHistory()))
	}
}
