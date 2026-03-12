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

func TestSendMessage_RetriesOn429(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(fixture(t, "error_rate_limited.json"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
			_, _ = w.Write(fixture(t, "error_server.json"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_recovered.json"))
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
		_, _ = w.Write(fixture(t, "error_rate_limited.json"))
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
		_, _ = w.Write(fixture(t, "error_bad_request.json"))
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
		_, _ = w.Write([]byte(`invalid json`))
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
		_, _ = w.Write(fixture(t, "error_server.json"))
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
			_, _ = w.Write(fixture(t, "error_rate_limited.json"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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
		_, _ = w.Write(fixture(t, "response_ok.json"))
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

func TestParseRetryAfter_Seconds(t *testing.T) {
	got := parseRetryAfter("30")
	if got != 30*time.Second {
		t.Errorf("parseRetryAfter(\"30\") = %v, want 30s", got)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Use an HTTP date 60 seconds in the future
	future := time.Now().Add(60 * time.Second).UTC()
	header := future.Format(http.TimeFormat)
	got := parseRetryAfter(header)
	if got <= 0 {
		t.Errorf("parseRetryAfter(future HTTP date) = %v, want positive duration", got)
	}
	// Should be roughly 60 seconds (allow some tolerance)
	if got > 65*time.Second || got < 55*time.Second {
		t.Errorf("parseRetryAfter(future HTTP date) = %v, want ~60s", got)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	got := parseRetryAfter("")
	if got != 0 {
		t.Errorf("parseRetryAfter(\"\") = %v, want 0", got)
	}
}

func TestParseRetryAfter_InvalidString(t *testing.T) {
	got := parseRetryAfter("bogus")
	if got != 0 {
		t.Errorf("parseRetryAfter(\"bogus\") = %v, want 0", got)
	}
}

func TestComputeRetryDelay_ExponentialBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	if got := computeRetryDelay(base, 1, nil); got != base {
		t.Errorf("computeRetryDelay(base, 1, nil) = %v, want %v", got, base)
	}
	if got := computeRetryDelay(base, 2, nil); got != 2*base {
		t.Errorf("computeRetryDelay(base, 2, nil) = %v, want %v", got, 2*base)
	}
}

func TestComputeRetryDelay_RespectsRetryAfterWhenLarger(t *testing.T) {
	base := 100 * time.Millisecond
	// RetryAfter = 5s >> base * 2^0 = 100ms
	err := &StatusError{StatusCode: 429, RetryAfter: 5 * time.Second}
	got := computeRetryDelay(base, 1, err)
	if got != 5*time.Second {
		t.Errorf("computeRetryDelay with large RetryAfter = %v, want 5s", got)
	}
}

func TestIsRetryableError_ParentDeadlineExpired_NotRetryable(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()
	// Even though DeadlineExceeded looks like a per-attempt timeout, an expired
	// parent deadline must not trigger a retry.
	if got := isRetryableError(ctx, context.DeadlineExceeded); got {
		t.Error("isRetryableError with expired parent deadline = true, want false")
	}
}

func TestSendMessage_ParentDeadlineExpired_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond) // outlast the parent deadline
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	client := NewClient(server.URL, "key", WithMaxRetries(3), WithRetryDelay(0))
	_, err := client.SendMessage(ctx, "test")
	if err == nil {
		t.Fatal("expected error when parent deadline expires")
	}
	// Should not retry after the parent context deadline has expired.
	if got := calls.Load(); got > 1 {
		t.Errorf("expected 1 request (no retry after parent deadline), got %d", got)
	}
}

func TestComputeRetryDelay_IgnoresRetryAfterWhenSmaller(t *testing.T) {
	base := 10 * time.Second
	// base * 2^1 = 20s > RetryAfter = 1s
	err := &StatusError{StatusCode: 429, RetryAfter: 1 * time.Second}
	got := computeRetryDelay(base, 2, err)
	if got != 20*time.Second {
		t.Errorf("computeRetryDelay with small RetryAfter = %v, want 20s", got)
	}
}

func TestSendMessage_429WithRetryAfterHeader_SetsRetryAfterOnStatusError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(fixture(t, "error_rate_limited.json"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "response_ok.json"))
	}))
	defer server.Close()

	var capturedErr error
	ctx := WithRetryNotifier(context.Background(), func(attempt int, err error) {
		capturedErr = err
	})

	client := NewClient(server.URL, "key", WithMaxRetries(2), WithRetryDelay(0))
	_, err := client.SendMessage(ctx, "test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}

	if capturedErr == nil {
		t.Fatal("expected retry notifier to be called with the 429 error")
	}
	var statusErr *StatusError
	if !errors.As(capturedErr, &statusErr) {
		t.Fatalf("expected *StatusError, got %T", capturedErr)
	}
	if statusErr.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", statusErr.StatusCode)
	}
	if statusErr.RetryAfter != 5*time.Second {
		t.Errorf("RetryAfter = %v, want 5s", statusErr.RetryAfter)
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
			_, _ = w.Write(fixture(t, "response_ok.json"))
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
			_, _ = w.Write(fixture(t, "response_ok.json"))
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
