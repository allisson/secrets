// Package repository implements data persistence for authentication and authorization entities.
//
// Provides PostgreSQL and MySQL implementations with transaction support via database.GetTx().
// PostgreSQL uses native UUID types, MySQL uses BINARY(16) types.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLClientRepository implements Client persistence for PostgreSQL.
// Uses native UUID types with transaction support via database.GetTx().
type PostgreSQLClientRepository struct {
	db *sql.DB
}

// Create inserts a new Client into the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns an error if policy marshaling or database insertion fails.
func (p *PostgreSQLClientRepository) Create(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, p.db)

	policiesJSON, err := json.Marshal(client.Policies)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client policies")
	}

	query := `INSERT INTO clients (id, secret, name, is_active, policies, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = querier.ExecContext(
		ctx,
		query,
		client.ID,
		client.Secret,
		client.Name,
		client.IsActive,
		policiesJSON,
		client.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create client")
	}
	return nil
}

// Update modifies an existing Client in the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns an error if policy marshaling or database update fails.
func (p *PostgreSQLClientRepository) Update(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, p.db)

	policiesJSON, err := json.Marshal(client.Policies)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client policies")
	}

	query := `UPDATE clients 
			  SET secret = $1, 
			  	  name = $2,
				  is_active = $3,
				  policies = $4,
				  created_at = $5
			  WHERE id = $6`

	_, err = querier.ExecContext(
		ctx,
		query,
		client.Secret,
		client.Name,
		client.IsActive,
		policiesJSON,
		client.CreatedAt,
		client.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update client")
	}

	return nil
}

// Get retrieves a Client by ID from the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns ErrClientNotFound if the client doesn't exist, or an error
// if policy unmarshaling or database query fails.
func (p *PostgreSQLClientRepository) Get(
	ctx context.Context,
	clientID uuid.UUID,
) (*authDomain.Client, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, secret, name, is_active, policies, failed_attempts, locked_until, created_at FROM clients WHERE id = $1`

	var client authDomain.Client
	var policiesJSON []byte

	err := querier.QueryRowContext(ctx, query, clientID).Scan(
		&client.ID,
		&client.Secret,
		&client.Name,
		&client.IsActive,
		&policiesJSON,
		&client.FailedAttempts,
		&client.LockedUntil,
		&client.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authDomain.ErrClientNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get client")
	}

	if err := json.Unmarshal(policiesJSON, &client.Policies); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal client policies")
	}

	return &client, nil
}

// List retrieves clients ordered by ID descending with pagination support. Uses transaction
// support via database.GetTx(). Returns empty slice if no clients found, or an error if
// policy unmarshaling or database query fails.
func (p *PostgreSQLClientRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*authDomain.Client, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, secret, name, is_active, policies, failed_attempts, locked_until, created_at
			  FROM clients
			  ORDER BY id DESC
			  LIMIT $1 OFFSET $2`

	rows, err := querier.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list clients")
	}
	defer func() {
		_ = rows.Close()
	}()

	// Initialize empty slice to avoid returning nil for empty results
	clients := make([]*authDomain.Client, 0)
	for rows.Next() {
		var client authDomain.Client
		var policiesJSON []byte

		err := rows.Scan(
			&client.ID,
			&client.Secret,
			&client.Name,
			&client.IsActive,
			&policiesJSON,
			&client.FailedAttempts,
			&client.LockedUntil,
			&client.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan client row")
		}

		if err := json.Unmarshal(policiesJSON, &client.Policies); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal client policies")
		}

		clients = append(clients, &client)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "error iterating client rows")
	}

	return clients, nil
}

// UpdateLockState atomically updates the failed attempt counter and lock expiry for a client.
func (p *PostgreSQLClientRepository) UpdateLockState(
	ctx context.Context,
	clientID uuid.UUID,
	failedAttempts int,
	lockedUntil *time.Time,
) error {
	querier := database.GetTx(ctx, p.db)
	query := `UPDATE clients SET failed_attempts = $1, locked_until = $2 WHERE id = $3`
	_, err := querier.ExecContext(ctx, query, failedAttempts, lockedUntil, clientID)
	if err != nil {
		return apperrors.Wrap(err, "failed to update client lock state")
	}
	return nil
}

// NewPostgreSQLClientRepository creates a new PostgreSQL Client repository.
func NewPostgreSQLClientRepository(db *sql.DB) *PostgreSQLClientRepository {
	return &PostgreSQLClientRepository{db: db}
}
