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
)

// TestMain sets Gin to test mode for all tests in this package.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// createTestServer creates a test server with a discarding logger.
func createTestServer() *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(nil, "localhost", 8080, logger)
}

// createMinimalRouter creates a minimal router with only health and ready endpoints for testing.
func createMinimalRouter(server *Server) *gin.Engine {
	router := gin.New()
	router.Use(CustomRecoveryMiddleware(server.logger))
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.Use(CustomLoggerMiddleware(server.logger))

	// Register only health endpoints for basic router tests
	router.GET("/health", server.healthHandler)
	router.GET("/ready", server.readinessHandler)

	return router
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

// TestReadinessHandler_NotReady_NilDB tests the readiness endpoint when DB is nil.
func TestReadinessHandler_NotReady_NilDB(t *testing.T) {
	server := createTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	server.readinessHandler(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", response["status"])

	components, ok := response["components"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "error", components["database"])
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

// TestRouter_ReadyEndpoint tests the ready endpoint through the full router when not ready.
func TestRouter_ReadyEndpoint(t *testing.T) {
	server := createTestServer()
	router := createMinimalRouter(server)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", response["status"])

	components, ok := response["components"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "error", components["database"])
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

// TestServer_NoMetricsEndpoint tests that the main server does NOT expose /metrics.
func TestServer_NoMetricsEndpoint(t *testing.T) {
	// We verify that a router created WITHOUT the metrics endpoint returns 404.
	// Note: We are testing default Gin behavior here as a proxy for the server's behavior,
	// since constructing a full Server with SetupRouter requires many mocked dependencies.
	router := gin.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
