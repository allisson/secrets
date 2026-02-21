package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/allisson/secrets/internal/app"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
)

// RunVerifyAuditLogs verifies cryptographic integrity of audit logs within a time range.
// Validates HMAC-SHA256 signatures against KEK-derived signing keys for tamper detection.
//
// Requirements: Database must be migrated with signature columns and KEK chain loaded.
func RunVerifyAuditLogs(ctx context.Context, startDate, endDate string, format string) error {
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

	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("verifying audit logs",
		slog.Time("start_date", start),
		slog.Time("end_date", end),
	)

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get audit log use case from container
	auditLogUseCase, err := container.AuditLogUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize audit log use case: %w", err)
	}

	// Execute batch verification
	report, err := auditLogUseCase.VerifyBatch(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to verify audit logs: %w", err)
	}

	// Output result based on format
	if format == "json" {
		if err := outputVerifyJSON(report); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	} else {
		outputVerifyText(report, start, end)
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
func outputVerifyText(report *authUseCase.VerificationReport, start, end time.Time) {
	fmt.Printf("Audit Log Integrity Verification\n")
	fmt.Printf("=================================\n\n")
	fmt.Printf(
		"Time Range: %s to %s\n\n",
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)

	fmt.Printf("Total Checked:  %d\n", report.TotalChecked)
	fmt.Printf("Signed:         %d\n", report.SignedCount)
	fmt.Printf("Unsigned:       %d (legacy)\n", report.UnsignedCount)
	fmt.Printf("Valid:          %d\n", report.ValidCount)
	fmt.Printf("Invalid:        %d\n\n", report.InvalidCount)

	switch {
	case report.InvalidCount > 0:
		fmt.Printf("WARNING: %d log(s) failed integrity check!\n\n", report.InvalidCount)
		fmt.Printf("Invalid Log IDs:\n")
		for _, id := range report.InvalidLogs {
			fmt.Printf("  - %s\n", id)
		}
		fmt.Printf("\nStatus: FAILED ❌\n")
	case report.TotalChecked == 0:
		fmt.Printf("Status: No logs found in specified time range\n")
	default:
		fmt.Printf("Status: PASSED ✓\n")
	}
}

// outputVerifyJSON outputs the verification result in JSON format for machine consumption.
func outputVerifyJSON(report *authUseCase.VerificationReport) error {
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

	fmt.Println(string(jsonBytes))
	return nil
}
