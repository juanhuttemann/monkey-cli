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

func TestWithMaxRetries_SetsField(t *testing.T) {
	client := NewClient("https://example.com", "key", WithMaxRetries(3))
	if client.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", client.maxRetries)
	}
}

func TestWithRetryDelay_SetsField(t *testing.T) {
	client := NewClient("https://example.com", "key", WithRetryDelay(500*time.Millisecond))
	if client.retryDelay != 500*time.Millisecond {
		t.Errorf("retryDelay = %v, want 500ms", client.retryDelay)
	}
}

func TestSendMessage_NoRetryByDefault(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
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

func TestSendMessage_RetriesOn429(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	result, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("expected 3 requests (1 initial + 2 retries), got %d", got)
	}
}

func TestSendMessage_RetriesOn500(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "recovered"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(2), WithRetryDelay(0))
	result, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result != "recovered" {
		t.Errorf("result = %q, want %q", result, "recovered")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 requests (1 initial + 1 retry), got %d", got)
	}
}

func TestSendMessage_MaxRetriesExceeded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error after max retries exceeded")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention 429, got: %v", err)
	}
	if got := calls.Load(); got != 4 { // 1 initial + 3 retries
		t.Errorf("expected 4 requests (1 initial + 3 retries), got %d", got)
	}
}

func TestSendMessage_NoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 request (no retry for 4xx), got %d", got)
	}
}

func TestSendMessage_NoRetryOnParseError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	_, err := client.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 request (no retry for parse errors), got %d", got)
	}
}

func TestSendMessage_ContextCancelledDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(10), WithRetryDelay(20*time.Millisecond))
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err := client.SendMessage(ctx, "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got >= 10 {
		t.Errorf("expected retries to stop when context cancelled, but got %d calls", got)
	}
}

func TestRetryNotifier_CalledBeforeEachRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	var notifiedAttempts []int
	ctx := WithRetryNotifier(context.Background(), func(attempt int, err error) {
		notifiedAttempts = append(notifiedAttempts, attempt)
	})

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
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

func TestStatusError_IsRetryable(t *testing.T) {
	cases := []struct {
		statusCode int
		retryable  bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
	}
	for _, tc := range cases {
		err := &StatusError{StatusCode: tc.statusCode}
		got := isRetryableError(context.Background(), err)
		if got != tc.retryable {
			t.Errorf("isRetryableError(status %d) = %v, want %v", tc.statusCode, got, tc.retryable)
		}
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

func TestSendMessage_RetriesOnDeadlineExceeded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// Simulate slow first response — longer than per-attempt timeout
			time.Sleep(100 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	// Per-attempt timeout shorter than the first response delay
	ctx := WithPerAttemptTimeout(context.Background(), 30*time.Millisecond)
	client := NewClient(server.URL, "key", WithMaxRetries(2), WithRetryDelay(0))
	result, err := client.SendMessage(ctx, "test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 requests (1 timed-out + 1 retry), got %d", got)
	}
}

func TestSendMessage_RetriesOnConnectionReset(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// Abruptly close the connection to simulate a connection reset
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("server does not support hijacking")
				return
			}
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(2), WithRetryDelay(0))
	result, err := client.SendMessage(context.Background(), "test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 requests (1 reset + 1 retry), got %d", got)
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

func TestSendMessage_RetriesAfterTimeoutThenConnectionReset(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		switch n {
		case 1:
			// First attempt: slow response → per-attempt timeout fires
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
		case 2:
			// Second attempt: connection reset
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("server does not support hijacking")
				return
			}
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		default:
			// Third attempt: success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"content": [{"type": "text", "text": "ok"}]}`))
		}
	}))
	defer server.Close()

	ctx := WithPerAttemptTimeout(context.Background(), 30*time.Millisecond)
	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	result, err := client.SendMessage(ctx, "test")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("expected 3 requests (timeout + reset + success), got %d", got)
	}
}
