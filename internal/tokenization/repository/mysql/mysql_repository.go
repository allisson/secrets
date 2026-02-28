package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// MySQLTokenizationKeyRepository implements tokenization key persistence for MySQL databases.
type MySQLTokenizationKeyRepository struct {
	db *sql.DB
}

// Create inserts a new tokenization key into the MySQL database.
func (m *MySQLTokenizationKeyRepository) Create(
	ctx context.Context,
	key *tokenizationDomain.TokenizationKey,
) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO tokenization_keys (id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	id, err := key.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal tokenization key id")
	}

	dekID, err := key.DekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		key.Name,
		key.Version,
		key.FormatType,
		key.IsDeterministic,
		dekID,
		key.CreatedAt,
		key.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create tokenization key")
	}
	return nil
}

// Delete soft-deletes a tokenization key by setting its deleted_at timestamp.
func (m *MySQLTokenizationKeyRepository) Delete(ctx context.Context, keyID uuid.UUID) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE tokenization_keys SET deleted_at = NOW() WHERE id = ?`

	id, err := keyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal tokenization key id")
	}

	_, err = querier.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a tokenization key by name.
func (m *MySQLTokenizationKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE name = ? AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var key tokenizationDomain.TokenizationKey
	var id, dekID []byte
	var formatType string

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&id,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&dekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name")
	}

	if err := key.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
	}

	if err := key.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// Get retrieves a tokenization key by its ID.
func (m *MySQLTokenizationKeyRepository) Get(
	ctx context.Context,
	keyID uuid.UUID,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE id = ? AND deleted_at IS NULL`

	id, err := keyID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal tokenization key id")
	}

	var key tokenizationDomain.TokenizationKey
	var keyIDBinary, dekID []byte
	var formatType string

	err = querier.QueryRowContext(ctx, query, id).Scan(
		&keyIDBinary,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&dekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by id")
	}

	if err := key.ID.UnmarshalBinary(keyIDBinary); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
	}

	if err := key.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// GetByNameAndVersion retrieves a specific version of a tokenization key by name and version.
func (m *MySQLTokenizationKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, format_type, is_deterministic, dek_id, created_at, deleted_at 
			  FROM tokenization_keys 
			  WHERE name = ? AND version = ? AND deleted_at IS NULL`

	var key tokenizationDomain.TokenizationKey
	var id, dekID []byte
	var formatType string

	err := querier.QueryRowContext(ctx, query, name, version).Scan(
		&id,
		&key.Name,
		&key.Version,
		&formatType,
		&key.IsDeterministic,
		&dekID,
		&key.CreatedAt,
		&key.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tokenizationDomain.ErrTokenizationKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name and version")
	}

	if err := key.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
	}

	if err := key.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	key.FormatType = tokenizationDomain.FormatType(formatType)
	return &key, nil
}

// List retrieves tokenization keys ordered by name ascending with pagination.
// Returns the latest version for each key.
func (m *MySQLTokenizationKeyRepository) List(
	ctx context.Context,
	offset, limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `
		SELECT tk.id, tk.name, tk.version, tk.format_type, tk.is_deterministic, tk.dek_id, tk.created_at, tk.deleted_at 
		FROM tokenization_keys tk
		INNER JOIN (
			SELECT name, MAX(version) as max_version
			FROM tokenization_keys
			WHERE deleted_at IS NULL
			GROUP BY name
			ORDER BY name ASC
			LIMIT ? OFFSET ?
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
		var id, dekID []byte
		var formatType string

		err := rows.Scan(
			&id,
			&key.Name,
			&key.Version,
			&formatType,
			&key.IsDeterministic,
			&dekID,
			&key.CreatedAt,
			&key.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan tokenization key")
		}

		if err := key.ID.UnmarshalBinary(id); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
		}

		if err := key.DekID.UnmarshalBinary(dekID); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
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

// NewMySQLTokenizationKeyRepository creates a new MySQL tokenization key repository instance.
func NewMySQLTokenizationKeyRepository(db *sql.DB) *MySQLTokenizationKeyRepository {
	return &MySQLTokenizationKeyRepository{db: db}
}

// MySQLTokenRepository implements token persistence for MySQL databases.
type MySQLTokenRepository struct {
	db *sql.DB
}

// Create inserts a new token mapping into the MySQL database.
func (m *MySQLTokenRepository) Create(
	ctx context.Context,
	token *tokenizationDomain.Token,
) error {
	querier := database.GetTx(ctx, m.db)

	// Convert metadata to JSON
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
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	id, err := token.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal token id")
	}

	keyID, err := token.TokenizationKeyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal tokenization key id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		keyID,
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
		// Check for duplicate entry error (MySQL error number 1062)
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}
		return apperrors.Wrap(err, "failed to create token")
	}
	return nil
}

// GetByToken retrieves a token mapping by its token string.
func (m *MySQLTokenRepository) GetByToken(
	ctx context.Context,
	tokenStr string,
) (*tokenizationDomain.Token, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at 
			  FROM tokenization_tokens 
			  WHERE token = ?`

	var token tokenizationDomain.Token
	var id, keyID []byte
	var metadataJSON []byte

	err := querier.QueryRowContext(ctx, query, tokenStr).Scan(
		&id,
		&keyID,
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

	if err := token.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal token id")
	}

	if err := token.TokenizationKeyID.UnmarshalBinary(keyID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
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
func (m *MySQLTokenRepository) GetByValueHash(
	ctx context.Context,
	keyID uuid.UUID,
	valueHash string,
) (*tokenizationDomain.Token, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, tokenization_key_id, token, value_hash, ciphertext, nonce, metadata, created_at, expires_at, revoked_at 
			  FROM tokenization_tokens 
			  WHERE tokenization_key_id = ? AND value_hash = ?`

	keyIDBinary, err := keyID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal tokenization key id")
	}

	var token tokenizationDomain.Token
	var id, tokenKeyID []byte
	var metadataJSON []byte

	err = querier.QueryRowContext(ctx, query, keyIDBinary, valueHash).Scan(
		&id,
		&tokenKeyID,
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

	if err := token.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal token id")
	}

	if err := token.TokenizationKeyID.UnmarshalBinary(tokenKeyID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal tokenization key id")
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
func (m *MySQLTokenRepository) Revoke(ctx context.Context, tokenStr string) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE tokenization_tokens SET revoked_at = NOW() WHERE token = ?`

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
func (m *MySQLTokenRepository) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	if olderThan.IsZero() {
		return 0, apperrors.New("olderThan timestamp cannot be zero")
	}

	querier := database.GetTx(ctx, m.db)

	query := `DELETE FROM tokenization_tokens WHERE expires_at < ?`

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
func (m *MySQLTokenRepository) CountExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	if olderThan.IsZero() {
		return 0, apperrors.New("olderThan timestamp cannot be zero")
	}

	querier := database.GetTx(ctx, m.db)

	query := `SELECT COUNT(*) FROM tokenization_tokens WHERE expires_at < ?`

	var count int64
	err := querier.QueryRowContext(ctx, query, olderThan).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to count expired tokens")
	}

	return count, nil
}

// NewMySQLTokenRepository creates a new MySQL token repository instance.
func NewMySQLTokenRepository(db *sql.DB) *MySQLTokenRepository {
	return &MySQLTokenRepository{db: db}
}
