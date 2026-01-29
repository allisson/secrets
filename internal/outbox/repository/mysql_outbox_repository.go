// Package repository provides data persistence implementations for outbox entities.
package repository

import (
	"context"
	"database/sql"

	"github.com/allisson/go-project-template/internal/database"
	"github.com/allisson/go-project-template/internal/outbox/domain"
)

// MySQLOutboxEventRepository handles outbox event persistence for MySQL
type MySQLOutboxEventRepository struct {
	db *sql.DB
}

// NewMySQLOutboxEventRepository creates a new MySQLOutboxEventRepository
func NewMySQLOutboxEventRepository(db *sql.DB) *MySQLOutboxEventRepository {
	return &MySQLOutboxEventRepository{
		db: db,
	}
}

// Create inserts a new outbox event
func (r *MySQLOutboxEventRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	querier := database.GetTx(ctx, r.db)

	query := `INSERT INTO outbox_events (id, event_type, payload, status, retries, last_error, processed_at, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`

	// Convert UUID to bytes for MySQL BINARY(16)
	idBytes, err := event.ID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = querier.ExecContext(ctx, query, idBytes, event.EventType, event.Payload, event.Status,
		event.Retries, event.LastError, event.ProcessedAt)

	return err
}

// GetPendingEvents retrieves pending events with limit
func (r *MySQLOutboxEventRepository) GetPendingEvents(
	ctx context.Context,
	limit int,
) ([]*domain.OutboxEvent, error) {
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, event_type, payload, status, retries, last_error, processed_at, created_at, updated_at 
			  FROM outbox_events 
			  WHERE status = ? 
			  ORDER BY created_at ASC 
			  LIMIT ? 
			  FOR UPDATE SKIP LOCKED`

	rows, err := querier.QueryContext(ctx, query, domain.OutboxEventStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []*domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent
		var idBytes []byte

		err := rows.Scan(&idBytes, &event.EventType, &event.Payload, &event.Status,
			&event.Retries, &event.LastError, &event.ProcessedAt, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Convert bytes back to UUID
		if err := event.ID.UnmarshalBinary(idBytes); err != nil {
			return nil, err
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// Update updates an outbox event
func (r *MySQLOutboxEventRepository) Update(ctx context.Context, event *domain.OutboxEvent) error {
	querier := database.GetTx(ctx, r.db)

	query := `UPDATE outbox_events 
			  SET event_type = ?, payload = ?, status = ?, retries = ?, last_error = ?, 
			      processed_at = ?, updated_at = NOW() 
			  WHERE id = ?`

	// Convert UUID to bytes for MySQL BINARY(16)
	idBytes, err := event.ID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = querier.ExecContext(ctx, query, event.EventType, event.Payload, event.Status,
		event.Retries, event.LastError, event.ProcessedAt, idBytes)

	return err
}
