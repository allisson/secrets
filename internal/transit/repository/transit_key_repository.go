// Package repository implements data persistence for transit encryption key management.
// Transit keys support versioning and soft deletion, with PostgreSQL persistence.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// TransitKeyRepository implements transit key persistence for PostgreSQL databases.
type TransitKeyRepository struct {
	db *sql.DB
}

// Create inserts a new transit key into the PostgreSQL database.
func (p *TransitKeyRepository) Create(
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

// Delete soft-deletes all versions of a transit key by name.
func (p *TransitKeyRepository) Delete(ctx context.Context, name string) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE transit_keys SET deleted_at = NOW() WHERE name = $1 AND deleted_at IS NULL`

	_, err := querier.ExecContext(ctx, query, name)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete transit key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a transit key by name.
func (p *TransitKeyRepository) GetByName(
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
func (p *TransitKeyRepository) GetByNameAndVersion(
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

// GetTransitKey retrieves a transit key version by name and optional version (0 for latest),
// including its associated encryption algorithm (populated in TransitKey.Algorithm).
// Returns ErrTransitKeyNotFound if not found.
func (p *TransitKeyRepository) GetTransitKey(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	var query string
	var args []interface{}

	if version == 0 {
		query = `SELECT tk.id, tk.name, tk.version, tk.dek_id, tk.created_at, tk.deleted_at, d.algorithm
				  FROM transit_keys tk
				  JOIN deks d ON tk.dek_id = d.id
				  WHERE tk.name = $1 AND tk.deleted_at IS NULL
				  ORDER BY tk.version DESC
				  LIMIT 1`
		args = []interface{}{name}
	} else {
		query = `SELECT tk.id, tk.name, tk.version, tk.dek_id, tk.created_at, tk.deleted_at, d.algorithm
				  FROM transit_keys tk
				  JOIN deks d ON tk.dek_id = d.id
				  WHERE tk.name = $1 AND tk.version = $2 AND tk.deleted_at IS NULL`
		args = []interface{}{name, version}
	}

	var transitKey transitDomain.TransitKey
	err := querier.QueryRowContext(ctx, query, args...).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
		&transitKey.Algorithm,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key")
	}

	return &transitKey, nil
}

// ListCursor retrieves transit keys ordered by name ascending using cursor-based pagination.
// Returns the latest version for each key.
func (p *TransitKeyRepository) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*transitDomain.TransitKey, error) {
	records, err := database.ListLatestCursor(
		ctx,
		database.GetTx(ctx, p.db),
		"transit_keys",
		"name",
		"t.id, t.name, t.version, t.dek_id, t.created_at, t.deleted_at",
		afterName,
		limit,
		func(rows *sql.Rows) (*transitDomain.TransitKey, error) {
			var key transitDomain.TransitKey
			if err := rows.Scan(
				&key.ID,
				&key.Name,
				&key.Version,
				&key.DekID,
				&key.CreatedAt,
				&key.DeletedAt,
			); err != nil {
				return nil, err
			}
			return &key, nil
		},
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list transit keys with cursor")
	}
	return records, nil
}

// HardDelete permanently removes soft-deleted transit keys older than the specified time.
func (p *TransitKeyRepository) HardDelete(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	count, err := database.HardDeleteOlderThan(
		ctx,
		database.GetTx(ctx, p.db),
		"transit_keys",
		olderThan,
		dryRun,
	)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to hard delete transit keys")
	}
	return count, nil
}

// NewTransitKeyRepository creates a new PostgreSQL transit key repository instance.
func NewTransitKeyRepository(db *sql.DB) *TransitKeyRepository {
	return &TransitKeyRepository{db: db}
}
