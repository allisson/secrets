package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BusinessMetrics defines the interface for recording business operation metrics.
// Implementations track operation counts and durations for observability across
// different business domains (auth, secrets, transit).
type BusinessMetrics interface {
	// RecordOperation records a business operation with its status.
	// Domain examples: "auth", "secrets", "transit"
	// Operation examples: "client_create", "secret_get", "transit_encrypt"
	// Status examples: "success", "error"
	RecordOperation(ctx context.Context, domain, operation, status string)

	// RecordDuration records the duration of a business operation with its status.
	// Duration is recorded in seconds as a histogram for percentile calculations.
	RecordDuration(ctx context.Context, domain, operation string, duration time.Duration, status string)
}

// businessMetrics implements BusinessMetrics using OpenTelemetry metrics.
type businessMetrics struct {
	operationCounter metric.Int64Counter
	durationHisto    metric.Float64Histogram
}

// NewBusinessMetrics creates a new BusinessMetrics implementation using the provided meter provider.
// The namespace parameter is used as a prefix for all metric names (e.g., "secrets").
// Returns error if meters cannot be initialized.
func NewBusinessMetrics(meterProvider metric.MeterProvider, namespace string) (BusinessMetrics, error) {
	meter := meterProvider.Meter(namespace)

	// Create counter for total operations
	operationCounter, err := meter.Int64Counter(
		fmt.Sprintf("%s_operations_total", namespace),
		metric.WithDescription("Total number of business operations"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation counter: %w", err)
	}

	// Create histogram for operation durations
	durationHisto, err := meter.Float64Histogram(
		fmt.Sprintf("%s_operation_duration_seconds", namespace),
		metric.WithDescription("Duration of business operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration histogram: %w", err)
	}

	return &businessMetrics{
		operationCounter: operationCounter,
		durationHisto:    durationHisto,
	}, nil
}

// RecordOperation increments the operation counter with domain, operation, and status labels.
func (b *businessMetrics) RecordOperation(ctx context.Context, domain, operation, status string) {
	b.operationCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("domain", domain),
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)
}

// RecordDuration records the operation duration in seconds with domain, operation, and status labels.
func (b *businessMetrics) RecordDuration(
	ctx context.Context,
	domain, operation string,
	duration time.Duration,
	status string,
) {
	b.durationHisto.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("domain", domain),
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)
}

// NoOpBusinessMetrics is a no-op implementation of BusinessMetrics for when metrics are disabled.
type NoOpBusinessMetrics struct{}

// NewNoOpBusinessMetrics creates a no-op BusinessMetrics implementation.
func NewNoOpBusinessMetrics() BusinessMetrics {
	return &NoOpBusinessMetrics{}
}

// RecordOperation does nothing when metrics are disabled.
func (n *NoOpBusinessMetrics) RecordOperation(ctx context.Context, domain, operation, status string) {
	// No-op
}

// RecordDuration does nothing when metrics are disabled.
func (n *NoOpBusinessMetrics) RecordDuration(
	ctx context.Context,
	domain, operation string,
	duration time.Duration,
	status string,
) {
	// No-op
}
