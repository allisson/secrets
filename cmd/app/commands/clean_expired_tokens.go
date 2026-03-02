package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// CleanExpiredTokensResult holds the result of the expired token cleanup operation.
type CleanExpiredTokensResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the cleanup result.
func (r *CleanExpiredTokensResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would delete %d expired token(s) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf(
		"Successfully deleted %d expired token(s) older than %d day(s)",
		r.Count,
		r.Days,
	)
}

// ToJSON returns a JSON representation of the cleanup result.
func (r *CleanExpiredTokensResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunCleanExpiredTokens deletes expired tokens older than the specified number of days.
// Supports dry-run mode and multiple output formats.
func RunCleanExpiredTokens(
	ctx context.Context,
	tokenizationUseCase tokenizationUseCase.TokenizationUseCase,
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

	logger.Info("cleaning expired tokens",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Execute deletion or count operation
	count, err := tokenizationUseCase.CleanupExpired(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	// Output result
	result := &CleanExpiredTokensResult{
		Count:  count,
		Days:   days,
		DryRun: dryRun,
	}
	WriteOutput(writer, format, result)

	logger.Info("cleanup completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}
