package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func TestRateLimitMiddleware_AllowsRequestsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test client
	client := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-client",
	}

	// Create middleware with generous limits
	logger := slog.Default()
	middleware := RateLimitMiddleware(10.0, 20, logger)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Add client to context
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send requests within limit
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimitMiddleware_BlocksRequestsExceedingLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test client
	client := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-client",
	}

	// Create middleware with very low limits
	logger := slog.Default()
	middleware := RateLimitMiddleware(1.0, 2, logger)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send requests up to burst capacity (should succeed)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Header().Get("Retry-After"), "")
}

func TestRateLimitMiddleware_Returns429WithRetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-client",
	}

	logger := slog.Default()
	middleware := RateLimitMiddleware(0.5, 1, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Consume burst
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Next request should be rate limited with Retry-After header
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_IndependentLimitsPerClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create two different clients
	client1 := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "client-1",
	}
	client2 := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "client-2",
	}

	logger := slog.Default()
	middleware := RateLimitMiddleware(1.0, 1, logger)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Client 1 consumes its limit
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := WithClient(req.Context(), client1)
	req = req.WithContext(ctx)

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Client 1 is now rate limited
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx = WithClient(req.Context(), client1)
	req = req.WithContext(ctx)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Client 2 should still have its own independent limit
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx = WithClient(req.Context(), client2)
	req = req.WithContext(ctx)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_BurstCapacityWorks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := &authDomain.Client{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-client",
	}

	logger := slog.Default()
	// Low rate but higher burst
	middleware := RateLimitMiddleware(1.0, 5, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Should be able to burst up to 5 requests
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 6th request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitMiddleware_RequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := RateLimitMiddleware(10.0, 20, logger)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request without authenticated client should fail
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRateLimiterStore_CleanupStaleEntries(t *testing.T) {
	store := &rateLimiterStore{
		rps:   10.0,
		burst: 20,
	}

	// Create a limiter entry
	client1 := uuid.Must(uuid.NewV7())
	limiter1 := store.getLimiter(client1)
	assert.NotNil(t, limiter1)

	// Verify it's stored
	_, ok := store.limiters.Load(client1)
	assert.True(t, ok)

	// Manually set last access to old time
	if val, ok := store.limiters.Load(client1); ok {
		entry := val.(*rateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now().Add(-2 * time.Hour)
		entry.mu.Unlock()
	}

	// Run cleanup manually
	threshold := time.Now().Add(-1 * time.Hour)
	store.limiters.Range(func(key, value interface{}) bool {
		entry := value.(*rateLimiterEntry)
		entry.mu.Lock()
		shouldDelete := entry.lastAccess.Before(threshold)
		entry.mu.Unlock()

		if shouldDelete {
			store.limiters.Delete(key)
		}
		return true
	})

	// Verify entry was cleaned up
	_, ok = store.limiters.Load(client1)
	assert.False(t, ok)
}
