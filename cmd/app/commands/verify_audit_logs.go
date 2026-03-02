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

// VerifyAuditLogsResult holds the result of the audit log verification operation.
type VerifyAuditLogsResult struct {
	TotalChecked  int64     `json:"total_checked"`
	SignedCount   int64     `json:"signed_count"`
	UnsignedCount int64     `json:"unsigned_count"`
	ValidCount    int64     `json:"valid_count"`
	InvalidCount  int64     `json:"invalid_count"`
	InvalidLogs   []string  `json:"invalid_logs"`
	Passed        bool      `json:"passed"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

// ToText returns a human-readable representation of the verification result.
func (r *VerifyAuditLogsResult) ToText() string {
	output := "Audit Log Integrity Verification\n"
	output += "=================================\n\n"
	output += fmt.Sprintf(
		"Time Range: %s to %s\n\n",
		r.StartDate.Format("2006-01-02 15:04:05"),
		r.EndDate.Format("2006-01-02 15:04:05"),
	)

	output += fmt.Sprintf("Total Checked:  %d\n", r.TotalChecked)
	output += fmt.Sprintf("Signed:         %d\n", r.SignedCount)
	output += fmt.Sprintf("Unsigned:       %d (legacy)\n", r.UnsignedCount)
	output += fmt.Sprintf("Valid:          %d\n", r.ValidCount)
	output += fmt.Sprintf("Invalid:        %d\n\n", r.InvalidCount)

	switch {
	case r.InvalidCount > 0:
		output += fmt.Sprintf("WARNING: %d log(s) failed integrity check!\n\n", r.InvalidCount)
		output += "Invalid Log IDs:\n"
		for _, id := range r.InvalidLogs {
			output += fmt.Sprintf("  - %s\n", id)
		}
		output += "\nStatus: FAILED ❌"
	case r.TotalChecked == 0:
		output += "Status: No logs found in specified time range"
	default:
		output += "Status: PASSED ✓"
	}
	return output
}

// ToJSON returns a JSON representation of the verification result.
func (r *VerifyAuditLogsResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunVerifyAuditLogs verifies cryptographic integrity of audit logs within a time range.
func RunVerifyAuditLogs(
	ctx context.Context,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
	writer io.Writer,
	startDateStr, endDateStr string,
	format string,
) error {
	// Parse date strings to time.Time
	start, err := parseDate(startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	end, err := parseDate(endDateStr)
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

	// Convert UUIDs to strings
	invalidLogs := make([]string, len(report.InvalidLogs))
	for i, id := range report.InvalidLogs {
		invalidLogs[i] = id.String()
	}

	// Output result
	result := &VerifyAuditLogsResult{
		TotalChecked:  report.TotalChecked,
		SignedCount:   report.SignedCount,
		UnsignedCount: report.UnsignedCount,
		ValidCount:    report.ValidCount,
		InvalidCount:  report.InvalidCount,
		InvalidLogs:   invalidLogs,
		Passed:        report.InvalidCount == 0,
		StartDate:     start,
		EndDate:       end,
	}
	WriteOutput(writer, format, result)

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
