package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/allisson/secrets/internal/metrics"
)

// SweepSpec describes a single retention sweep: a CLI command that deletes
// rows older than a --days threshold. See the "Retention sweep" entry in
// CONTEXT.md. The six sweep commands differ only in the fields below; the
// shared validate -> log -> track -> output machinery lives in
// RunRetentionSweep.
type SweepSpec struct {
	// Verb and VerbPast are the action words used in output, e.g. "delete"
	// and "deleted", or "purge" and "purged".
	Verb     string
	VerbPast string
	// Subject is the noun phrase describing what is swept, e.g. "secret(s)"
	// or "expired/revoked authentication token(s)".
	Subject string
	// MetricModule and MetricOp are the metrics.Track labels for the sweep.
	MetricModule string
	MetricOp     string
	// SupportsDryRun is false only for sweeps whose underlying usecase cannot
	// count without deleting (auth-token purge). When false and a dry run is
	// requested, the sweep emits a notice and deletes nothing.
	SupportsDryRun bool
	// Sweep deletes rows older than days and returns the affected count.
	// dryRun is honored only when SupportsDryRun is true.
	Sweep func(ctx context.Context, days int, dryRun bool) (int64, error)
}

// SweepResult holds the outcome of a retention sweep. The verb/subject fields
// are unexported so JSON output stays {count, days, dry_run}.
type SweepResult struct {
	verb     string
	verbPast string
	subject  string

	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the sweep result.
func (r *SweepResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would %s %d %s older than %d day(s)",
			r.verb,
			r.Count,
			r.subject,
			r.Days,
		)
	}
	return fmt.Sprintf(
		"Successfully %s %d %s older than %d day(s)",
		r.verbPast,
		r.Count,
		r.subject,
		r.Days,
	)
}

// ToJSON returns a JSON representation of the sweep result.
func (r *SweepResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunRetentionSweep executes a retention sweep described by spec: it validates
// days, tracks the sweep under the spec's metric labels, and writes the result
// in the requested format. A dry run against a spec that does not support one
// emits a notice and deletes nothing.
func RunRetentionSweep(
	ctx context.Context,
	spec SweepSpec,
	bm metrics.BusinessMetrics,
	logger *slog.Logger,
	writer io.Writer,
	days int,
	dryRun bool,
	format string,
) error {
	if days < 0 {
		return fmt.Errorf("days must be a non-negative number, got: %d", days)
	}

	logger.Info("running retention sweep",
		slog.String("subject", spec.Subject),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	if dryRun && !spec.SupportsDryRun {
		_, _ = fmt.Fprintf(
			writer,
			"Notice: Dry-run is not supported for %s. No rows were deleted.\n",
			spec.Subject,
		)
		WriteOutput(writer, format, &SweepResult{
			verb:     spec.Verb,
			verbPast: spec.VerbPast,
			subject:  spec.Subject,
			Days:     days,
			DryRun:   dryRun,
		})
		return nil
	}

	var count int64
	if err := metrics.Track(ctx, bm, spec.MetricModule, spec.MetricOp, func() error {
		var e error
		count, e = spec.Sweep(ctx, days, dryRun)
		return e
	}); err != nil {
		return fmt.Errorf("failed to %s %s: %w", spec.Verb, spec.Subject, err)
	}

	WriteOutput(writer, format, &SweepResult{
		verb:     spec.Verb,
		verbPast: spec.VerbPast,
		subject:  spec.Subject,
		Count:    count,
		Days:     days,
		DryRun:   dryRun,
	})

	logger.Info("retention sweep completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}
