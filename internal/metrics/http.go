package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// httpMetrics holds HTTP-specific metric instruments.
type httpMetrics struct {
	requestCounter metric.Int64Counter
	durationHisto  metric.Float64Histogram
}

// HTTPMetricsMiddleware returns a Gin middleware that records HTTP request metrics.
// Tracks total requests and request durations with method, path, and status_code labels.
// The path is sanitized to route patterns (e.g., /v1/secrets/:path) to prevent high cardinality.
func HTTPMetricsMiddleware(meterProvider metric.MeterProvider, namespace string) gin.HandlerFunc {
	meter := meterProvider.Meter(namespace)

	// Create counter for total HTTP requests
	requestCounter, err := meter.Int64Counter(
		fmt.Sprintf("%s_http_requests_total", namespace),
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		// If metric creation fails, return a no-op middleware
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Create histogram for HTTP request durations
	durationHisto, err := meter.Float64Histogram(
		fmt.Sprintf("%s_http_request_duration_seconds", namespace),
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// If metric creation fails, return a no-op middleware
		return func(c *gin.Context) {
			c.Next()
		}
	}

	metrics := &httpMetrics{
		requestCounter: requestCounter,
		durationHisto:  durationHisto,
	}

	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start)
		method := c.Request.Method
		path := sanitizePath(c.FullPath()) // Use route pattern, not actual path
		statusCode := strconv.Itoa(c.Writer.Status())

		attrs := []attribute.KeyValue{
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.String("status_code", statusCode),
		}

		// Record request count
		metrics.requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))

		// Record request duration
		metrics.durationHisto.Record(c.Request.Context(), duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// sanitizePath converts actual request paths to route patterns for metrics.
// Returns the route pattern if available, otherwise returns the actual path.
// If path is empty (route not matched), returns "unknown".
func sanitizePath(fullPath string) string {
	if fullPath == "" {
		return "unknown"
	}
	return fullPath
}
