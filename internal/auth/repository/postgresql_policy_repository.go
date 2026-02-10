package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLPolicyRepository implements Policy persistence for PostgreSQL.
// Uses native UUID types with transaction support via database.GetTx().
type PostgreSQLPolicyRepository struct {
	db *sql.DB
}

// Create inserts a new Policy into the PostgreSQL database.
func (p *PostgreSQLPolicyRepository) Create(ctx context.Context, policy *authDomain.Policy) error {
	querier := database.GetTx(ctx, p.db)

	documentJSON, err := json.Marshal(policy.Document)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy document")
	}

	query := `INSERT INTO policies (id, name, document, created_at) 
			  VALUES ($1, $2, $3, $4)`

	_, err = querier.ExecContext(
		ctx,
		query,
		policy.ID,
		policy.Name,
		documentJSON,
		policy.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create policy")
	}
	return nil
}

// Update modifies an existing Policy in the PostgreSQL database.
func (p *PostgreSQLPolicyRepository) Update(ctx context.Context, policy *authDomain.Policy) error {
	querier := database.GetTx(ctx, p.db)

	documentJSON, err := json.Marshal(policy.Document)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy document")
	}

	query := `UPDATE policies 
			  SET document = $1, 
			  	  created_at = $2
			  WHERE name = $3`

	_, err = querier.ExecContext(
		ctx,
		query,
		documentJSON,
		policy.CreatedAt,
		policy.Name,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update policy")
	}

	return nil
}

// Get retrieves a Policy by name from the PostgreSQL database.
func (p *PostgreSQLPolicyRepository) Get(ctx context.Context, name string) (*authDomain.Policy, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, document, created_at FROM policies WHERE name = $1`

	var policy authDomain.Policy
	var documentJSON []byte

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&policy.ID,
		&policy.Name,
		&documentJSON,
		&policy.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authDomain.ErrPolicyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get policy")
	}

	if err := json.Unmarshal(documentJSON, &policy.Document); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal policy document")
	}

	return &policy, nil
}

// Delete removes a Policy by name from the PostgreSQL database.
func (p *PostgreSQLPolicyRepository) Delete(ctx context.Context, name string) error {
	querier := database.GetTx(ctx, p.db)

	query := `DELETE FROM policies WHERE name = $1`

	result, err := querier.ExecContext(ctx, query, name)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete policy")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return authDomain.ErrPolicyNotFound
	}

	return nil
}

// NewPostgreSQLPolicyRepository creates a new PostgreSQL Policy repository.
func NewPostgreSQLPolicyRepository(db *sql.DB) *PostgreSQLPolicyRepository {
	return &PostgreSQLPolicyRepository{db: db}
}
