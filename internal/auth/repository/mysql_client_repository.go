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

// MySQLClientRepository implements Client persistence for MySQL.
// Uses BINARY(16) for UUID storage with transaction support via database.GetTx().
type MySQLClientRepository struct {
	db *sql.DB
}

// Create inserts a new Client into the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns an error if UUID/policy
// marshaling or database insertion fails.
func (m *MySQLClientRepository) Create(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, m.db)

	policiesJSON, err := json.Marshal(client.Policies)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client policies")
	}

	query := `INSERT INTO clients (id, secret, name, is_active, policies, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := client.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
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

// Update modifies an existing Client in the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns an error if UUID/policy
// marshaling or database update fails.
func (m *MySQLClientRepository) Update(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, m.db)

	policiesJSON, err := json.Marshal(client.Policies)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client policies")
	}

	query := `UPDATE clients 
			  SET secret = ?, 
			  	  name = ?,
				  is_active = ?,
				  policies = ?,
				  created_at = ?
			  WHERE id = ?`

	id, err := client.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		client.Secret,
		client.Name,
		client.IsActive,
		policiesJSON,
		client.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update client")
	}

	return nil
}

// Get retrieves a Client by ID from the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns ErrClientNotFound if the client
// doesn't exist, or an error if UUID/policy unmarshaling or database query fails.
func (m *MySQLClientRepository) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, secret, name, is_active, policies, failed_attempts, locked_until, created_at FROM clients WHERE id = ?`

	id, err := clientID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal client id")
	}

	var client authDomain.Client
	var idBytes []byte
	var policiesJSON []byte

	err = querier.QueryRowContext(ctx, query, id).Scan(
		&idBytes,
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

	if err := client.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal client id")
	}

	if err := json.Unmarshal(policiesJSON, &client.Policies); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal client policies")
	}

	return &client, nil
}

// List retrieves clients ordered by ID descending with pagination support using BINARY(16)
// for UUIDs. Uses transaction support via database.GetTx(). Returns empty slice if no clients
// found, or an error if UUID/policy unmarshaling or database query fails.
func (m *MySQLClientRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*authDomain.Client, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, secret, name, is_active, policies, failed_attempts, locked_until, created_at
			  FROM clients
			  ORDER BY id DESC
			  LIMIT ? OFFSET ?`

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
		var idBytes []byte
		var policiesJSON []byte

		err := rows.Scan(
			&idBytes,
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

		if err := client.ID.UnmarshalBinary(idBytes); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal client id")
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
func (m *MySQLClientRepository) UpdateLockState(
	ctx context.Context,
	clientID uuid.UUID,
	failedAttempts int,
	lockedUntil *time.Time,
) error {
	querier := database.GetTx(ctx, m.db)

	id, err := clientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	query := `UPDATE clients SET failed_attempts = ?, locked_until = ? WHERE id = ?`
	_, err = querier.ExecContext(ctx, query, failedAttempts, lockedUntil, id)
	if err != nil {
		return apperrors.Wrap(err, "failed to update client lock state")
	}
	return nil
}

// NewMySQLClientRepository creates a new MySQL Client repository.
func NewMySQLClientRepository(db *sql.DB) *MySQLClientRepository {
	return &MySQLClientRepository{db: db}
}
