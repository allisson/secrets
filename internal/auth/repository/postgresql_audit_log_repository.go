package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

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
// via database.GetTx(). Handles nil metadata as database NULL. Includes cryptographic
// signature fields (signature, kek_id, is_signed) for tamper detection. Returns an error
// if metadata marshaling or database insertion fails.
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

	query := `INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, signature, kek_id, is_signed, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = querier.ExecContext(
		ctx,
		query,
		auditLog.ID,
		auditLog.RequestID,
		auditLog.ClientID,
		string(auditLog.Capability),
		auditLog.Path,
		metadataJSON,
		auditLog.Signature,
		auditLog.KekID,
		auditLog.IsSigned,
		auditLog.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create audit log")
	}

	return nil
}

// Get retrieves a single audit log by ID from the PostgreSQL database. Returns
// error if the audit log is not found or if database operation fails.
func (p *PostgreSQLAuditLogRepository) Get(ctx context.Context, id uuid.UUID) (*authDomain.AuditLog, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, request_id, client_id, capability, path, metadata, signature, kek_id, is_signed, created_at
			  FROM audit_logs
			  WHERE id = $1`

	var auditLog authDomain.AuditLog
	var metadataJSON []byte
	var capability string

	err := querier.QueryRowContext(ctx, query, id).Scan(
		&auditLog.ID,
		&auditLog.RequestID,
		&auditLog.ClientID,
		&capability,
		&auditLog.Path,
		&metadataJSON,
		&auditLog.Signature,
		&auditLog.KekID,
		&auditLog.IsSigned,
		&auditLog.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.Wrap(apperrors.ErrNotFound, "audit log not found")
	}
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get audit log")
	}

	auditLog.Capability = authDomain.Capability(capability)

	// Unmarshal metadata if not NULL
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &auditLog.Metadata); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal audit log metadata")
		}
	}

	return &auditLog, nil
}

// List retrieves audit logs ordered by created_at descending (newest first) with pagination
// and optional time-based filtering. Accepts createdAtFrom and createdAtTo as optional filters
// (nil means no filter). Both boundaries are inclusive (>= and <=). All timestamps are expected
// in UTC. Returns empty slice if no audit logs found. Handles NULL metadata gracefully by
// returning nil map for those entries.
func (p *PostgreSQLAuditLogRepository) List(
	ctx context.Context,
	offset, limit int,
	createdAtFrom, createdAtTo *time.Time,
) ([]*authDomain.AuditLog, error) {
	querier := database.GetTx(ctx, p.db)

	// Build dynamic WHERE clause based on provided filters
	var conditions []string
	var args []interface{}
	paramIndex := 1

	if createdAtFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", paramIndex))
		args = append(args, *createdAtFrom)
		paramIndex++
	}

	if createdAtTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", paramIndex))
		args = append(args, *createdAtTo)
		paramIndex++
	}

	// Build query
	query := `SELECT id, request_id, client_id, capability, path, metadata, signature, kek_id, is_signed, created_at 
			  FROM audit_logs`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)

	// Add limit and offset to args
	args = append(args, limit, offset)

	rows, err := querier.QueryContext(ctx, query, args...)
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
			&auditLog.Signature,
			&auditLog.KekID,
			&auditLog.IsSigned,
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

// DeleteOlderThan removes audit logs created before the specified timestamp.
// When dryRun is true, returns count via SELECT COUNT(*) without deletion. When false,
// executes DELETE and returns affected rows. Uses transaction support via database.GetTx().
// All timestamps are expected in UTC.
func (p *PostgreSQLAuditLogRepository) DeleteOlderThan(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	querier := database.GetTx(ctx, p.db)

	if dryRun {
		// Use COUNT query for dry-run mode
		query := `SELECT COUNT(*) FROM audit_logs WHERE created_at < $1`
		var count int64
		err := querier.QueryRowContext(ctx, query, olderThan).Scan(&count)
		if err != nil {
			return 0, apperrors.Wrap(err, "failed to count audit logs")
		}
		return count, nil
	}

	// Execute actual deletion
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	result, err := querier.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to delete audit logs")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to get affected rows count")
	}

	return count, nil
}

// NewPostgreSQLAuditLogRepository creates a new PostgreSQL AuditLog repository.
func NewPostgreSQLAuditLogRepository(db *sql.DB) *PostgreSQLAuditLogRepository {
	return &PostgreSQLAuditLogRepository{db: db}
}
