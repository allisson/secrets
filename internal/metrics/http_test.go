package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success_RecordHTTPMetrics", func(t *testing.T) {
		provider, err := NewProvider("test_app")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, provider.Shutdown(context.Background()))
		}()

		middleware := HTTPMetricsMiddleware(provider.MeterProvider(), "test_app")

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Success_RecordMultipleRequests", func(t *testing.T) {
		provider, err := NewProvider("test_app")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, provider.Shutdown(context.Background()))
		}()

		middleware := HTTPMetricsMiddleware(provider.MeterProvider(), "test_app")

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})
		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
		})

		// Record multiple requests
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Record POST request
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Record error request
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/error", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success_RecordWithPathParams", func(t *testing.T) {
		provider, err := NewProvider("test_app")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, provider.Shutdown(context.Background()))
		}()

		middleware := HTTPMetricsMiddleware(provider.MeterProvider(), "test_app")

		router := gin.New()
		router.Use(middleware)
		router.GET("/users/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
		})

		// Request with different path params should use route pattern
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/users/456", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RoutePattern",
			input:    "/v1/secrets/:path",
			expected: "/v1/secrets/:path",
		},
		{
			name:     "EmptyPath",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "RootPath",
			input:    "/",
			expected: "/",
		},
		{
			name:     "WildcardPath",
			input:    "/v1/secrets/*path",
			expected: "/v1/secrets/*path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
