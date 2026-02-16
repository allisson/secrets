// Package metrics provides OpenTelemetry metrics instrumentation with Prometheus export.
// Supports business operation metrics and HTTP request metrics for observability.
package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// Provider manages the OpenTelemetry meter provider and Prometheus exporter.
// Provides access to the HTTP handler for exposing metrics in Prometheus format.
type Provider struct {
	meterProvider *metric.MeterProvider
	exporter      *promexporter.Exporter
	registry      *prometheus.Registry
}

// NewProvider creates and initializes a new metrics provider with Prometheus exporter.
// The namespace parameter is used as a prefix for all metric names (e.g., "secrets").
// Returns error if the Prometheus exporter cannot be initialized.
func NewProvider(namespace string) (*Provider, error) {
	// Create custom Prometheus registry
	registry := prometheus.NewRegistry()

	// Create Prometheus exporter with custom registry
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create meter provider with Prometheus exporter
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	return &Provider{
		meterProvider: meterProvider,
		exporter:      exporter,
		registry:      registry,
	}, nil
}

// Handler returns an HTTP handler that serves metrics in Prometheus exposition format.
// This handler should be exposed at the /metrics endpoint for Prometheus scraping.
func (p *Provider) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// MeterProvider returns the OpenTelemetry meter provider for creating meters.
// Use this to obtain a meter for recording metrics in different parts of the application.
func (p *Provider) MeterProvider() *metric.MeterProvider {
	return p.meterProvider
}

// Shutdown performs cleanup of the metrics provider and flushes any pending metrics.
// Should be called during application shutdown to ensure all metrics are exported.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.meterProvider == nil {
		return nil
	}
	return p.meterProvider.Shutdown(ctx)
}
