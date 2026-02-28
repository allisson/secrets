package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// RunVerifyAuditLogs verifies cryptographic integrity of audit logs within a time range.
// Validates HMAC-SHA256 signatures against KEK-derived signing keys for tamper detection.
//
// Requirements: Database must be migrated with signature columns and KEK chain loaded.
func RunVerifyAuditLogs(
	ctx context.Context,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
	writer io.Writer,
	startDate, endDate string,
	format string,
) error {
	// Parse date strings to time.Time
	start, err := parseDate(startDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	end, err := parseDate(endDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	// Validate time range
	if !end.After(start) {
		return fmt.Errorf("end date must be after start date")
	}

	logger.Info("verifying audit logs",
		slog.Time("start_date", start),
		slog.Time("end_date", end),
	)

	// Execute batch verification
	report, err := auditLogUseCase.VerifyBatch(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to verify audit logs: %w", err)
	}

	// Output result based on format
	if format == "json" {
		if err := outputVerifyJSON(writer, report); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	} else {
		outputVerifyText(writer, report, start, end)
	}

	// Log summary
	logger.Info("verification completed",
		slog.Int64("total_checked", report.TotalChecked),
		slog.Int64("valid", report.ValidCount),
		slog.Int64("invalid", report.InvalidCount),
		slog.Int64("unsigned", report.UnsignedCount),
	)

	// Exit with error code if integrity check failed
	if report.InvalidCount > 0 {
		return fmt.Errorf("integrity check failed: %d invalid signature(s)", report.InvalidCount)
	}

	return nil
}

// parseDate parses a date string in format "YYYY-MM-DD" or "YYYY-MM-DD HH:MM:SS" to time.Time.
func parseDate(dateStr string) (time.Time, error) {
	// Try full datetime format first
	t, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err == nil {
		return t, nil
	}

	// Try date-only format (defaults to start of day)
	t, err = time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid date format (expected YYYY-MM-DD or YYYY-MM-DD HH:MM:SS): %s",
			dateStr,
		)
	}

	return t, nil
}

// outputVerifyText outputs the verification result in human-readable text format.
func outputVerifyText(writer io.Writer, report *authUseCase.VerificationReport, start, end time.Time) {
	_, _ = fmt.Fprintf(writer, "Audit Log Integrity Verification\n")
	_, _ = fmt.Fprintf(writer, "=================================\n\n")
	_, _ = fmt.Fprintf(writer,
		"Time Range: %s to %s\n\n",
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)

	_, _ = fmt.Fprintf(writer, "Total Checked:  %d\n", report.TotalChecked)
	_, _ = fmt.Fprintf(writer, "Signed:         %d\n", report.SignedCount)
	_, _ = fmt.Fprintf(writer, "Unsigned:       %d (legacy)\n", report.UnsignedCount)
	_, _ = fmt.Fprintf(writer, "Valid:          %d\n", report.ValidCount)
	_, _ = fmt.Fprintf(writer, "Invalid:        %d\n\n", report.InvalidCount)

	switch {
	case report.InvalidCount > 0:
		_, _ = fmt.Fprintf(writer, "WARNING: %d log(s) failed integrity check!\n\n", report.InvalidCount)
		_, _ = fmt.Fprintf(writer, "Invalid Log IDs:\n")
		for _, id := range report.InvalidLogs {
			_, _ = fmt.Fprintf(writer, "  - %s\n", id)
		}
		_, _ = fmt.Fprintf(writer, "\nStatus: FAILED ❌\n")
	case report.TotalChecked == 0:
		_, _ = fmt.Fprintf(writer, "Status: No logs found in specified time range\n")
	default:
		_, _ = fmt.Fprintf(writer, "Status: PASSED ✓\n")
	}
}

// outputVerifyJSON outputs the verification result in JSON format for machine consumption.
func outputVerifyJSON(writer io.Writer, report *authUseCase.VerificationReport) error {
	result := map[string]interface{}{
		"total_checked":  report.TotalChecked,
		"signed_count":   report.SignedCount,
		"unsigned_count": report.UnsignedCount,
		"valid_count":    report.ValidCount,
		"invalid_count":  report.InvalidCount,
		"invalid_logs":   report.InvalidLogs,
		"passed":         report.InvalidCount == 0,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	_, _ = fmt.Fprintln(writer, string(jsonBytes))
	return nil
}
