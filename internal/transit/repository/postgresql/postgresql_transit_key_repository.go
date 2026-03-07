// Package repository implements data persistence for transit encryption key management.
// Transit keys support versioning and soft deletion, with implementations for both PostgreSQL and MySQL.
package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
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

// GetTransitKey retrieves a transit key version by name and optional version (0 for latest), 
// including its associated encryption algorithm. Returns ErrTransitKeyNotFound if not found.
func (p *PostgreSQLTransitKeyRepository) GetTransitKey(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, cryptoDomain.Algorithm, error) {
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
	var algorithm cryptoDomain.Algorithm
	err := querier.QueryRowContext(ctx, query, args...).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
		&algorithm,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", transitDomain.ErrTransitKeyNotFound
		}
		return nil, "", apperrors.Wrap(err, "failed to get transit key")
	}

	return &transitKey, algorithm, nil
}

// ListCursor retrieves transit keys ordered by name ascending using cursor-based pagination.
// Returns the latest version for each key.
func (p *PostgreSQLTransitKeyRepository) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	var query string
	var args []interface{}

	if afterName == nil {
		// First page: no cursor
		query = `
			SELECT tk.id, tk.name, tk.version, tk.dek_id, tk.created_at, tk.deleted_at
			FROM transit_keys tk
			INNER JOIN (
				SELECT name, MAX(version) as max_version
				FROM transit_keys
				WHERE deleted_at IS NULL
				GROUP BY name
				ORDER BY name ASC
				LIMIT $1
			) latest ON tk.name = latest.name AND tk.version = latest.max_version
			ORDER BY tk.name ASC`
		args = []interface{}{limit}
	} else {
		// Subsequent pages: use cursor (name > afterName)
		query = `
			SELECT tk.id, tk.name, tk.version, tk.dek_id, tk.created_at, tk.deleted_at
			FROM transit_keys tk
			INNER JOIN (
				SELECT name, MAX(version) as max_version
				FROM transit_keys
				WHERE deleted_at IS NULL AND name > $1
				GROUP BY name
				ORDER BY name ASC
				LIMIT $2
			) latest ON tk.name = latest.name AND tk.version = latest.max_version
			ORDER BY tk.name ASC`
		args = []interface{}{*afterName, limit}
	}

	rows, err := querier.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list transit keys with cursor")
	}
	defer func() {
		_ = rows.Close()
	}()

	var transitKeys []*transitDomain.TransitKey
	for rows.Next() {
		var transitKey transitDomain.TransitKey
		err := rows.Scan(
			&transitKey.ID,
			&transitKey.Name,
			&transitKey.Version,
			&transitKey.DekID,
			&transitKey.CreatedAt,
			&transitKey.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan transit key")
		}
		transitKeys = append(transitKeys, &transitKey)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "error iterating transit keys")
	}

	if transitKeys == nil {
		transitKeys = make([]*transitDomain.TransitKey, 0)
	}

	return transitKeys, nil
}

// HardDelete permanently removes soft-deleted transit keys older than the specified time.
func (p *PostgreSQLTransitKeyRepository) HardDelete(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	querier := database.GetTx(ctx, p.db)

	if dryRun {
		query := `SELECT COUNT(*) FROM transit_keys WHERE deleted_at IS NOT NULL AND deleted_at < $1`
		var count int64
		err := querier.QueryRowContext(ctx, query, olderThan).Scan(&count)
		if err != nil {
			return 0, apperrors.Wrap(err, "failed to count transit keys for hard delete")
		}
		return count, nil
	}

	query := `DELETE FROM transit_keys WHERE deleted_at IS NOT NULL AND deleted_at < $1`
	result, err := querier.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to hard delete transit keys")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to get rows affected for hard delete")
	}

	return count, nil
}

// NewPostgreSQLTransitKeyRepository creates a new PostgreSQL transit key repository instance.
func NewPostgreSQLTransitKeyRepository(db *sql.DB) *PostgreSQLTransitKeyRepository {
	return &PostgreSQLTransitKeyRepository{db: db}
}
