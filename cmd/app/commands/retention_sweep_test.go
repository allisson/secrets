package commands

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/metrics"
)

// recordingMetrics captures the (domain, operation, status) labels passed to
// RecordOperation so tests can assert that a sweep was tracked.
type recordingMetrics struct {
	operations []recordedOp
}

type recordedOp struct {
	domain, operation, status string
}

func (m *recordingMetrics) RecordOperation(_ context.Context, domain, operation, status string) {
	m.operations = append(m.operations, recordedOp{domain, operation, status})
}

func (m *recordingMetrics) RecordDuration(_ context.Context, _, _ string, _ time.Duration, _ string) {
}

func deleteSecretsSpec(sweep func(context.Context, int, bool) (int64, error)) SweepSpec {
	return SweepSpec{
		Verb:           "delete",
		VerbPast:       "deleted",
		Subject:        "secret(s)",
		MetricModule:   "secrets",
		MetricOp:       "secret_purge_deleted",
		SupportsDryRun: true,
		Sweep:          sweep,
	}
}

func TestRunRetentionSweep(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	days := 30

	t.Run("text-output", func(t *testing.T) {
		spec := deleteSecretsSpec(func(_ context.Context, _ int, _ bool) (int64, error) {
			return 100, nil
		})

		var out bytes.Buffer
		err := RunRetentionSweep(
			ctx,
			spec,
			metrics.NewNopBusinessMetrics(),
			logger,
			&out,
			days,
			false,
			"text",
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 100 secret(s) older than 30 day(s)")
	})

	t.Run("text-output-dry-run", func(t *testing.T) {
		spec := deleteSecretsSpec(func(_ context.Context, _ int, dryRun bool) (int64, error) {
			require.True(t, dryRun)
			return 75, nil
		})

		var out bytes.Buffer
		err := RunRetentionSweep(ctx, spec, metrics.NewNopBusinessMetrics(), logger, &out, days, true, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Dry-run mode: Would delete 75 secret(s) older than 30 day(s)")
	})

	t.Run("json-output", func(t *testing.T) {
		spec := deleteSecretsSpec(func(_ context.Context, _ int, _ bool) (int64, error) {
			return 50, nil
		})

		var out bytes.Buffer
		err := RunRetentionSweep(ctx, spec, metrics.NewNopBusinessMetrics(), logger, &out, days, true, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 50`)
		require.Contains(t, out.String(), `"days": 30`)
		require.Contains(t, out.String(), `"dry_run": true`)
	})

	t.Run("custom-verb-and-subject", func(t *testing.T) {
		// The auth-token sweep purges rather than deletes and carries a richer subject.
		spec := SweepSpec{
			Verb:           "purge",
			VerbPast:       "purged",
			Subject:        "expired/revoked authentication token(s)",
			MetricModule:   "auth",
			MetricOp:       "token_purge",
			SupportsDryRun: false,
			Sweep: func(_ context.Context, _ int, _ bool) (int64, error) {
				return 7, nil
			},
		}

		var out bytes.Buffer
		err := RunRetentionSweep(
			ctx,
			spec,
			metrics.NewNopBusinessMetrics(),
			logger,
			&out,
			days,
			false,
			"text",
		)

		require.NoError(t, err)
		require.Contains(t,
			out.String(),
			"Successfully purged 7 expired/revoked authentication token(s) older than 30 day(s)",
		)
	})

	t.Run("invalid-days-negative", func(t *testing.T) {
		called := false
		spec := deleteSecretsSpec(func(_ context.Context, _ int, _ bool) (int64, error) {
			called = true
			return 0, nil
		})

		err := RunRetentionSweep(
			ctx,
			spec,
			metrics.NewNopBusinessMetrics(),
			logger,
			&bytes.Buffer{},
			-1,
			false,
			"text",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "days must be a non-negative number")
		require.False(t, called, "sweep must not run for invalid days")
	})

	t.Run("zero-days-allowed", func(t *testing.T) {
		spec := deleteSecretsSpec(func(_ context.Context, d int, _ bool) (int64, error) {
			require.Equal(t, 0, d)
			return 10, nil
		})

		var out bytes.Buffer
		err := RunRetentionSweep(ctx, spec, metrics.NewNopBusinessMetrics(), logger, &out, 0, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 10 secret(s) older than 0 day(s)")
	})

	t.Run("sweep-error-wrapped", func(t *testing.T) {
		spec := deleteSecretsSpec(func(_ context.Context, _ int, _ bool) (int64, error) {
			return 0, errors.New("db down")
		})

		err := RunRetentionSweep(
			ctx,
			spec,
			metrics.NewNopBusinessMetrics(),
			logger,
			&bytes.Buffer{},
			days,
			false,
			"text",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to delete secret(s)")
		require.Contains(t, err.Error(), "db down")
	})

	t.Run("dry-run-unsupported-emits-notice-and-skips", func(t *testing.T) {
		called := false
		spec := SweepSpec{
			Verb:           "purge",
			VerbPast:       "purged",
			Subject:        "expired/revoked authentication token(s)",
			MetricModule:   "auth",
			MetricOp:       "token_purge",
			SupportsDryRun: false,
			Sweep: func(_ context.Context, _ int, _ bool) (int64, error) {
				called = true
				return 99, nil
			},
		}

		var out bytes.Buffer
		err := RunRetentionSweep(ctx, spec, metrics.NewNopBusinessMetrics(), logger, &out, days, true, "text")

		require.NoError(t, err)
		require.False(t, called, "sweep must not run on an unsupported dry run")
		require.Contains(
			t,
			out.String(),
			"Dry-run is not supported for expired/revoked authentication token(s)",
		)
		require.Contains(t, out.String(), "Would purge 0 expired/revoked authentication token(s)")
	})

	t.Run("every-sweep-is-tracked", func(t *testing.T) {
		// Includes audit-log cleanup, which previously ran without metrics.
		rec := &recordingMetrics{}
		spec := SweepSpec{
			Verb:           "delete",
			VerbPast:       "deleted",
			Subject:        "audit log(s)",
			MetricModule:   "auth",
			MetricOp:       "audit_log_clean",
			SupportsDryRun: true,
			Sweep: func(_ context.Context, _ int, _ bool) (int64, error) {
				return 3, nil
			},
		}

		err := RunRetentionSweep(ctx, spec, rec, logger, &bytes.Buffer{}, days, false, "text")

		require.NoError(t, err)
		require.Equal(t, []recordedOp{{"auth", "audit_log_clean", "success"}}, rec.operations)
	})
}

var _ metrics.BusinessMetrics = (*recordingMetrics)(nil)
