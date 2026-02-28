package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertBizMetricLine checks that the Prometheus output contains a business metric
// matching the given name, partial label pattern, and value. Uses regex to handle
// extra OTel scope labels injected by the Prometheus exporter.
func assertBizMetricLine(t *testing.T, output, name, labels, value string) {
	t.Helper()
	pattern := name + `\{[^}]*` + labels + `[^}]*\} ` + value
	assert.Regexp(t, pattern, output)
}

func TestNewBusinessMetrics(t *testing.T) {
	t.Run("Success_CreateBusinessMetrics", func(t *testing.T) {
		provider, err := NewProvider("test_app")
		require.NoError(t, err)

		businessMetrics, err := NewBusinessMetrics(provider.MeterProvider(), "test_app")

		require.NoError(t, err)
		assert.NotNil(t, businessMetrics)
	})
}

func TestBusinessMetrics_RecordOperation(t *testing.T) {
	provider, err := NewProvider("test_app")
	require.NoError(t, err)

	bm, err := NewBusinessMetrics(provider.MeterProvider(), "test_app")
	require.NoError(t, err)

	t.Run("Success_RecordSuccessfulOperation", func(t *testing.T) {
		// Should not panic
		bm.RecordOperation(context.Background(), "auth", "create_client", "success")
	})

	t.Run("Success_RecordFailedOperation", func(t *testing.T) {
		// Should not panic
		bm.RecordOperation(context.Background(), "auth", "create_client", "error")
	})

	t.Run("Success_RecordMultipleDomains", func(t *testing.T) {
		bm.RecordOperation(context.Background(), "auth", "create_client", "success")
		bm.RecordOperation(context.Background(), "secrets", "encrypt", "success")
		bm.RecordOperation(context.Background(), "transit", "rotate_key", "error")
	})
}

func TestBusinessMetrics_RecordDuration(t *testing.T) {
	provider, err := NewProvider("test_app")
	require.NoError(t, err)

	bm, err := NewBusinessMetrics(provider.MeterProvider(), "test_app")
	require.NoError(t, err)

	t.Run("Success_RecordSuccessfulDuration", func(t *testing.T) {
		// Should not panic
		bm.RecordDuration(context.Background(), "auth", "create_client", 123*time.Millisecond, "success")
	})

	t.Run("Success_RecordFailedDuration", func(t *testing.T) {
		// Should not panic
		bm.RecordDuration(context.Background(), "auth", "create_client", 456*time.Millisecond, "error")
	})

	t.Run("Success_RecordMultipleDomains", func(t *testing.T) {
		bm.RecordDuration(context.Background(), "auth", "create_client", 100*time.Millisecond, "success")
		bm.RecordDuration(context.Background(), "secrets", "encrypt", 200*time.Millisecond, "success")
		bm.RecordDuration(context.Background(), "transit", "rotate_key", 300*time.Millisecond, "error")
	})
}

func TestNewNoOpBusinessMetrics(t *testing.T) {
	noOpMetrics := NewNoOpBusinessMetrics()

	assert.NotNil(t, noOpMetrics)
	assert.IsType(t, &NoOpBusinessMetrics{}, noOpMetrics)

	t.Run("NoOp_RecordOperationDoesNotPanic", func(t *testing.T) {
		// Should not panic or do anything
		noOpMetrics.RecordOperation(context.Background(), "auth", "create_client", "success")
		noOpMetrics.RecordOperation(context.Background(), "secrets", "encrypt", "error")
	})

	t.Run("NoOp_RecordDurationDoesNotPanic", func(t *testing.T) {
		// Should not panic or do anything
		noOpMetrics.RecordDuration(
			context.Background(),
			"auth",
			"create_client",
			100*time.Millisecond,
			"success",
		)
		noOpMetrics.RecordDuration(context.Background(), "secrets", "encrypt", 200*time.Millisecond, "error")
	})
}

func TestBusinessMetrics_Integration(t *testing.T) {
	provider, err := NewProvider("integration_test")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	bm, err := NewBusinessMetrics(provider.MeterProvider(), "integration_test")
	require.NoError(t, err)

	// Record various operations
	ctx := context.Background()

	// Record operation counts
	bm.RecordOperation(ctx, "auth", "create_client", "success")
	bm.RecordOperation(ctx, "auth", "create_client", "success")
	bm.RecordOperation(ctx, "auth", "create_client", "error")
	bm.RecordOperation(ctx, "secrets", "encrypt", "success")
	bm.RecordOperation(ctx, "secrets", "decrypt", "success")
	bm.RecordOperation(ctx, "transit", "rotate_key", "success")

	// Record operation durations
	bm.RecordDuration(ctx, "auth", "create_client", 50*time.Millisecond, "success")
	bm.RecordDuration(ctx, "auth", "create_client", 60*time.Millisecond, "success")
	bm.RecordDuration(ctx, "auth", "create_client", 100*time.Millisecond, "error")
	bm.RecordDuration(ctx, "secrets", "encrypt", 10*time.Millisecond, "success")
	bm.RecordDuration(ctx, "secrets", "decrypt", 20*time.Millisecond, "success")
	bm.RecordDuration(ctx, "transit", "rotate_key", 150*time.Millisecond, "success")

	// Metrics should be recorded without errors
	// Verify metrics in Prometheus registry
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	provider.Handler().ServeHTTP(w, req)

	output := w.Body.String()

	// Check operation counts
	assertBizMetricLine(
		t,
		output,
		`integration_test_operations_total`,
		`domain="auth".*operation="create_client".*status="success"`,
		`2`,
	)
	assertBizMetricLine(
		t,
		output,
		`integration_test_operations_total`,
		`domain="auth".*operation="create_client".*status="error"`,
		`1`,
	)
	assertBizMetricLine(
		t,
		output,
		`integration_test_operations_total`,
		`domain="secrets".*operation="encrypt".*status="success"`,
		`1`,
	)

	// Check durations (existence)
	assertBizMetricLine(
		t,
		output,
		`integration_test_operation_duration_seconds_count`,
		`domain="auth".*operation="create_client".*status="success"`,
		`2`,
	)
	assertBizMetricLine(
		t,
		output,
		`integration_test_operation_duration_seconds_sum`,
		`domain="auth".*operation="create_client".*status="success"`,
		``,
	)
}
