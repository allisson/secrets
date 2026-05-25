package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/allisson/secrets/internal/metrics"
)

// BusinessMetricsMiddleware records operation count and duration for a named route.
// Must be placed before the handler in the middleware chain.
// Uses HTTP response status to determine success (< 400) vs error (>= 400).
func BusinessMetricsMiddleware(bm metrics.BusinessMetrics, domain, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := "success"
		if c.Writer.Status() >= http.StatusBadRequest {
			status = "error"
		}
		bm.RecordOperation(c.Request.Context(), domain, operation, status)
		bm.RecordDuration(c.Request.Context(), domain, operation, time.Since(start), status)
	}
}
