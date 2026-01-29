// Package repository provides data persistence implementations for user entities.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/allisson/go-project-template/internal/database"
	"github.com/allisson/go-project-template/internal/user/domain"

	apperrors "github.com/allisson/go-project-template/internal/errors"
)

// PostgreSQLUserRepository handles user persistence for PostgreSQL
type PostgreSQLUserRepository struct {
	db *sql.DB
}

// NewPostgreSQLUserRepository creates a new PostgreSQLUserRepository
func NewPostgreSQLUserRepository(db *sql.DB) *PostgreSQLUserRepository {
	return &PostgreSQLUserRepository{
		db: db,
	}
}

// Create inserts a new user
func (r *PostgreSQLUserRepository) Create(ctx context.Context, user *domain.User) error {
	querier := database.GetTx(ctx, r.db)

	query := `INSERT INTO users (id, name, email, password, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, NOW(), NOW())`

	_, err := querier.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.Password)
	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if isPostgreSQLUniqueViolation(err) {
			return domain.ErrUserAlreadyExists
		}
		return apperrors.Wrap(err, "failed to create user")
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *PostgreSQLUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, name, email, password, created_at, updated_at 
			  FROM users WHERE id = $1`

	err := querier.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get user by id")
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgreSQLUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, name, email, password, created_at, updated_at 
			  FROM users WHERE email = $1`

	err := querier.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get user by email")
	}

	return &user, nil
}

// isPostgreSQLUniqueViolation checks if the error is a PostgreSQL unique constraint violation
func isPostgreSQLUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	// PostgreSQL: "duplicate key value violates unique constraint" or "pq: duplicate key"
	return strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint")
}
