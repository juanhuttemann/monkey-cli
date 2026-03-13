package api

import (
	"context"
	"time"
)

// retryNotifierKey is the context key for the per-request retry callback.
type retryNotifierKey struct{}

// perAttemptTimeoutKey is the context key for the per-attempt timeout duration.
type perAttemptTimeoutKey struct{}

// WithPerAttemptTimeout returns a context that will apply the given timeout to each
// individual request attempt. The SDK client extracts this value and passes it as
// option.WithRequestTimeout, so each retry gets a fresh timeout.
func WithPerAttemptTimeout(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, perAttemptTimeoutKey{}, d)
}

// WithRetryNotifier returns a context that carries a callback invoked before each retry
// attempt. attempt is 1-based; err is the error that triggered the retry.
func WithRetryNotifier(ctx context.Context, fn func(attempt int, err error)) context.Context {
	return context.WithValue(ctx, retryNotifierKey{}, fn)
}
