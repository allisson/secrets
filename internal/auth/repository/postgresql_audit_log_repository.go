package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLAuditLogRepository implements AuditLog persistence for PostgreSQL.
// Uses native UUID types with transaction support via database.GetTx().
type PostgreSQLAuditLogRepository struct {
	db *sql.DB
}

// Create inserts a new AuditLog into the PostgreSQL database. Uses transaction support
// via database.GetTx(). Handles nil metadata as database NULL. Returns an error if
// metadata marshaling or database insertion fails.
func (p *PostgreSQLAuditLogRepository) Create(ctx context.Context, auditLog *authDomain.AuditLog) error {
	querier := database.GetTx(ctx, p.db)

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
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = querier.ExecContext(
		ctx,
		query,
		auditLog.ID,
		auditLog.RequestID,
		auditLog.ClientID,
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

// NewPostgreSQLAuditLogRepository creates a new PostgreSQL AuditLog repository.
func NewPostgreSQLAuditLogRepository(db *sql.DB) *PostgreSQLAuditLogRepository {
	return &PostgreSQLAuditLogRepository{db: db}
}
