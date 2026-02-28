package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

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

	// Output result based on format
	if format == "json" {
		outputCleanExpiredJSON(writer, count, days, dryRun)
	} else {
		outputCleanExpiredText(writer, count, days, dryRun)
	}

	logger.Info("cleanup completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}

// outputCleanExpiredText outputs the result in human-readable text format.
func outputCleanExpiredText(writer io.Writer, count int64, days int, dryRun bool) {
	if dryRun {
		_, _ = fmt.Fprintf(
			writer,
			"Dry-run mode: Would delete %d expired token(s) older than %d day(s)\n",
			count,
			days,
		)
	} else {
		_, _ = fmt.Fprintf(
			writer,
			"Successfully deleted %d expired token(s) older than %d day(s)\n",
			count,
			days,
		)
	}
}

// outputCleanExpiredJSON outputs the result in JSON format for machine consumption.
func outputCleanExpiredJSON(writer io.Writer, count int64, days int, dryRun bool) {
	result := map[string]interface{}{
		"count":   count,
		"days":    days,
		"dry_run": dryRun,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(writer, string(jsonBytes))
}
