// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// TokenRateLimitMiddleware enforces per-IP rate limiting on token issuance endpoint.
//
// Designed for unauthenticated endpoints to prevent credential stuffing and brute force
// attacks. Uses token bucket algorithm via golang.org/x/time/rate. Each IP address gets
// an independent rate limiter.
//
// Uses c.ClientIP() which automatically handles:
//   - X-Forwarded-For header (takes first IP)
//   - X-Real-IP header
//   - Direct connection remote address
//
// Configuration:
//   - ctx: Application context for cleanup goroutine
//   - rps: Requests per second allowed per IP address
//   - burst: Maximum burst capacity for temporary spikes
//
// Returns:
//   - 429 Too Many Requests: Rate limit exceeded (includes Retry-After header)
//   - Continues: Request allowed within rate limit
func TokenRateLimitMiddleware(
	ctx context.Context,
	rps float64,
	burst int,
	logger *slog.Logger,
) gin.HandlerFunc {
	return newRateLimitMiddleware(ctx, rps, burst, logger,
		func(c *gin.Context) (string, bool) { return c.ClientIP(), true },
		func(ip string) slog.Attr { return slog.String("client_ip", ip) },
		"token rate limit exceeded",
		"Too many token requests from this IP. Please retry after the specified delay.",
	)
}
