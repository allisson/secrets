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

// MySQLTransitKeyRepository implements transit key persistence for MySQL databases.
type MySQLTransitKeyRepository struct {
	db *sql.DB
}

// Create inserts a new transit key into the MySQL database.
func (m *MySQLTransitKeyRepository) Create(ctx context.Context, transitKey *transitDomain.TransitKey) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := transitKey.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal transit key id")
	}

	dekID, err := transitKey.DekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		transitKey.Name,
		transitKey.Version,
		dekID,
		transitKey.CreatedAt,
		transitKey.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create transit key")
	}
	return nil
}

// Delete soft-deletes a transit key by setting its deleted_at timestamp.
func (m *MySQLTransitKeyRepository) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE transit_keys SET deleted_at = NOW() WHERE id = ?`

	id, err := transitKeyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal transit key id")
	}

	_, err = querier.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete transit key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a transit key by name.
func (m *MySQLTransitKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = ? AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var transitKey transitDomain.TransitKey
	var id []byte
	var dekID []byte

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&id,
		&transitKey.Name,
		&transitKey.Version,
		&dekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name")
	}

	if err := transitKey.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal transit key id")
	}

	if err := transitKey.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	return &transitKey, nil
}

// GetByNameAndVersion retrieves a specific version of a transit key by name and version.
func (m *MySQLTransitKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = ? AND version = ? AND deleted_at IS NULL`

	var transitKey transitDomain.TransitKey
	var id []byte
	var dekID []byte

	err := querier.QueryRowContext(ctx, query, name, version).Scan(
		&id,
		&transitKey.Name,
		&transitKey.Version,
		&dekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name and version")
	}

	if err := transitKey.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal transit key id")
	}

	if err := transitKey.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	return &transitKey, nil
}

// List retrieves transit keys ordered by name ascending with pagination.
// Returns the latest version for each key.
func (m *MySQLTransitKeyRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `
		SELECT tk.id, tk.name, tk.version, tk.dek_id, tk.created_at, tk.deleted_at
		FROM transit_keys tk
		INNER JOIN (
			SELECT name, MAX(version) as max_version
			FROM transit_keys
			WHERE deleted_at IS NULL
			GROUP BY name
			ORDER BY name ASC
			LIMIT ? OFFSET ?
		) latest ON tk.name = latest.name AND tk.version = latest.max_version
		ORDER BY tk.name ASC`

	rows, err := querier.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list transit keys")
	}
	defer func() {
		_ = rows.Close()
	}()

	var transitKeys []*transitDomain.TransitKey
	for rows.Next() {
		var transitKey transitDomain.TransitKey
		var id, dekID []byte

		err := rows.Scan(
			&id,
			&transitKey.Name,
			&transitKey.Version,
			&dekID,
			&transitKey.CreatedAt,
			&transitKey.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan transit key")
		}

		if err := transitKey.ID.UnmarshalBinary(id); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal transit key id")
		}

		if err := transitKey.DekID.UnmarshalBinary(dekID); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
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

// NewMySQLTransitKeyRepository creates a new MySQL transit key repository instance.
func NewMySQLTransitKeyRepository(db *sql.DB) *MySQLTransitKeyRepository {
	return &MySQLTransitKeyRepository{db: db}
}
