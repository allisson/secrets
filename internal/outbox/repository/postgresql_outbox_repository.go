// Package repository provides data persistence implementations for outbox entities.
package repository

import (
	"context"
	"database/sql"

	"github.com/allisson/go-project-template/internal/database"
	"github.com/allisson/go-project-template/internal/outbox/domain"
)

// PostgreSQLOutboxEventRepository handles outbox event persistence for PostgreSQL
type PostgreSQLOutboxEventRepository struct {
	db *sql.DB
}

// NewPostgreSQLOutboxEventRepository creates a new PostgreSQLOutboxEventRepository
func NewPostgreSQLOutboxEventRepository(db *sql.DB) *PostgreSQLOutboxEventRepository {
	return &PostgreSQLOutboxEventRepository{
		db: db,
	}
}

// Create inserts a new outbox event
func (r *PostgreSQLOutboxEventRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	querier := database.GetTx(ctx, r.db)

	query := `INSERT INTO outbox_events (id, event_type, payload, status, retries, last_error, processed_at, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`

	_, err := querier.ExecContext(ctx, query, event.ID, event.EventType, event.Payload, event.Status,
		event.Retries, event.LastError, event.ProcessedAt)

	return err
}

// GetPendingEvents retrieves pending events with limit
func (r *PostgreSQLOutboxEventRepository) GetPendingEvents(
	ctx context.Context,
	limit int,
) ([]*domain.OutboxEvent, error) {
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, event_type, payload, status, retries, last_error, processed_at, created_at, updated_at 
			  FROM outbox_events 
			  WHERE status = $1 
			  ORDER BY created_at ASC 
			  LIMIT $2 
			  FOR UPDATE SKIP LOCKED`

	rows, err := querier.QueryContext(ctx, query, domain.OutboxEventStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []*domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent

		err := rows.Scan(&event.ID, &event.EventType, &event.Payload, &event.Status,
			&event.Retries, &event.LastError, &event.ProcessedAt, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
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
func (r *PostgreSQLOutboxEventRepository) Update(ctx context.Context, event *domain.OutboxEvent) error {
	querier := database.GetTx(ctx, r.db)

	query := `UPDATE outbox_events 
			  SET event_type = $1, payload = $2, status = $3, retries = $4, last_error = $5, 
			      processed_at = $6, updated_at = NOW() 
			  WHERE id = $7`

	_, err := querier.ExecContext(ctx, query, event.EventType, event.Payload, event.Status,
		event.Retries, event.LastError, event.ProcessedAt, event.ID)

	return err
}
