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

// MySQLUserRepository handles user persistence for MySQL
type MySQLUserRepository struct {
	db *sql.DB
}

// NewMySQLUserRepository creates a new MySQLUserRepository
func NewMySQLUserRepository(db *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{
		db: db,
	}
}

// Create inserts a new user
func (r *MySQLUserRepository) Create(ctx context.Context, user *domain.User) error {
	querier := database.GetTx(ctx, r.db)

	query := `INSERT INTO users (id, name, email, password, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, NOW(), NOW())`

	// Convert UUID to bytes for MySQL BINARY(16)
	uuidBytes, err := user.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal UUID")
	}

	_, err = querier.ExecContext(ctx, query, uuidBytes, user.Name, user.Email, user.Password)
	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if isMySQLUniqueViolation(err) {
			return domain.ErrUserAlreadyExists
		}
		return apperrors.Wrap(err, "failed to create user")
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *MySQLUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, name, email, password, created_at, updated_at 
			  FROM users WHERE id = ?`

	// Convert UUID to bytes for MySQL BINARY(16)
	uuidBytes, err := id.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal UUID")
	}

	var idBytes []byte
	err = querier.QueryRowContext(ctx, query, uuidBytes).Scan(
		&idBytes, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get user by id")
	}

	// Convert bytes back to UUID
	if err := user.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal UUID")
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *MySQLUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	querier := database.GetTx(ctx, r.db)

	query := `SELECT id, name, email, password, created_at, updated_at 
			  FROM users WHERE email = ?`

	var idBytes []byte
	err := querier.QueryRowContext(ctx, query, email).Scan(
		&idBytes, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get user by email")
	}

	// Convert bytes back to UUID
	if err := user.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal UUID")
	}

	return &user, nil
}

// isMySQLUniqueViolation checks if the error is a MySQL unique constraint violation
func isMySQLUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	// MySQL: "Error 1062: Duplicate entry"
	return strings.Contains(errMsg, "duplicate entry") || strings.Contains(errMsg, "1062")
}
