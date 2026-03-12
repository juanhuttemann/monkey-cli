package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// retryNotifierKey is the context key for the per-request retry callback.
type retryNotifierKey struct{}

// perAttemptTimeoutKey is the context key for the per-attempt timeout duration.
type perAttemptTimeoutKey struct{}

// WithPerAttemptTimeout returns a context that will apply the given timeout to each
// individual request attempt. This allows retrying after a timeout, as each retry
// gets a fresh timeout rather than sharing an already-expired one.
func WithPerAttemptTimeout(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, perAttemptTimeoutKey{}, d)
}

// WithRetryNotifier returns a context that carries a callback invoked before each retry attempt.
// attempt is 1-based; err is the error that triggered the retry.
func WithRetryNotifier(ctx context.Context, fn func(attempt int, err error)) context.Context {
	return context.WithValue(ctx, retryNotifierKey{}, fn)
}

// applyPerAttemptTimeout wraps ctx in a fresh timeout if perAttemptTimeoutKey is set.
// The returned CancelFunc must always be called (safe to call on a no-op cancel).
func applyPerAttemptTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if d, ok := ctx.Value(perAttemptTimeoutKey{}).(time.Duration); ok && d > 0 {
		return context.WithTimeout(ctx, d)
	}
	return ctx, func() {}
}

// parseRetryAfter parses the value of a Retry-After HTTP header.
// It first tries to parse it as an integer number of seconds, then as an HTTP
// date (e.g. "Wed, 21 Oct 2015 07:28:00 GMT"). Returns 0 for empty or invalid
// values, and 0 if an HTTP date is in the past.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	// Try integer seconds first.
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second
	}
	// Try HTTP date format.
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// computeRetryDelay returns the delay to wait before attempt number attempt (1-based).
// It computes base * 2^(attempt-1) as the exponential backoff, but uses
// lastErr's RetryAfter field instead when that is larger.
func computeRetryDelay(base time.Duration, attempt int, lastErr error) time.Duration {
	delay := base * time.Duration(1<<uint(attempt-1))
	var statusErr *StatusError
	if errors.As(lastErr, &statusErr) && statusErr.RetryAfter > delay {
		delay = statusErr.RetryAfter
	}
	return delay
}

// isRetryableError reports whether err warrants a retry attempt.
// ctx should be the parent (non-per-attempt) context; if it is done (cancelled
// or deadline exceeded) the function returns false regardless of the error.
func isRetryableError(ctx context.Context, err error) bool {
	// Parent context is done — do not retry.
	if ctx.Err() != nil {
		return false
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusTooManyRequests || statusErr.StatusCode >= 500
	}
	// Per-attempt timeout expired (parent ctx is still alive) — retry with fresh timeout.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	// Network-level transport errors (connection reset, EOF, etc.) are retryable.
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
