package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStatusError_ErrorMessage(t *testing.T) {
	err := &StatusError{StatusCode: 429, Body: "rate limit exceeded"}
	got := err.Error()
	if !strings.Contains(got, "429") {
		t.Errorf("StatusError.Error() should contain status code, got: %q", got)
	}
	if !strings.Contains(got, "rate limit exceeded") {
		t.Errorf("StatusError.Error() should contain body, got: %q", got)
	}
}

func TestStatusError_ErrorsAs(t *testing.T) {
	err := &StatusError{StatusCode: 500, Body: "server error"}
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Error("errors.As should find *StatusError")
	}
	if statusErr.StatusCode != 500 {
		t.Errorf("statusErr.StatusCode = %d, want 500", statusErr.StatusCode)
	}
}

func TestWithRetryNotifier_InjectsIntoContext(t *testing.T) {
	var called bool
	fn := func(attempt int, err error) { called = true }
	ctx := WithRetryNotifier(context.Background(), fn)

	notifier, ok := ctx.Value(retryNotifierKey{}).(func(int, error))
	if !ok {
		t.Fatal("context should contain retry notifier function")
	}
	notifier(1, nil)
	if !called {
		t.Error("notifier function should have been called")
	}
}

func TestWithPerAttemptTimeout_InjectsIntoContext(t *testing.T) {
	ctx := WithPerAttemptTimeout(context.Background(), 30*time.Second)
	got, ok := ctx.Value(perAttemptTimeoutKey{}).(time.Duration)
	if !ok {
		t.Fatal("context should contain per-attempt timeout duration")
	}
	if got != 30*time.Second {
		t.Errorf("per-attempt timeout = %v, want 30s", got)
	}
}

func TestSendMessage_NoRetryByDefault(t *testing.T) {
	// Default client has maxRetries=0, so a 429 should not be retried.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(fixture(t, "error_rate_limited.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 request (no retry by default), got %d", got)
	}
}

func TestSendMessage_NoRetryOn4xx(t *testing.T) {
	// 400 is not retryable; should return immediately regardless of maxRetries.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_bad_request.json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(3))
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 request (no retry for 4xx), got %d", got)
	}
}

func TestRetryNotifier_CalledBeforeEachRetry(t *testing.T) {
	// Server returns 429 for the first 2 calls, then 200.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(fixture(t, "error_rate_limited.json"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	var notifiedAttempts []int
	ctx := WithRetryNotifier(context.Background(), func(attempt int, err error) {
		notifiedAttempts = append(notifiedAttempts, attempt)
	})

	client := NewClient(server.URL, "key", WithMaxRetries(3))
	_, err := client.SendMessage(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifiedAttempts) != 2 {
		t.Errorf("expected 2 retry notifications, got %d: %v", len(notifiedAttempts), notifiedAttempts)
	}
	if len(notifiedAttempts) >= 1 && notifiedAttempts[0] != 1 {
		t.Errorf("first retry attempt = %d, want 1", notifiedAttempts[0])
	}
	if len(notifiedAttempts) >= 2 && notifiedAttempts[1] != 2 {
		t.Errorf("second retry attempt = %d, want 2", notifiedAttempts[1])
	}
}
