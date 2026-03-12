package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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

// isRetryableError reports whether err warrants a retry attempt.
// ctx should be the parent (non-per-attempt) context; if it is cancelled the
// function returns false regardless of the error.
func isRetryableError(ctx context.Context, err error) bool {
	// Explicit user cancellation — do not retry.
	if errors.Is(ctx.Err(), context.Canceled) {
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
