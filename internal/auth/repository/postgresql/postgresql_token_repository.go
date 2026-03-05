package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLTokenRepository implements Token persistence for PostgreSQL.
// Uses native UUID types with transaction support via database.GetTx().
type PostgreSQLTokenRepository struct {
	db *sql.DB
}

// Create inserts a new Token into the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns an error if database insertion fails.
func (p *PostgreSQLTokenRepository) Create(ctx context.Context, token *authDomain.Token) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		token.ID,
		token.TokenHash,
		token.ClientID,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create token")
	}
	return nil
}

// Update modifies an existing Token in the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns an error if database update fails.
func (p *PostgreSQLTokenRepository) Update(ctx context.Context, token *authDomain.Token) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokens 
			  SET token_hash = $1, 
			  	  client_id = $2,
				  expires_at = $3,
				  revoked_at = $4,
				  created_at = $5
			  WHERE id = $6`

	_, err := querier.ExecContext(
		ctx,
		query,
		token.TokenHash,
		token.ClientID,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
		token.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update token")
	}

	return nil
}

// Get retrieves a Token by ID from the PostgreSQL database. Uses transaction support
// via database.GetTx(). Returns ErrTokenNotFound if the token doesn't exist, or an error
// if database query fails.
func (p *PostgreSQLTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*authDomain.Token, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, token_hash, client_id, expires_at, revoked_at, created_at 
			  FROM tokens WHERE id = $1`

	var token authDomain.Token

	err := querier.QueryRowContext(ctx, query, tokenID).Scan(
		&token.ID,
		&token.TokenHash,
		&token.ClientID,
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

	return &token, nil
}

// GetByTokenHash retrieves a Token by token hash from the PostgreSQL database. Uses transaction
// support via database.GetTx(). Returns ErrTokenNotFound if the token doesn't exist, or an error
// if database query fails.
func (p *PostgreSQLTokenRepository) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*authDomain.Token, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, token_hash, client_id, expires_at, revoked_at, created_at 
			  FROM tokens WHERE token_hash = $1`

	var token authDomain.Token

	err := querier.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.TokenHash,
		&token.ClientID,
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

	return &token, nil
}

// RevokeByTokenID marks a specific token as revoked by setting its revoked_at timestamp.
func (p *PostgreSQLTokenRepository) RevokeByTokenID(ctx context.Context, tokenID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokens SET revoked_at = $1 WHERE id = $2`

	_, err := querier.ExecContext(ctx, query, time.Now().UTC(), tokenID)
	if err != nil {
		return apperrors.Wrap(err, "failed to revoke token")
	}

	return nil
}

// RevokeByClientID marks all active tokens for a specific client as revoked.
func (p *PostgreSQLTokenRepository) RevokeByClientID(ctx context.Context, clientID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE tokens SET revoked_at = $1 WHERE client_id = $2 AND revoked_at IS NULL`

	_, err := querier.ExecContext(ctx, query, time.Now().UTC(), clientID)
	if err != nil {
		return apperrors.Wrap(err, "failed to revoke tokens by client id")
	}

	return nil
}

// PurgeExpiredAndRevoked permanently deletes tokens that are either expired or revoked
// and were created before the specified timestamp. Returns the number of deleted tokens.
func (p *PostgreSQLTokenRepository) PurgeExpiredAndRevoked(
	ctx context.Context,
	olderThan time.Time,
) (int64, error) {
	querier := database.GetTx(ctx, p.db)

	query := `DELETE FROM tokens WHERE expires_at < $1 OR revoked_at < $1`

	result, err := querier.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to purge tokens")
	}

	return result.RowsAffected()
}

// NewPostgreSQLTokenRepository creates a new PostgreSQL Token repository.
func NewPostgreSQLTokenRepository(db *sql.DB) *PostgreSQLTokenRepository {
	return &PostgreSQLTokenRepository{db: db}
}
