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

// RunCleanAuditLogs deletes audit logs older than the specified number of days.
// Supports dry-run mode to preview deletion count and both text/JSON output formats.
//
// Requirements: Database must be migrated and accessible.
func RunCleanAuditLogs(ctx context.Context, days int, dryRun bool, format string) error {
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
	logger.Info("cleaning audit logs",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get audit log use case from container
	auditLogUseCase, err := container.AuditLogUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize audit log use case: %w", err)
	}

	// Execute deletion or count operation
	count, err := auditLogUseCase.DeleteOlderThan(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to delete audit logs: %w", err)
	}

	// Output result based on format
	if format == "json" {
		outputCleanJSON(count, days, dryRun)
	} else {
		outputCleanText(count, days, dryRun)
	}

	logger.Info("cleanup completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}

// outputCleanText outputs the result in human-readable text format.
func outputCleanText(count int64, days int, dryRun bool) {
	if dryRun {
		fmt.Printf("Dry-run mode: Would delete %d audit log(s) older than %d day(s)\n", count, days)
	} else {
		fmt.Printf("Successfully deleted %d audit log(s) older than %d day(s)\n", count, days)
	}
}

// outputCleanJSON outputs the result in JSON format for machine consumption.
func outputCleanJSON(count int64, days int, dryRun bool) {
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
