package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

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

// List retrieves audit logs ordered by created_at descending (newest first) with pagination
// and optional time-based filtering. Accepts createdAtFrom and createdAtTo as optional filters
// (nil means no filter). Both boundaries are inclusive (>= and <=). All timestamps are expected
// in UTC. Returns empty slice if no audit logs found. Handles NULL metadata gracefully by
// returning nil map for those entries. UUIDs are stored as BINARY(16) and must be unmarshaled.
func (m *MySQLAuditLogRepository) List(
	ctx context.Context,
	offset, limit int,
	createdAtFrom, createdAtTo *time.Time,
) ([]*authDomain.AuditLog, error) {
	querier := database.GetTx(ctx, m.db)

	// Build dynamic WHERE clause based on provided filters
	var conditions []string
	var args []interface{}

	if createdAtFrom != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *createdAtFrom)
	}

	if createdAtTo != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *createdAtTo)
	}

	// Build query
	query := `SELECT id, request_id, client_id, capability, path, metadata, created_at 
			  FROM audit_logs`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

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
		var idBinary, requestIDBinary, clientIDBinary []byte
		var metadataJSON []byte
		var capability string

		err := rows.Scan(
			&idBinary,
			&requestIDBinary,
			&clientIDBinary,
			&capability,
			&auditLog.Path,
			&metadataJSON,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan audit log")
		}

		// Unmarshal UUIDs from BINARY(16)
		if err := auditLog.ID.UnmarshalBinary(idBinary); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal audit log id")
		}

		if err := auditLog.RequestID.UnmarshalBinary(requestIDBinary); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal audit log request_id")
		}

		if err := auditLog.ClientID.UnmarshalBinary(clientIDBinary); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal audit log client_id")
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

// NewMySQLAuditLogRepository creates a new MySQL AuditLog repository.
func NewMySQLAuditLogRepository(db *sql.DB) *MySQLAuditLogRepository {
	return &MySQLAuditLogRepository{db: db}
}
