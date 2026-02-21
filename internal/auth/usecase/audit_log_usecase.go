// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// auditLogUseCase implements AuditLogUseCase interface for recording and verifying audit logs.
// Provides cryptographic signing with HMAC-SHA256 for tamper detection (PCI DSS Requirement 10.2.2).
type auditLogUseCase struct {
	auditLogRepo AuditLogRepository
	auditSigner  authService.AuditSigner
	kekChain     *cryptoDomain.KekChain
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
	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  requestID,
		ClientID:   clientID,
		Capability: capability,
		Path:       path,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
		IsSigned:   false, // Default to unsigned
	}

	// Sign the audit log if KekChain and AuditSigner are available
	if a.kekChain != nil && a.auditSigner != nil {
		// Get active KEK ID from chain
		activeKekID := a.kekChain.ActiveKekID()

		// Retrieve active KEK for signing
		kek, ok := a.kekChain.Get(activeKekID)
		if !ok {
			return apperrors.Wrap(cryptoDomain.ErrKekNotFound, "active kek not found in chain")
		}

		// Sign the audit log with HMAC-SHA256
		signature, err := a.auditSigner.Sign(kek.Key, auditLog)
		if err != nil {
			return apperrors.Wrap(err, "failed to sign audit log")
		}

		// Populate signature fields
		auditLog.Signature = signature
		auditLog.KekID = &activeKekID
		auditLog.IsSigned = true
	}

	// Persist the audit log (signed or unsigned)
	if err := a.auditLogRepo.Create(ctx, auditLog); err != nil {
		return apperrors.Wrap(err, "failed to create audit log")
	}

	return nil
}

// List retrieves audit logs ordered by created_at descending (newest first) with pagination
// and optional time-based filtering. Accepts createdAtFrom and createdAtTo as optional filters
// (nil means no filter). Both boundaries are inclusive (>= and <=). All timestamps are expected
// in UTC. Returns empty slice if no audit logs found.
func (a *auditLogUseCase) List(
	ctx context.Context,
	offset, limit int,
	createdAtFrom, createdAtTo *time.Time,
) ([]*authDomain.AuditLog, error) {
	auditLogs, err := a.auditLogRepo.List(ctx, offset, limit, createdAtFrom, createdAtTo)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list audit logs")
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

	// Get KEK by ID (historical KEK used for signing)
	kek, ok := a.kekChain.Get(*auditLog.KekID)
	if !ok {
		return authDomain.ErrKekNotFoundForLog
	}

	// Verify signature using KEK
	if err := a.auditSigner.Verify(kek.Key, auditLog); err != nil {
		return apperrors.Wrap(err, "audit log signature verification failed")
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

	// Paginate through logs in batches
	const pageSize = 1000
	offset := 0

	for {
		// Retrieve logs in time range
		logs, err := a.auditLogRepo.List(ctx, offset, pageSize, &startTime, &endTime)
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

			// Get KEK for verification
			kek, ok := a.kekChain.Get(*log.KekID)
			if !ok {
				report.InvalidCount++
				report.InvalidLogs = append(report.InvalidLogs, log.ID)
				continue
			}

			// Verify signature
			if err := a.auditSigner.Verify(kek.Key, log); err != nil {
				report.InvalidCount++
				report.InvalidLogs = append(report.InvalidLogs, log.ID)
				continue
			}

			report.ValidCount++
		}

		offset += pageSize
	}

	return report, nil
}

// NewAuditLogUseCase creates a new AuditLogUseCase with the provided dependencies.
// Requires audit log repository, audit signer for HMAC operations, and KEK chain
// for signature verification across KEK rotations.
func NewAuditLogUseCase(
	auditLogRepo AuditLogRepository,
	auditSigner authService.AuditSigner,
	kekChain *cryptoDomain.KekChain,
) AuditLogUseCase {
	return &auditLogUseCase{
		auditLogRepo: auditLogRepo,
		auditSigner:  auditSigner,
		kekChain:     kekChain,
	}
}
