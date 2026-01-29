package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/allisson/go-project-template/internal/errors"
	"github.com/allisson/go-project-template/internal/testutil"
	"github.com/allisson/go-project-template/internal/user/domain"
)

func TestNewPostgreSQLUserRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestPostgreSQLUserRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	user := &domain.User{
		ID:       uuid1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashed_password",
	}

	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	// Verify the user was created
	createdUser, err := repo.GetByID(ctx, uuid1)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, createdUser.ID)
	assert.Equal(t, user.Name, createdUser.Name)
	assert.Equal(t, user.Email, createdUser.Email)
	assert.Equal(t, user.Password, createdUser.Password)
	assert.False(t, createdUser.CreatedAt.IsZero())
	assert.False(t, createdUser.UpdatedAt.IsZero())
}

func TestPostgreSQLUserRepository_GetByID(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	expectedUser := &domain.User{
		ID:       uuid1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashed_password",
	}

	// Create the user first
	err := repo.Create(ctx, expectedUser)
	require.NoError(t, err)

	// Get the user by ID
	user, err := repo.GetByID(ctx, uuid1)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Name, user.Name)
	assert.Equal(t, expectedUser.Email, user.Email)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestPostgreSQLUserRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	ctx := context.Background()

	notFoundUUID := uuid.Must(uuid.NewV7())
	user, err := repo.GetByID(ctx, notFoundUUID)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Is(err, domain.ErrUserNotFound))
}

func TestPostgreSQLUserRepository_GetByEmail(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	expectedUser := &domain.User{
		ID:       uuid1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashed_password",
	}

	// Create the user first
	err := repo.Create(ctx, expectedUser)
	require.NoError(t, err)

	// Get the user by email
	user, err := repo.GetByEmail(ctx, "john@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)
}

func TestPostgreSQLUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLUserRepository(db)
	ctx := context.Background()

	user, err := repo.GetByEmail(ctx, "notfound@example.com")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Is(err, domain.ErrUserNotFound))
}
