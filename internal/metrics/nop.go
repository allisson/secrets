package metrics

import (
	"context"
	"time"
)

type nopBusinessMetrics struct{}

func (nopBusinessMetrics) RecordOperation(_ context.Context, _, _, _ string) {}
func (nopBusinessMetrics) RecordDuration(_ context.Context, _, _ string, _ time.Duration, _ string) {
}

// NewNopBusinessMetrics returns a BusinessMetrics that discards all recordings.
func NewNopBusinessMetrics() BusinessMetrics { return nopBusinessMetrics{} }
