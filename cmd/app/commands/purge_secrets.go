package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
)

// PurgeSecretsResult holds the result of the secret purge operation.
type PurgeSecretsResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the purge result.
func (r *PurgeSecretsResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would delete %d secret(s) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf("Successfully deleted %d secret(s) older than %d day(s)", r.Count, r.Days)
}

// ToJSON returns a JSON representation of the purge result.
func (r *PurgeSecretsResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunPurgeSecrets permanently deletes soft-deleted secrets older than the specified number of days.
// Supports dry-run mode and multiple output formats.
func RunPurgeSecrets(
	ctx context.Context,
	secretUseCase secretsUseCase.SecretUseCase,
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

	logger.Info("purging deleted secrets",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Execute purge operation
	count, err := secretUseCase.PurgeDeleted(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to purge secrets: %w", err)
	}

	// Output result
	result := &PurgeSecretsResult{
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
