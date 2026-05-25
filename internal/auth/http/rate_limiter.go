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

// rateLimiterEntry holds a rate limiter and last access time for cleanup.
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

// rateLimiterStore holds per-key rate limiters with automatic cleanup.
type rateLimiterStore[K comparable] struct {
	limiters sync.Map
	rps      float64
	burst    int
}

func (s *rateLimiterStore[K]) getLimiter(key K) *rate.Limiter {
	if val, ok := s.limiters.Load(key); ok {
		entry := val.(*rateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now()
		entry.mu.Unlock()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(s.rps), s.burst)
	entry := &rateLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	s.limiters.Store(key, entry)
	return limiter
}

func (s *rateLimiterStore[K]) cleanupStale(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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

// newRateLimitMiddleware builds a gin.HandlerFunc that rate-limits by an
// arbitrary key K extracted from the request.
//
// keyFn returns (key, true) on success; on failure it must write the error
// response and call c.Abort() itself, then return (zero, false).
// logAttr converts the key into a slog.Attr for the "rate limit exceeded" log line.
// logMsg and errMsg customise the debug log and the JSON error body respectively.
func newRateLimitMiddleware[K comparable](
	ctx context.Context,
	rps float64,
	burst int,
	logger *slog.Logger,
	keyFn func(*gin.Context) (K, bool),
	logAttr func(K) slog.Attr,
	logMsg string,
	errMsg string,
) gin.HandlerFunc {
	store := &rateLimiterStore[K]{rps: rps, burst: burst}
	go store.cleanupStale(ctx, 5*time.Minute)

	return func(c *gin.Context) {
		key, ok := keyFn(c)
		if !ok {
			return
		}

		limiter := store.getLimiter(key)
		if !limiter.Allow() {
			reservation := limiter.Reserve()
			retryAfter := int(reservation.Delay().Seconds())
			reservation.Cancel()

			logger.Debug(logMsg, logAttr(key), slog.Int("retry_after", retryAfter))
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": errMsg,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
