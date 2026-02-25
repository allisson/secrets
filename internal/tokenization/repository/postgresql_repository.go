// Package repository implements data persistence for tokenization key and token management.
// Supports versioning, soft deletion, deterministic token lookups, and dual database support (PostgreSQL and MySQL).
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// PostgreSQLTokenizationKeyRepository implements tokenization key persistence for PostgreSQL databases.
type PostgreSQLTokenizationKeyRepository struct {
	db *sql.DB
}

// Create inserts a new tokenization key into the PostgreSQL database.
func (p *PostgreSQLTokenizationKeyRepository) Create(
	ctx context.Context,
	key *tokenizationDomain.TokenizationKey,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO tokenization_keys (id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := querier.ExecContext(
		ctx,
		query,
		key.ID,
		key.Name,
		key.Version,
		key.FormatType,
		key.IsDeterministic,
		key.DekID,
		key.CreatedAt,
		key.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create tokenization key")
	}
	return nil
}

// Delete soft-deletes a tokenization key by setting its deleted_at timestamp.
func (p *PostgreSQLTokenizationKeyRepository) Delete(ctx context.Context, keyID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokenization_keys SET deleted_at = NOW() WHERE id = $1`

	_, err := querier.ExecContext(ctx, query, keyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a tokenization key by name.
func (p *PostgreSQLTokenizationKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE name = $1 AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var key tokenizationDomain.TokenizationKey
	var formatType string

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&key.ID,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&key.DekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// Get retrieves a tokenization key by its ID.
func (p *PostgreSQLTokenizationKeyRepository) Get(
	ctx context.Context,
	keyID uuid.UUID,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE id = $1 AND deleted_at IS NULL`

	var key tokenizationDomain.TokenizationKey
	var formatType string

	err := querier.QueryRowContext(ctx, query, keyID).Scan(
		&key.ID,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&key.DekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by id")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// GetByNameAndVersion retrieves a specific version of a tokenization key by name and version.
func (p *PostgreSQLTokenizationKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE name = $1 AND version = $2 AND deleted_at IS NULL`

	var key tokenizationDomain.TokenizationKey
	var formatType string

	err := querier.QueryRowContext(ctx, query, name, version).Scan(
		&key.ID,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&key.DekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name and version")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// List retrieves tokenization keys ordered by name ascending with pagination.
// Returns the latest version for each key.
func (p *PostgreSQLTokenizationKeyRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `
		SELECT tk.id, tk.name, tk.version, tk.format_type, tk.is_deterministic, tk.dek_id, tk.created_at, tk.deleted_at 
		FROM tokenization_keys tk
		INNER JOIN (
			SELECT name, MAX(version) as max_version
			FROM tokenization_keys
			WHERE deleted_at IS NULL
			GROUP BY name
			ORDER BY name ASC
			LIMIT $1 OFFSET $2
		) latest ON tk.name = latest.name AND tk.version = latest.max_version
		ORDER BY tk.name ASC`

	rows, err := querier.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list tokenization keys")
	}
	defer func() {
		_ = rows.Close()
	}()

	var keys []*tokenizationDomain.TokenizationKey
	for rows.Next() {
		var key tokenizationDomain.TokenizationKey
		var formatType string

		err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.Version,
			&formatType,
			&key.IsDeterministic,
			&key.DekID,
			&key.CreatedAt,
			&key.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan tokenization key")
		}

		key.FormatType = tokenizationDomain.FormatType(formatType)
		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "error iterating tokenization keys")
	}

	if keys == nil {
		keys = make([]*tokenizationDomain.TokenizationKey, 0)
	}

	return keys, nil
}

// NewPostgreSQLTokenizationKeyRepository creates a new PostgreSQL tokenization key repository instance.
func NewPostgreSQLTokenizationKeyRepository(db *sql.DB) *PostgreSQLTokenizationKeyRepository {
	return &PostgreSQLTokenizationKeyRepository{db: db}
}

// PostgreSQLTokenRepository implements token persistence for PostgreSQL databases.
type PostgreSQLTokenRepository struct {
	db *sql.DB
}

// Create inserts a new token mapping into the PostgreSQL database.
func (p *PostgreSQLTokenRepository) Create(
	ctx context.Context,
	token *tokenizationDomain.Token,
) error {
	querier := database.GetTx(ctx, p.db)

	// Convert metadata to JSONB
	var metadataJSON []byte
	var err error
	if token.Metadata != nil {
		metadataJSON, err = json.Marshal(token.Metadata)
		if err != nil {
			return apperrors.Wrap(err, "failed to marshal metadata")
		}
	}

	query := `INSERT INTO tokenization_tokens 
			  (id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = querier.ExecContext(
		ctx,
		query,
		token.ID,
		token.TokenizationKeyID,
		token.Token,
		token.ValueHash,
		token.Ciphertext,
		token.Nonce,
		metadataJSON,
		token.CreatedAt,
		token.ExpiresAt,
		token.RevokedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create token")
	}
	return nil
}

// GetByToken retrieves a token mapping by its token string.
func (p *PostgreSQLTokenRepository) GetByToken(
	ctx context.Context,
	tokenStr string,
) (*tokenizationDomain.Token, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at 
			  FROM tokenization_tokens 
			  WHERE token = $1`

	var token tokenizationDomain.Token
	var metadataJSON []byte

	err := querier.QueryRowContext(ctx, query, tokenStr).Scan(
		&token.ID,
		&token.TokenizationKeyID,
		&token.Token,
		&token.ValueHash,
		&token.Ciphertext,
		&token.Nonce,
		&metadataJSON,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get token by token string")
	}

	// Parse metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &token.Metadata); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	return &token, nil
}

// GetByValueHash retrieves a token by its value hash (for deterministic mode).
func (p *PostgreSQLTokenRepository) GetByValueHash(
	ctx context.Context,
	keyID uuid.UUID,
	valueHash string,
) (*tokenizationDomain.Token, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at 
			  FROM tokenization_tokens 
			  WHERE tokenization_key_id = $1 AND value_hash = $2`

	var token tokenizationDomain.Token
	var metadataJSON []byte

	err := querier.QueryRowContext(ctx, query, keyID, valueHash).Scan(
		&token.ID,
		&token.TokenizationKeyID,
		&token.Token,
		&token.ValueHash,
		&token.Ciphertext,
		&token.Nonce,
		&metadataJSON,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get token by value hash")
	}

	// Parse metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &token.Metadata); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	return &token, nil
}

// Revoke marks a token as revoked by setting its revoked_at timestamp.
func (p *PostgreSQLTokenRepository) Revoke(ctx context.Context, tokenStr string) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokenization_tokens SET revoked_at = NOW() WHERE token = $1`

	result, err := querier.ExecContext(ctx, query, tokenStr)
	if err != nil {
		return apperrors.Wrap(err, "failed to revoke token")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return tokenizationDomain.ErrTokenNotFound
	}

	return nil
}

// DeleteExpired deletes tokens that expired before the specified timestamp.
// Returns the number of deleted tokens. Uses transaction support via database.GetTx().
// All timestamps are expected in UTC.
func (p *PostgreSQLTokenRepository) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	if olderThan.IsZero() {
		return 0, apperrors.New("olderThan timestamp cannot be zero")
	}

	querier := database.GetTx(ctx, p.db)

	query := `DELETE FROM tokenization_tokens WHERE expires_at < $1`

	result, err := querier.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to delete expired tokens")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to get rows affected")
	}

	return rowsAffected, nil
}

// CountExpired counts tokens that expired before the specified timestamp without deleting them.
// Returns the count of matching tokens. Uses transaction support via database.GetTx().
// All timestamps are expected in UTC.
func (p *PostgreSQLTokenRepository) CountExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	if olderThan.IsZero() {
		return 0, apperrors.New("olderThan timestamp cannot be zero")
	}

	querier := database.GetTx(ctx, p.db)

	query := `SELECT COUNT(*) FROM tokenization_tokens WHERE expires_at < $1`

	var count int64
	err := querier.QueryRowContext(ctx, query, olderThan).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to count expired tokens")
	}

	return count, nil
}

// NewPostgreSQLTokenRepository creates a new PostgreSQL token repository instance.
func NewPostgreSQLTokenRepository(db *sql.DB) *PostgreSQLTokenRepository {
	return &PostgreSQLTokenRepository{db: db}
}
