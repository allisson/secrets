package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/allisson/secrets/internal/auth/usecase"
)

// PurgeAuthTokensResult holds the result of the authentication token purge operation.
type PurgeAuthTokensResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the purge result.
func (r *PurgeAuthTokensResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would purge %d expired/revoked authentication token(s) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf(
		"Successfully purged %d expired/revoked authentication token(s) older than %d day(s)",
		r.Count,
		r.Days,
	)
}

// ToJSON returns a JSON representation of the purge result.
func (r *PurgeAuthTokensResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunPurgeAuthTokens deletes expired and revoked authentication tokens older than the specified number of days.
// Supports dry-run mode (if implemented in usecase) and multiple output formats.
func RunPurgeAuthTokens(
	ctx context.Context,
	tokenUseCase usecase.TokenUseCase,
	logger *slog.Logger,
	writer io.Writer,
	days int,
	dryRun bool,
	format string,
) error {
	// Validate days parameter
	if days < 0 {
		return fmt.Errorf("days must be a non-negative number, got: %d", days)
	}

	logger.Info("purging authentication tokens",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Note: dryRun is not yet supported in TokenUseCase.PurgeExpiredAndRevoked
	// For now, we will only proceed if dryRun is false or inform that it's not supported.
	if dryRun {
		result := &PurgeAuthTokensResult{
			Count:  0,
			Days:   days,
			DryRun: dryRun,
		}
		_, _ = fmt.Fprintln(
			writer,
			"Notice: Dry-run is not yet implemented for auth token purging. No tokens were deleted.",
		)
		WriteOutput(writer, format, result)
		return nil
	}

	// Execute purge
	count, err := tokenUseCase.PurgeExpiredAndRevoked(ctx, days)
	if err != nil {
		return fmt.Errorf("failed to purge authentication tokens: %w", err)
	}

	// Output result
	result := &PurgeAuthTokensResult{
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
