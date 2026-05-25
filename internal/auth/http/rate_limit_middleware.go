// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// RateLimitMiddleware enforces per-client rate limiting on authenticated requests.
//
// MUST be used after AuthenticationMiddleware (requires authenticated client in context).
// Uses token bucket algorithm via golang.org/x/time/rate. Each client gets independent
// rate limiter based on their client ID.
//
// Configuration:
//   - ctx: Application context for cleanup goroutine
//   - rps: Requests per second allowed per client
//   - burst: Maximum burst capacity for temporary spikes
//
// Returns:
//   - 429 Too Many Requests: Rate limit exceeded (includes Retry-After header)
//   - Continues: Request allowed within rate limit
func RateLimitMiddleware(ctx context.Context, rps float64, burst int, logger *slog.Logger) gin.HandlerFunc {
	return newRateLimitMiddleware(ctx, rps, burst, logger,
		func(c *gin.Context) (uuid.UUID, bool) {
			client, ok := GetClient(c.Request.Context())
			if !ok || client == nil {
				logger.Error("rate limit middleware: no authenticated client in context")
				httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
				c.Abort()
				return uuid.UUID{}, false
			}
			return client.ID, true
		},
		func(id uuid.UUID) slog.Attr { return slog.String("client_id", id.String()) },
		"rate limit exceeded",
		"Too many requests. Please retry after the specified delay.",
	)
}
