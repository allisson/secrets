package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// PurgeTokenizationKeysResult holds the result of the tokenization key purge operation.
type PurgeTokenizationKeysResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the purge result.
func (r *PurgeTokenizationKeysResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would delete %d tokenization key(s) (and associated tokens) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf(
		"Successfully deleted %d tokenization key(s) (and associated tokens) older than %d day(s)",
		r.Count,
		r.Days,
	)
}

// ToJSON returns a JSON representation of the purge result.
func (r *PurgeTokenizationKeysResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunPurgeTokenizationKeys permanently deletes soft-deleted tokenization keys and their tokens older than the specified number of days.
// Supports dry-run mode and multiple output formats.
func RunPurgeTokenizationKeys(
	ctx context.Context,
	tokenizationUseCase tokenizationUseCase.TokenizationKeyUseCase,
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

	logger.Info("purging deleted tokenization keys",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Execute purge operation
	count, err := tokenizationUseCase.PurgeDeleted(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to purge tokenization keys: %w", err)
	}

	// Output result
	result := &PurgeTokenizationKeysResult{
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
