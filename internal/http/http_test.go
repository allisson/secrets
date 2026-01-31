package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/httputil"
)

func TestMakeJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		body         interface{}
		statusCode   int
		expectedBody string
	}{
		{
			name:         "success response",
			body:         map[string]string{"status": "ok"},
			statusCode:   http.StatusOK,
			expectedBody: `{"status":"ok"}`,
		},
		{
			name:         "error response",
			body:         map[string]string{"error": "something went wrong"},
			statusCode:   http.StatusInternalServerError,
			expectedBody: `{"error":"something went wrong"}`,
		},
		{
			name: "complex object",
			body: map[string]interface{}{
				"id":   1,
				"name": "Test",
				"data": map[string]string{"key": "value"},
			},
			statusCode:   http.StatusOK,
			expectedBody: `{"data":{"key":"value"},"id":1,"name":"Test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httputil.MakeJSONResponse(w, tt.statusCode, tt.body)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestHealthHandler(t *testing.T) {
	handler := HealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestReadinessHandler_Ready(t *testing.T) {
	ctx := context.Background()
	handler := ReadinessHandler(ctx)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}

func TestReadinessHandler_NotReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context to simulate shutdown

	handler := ReadinessHandler(ctx)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not ready", response["status"])
}

func TestLoggingMiddleware(t *testing.T) {
	// Create a test logger
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	// Create a simple handler that returns 200 OK
	simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck,gosec
	})

	// Wrap with logging middleware
	wrapped := LoggingMiddleware(logger)(simpleHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a test logger
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	wrapped := RecoveryMiddleware(logger)(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", response["error"])
}

func TestChainMiddleware(t *testing.T) {
	// Create middleware that adds headers
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-1", "value1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-2", "value2")
			next.ServeHTTP(w, r)
		})
	}

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain middlewares
	chained := ChainMiddleware(middleware1, middleware2)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "value1", w.Header().Get("X-Test-1"))
	assert.Equal(t, "value2", w.Header().Get("X-Test-2"))
}
