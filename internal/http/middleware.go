// Package http provides HTTP server implementation and request handlers.
package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// CustomLoggerMiddleware provides structured logging using slog.
// This replaces Gin's default logger to maintain consistency with
// the application's existing logging patterns.
func CustomLoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Get request body size
		requestSize := c.Request.ContentLength

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(start)

		logger.Info("http request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("request_size", requestSize),
			slog.Int("response_size", c.Writer.Size()),
			slog.Duration("duration", duration),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("request_id", requestid.Get(c)),
		)
	}
}

// CustomRecoveryMiddleware provides panic recovery using slog.
// This replaces Gin's default recovery middleware to provide
// structured error logs for critical failures.
func CustomRecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					slog.Any("error", err),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("request_id", requestid.Get(c)),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
