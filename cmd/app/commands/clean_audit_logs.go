package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// CleanAuditLogsResult holds the result of the audit log cleanup operation.
type CleanAuditLogsResult struct {
	Count  int64 `json:"count"`
	Days   int   `json:"days"`
	DryRun bool  `json:"dry_run"`
}

// ToText returns a human-readable representation of the cleanup result.
func (r *CleanAuditLogsResult) ToText() string {
	if r.DryRun {
		return fmt.Sprintf(
			"Dry-run mode: Would delete %d audit log(s) older than %d day(s)",
			r.Count,
			r.Days,
		)
	}
	return fmt.Sprintf("Successfully deleted %d audit log(s) older than %d day(s)", r.Count, r.Days)
}

// ToJSON returns a JSON representation of the cleanup result.
func (r *CleanAuditLogsResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunCleanAuditLogs deletes audit logs older than the specified number of days.
// Supports dry-run mode and multiple output formats.
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

	// Output result
	result := &CleanAuditLogsResult{
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
