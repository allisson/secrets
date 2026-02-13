// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// auditLogUseCase implements AuditLogUseCase interface for recording audit logs.
type auditLogUseCase struct {
	auditLogRepo AuditLogRepository
}

// Create records an audit log entry for an authenticated operation. Generates a unique
// UUIDv7 identifier and timestamp. The metadata parameter is optional and can be nil.
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
	}

	// Persist the audit log
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

// NewAuditLogUseCase creates a new AuditLogUseCase with the provided dependencies.
func NewAuditLogUseCase(auditLogRepo AuditLogRepository) AuditLogUseCase {
	return &auditLogUseCase{
		auditLogRepo: auditLogRepo,
	}
}
