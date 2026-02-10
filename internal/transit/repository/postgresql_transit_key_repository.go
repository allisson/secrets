// Package repository implements data persistence for transit encryption key management.
// Transit keys support versioning and soft deletion, with implementations for both PostgreSQL and MySQL.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// PostgreSQLTransitKeyRepository implements transit key persistence for PostgreSQL databases.
type PostgreSQLTransitKeyRepository struct {
	db *sql.DB
}

// Create inserts a new transit key into the PostgreSQL database.
func (p *PostgreSQLTransitKeyRepository) Create(
	ctx context.Context,
	transitKey *transitDomain.TransitKey,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		transitKey.ID,
		transitKey.Name,
		transitKey.Version,
		transitKey.DekID,
		transitKey.CreatedAt,
		transitKey.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create transit key")
	}
	return nil
}

// Delete soft-deletes a transit key by setting its deleted_at timestamp.
func (p *PostgreSQLTransitKeyRepository) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE transit_keys SET deleted_at = NOW() WHERE id = $1`

	_, err := querier.ExecContext(ctx, query, transitKeyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete transit key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a transit key by name.
func (p *PostgreSQLTransitKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = $1 AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var transitKey transitDomain.TransitKey
	err := querier.QueryRowContext(ctx, query, name).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name")
	}

	return &transitKey, nil
}

// GetByNameAndVersion retrieves a specific version of a transit key by name and version.
func (p *PostgreSQLTransitKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = $1 AND version = $2 AND deleted_at IS NULL`

	var transitKey transitDomain.TransitKey
	err := querier.QueryRowContext(ctx, query, name, version).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name and version")
	}

	return &transitKey, nil
}

// NewPostgreSQLTransitKeyRepository creates a new PostgreSQL transit key repository instance.
func NewPostgreSQLTransitKeyRepository(db *sql.DB) *PostgreSQLTransitKeyRepository {
	return &PostgreSQLTransitKeyRepository{db: db}
}
