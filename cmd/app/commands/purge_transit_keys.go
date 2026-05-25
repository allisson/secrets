package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/allisson/secrets/internal/metrics"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

// PurgeTransitKeysResult holds the result of the transit key purge operation.
type PurgeTransitKeysResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the purge result.
func (r *PurgeTransitKeysResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would delete %d transit key(s) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf("Successfully deleted %d transit key(s) older than %d day(s)", r.Count, r.Days)
}

// ToJSON returns a JSON representation of the purge result.
func (r *PurgeTransitKeysResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunPurgeTransitKeys permanently deletes soft-deleted transit keys older than the specified number of days.
// Supports dry-run mode and multiple output formats.
func RunPurgeTransitKeys(
	ctx context.Context,
	transitUseCase transitUseCase.TransitKeyUseCase,
	bm metrics.BusinessMetrics,
	logger *slog.Logger,
	writer io.Writer,
	days int,
	dryRun bool,
	format string,
) error {
	// Validate days parameter
	if days < 0 {
		return fmt.Errorf("days must be a positive number, got: %d", days)
	}

	logger.Info("purging deleted transit keys",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Execute purge operation
	var count int64
	if err := metrics.Track(ctx, bm, "transit", "transit_key_purge_deleted", func() error {
		var e error
		count, e = transitUseCase.PurgeDeleted(ctx, days, dryRun)
		return e
	}); err != nil {
		return fmt.Errorf("failed to purge transit keys: %w", err)
	}

	// Output result
	result := &PurgeTransitKeysResult{
		Count:  count,
		Days:   days,
		DryRun: dryRun,
	}
	WriteOutput(writer, format, result)

	logger.Info("purge completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}
