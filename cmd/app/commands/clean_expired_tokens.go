package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunCleanExpiredTokens deletes expired tokens older than the specified number of days.
// Supports dry-run mode to preview deletion count and both text/JSON output formats.
//
// Requirements: Database must be migrated and accessible.
func RunCleanExpiredTokens(ctx context.Context, days int, dryRun bool, format string) error {
	// Validate days parameter
	if days < 0 {
		return fmt.Errorf("days must be a positive number, got: %d", days)
	}

	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("cleaning expired tokens",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get tokenization use case from container
	tokenizationUseCase, err := container.TokenizationUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize tokenization use case: %w", err)
	}

	// Execute deletion or count operation
	count, err := tokenizationUseCase.CleanupExpired(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	// Output result based on format
	if format == "json" {
		outputCleanExpiredJSON(count, days, dryRun)
	} else {
		outputCleanExpiredText(count, days, dryRun)
	}

	logger.Info("cleanup completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}

// outputCleanExpiredText outputs the result in human-readable text format.
func outputCleanExpiredText(count int64, days int, dryRun bool) {
	if dryRun {
		fmt.Printf("Dry-run mode: Would delete %d expired token(s) older than %d day(s)\n", count, days)
	} else {
		fmt.Printf("Successfully deleted %d expired token(s) older than %d day(s)\n", count, days)
	}
}

// outputCleanExpiredJSON outputs the result in JSON format for machine consumption.
func outputCleanExpiredJSON(count int64, days int, dryRun bool) {
	result := map[string]interface{}{
		"count":   count,
		"days":    days,
		"dry_run": dryRun,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonBytes))
}
