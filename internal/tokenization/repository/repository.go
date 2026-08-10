// Package repository implements data persistence for tokenization key and token management.
// Supports versioning, soft deletion, deterministic token lookups, and PostgreSQL persistence.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// TokenizationKeyRepository implements tokenization key persistence for PostgreSQL databases.
type TokenizationKeyRepository struct {
	db *sql.DB
}

// Create inserts a new tokenization key into the PostgreSQL database.
func (p *TokenizationKeyRepository) Create(
	ctx context.Context,
	key *tokenizationDomain.TokenizationKey,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO tokenization_keys (id, name, version, format_type, is_deterministic, salt, dek_id, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := querier.ExecContext(
		ctx,
		query,
		key.ID,
		key.Name,
		key.Version,
		key.FormatType,
		key.IsDeterministic,
		key.Salt,
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
func (p *TokenizationKeyRepository) Delete(ctx context.Context, name string) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokenization_keys SET deleted_at = NOW() WHERE name = $1`

	_, err := querier.ExecContext(ctx, query, name)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a tokenization key by name.
func (p *TokenizationKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, salt, dek_id, created_at, deleted_at 
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
		&key.Salt,
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
func (p *TokenizationKeyRepository) Get(
	ctx context.Context,
	keyID uuid.UUID,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, salt, dek_id, created_at, deleted_at 
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
		&key.Salt,
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
func (p *TokenizationKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, format_type, is_deterministic, salt, dek_id, created_at, deleted_at 
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
		&key.Salt,
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

// ListCursor retrieves tokenization keys ordered by name ascending using cursor-based pagination.
// Returns the latest version for each key.
func (p *TokenizationKeyRepository) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	records, err := database.ListLatestCursor(
		ctx,
		database.GetTx(ctx, p.db),
		"tokenization_keys",
		"name",
		"t.id, t.name, t.version, t.format_type, t.is_deterministic, t.salt, t.dek_id, t.created_at, t.deleted_at",
		afterName,
		limit,
		func(rows *sql.Rows) (*tokenizationDomain.TokenizationKey, error) {
			var key tokenizationDomain.TokenizationKey
			var formatType string
			if err := rows.Scan(
				&key.ID,
				&key.Name,
				&key.Version,
				&formatType,
				&key.IsDeterministic,
				&key.Salt,
				&key.DekID,
				&key.CreatedAt,
				&key.DeletedAt,
			); err != nil {
				return nil, err
			}
			key.FormatType = tokenizationDomain.FormatType(formatType)
			return &key, nil
		},
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list tokenization keys with cursor")
	}
	return records, nil
}

// HardDelete permanently removes soft-deleted tokenization keys and their associated tokens.
func (p *TokenizationKeyRepository) HardDelete(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	querier := database.GetTx(ctx, p.db)

	if dryRun {
		query := `SELECT COUNT(*) FROM tokenization_keys WHERE deleted_at IS NOT NULL AND deleted_at < $1`
		var count int64
		err := querier.QueryRowContext(ctx, query, olderThan).Scan(&count)
		if err != nil {
			return 0, apperrors.Wrap(err, "failed to count tokenization keys for hard delete")
		}
		return count, nil
	}

	// Delete associated tokens first
	//nolint:gosec // false positive: this is a SQL query, not a credential
	deleteTokensQuery := `
		DELETE FROM tokenization_tokens 
		WHERE tokenization_key_id IN (
			SELECT id FROM tokenization_keys WHERE deleted_at IS NOT NULL AND deleted_at < $1
		)`
	_, err := querier.ExecContext(ctx, deleteTokensQuery, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to delete associated tokens")
	}

	// Delete keys
	deleteKeysQuery := `DELETE FROM tokenization_keys WHERE deleted_at IS NOT NULL AND deleted_at < $1`
	result, err := querier.ExecContext(ctx, deleteKeysQuery, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to hard delete tokenization keys")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to get rows affected for hard delete")
	}

	return count, nil
}

// NewTokenizationKeyRepository creates a new PostgreSQL tokenization key repository instance.
func NewTokenizationKeyRepository(db *sql.DB) *TokenizationKeyRepository {
	return &TokenizationKeyRepository{db: db}
}

// TokenRepository implements token persistence for PostgreSQL databases.
type TokenRepository struct {
	db *sql.DB
}

// Create inserts a new token mapping into the PostgreSQL database.
func (p *TokenRepository) Create(
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
		// Check for unique constraint violation (PostgreSQL error code 23505)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return apperrors.ErrConflict
		}
		return apperrors.Wrap(err, "failed to create token")
	}
	return nil
}

// CreateBatch inserts multiple token mappings into the PostgreSQL database.
func (p *TokenRepository) CreateBatch(
	ctx context.Context,
	tokens []*tokenizationDomain.Token,
) error {
	if len(tokens) == 0 {
		return nil
	}

	for _, token := range tokens {
		if err := p.Create(ctx, token); err != nil {
			return apperrors.Wrap(err, "failed to create token in batch")
		}
	}
	return nil
}

// GetByToken retrieves a token mapping by its token string.
func (p *TokenRepository) GetByToken(
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

// GetBatchByTokens retrieves multiple token mappings by their token strings.
func (p *TokenRepository) GetBatchByTokens(
	ctx context.Context,
	tokenStrings []string,
) ([]*tokenizationDomain.Token, error) {
	if len(tokenStrings) == 0 {
		return []*tokenizationDomain.Token{}, nil
	}

	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at 
			  FROM tokenization_tokens 
			  WHERE token = ANY($1)`

	rows, err := querier.QueryContext(ctx, query, pq.Array(tokenStrings))
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get tokens by batch")
	}
	defer func() {
		_ = rows.Close()
	}()

	var tokens []*tokenizationDomain.Token
	for rows.Next() {
		var token tokenizationDomain.Token
		var metadataJSON []byte

		err := rows.Scan(
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
			return nil, apperrors.Wrap(err, "failed to scan token")
		}

		// Parse metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &token.Metadata); err != nil {
				return nil, apperrors.Wrap(err, "failed to unmarshal metadata")
			}
		}

		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "error iterating tokens")
	}

	if tokens == nil {
		tokens = make([]*tokenizationDomain.Token, 0)
	}

	return tokens, nil
}

// GetByValueHash retrieves a token by its value hash (for deterministic mode).
func (p *TokenRepository) GetByValueHash(
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
func (p *TokenRepository) Revoke(ctx context.Context, tokenStr string) error {
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
func (p *TokenRepository) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
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
func (p *TokenRepository) CountExpired(ctx context.Context, olderThan time.Time) (int64, error) {
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

// NewTokenRepository creates a new PostgreSQL token repository instance.
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}
