// Package http provides HTTP server implementation and request handlers.
package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/metrics"
)

// TestMain sets Gin to test mode for all tests in this package.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// createTestServer creates a test server with a discarding logger.
func createTestServer() *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer("localhost", 8080, logger)
}

// TestHealthHandler tests the health check endpoint handler.
func TestHealthHandler(t *testing.T) {
	server := createTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	server.healthHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// TestReadinessHandler_Ready tests the readiness endpoint when server is ready.
func TestReadinessHandler_Ready(t *testing.T) {
	server := createTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	server.readinessHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}

// TestCustomLoggerMiddleware tests the custom logging middleware.
func TestCustomLoggerMiddleware(t *testing.T) {
	// Create a test logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.Use(CustomLoggerMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test", response["message"])
}

// TestRecoveryMiddleware tests Gin's built-in recovery middleware.
func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(CustomLoggerMiddleware(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// Should not panic - Recovery middleware catches it
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// createMinimalRouter creates a minimal router with only health and ready endpoints for testing.
func createMinimalRouter(server *Server) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.Use(CustomLoggerMiddleware(server.logger))

	// Register only health endpoints for basic router tests
	router.GET("/health", server.healthHandler)
	router.GET("/ready", server.readinessHandler)

	return router
}

// TestRouter_HealthEndpoint tests the health endpoint through the full router.
func TestRouter_HealthEndpoint(t *testing.T) {
	server := createTestServer()
	router := createMinimalRouter(server)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// TestRouter_ReadyEndpoint tests the ready endpoint through the full router.
func TestRouter_ReadyEndpoint(t *testing.T) {
	server := createTestServer()
	router := createMinimalRouter(server)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}

// TestRouter_NotFoundEndpoint tests 404 handling.
func TestRouter_NotFoundEndpoint(t *testing.T) {
	server := createTestServer()
	router := createMinimalRouter(server)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestServer_ShutdownGracefully tests graceful server shutdown.
func TestServer_ShutdownGracefully(t *testing.T) {
	server := createTestServer()

	// Initialize router with minimal setup
	router := createMinimalRouter(server)
	server.router = router

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Verify no startup errors
	select {
	case err := <-errChan:
		t.Fatalf("server startup failed: %v", err)
	default:
		// No error, good
	}
}

// TestRequestIDMiddleware_HeaderPresent verifies X-Request-Id header is present in response.
func TestRequestIDMiddleware_HeaderPresent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify X-Request-Id header is present
	requestID := w.Header().Get("X-Request-Id")
	assert.NotEmpty(t, requestID, "X-Request-Id header should be present")

	// Verify it's a valid UUID
	parsedUUID, err := uuid.Parse(requestID)
	require.NoError(t, err, "X-Request-Id should be a valid UUID")
	assert.NotEqual(t, uuid.Nil, parsedUUID, "X-Request-Id should not be nil UUID")

	_ = logger // Prevent unused variable error
}

// TestRouter_MetricsEndpoint tests the /metrics endpoint when metrics are enabled.
func TestRouter_MetricsEndpoint(t *testing.T) {
	server := createTestServer()

	// Create metrics provider
	provider, err := metrics.NewProvider("test_app")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	// Create router with metrics endpoint
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.Use(CustomLoggerMiddleware(server.logger))

	// Add metrics middleware
	router.Use(metrics.HTTPMetricsMiddleware(provider.MeterProvider(), "test_app"))

	// Add metrics endpoint
	router.GET("/metrics", gin.WrapH(provider.Handler()))

	// Add a test endpoint to generate metrics
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Generate some metrics by calling the test endpoint
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Now request the metrics endpoint
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response is in Prometheus format (contains metric lines)
	body := w.Body.String()
	assert.NotEmpty(t, body, "metrics response should not be empty")

	// Check for expected metric names (OpenTelemetry automatically exposes these)
	assert.Contains(t, body, "test_app_http_requests_total", "should contain HTTP requests counter metric")
	assert.Contains(
		t,
		body,
		"test_app_http_request_duration_seconds",
		"should contain HTTP duration histogram metric",
	)

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "text/plain", "metrics endpoint should return text/plain content type")
}

// TestRouter_MetricsEndpoint_NoAuth tests that /metrics endpoint does not require authentication.
func TestRouter_MetricsEndpoint_NoAuth(t *testing.T) {
	// Create metrics provider
	provider, err := metrics.NewProvider("test_app")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	// Create router with metrics endpoint (no auth middleware)
	router := gin.New()
	router.GET("/metrics", gin.WrapH(provider.Handler()))

	// Request without authentication should succeed
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
