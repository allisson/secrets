// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
)

// auditLogUseCase implements AuditLogUseCase interface for recording and verifying audit logs.
// Provides cryptographic signing with HMAC-SHA256 for tamper detection.
type auditLogUseCase struct {
	auditLogRepo AuditLogRepository
	keySigner    keyring.KeySigner
}

// Create records an audit log entry for an authenticated operation. Generates a unique
// UUIDv7 identifier and timestamp, then signs the log with HMAC-SHA256 using the active KEK
// if KekChain and AuditSigner are available. For legacy/testing scenarios without signing,
// creates unsigned audit logs. The metadata parameter is optional and can be nil.
func (a *auditLogUseCase) Create(
	ctx context.Context,
	requestID uuid.UUID,
	clientID uuid.UUID,
	capability authDomain.Capability,
	path string,
	metadata map[string]any,
) error {
	// Create the audit log entity
	// Truncate timestamp to microsecond precision to match database storage (PostgreSQL TIMESTAMPTZ
	// ). This ensures the signature matches the value
	// retrieved from the database during verification.
	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  requestID,
		ClientID:   clientID,
		Capability: capability,
		Path:       path,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		IsSigned:   false, // Default to unsigned
	}

	// Sign the audit log if a KeySigner is available
	if a.keySigner != nil {
		canonical, err := auditLog.Canonical()
		if err != nil {
			return apperrors.Wrap(err, "failed to canonicalize audit log for signing")
		}

		signature, kekID, err := a.keySigner.SignWithKey(canonical)
		if err != nil {
			return apperrors.Wrap(err, "failed to sign audit log")
		}

		auditLog.Signature = signature
		auditLog.KekID = &kekID
		auditLog.IsSigned = true
	}

	// Persist the audit log (signed or unsigned)
	if err := a.auditLogRepo.Create(ctx, auditLog); err != nil {
		return apperrors.Wrap(err, "failed to create audit log")
	}

	return nil
}

// ListCursor retrieves audit logs ordered by created_at descending (newest first) with cursor-based pagination
// and optional time-based filtering. If afterID is provided, returns logs with ID greater than afterID (UUIDv7 ordering).
// Accepts createdAtFrom and createdAtTo as optional filters (nil means no filter). Both boundaries are inclusive (>= and <=).
// Accepts clientID as an optional filter (nil means no filter).
// All timestamps are expected in UTC. Returns empty slice if no audit logs found. Limit is pre-validated (1-1000).
func (a *auditLogUseCase) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
	createdAtFrom, createdAtTo *time.Time,
	clientID *uuid.UUID,
) ([]*authDomain.AuditLog, error) {
	auditLogs, err := a.auditLogRepo.ListCursor(ctx, afterID, limit, createdAtFrom, createdAtTo, clientID)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list audit logs with cursor")
	}

	return auditLogs, nil
}

// DeleteOlderThan removes audit logs older than the specified number of days.
// When dryRun is true, returns count without deletion. When false, executes DELETE
// and returns affected rows. Calculates the cutoff timestamp as current UTC time
// minus the given days. All time calculations use UTC.
func (a *auditLogUseCase) DeleteOlderThan(ctx context.Context, days int, dryRun bool) (int64, error) {
	// Calculate cutoff date in UTC
	cutoffDate := time.Now().UTC().AddDate(0, 0, -days)

	// Delete audit logs older than cutoff date (or count if dry-run)
	count, err := a.auditLogRepo.DeleteOlderThan(ctx, cutoffDate, dryRun)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to delete old audit logs")
	}

	return count, nil
}

// VerifyIntegrity verifies the cryptographic signature of a specific audit log.
// Retrieves the log from the repository and validates its HMAC-SHA256 signature
// using the KEK referenced by log.KekID. Returns nil if valid, error otherwise.
func (a *auditLogUseCase) VerifyIntegrity(ctx context.Context, id uuid.UUID) error {
	// Retrieve audit log from repository
	auditLog, err := a.auditLogRepo.Get(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "failed to retrieve audit log")
	}

	// Check if legacy unsigned log
	if !auditLog.IsSigned || auditLog.KekID == nil {
		return authDomain.ErrSignatureMissing
	}

	canonical, err := auditLog.Canonical()
	if err != nil {
		return apperrors.Wrap(err, "failed to canonicalize audit log")
	}

	if err := a.keySigner.VerifyWithKey(*auditLog.KekID, canonical, auditLog.Signature); err != nil {
		if errors.Is(err, keyring.ErrKekNotFound) {
			return authDomain.ErrKekNotFoundForLog
		}
		return apperrors.Wrap(authDomain.ErrSignatureInvalid, "audit log signature verification failed")
	}

	return nil
}

// VerifyBatch performs batch verification of audit logs within a time range.
// Returns a detailed report with total checked, signed/unsigned counts, valid/invalid
// counts, and IDs of invalid logs. Processes logs in batches of 1000 for efficiency.
func (a *auditLogUseCase) VerifyBatch(
	ctx context.Context,
	startTime, endTime time.Time,
) (*VerificationReport, error) {
	report := &VerificationReport{
		InvalidLogs: []uuid.UUID{},
	}

	// Paginate through logs in batches using cursor-based pagination
	const pageSize = 1000
	var afterID *uuid.UUID

	for {
		// Retrieve logs in time range
		logs, err := a.auditLogRepo.ListCursor(ctx, afterID, pageSize, &startTime, &endTime, nil)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to list audit logs")
		}

		if len(logs) == 0 {
			break
		}

		// Verify each log in batch
		for _, log := range logs {
			report.TotalChecked++

			// Check if signed
			if !log.IsSigned || log.KekID == nil {
				report.UnsignedCount++
				continue
			}

			report.SignedCount++

			canonical, err := log.Canonical()
			if err != nil {
				report.InvalidCount++
				report.InvalidLogs = append(report.InvalidLogs, log.ID)
				continue
			}

			if err := a.keySigner.VerifyWithKey(*log.KekID, canonical, log.Signature); err != nil {
				report.InvalidCount++
				report.InvalidLogs = append(report.InvalidLogs, log.ID)
				continue
			}

			report.ValidCount++
		}

		// Check if we have more pages
		// If we got fewer items than requested, we've reached the end
		if len(logs) < pageSize {
			break
		}

		// Set cursor to last log's ID for next page
		lastLog := logs[len(logs)-1]
		afterID = &lastLog.ID
	}

	return report, nil
}

// NewAuditLogUseCase creates a new AuditLogUseCase with the provided dependencies.
// Pass nil for keySigner to create unsigned audit logs (legacy / testing mode).
func NewAuditLogUseCase(
	auditLogRepo AuditLogRepository,
	keySigner keyring.KeySigner,
) AuditLogUseCase {
	return &auditLogUseCase{
		auditLogRepo: auditLogRepo,
		keySigner:    keySigner,
	}
}
