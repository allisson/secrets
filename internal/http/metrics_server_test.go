package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/metrics"
)

// TestMetricsServer_Endpoints tests the metrics server endpoints.
func TestMetricsServer_Endpoints(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create metrics provider
	provider, err := metrics.NewProvider("test_app")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	// Create metrics server
	metricsServer := NewDefaultMetricsServer("localhost", 8081, logger, provider)
	require.NotNil(t, metricsServer)

	// Test the handler from metricsServer exactly as it's configured
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsServer.GetHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

// TestMetricsServer_Lifecycle tests the Start and Shutdown methods of MetricsServer.
func TestMetricsServer_Lifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create metrics provider
	provider, err := metrics.NewProvider("test_app")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	// Create metrics server with random port
	metricsServer := NewDefaultMetricsServer("localhost", 0, logger, provider)
	require.NotNil(t, metricsServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := metricsServer.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err = metricsServer.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Verify no startup errors
	select {
	case err := <-errChan:
		t.Fatalf("metrics server startup failed: %v", err)
	default:
		// No error, good
	}
}

// TestMetricsServer_ContextCancellation tests if Start returns when context is cancelled.
func TestMetricsServer_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create metrics server
	metricsServer := NewDefaultMetricsServer("localhost", 0, logger, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- metricsServer.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Start should return nil (after calling Shutdown internally)
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}
