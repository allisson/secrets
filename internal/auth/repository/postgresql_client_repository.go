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

// Create inserts a new Client into the PostgreSQL database.
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

// Update modifies an existing Client in the PostgreSQL database.
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

// Get retrieves a Client by ID from the PostgreSQL database.
func (p *PostgreSQLClientRepository) Get(
	ctx context.Context,
	clientID uuid.UUID,
) (*authDomain.Client, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, secret, name, is_active, policies, created_at FROM clients WHERE id = $1`

	var client authDomain.Client
	var policiesJSON []byte

	err := querier.QueryRowContext(ctx, query, clientID).Scan(
		&client.ID,
		&client.Secret,
		&client.Name,
		&client.IsActive,
		&policiesJSON,
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

// NewPostgreSQLClientRepository creates a new PostgreSQL Client repository.
func NewPostgreSQLClientRepository(db *sql.DB) *PostgreSQLClientRepository {
	return &PostgreSQLClientRepository{db: db}
}
