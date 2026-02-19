package http

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// createCORSMiddleware creates a CORS middleware based on configuration.
// Returns nil if CORS is disabled or no origins configured.
//
// CORS is disabled by default since Secrets is designed as a server-to-server API.
// Enable only if browser-based applications require direct API access.
//
// Configuration:
//   - enabled: Whether CORS is enabled
//   - allowOriginsStr: Comma-separated list of allowed origins
//
// Returns nil if disabled or no valid origins are configured.
func createCORSMiddleware(enabled bool, allowOriginsStr string, logger *slog.Logger) gin.HandlerFunc {
	if !enabled {
		return nil
	}

	if allowOriginsStr == "" {
		logger.Warn("CORS enabled but no origins configured - CORS will not be applied")
		return nil
	}

	// Parse comma-separated origins
	origins := parseOrigins(allowOriginsStr)
	if len(origins) == 0 {
		logger.Warn("CORS enabled but no valid origins found")
		return nil
	}

	logger.Info("CORS enabled",
		slog.Int("origin_count", len(origins)),
		slog.Any("origins", origins))

	config := cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
		},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
		},
		ExposeHeaders: []string{
			"X-Request-Id",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}

// parseOrigins parses comma-separated origin list and trims whitespace.
// Returns empty slice if input is empty.
func parseOrigins(originsStr string) []string {
	if originsStr == "" {
		return nil
	}

	parts := strings.Split(originsStr, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return origins
}
