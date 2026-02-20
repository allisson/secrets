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
	"golang.org/x/time/rate"
)

// tokenRateLimiterStore holds per-IP rate limiters with automatic cleanup.
type tokenRateLimiterStore struct {
	limiters sync.Map // map[string]*tokenRateLimiterEntry (IP -> limiter)
	rps      float64
	burst    int
}

// tokenRateLimiterEntry holds a rate limiter and last access time for cleanup.
type tokenRateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

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
//   - rps: Requests per second allowed per IP address
//   - burst: Maximum burst capacity for temporary spikes
//
// Returns:
//   - 429 Too Many Requests: Rate limit exceeded (includes Retry-After header)
//   - Continues: Request allowed within rate limit
func TokenRateLimitMiddleware(rps float64, burst int, logger *slog.Logger) gin.HandlerFunc {
	store := &tokenRateLimiterStore{
		rps:   rps,
		burst: burst,
	}

	// Start cleanup goroutine for stale limiters (every 5 minutes)
	go store.cleanupStale(context.Background(), 5*time.Minute)

	return func(c *gin.Context) {
		// Get client IP address
		clientIP := c.ClientIP()

		// Get or create rate limiter for this IP
		limiter := store.getLimiter(clientIP)

		// Check if request is allowed
		if !limiter.Allow() {
			// Calculate retry-after delay
			reservation := limiter.Reserve()
			retryAfter := int(reservation.Delay().Seconds())
			reservation.Cancel() // Cancel the reservation

			logger.Debug("token rate limit exceeded",
				slog.String("client_ip", clientIP),
				slog.Int("retry_after", retryAfter))

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many token requests from this IP. Please retry after the specified delay.",
			})
			c.Abort()
			return
		}

		// Request allowed, continue
		c.Next()
	}
}

// getLimiter retrieves or creates a rate limiter for an IP address.
func (s *tokenRateLimiterStore) getLimiter(ip string) *rate.Limiter {
	// Try to load existing limiter
	if val, ok := s.limiters.Load(ip); ok {
		entry := val.(*tokenRateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now()
		entry.mu.Unlock()
		return entry.limiter
	}

	// Create new limiter
	limiter := rate.NewLimiter(rate.Limit(s.rps), s.burst)
	entry := &tokenRateLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	// Store and return
	s.limiters.Store(ip, entry)
	return limiter
}

// cleanupStale removes rate limiters that haven't been accessed recently.
// Runs periodically to prevent unbounded memory growth from IP address churn.
func (s *tokenRateLimiterStore) cleanupStale(ctx context.Context, interval time.Duration) {
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
				entry := value.(*tokenRateLimiterEntry)
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
