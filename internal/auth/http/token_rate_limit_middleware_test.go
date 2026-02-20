package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTokenRateLimitMiddleware_AllowsRequestsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create middleware with generous limits
	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(10.0, 20, logger)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send requests within limit from same IP
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/token", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestTokenRateLimitMiddleware_BlocksRequestsExceedingLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create middleware with very low limits
	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(1.0, 2, logger)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send requests up to burst capacity (should succeed)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/token", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestTokenRateLimitMiddleware_Returns429WithRetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(0.5, 1, logger)

	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Consume burst
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Next request should be rate limited with Retry-After header
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))

	// Verify error message
	assert.Contains(t, w.Body.String(), "rate_limit_exceeded")
	assert.Contains(t, w.Body.String(), "Too many token requests from this IP")
}

func TestTokenRateLimitMiddleware_IndependentLimitsPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(1.0, 1, logger)

	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// IP 1 consumes its limit
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// IP 1 is now rate limited
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", nil)
	req.RemoteAddr = "192.168.1.100:12346" // Different port, same IP
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// IP 2 should still have its own independent limit
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", nil)
	req.RemoteAddr = "192.168.1.101:12345" // Different IP
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTokenRateLimitMiddleware_BurstCapacityWorks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	// Low rate but higher burst
	middleware := TokenRateLimitMiddleware(1.0, 5, logger)

	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Should be able to burst up to 5 requests
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/token", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 6th request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestTokenRateLimitMiddleware_NoAuthenticationRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(10.0, 20, logger)

	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request without any authentication context should work
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	router.ServeHTTP(w, req)

	// Should succeed (no authentication required)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTokenRateLimiterStore_CleanupStaleEntries(t *testing.T) {
	store := &tokenRateLimiterStore{
		rps:   10.0,
		burst: 20,
	}

	// Create a limiter entry
	ip1 := "192.168.1.100"
	limiter1 := store.getLimiter(ip1)
	assert.NotNil(t, limiter1)

	// Verify it's stored
	_, ok := store.limiters.Load(ip1)
	assert.True(t, ok)

	// Manually set last access to old time
	if val, ok := store.limiters.Load(ip1); ok {
		entry := val.(*tokenRateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now().Add(-2 * time.Hour)
		entry.mu.Unlock()
	}

	// Run cleanup manually
	threshold := time.Now().Add(-1 * time.Hour)
	store.limiters.Range(func(key, value interface{}) bool {
		entry := value.(*tokenRateLimiterEntry)
		entry.mu.Lock()
		shouldDelete := entry.lastAccess.Before(threshold)
		entry.mu.Unlock()

		if shouldDelete {
			store.limiters.Delete(key)
		}
		return true
	})

	// Verify entry was cleaned up
	_, ok = store.limiters.Load(ip1)
	assert.False(t, ok)
}

func TestTokenRateLimitMiddleware_HandlesXForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := TokenRateLimitMiddleware(1.0, 1, logger)

	router := gin.New()
	router.Use(middleware)
	router.POST("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First request with X-Forwarded-For header
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request from same IP in X-Forwarded-For should be rate limited
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Request from different IP in X-Forwarded-For should succeed
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTokenRateLimitMiddleware_RespectsConfiguredLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		rps               float64
		burst             int
		requestsToSend    int
		expectedSuccesses int
	}{
		{
			name:              "Conservative limits",
			rps:               3.0,
			burst:             5,
			requestsToSend:    10,
			expectedSuccesses: 5,
		},
		{
			name:              "Moderate limits",
			rps:               5.0,
			burst:             10,
			requestsToSend:    15,
			expectedSuccesses: 10,
		},
		{
			name:              "Permissive limits",
			rps:               10.0,
			burst:             20,
			requestsToSend:    25,
			expectedSuccesses: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.Default()
			middleware := TokenRateLimitMiddleware(tt.rps, tt.burst, logger)

			router := gin.New()
			router.Use(middleware)
			router.POST("/token", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			successes := 0
			for i := 0; i < tt.requestsToSend; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/token", nil)
				req.RemoteAddr = "192.168.1.50:12345" // Use unique IP for this test
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					successes++
				}
			}

			assert.Equal(t, tt.expectedSuccesses, successes,
				"Expected %d successes but got %d", tt.expectedSuccesses, successes)
		})
	}
}
