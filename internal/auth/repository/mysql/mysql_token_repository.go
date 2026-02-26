package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLTokenRepository implements Token persistence for MySQL.
// Uses BINARY(16) for UUIDs with transaction support via database.GetTx().
type MySQLTokenRepository struct {
	db *sql.DB
}

// Create inserts a new Token into the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns an error if UUID marshaling
// or database insertion fails.
func (m *MySQLTokenRepository) Create(ctx context.Context, token *authDomain.Token) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := token.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal token id")
	}

	clientID, err := token.ClientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		token.TokenHash,
		clientID,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create token")
	}
	return nil
}

// Update modifies an existing Token in the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns an error if UUID marshaling
// or database update fails.
func (m *MySQLTokenRepository) Update(ctx context.Context, token *authDomain.Token) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE tokens 
			  SET token_hash = ?, 
			  	  client_id = ?,
				  expires_at = ?,
				  revoked_at = ?,
				  created_at = ?
			  WHERE id = ?`

	id, err := token.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal token id")
	}

	clientID, err := token.ClientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		token.TokenHash,
		clientID,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update token")
	}

	return nil
}

// Get retrieves a Token by ID from the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns ErrTokenNotFound if the token
// doesn't exist, or an error if UUID unmarshaling or database query fails.
func (m *MySQLTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*authDomain.Token, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, token_hash, client_id, expires_at, revoked_at, created_at 
			  FROM tokens WHERE id = ?`

	id, err := tokenID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal token id")
	}

	var token authDomain.Token
	var idBytes []byte
	var clientIDBytes []byte

	err = querier.QueryRowContext(ctx, query, id).Scan(
		&idBytes,
		&token.TokenHash,
		&clientIDBytes,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authDomain.ErrTokenNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get token")
	}

	if err := token.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal token id")
	}

	if err := token.ClientID.UnmarshalBinary(clientIDBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal client id")
	}

	return &token, nil
}

// GetByTokenHash retrieves a Token by token hash from the MySQL database using BINARY(16) for UUIDs.
// Uses transaction support via database.GetTx(). Returns ErrTokenNotFound if the token doesn't exist,
// or an error if UUID unmarshaling or database query fails.
func (m *MySQLTokenRepository) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*authDomain.Token, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, token_hash, client_id, expires_at, revoked_at, created_at 
			  FROM tokens WHERE token_hash = ?`

	var token authDomain.Token
	var idBytes []byte
	var clientIDBytes []byte

	err := querier.QueryRowContext(ctx, query, tokenHash).Scan(
		&idBytes,
		&token.TokenHash,
		&clientIDBytes,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authDomain.ErrTokenNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get token by hash")
	}

	if err := token.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal token id")
	}

	if err := token.ClientID.UnmarshalBinary(clientIDBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal client id")
	}

	return &token, nil
}

// NewMySQLTokenRepository creates a new MySQL Token repository.
func NewMySQLTokenRepository(db *sql.DB) *MySQLTokenRepository {
	return &MySQLTokenRepository{db: db}
}
