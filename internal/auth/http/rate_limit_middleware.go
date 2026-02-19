// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// rateLimiterStore holds per-client rate limiters with automatic cleanup.
type rateLimiterStore struct {
	limiters sync.Map // map[uuid.UUID]*rateLimiterEntry
	rps      float64
	burst    int
}

// rateLimiterEntry holds a rate limiter and last access time for cleanup.
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

// RateLimitMiddleware enforces per-client rate limiting on authenticated requests.
//
// MUST be used after AuthenticationMiddleware (requires authenticated client in context).
// Uses token bucket algorithm via golang.org/x/time/rate. Each client gets independent
// rate limiter based on their client ID.
//
// Configuration:
//   - rps: Requests per second allowed per client
//   - burst: Maximum burst capacity for temporary spikes
//
// Returns:
//   - 429 Too Many Requests: Rate limit exceeded (includes Retry-After header)
//   - Continues: Request allowed within rate limit
func RateLimitMiddleware(rps float64, burst int, logger *slog.Logger) gin.HandlerFunc {
	store := &rateLimiterStore{
		rps:   rps,
		burst: burst,
	}

	// Start cleanup goroutine for stale limiters (every 5 minutes)
	go store.cleanupStale(context.Background(), 5*time.Minute)

	return func(c *gin.Context) {
		// Get authenticated client from context
		client, ok := GetClient(c.Request.Context())
		if !ok || client == nil {
			// Should never happen - authentication middleware should have caught this
			logger.Error("rate limit middleware: no authenticated client in context")
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
			c.Abort()
			return
		}

		// Get or create rate limiter for this client
		limiter := store.getLimiter(client.ID)

		// Check if request is allowed
		if !limiter.Allow() {
			// Calculate retry-after delay
			reservation := limiter.Reserve()
			retryAfter := int(reservation.Delay().Seconds())
			reservation.Cancel() // Cancel the reservation

			logger.Debug("rate limit exceeded",
				slog.String("client_id", client.ID.String()),
				slog.Int("retry_after", retryAfter))

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please retry after the specified delay.",
			})
			c.Abort()
			return
		}

		// Request allowed, continue
		c.Next()
	}
}

// getLimiter retrieves or creates a rate limiter for a client.
func (s *rateLimiterStore) getLimiter(clientID uuid.UUID) *rate.Limiter {
	// Try to load existing limiter
	if val, ok := s.limiters.Load(clientID); ok {
		entry := val.(*rateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now()
		entry.mu.Unlock()
		return entry.limiter
	}

	// Create new limiter
	limiter := rate.NewLimiter(rate.Limit(s.rps), s.burst)
	entry := &rateLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	// Store and return
	s.limiters.Store(clientID, entry)
	return limiter
}

// cleanupStale removes rate limiters that haven't been accessed recently.
// Runs periodically to prevent unbounded memory growth.
func (s *rateLimiterStore) cleanupStale(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Remove limiters not accessed in last hour
			threshold := time.Now().Add(-1 * time.Hour)
			s.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*rateLimiterEntry)
				entry.mu.Lock()
				shouldDelete := entry.lastAccess.Before(threshold)
				entry.mu.Unlock()

				if shouldDelete {
					s.limiters.Delete(key)
				}
				return true
			})
		}
	}
}
