package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomLoggerMiddleware tests the custom logging middleware.
func TestCustomLoggerMiddleware(t *testing.T) {
	// Create a test logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

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

// TestCustomRecoveryMiddleware tests the custom recovery middleware.
func TestCustomRecoveryMiddleware(t *testing.T) {
	// Create a test logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router := gin.New()
	router.Use(CustomRecoveryMiddleware(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// Should not panic - Recovery middleware catches it
	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRequestIDMiddleware_HeaderPresent verifies X-Request-Id header is present in response.
func TestRequestIDMiddleware_HeaderPresent(t *testing.T) {
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
}

// TestMaxRequestBodySizeMiddleware tests the request body size limit middleware.
func TestMaxRequestBodySizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		limit          int64
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "within limit",
			limit:          10,
			requestBody:    "small",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "at limit",
			limit:          10,
			requestBody:    "0123456789",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "exceeds limit",
			limit:          5,
			requestBody:    "too large",
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "no content length exceeds limit",
			limit:          5,
			requestBody:    "too large",
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(MaxRequestBodySizeMiddleware(tt.limit))
			router.POST("/test", func(c *gin.Context) {
				// We need to read the body to trigger MaxBytesReader
				_, err := io.ReadAll(c.Request.Body)
				if err != nil {
					return
				}
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.requestBody))
			if tt.name == "no content length exceeds limit" {
				req.ContentLength = -1
				req.Header.Del("Content-Length")
			}
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
