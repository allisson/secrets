package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

func RunCleanAuditLogs(
	ctx context.Context,
	auditLogUseCase authUseCase.AuditLogUseCase,
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

	logger.Info("cleaning audit logs",
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	// Execute deletion or count operation
	count, err := auditLogUseCase.DeleteOlderThan(ctx, days, dryRun)
	if err != nil {
		return fmt.Errorf("failed to delete audit logs: %w", err)
	}

	// Output result based on format
	if format == "json" {
		outputCleanAuditLogsJSON(writer, count, days, dryRun)
	} else {
		outputCleanAuditLogsText(writer, count, days, dryRun)
	}

	logger.Info("cleanup completed",
		slog.Int64("count", count),
		slog.Int("days", days),
		slog.Bool("dry_run", dryRun),
	)

	return nil
}

// outputCleanAuditLogsText outputs the result in human-readable text format.
func outputCleanAuditLogsText(writer io.Writer, count int64, days int, dryRun bool) {
	if dryRun {
		_, _ = fmt.Fprintf(
			writer,
			"Dry-run mode: Would delete %d audit log(s) older than %d day(s)\n",
			count,
			days,
		)
	} else {
		_, _ = fmt.Fprintf(writer, "Successfully deleted %d audit log(s) older than %d day(s)\n", count, days)
	}
}

// outputCleanAuditLogsJSON outputs the result in JSON format for machine consumption.
func outputCleanAuditLogsJSON(writer io.Writer, count int64, days int, dryRun bool) {
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
