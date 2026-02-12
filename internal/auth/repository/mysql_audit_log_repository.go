package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLAuditLogRepository implements AuditLog persistence for MySQL.
// Uses BINARY(16) for UUID storage with transaction support via database.GetTx().
type MySQLAuditLogRepository struct {
	db *sql.DB
}

// Create inserts a new AuditLog into the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Handles nil metadata as database NULL.
// Returns an error if UUID/metadata marshaling or database insertion fails.
func (m *MySQLAuditLogRepository) Create(ctx context.Context, auditLog *authDomain.AuditLog) error {
	querier := database.GetTx(ctx, m.db)

	var metadataJSON []byte
	var err error

	// Handle nil metadata as NULL
	if auditLog.Metadata != nil {
		metadataJSON, err = json.Marshal(auditLog.Metadata)
		if err != nil {
			return apperrors.Wrap(err, "failed to marshal audit log metadata")
		}
	}

	query := `INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	id, err := auditLog.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal audit log id")
	}

	requestID, err := auditLog.RequestID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal audit log request_id")
	}

	clientID, err := auditLog.ClientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal audit log client_id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		requestID,
		clientID,
		string(auditLog.Capability),
		auditLog.Path,
		metadataJSON,
		auditLog.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create audit log")
	}

	return nil
}

// NewMySQLAuditLogRepository creates a new MySQL AuditLog repository.
func NewMySQLAuditLogRepository(db *sql.DB) *MySQLAuditLogRepository {
	return &MySQLAuditLogRepository{db: db}
}
