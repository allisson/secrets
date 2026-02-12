// Package http provides HTTP server implementation and request handlers.
package http

import (
	"log/slog"
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

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(start)

		logger.Info("http request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", c.Writer.Status()),
			slog.Int("body_size", c.Writer.Size()),
			slog.Duration("duration", duration),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("request_id", requestid.Get(c)),
		)
	}
}
