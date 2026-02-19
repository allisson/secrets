package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateCORSMiddleware_DisabledReturnsNil(t *testing.T) {
	logger := slog.Default()
	middleware := createCORSMiddleware(false, "https://example.com", logger)
	assert.Nil(t, middleware)
}

func TestCreateCORSMiddleware_EnabledWithoutOriginsReturnsNil(t *testing.T) {
	logger := slog.Default()
	middleware := createCORSMiddleware(true, "", logger)
	assert.Nil(t, middleware)
}

func TestCreateCORSMiddleware_ParsesCommaSeparatedOrigins(t *testing.T) {
	logger := slog.Default()
	middleware := createCORSMiddleware(true, "https://app.example.com,https://admin.example.com", logger)
	assert.NotNil(t, middleware)
}

func TestCreateCORSMiddleware_TrimsWhitespace(t *testing.T) {
	logger := slog.Default()
	middleware := createCORSMiddleware(true, " https://app.example.com , https://admin.example.com ", logger)
	assert.NotNil(t, middleware)
}

func TestParseOrigins_ParsesCommaSeparated(t *testing.T) {
	origins := parseOrigins("https://app.example.com,https://admin.example.com")
	assert.Equal(t, 2, len(origins))
	assert.Equal(t, "https://app.example.com", origins[0])
	assert.Equal(t, "https://admin.example.com", origins[1])
}

func TestParseOrigins_TrimsWhitespace(t *testing.T) {
	origins := parseOrigins(" https://app.example.com , https://admin.example.com ")
	assert.Equal(t, 2, len(origins))
	assert.Equal(t, "https://app.example.com", origins[0])
	assert.Equal(t, "https://admin.example.com", origins[1])
}

func TestParseOrigins_HandlesEmptyString(t *testing.T) {
	origins := parseOrigins("")
	assert.Nil(t, origins)
}

func TestCORSIntegration_HeadersAddedWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := createCORSMiddleware(true, "https://app.example.com", logger)

	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSIntegration_NoHeadersWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := createCORSMiddleware(false, "https://app.example.com", logger)

	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSIntegration_PreflightRequestHandled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := slog.Default()
	middleware := createCORSMiddleware(true, "https://app.example.com", logger)

	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send preflight OPTIONS request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}
