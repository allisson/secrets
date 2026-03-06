package http

import (
	"context"
	"log/slog"
	"testing"

	"go.uber.org/goleak"
)

func TestRateLimitMiddleware_GoroutineLeak(t *testing.T) {
	// Ensure no goroutines are leaking after the test
	defer goleak.VerifyNone(t)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create middleware (this starts the goroutine)
	logger := slog.Default()
	_ = RateLimitMiddleware(ctx, 10.0, 20, logger)

	// Cancel the context - this should stop the goroutine
	cancel()
}

func TestTokenRateLimitMiddleware_GoroutineLeak(t *testing.T) {
	// Ensure no goroutines are leaking after the test
	defer goleak.VerifyNone(t)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create middleware (this starts the goroutine)
	logger := slog.Default()
	_ = TokenRateLimitMiddleware(ctx, 10.0, 20, logger)

	// Cancel the context - this should stop the goroutine
	cancel()
}
