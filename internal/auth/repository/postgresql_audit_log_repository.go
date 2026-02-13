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

// List retrieves audit logs ordered by ID descending (newest first) with pagination.
// Uses offset and limit for pagination control. Returns empty slice if no audit logs found.
// Handles NULL metadata gracefully by returning nil map for those entries.
func (p *PostgreSQLAuditLogRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*authDomain.AuditLog, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, request_id, client_id, capability, path, metadata, created_at 
			  FROM audit_logs 
			  ORDER BY id DESC 
			  LIMIT $1 OFFSET $2`

	rows, err := querier.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list audit logs")
	}
	defer func() {
		_ = rows.Close()
	}()

	// Initialize empty slice to avoid returning nil for empty results
	auditLogs := make([]*authDomain.AuditLog, 0)
	for rows.Next() {
		var auditLog authDomain.AuditLog
		var metadataJSON []byte
		var capability string

		err := rows.Scan(
			&auditLog.ID,
			&auditLog.RequestID,
			&auditLog.ClientID,
			&capability,
			&auditLog.Path,
			&metadataJSON,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan audit log")
		}

		auditLog.Capability = authDomain.Capability(capability)

		// Unmarshal metadata if not NULL
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &auditLog.Metadata); err != nil {
				return nil, apperrors.Wrap(err, "failed to unmarshal audit log metadata")
			}
		}

		auditLogs = append(auditLogs, &auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "failed to iterate audit logs")
	}

	return auditLogs, nil
}

// NewPostgreSQLAuditLogRepository creates a new PostgreSQL AuditLog repository.
func NewPostgreSQLAuditLogRepository(db *sql.DB) *PostgreSQLAuditLogRepository {
	return &PostgreSQLAuditLogRepository{db: db}
}
