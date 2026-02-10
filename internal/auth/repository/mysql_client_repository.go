package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLClientRepository implements Client persistence for MySQL.
// Uses BINARY(16) for UUIDs with transaction support via database.GetTx().
type MySQLClientRepository struct {
	db *sql.DB
}

// Create inserts a new Client into the MySQL database.
func (m *MySQLClientRepository) Create(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO clients (id, secret, name, is_active, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

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
		client.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create client")
	}
	return nil
}

// Update modifies an existing Client in the MySQL database.
func (m *MySQLClientRepository) Update(ctx context.Context, client *authDomain.Client) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE clients 
			  SET secret = ?, 
			  	  name = ?,
				  is_active = ?,
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
		client.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update client")
	}

	return nil
}

// Get retrieves a Client by ID from the MySQL database.
func (m *MySQLClientRepository) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, secret, name, is_active, created_at FROM clients WHERE id = ?`

	id, err := clientID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal client id")
	}

	var client authDomain.Client
	var idBytes []byte

	err = querier.QueryRowContext(ctx, query, id).Scan(
		&idBytes,
		&client.Secret,
		&client.Name,
		&client.IsActive,
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

	return &client, nil
}

// NewMySQLClientRepository creates a new MySQL Client repository.
func NewMySQLClientRepository(db *sql.DB) *MySQLClientRepository {
	return &MySQLClientRepository{db: db}
}
