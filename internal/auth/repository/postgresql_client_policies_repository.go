package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLClientPoliciesRepository implements ClientPolicies persistence for PostgreSQL.
// Uses native UUID types with transaction support via database.GetTx().
type PostgreSQLClientPoliciesRepository struct {
	db *sql.DB
}

// Create inserts a new ClientPolicies relationship into the PostgreSQL database.
func (p *PostgreSQLClientPoliciesRepository) Create(
	ctx context.Context,
	clientPolicies *authDomain.ClientPolicies,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO client_policies (client_id, policy_id) VALUES ($1, $2)`

	_, err := querier.ExecContext(ctx, query, clientPolicies.ClientID, clientPolicies.PolicyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to create client policies")
	}
	return nil
}

// Delete removes a ClientPolicies relationship from the PostgreSQL database.
func (p *PostgreSQLClientPoliciesRepository) Delete(
	ctx context.Context,
	clientID uuid.UUID,
	policyID uuid.UUID,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `DELETE FROM client_policies WHERE client_id = $1 AND policy_id = $2`

	result, err := querier.ExecContext(ctx, query, clientID, policyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete client policies")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return authDomain.ErrClientPoliciesNotFound
	}

	return nil
}

// NewPostgreSQLClientPoliciesRepository creates a new PostgreSQL ClientPolicies repository.
func NewPostgreSQLClientPoliciesRepository(db *sql.DB) *PostgreSQLClientPoliciesRepository {
	return &PostgreSQLClientPoliciesRepository{db: db}
}
