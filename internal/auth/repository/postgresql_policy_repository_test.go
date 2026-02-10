package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/testutil"
)

func TestNewPostgreSQLPolicyRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLPolicyRepository{}, repo)
}

func TestPostgreSQLPolicyRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read", "write"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Verify the policy was created by retrieving it
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	require.NoError(t, err)

	assert.Equal(t, policy.ID, retrievedPolicy.ID)
	assert.Equal(t, policy.Name, retrievedPolicy.Name)
	assert.Equal(t, policy.Document.Path, retrievedPolicy.Document.Path)
	assert.Equal(t, policy.Document.Capabilities, retrievedPolicy.Document.Capabilities)
	assert.WithinDuration(t, policy.CreatedAt, retrievedPolicy.CreatedAt, time.Second)
}

func TestPostgreSQLPolicyRepository_Create_WithComplexDocument(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "admin-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/admin/*",
			Capabilities: []string{"read", "write", "delete", "list"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Verify the policy was created with all capabilities
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	require.NoError(t, err)
	assert.Equal(t, 4, len(retrievedPolicy.Document.Capabilities))
	assert.Contains(t, retrievedPolicy.Document.Capabilities, "delete")
	assert.Contains(t, retrievedPolicy.Document.Capabilities, "list")
}

func TestPostgreSQLPolicyRepository_Create_MultiplePolicies(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create first policy
	policy1 := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "policy-1",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/app1/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create second policy
	policy2 := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "policy-2",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/app2/*",
			Capabilities: []string{"write"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err = repo.Create(ctx, policy2)
	require.NoError(t, err)

	// Verify both policies can be retrieved
	retrievedPolicy1, err := repo.Get(ctx, policy1.Name)
	require.NoError(t, err)
	assert.Equal(t, policy1.Name, retrievedPolicy1.Name)

	retrievedPolicy2, err := repo.Get(ctx, policy2.Name)
	require.NoError(t, err)
	assert.Equal(t, policy2.Name, retrievedPolicy2.Name)
}

func TestPostgreSQLPolicyRepository_Update(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create initial policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "update-test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/original/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Update the policy
	policy.Document.Path = "/updated/*"
	policy.Document.Capabilities = []string{"read", "write", "delete"}

	err = repo.Update(ctx, policy)
	require.NoError(t, err)

	// Verify the update
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	require.NoError(t, err)

	assert.Equal(t, policy.Name, retrievedPolicy.Name)
	assert.Equal(t, "/updated/*", retrievedPolicy.Document.Path)
	assert.Equal(t, 3, len(retrievedPolicy.Document.Capabilities))
	assert.Contains(t, retrievedPolicy.Document.Capabilities, "delete")
}

func TestPostgreSQLPolicyRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Try to update a non-existent policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "non-existent-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/test/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err := repo.Update(ctx, policy)
	assert.NoError(t, err)
}

func TestPostgreSQLPolicyRepository_Get_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "get-test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/api/*",
			Capabilities: []string{"read", "write"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Retrieve the policy
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	require.NoError(t, err)
	require.NotNil(t, retrievedPolicy)

	assert.Equal(t, policy.ID, retrievedPolicy.ID)
	assert.Equal(t, policy.Name, retrievedPolicy.Name)
	assert.Equal(t, policy.Document.Path, retrievedPolicy.Document.Path)
	assert.Equal(t, policy.Document.Capabilities, retrievedPolicy.Document.Capabilities)
	assert.WithinDuration(t, policy.CreatedAt, retrievedPolicy.CreatedAt, time.Second)
}

func TestPostgreSQLPolicyRepository_Get_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Try to get a non-existent policy
	policy, err := repo.Get(ctx, "non-existent-policy")

	assert.Error(t, err)
	assert.Nil(t, policy)
	assert.ErrorIs(t, err, authDomain.ErrPolicyNotFound)
}

func TestPostgreSQLPolicyRepository_Delete_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "delete-test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/temp/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Delete the policy
	err = repo.Delete(ctx, policy.Name)
	require.NoError(t, err)

	// Verify the policy was deleted
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	assert.Error(t, err)
	assert.Nil(t, retrievedPolicy)
	assert.ErrorIs(t, err, authDomain.ErrPolicyNotFound)
}

func TestPostgreSQLPolicyRepository_Delete_NonExistent(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Try to delete a non-existent policy
	err := repo.Delete(ctx, "non-existent-policy")

	assert.Error(t, err)
	assert.ErrorIs(t, err, authDomain.ErrPolicyNotFound)
}

func TestPostgreSQLPolicyRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "tx-test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/tx/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	// Test rollback behavior using a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create policy within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO policies (id, name, document, created_at) VALUES ($1, $2, $3, $4)`,
		policy.ID,
		policy.Name,
		`{"path":"/tx/*","capabilities":["read"]}`,
		policy.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the policy was not created (rollback worked)
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	assert.Error(t, err)
	assert.Nil(t, retrievedPolicy)
	assert.ErrorIs(t, err, authDomain.ErrPolicyNotFound)
}

func TestPostgreSQLPolicyRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create initial policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "tx-update-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/original/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE policies SET document = $1, created_at = $2 WHERE name = $3`,
		`{"path":"/updated/*","capabilities":["read","write"]}`,
		policy.CreatedAt,
		policy.Name,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the policy was not updated (rollback worked)
	retrievedPolicy, err := repo.Get(ctx, policy.Name)
	require.NoError(t, err)
	assert.Equal(t, "/original/*", retrievedPolicy.Document.Path)
	assert.Equal(t, 1, len(retrievedPolicy.Document.Capabilities))
}

func TestPostgreSQLPolicyRepository_Get_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create a policy outside transaction
	policy1 := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "policy-outside-tx",
		Document: authDomain.PolicyDocument{
			Path:         "/outside/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, policy1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another policy inside transaction
	time.Sleep(time.Millisecond)
	policy2 := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "policy-inside-tx",
		Document: authDomain.PolicyDocument{
			Path:         "/inside/*",
			Capabilities: []string{"write"},
		},
		CreatedAt: time.Now().UTC(),
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO policies (id, name, document, created_at) VALUES ($1, $2, $3, $4)`,
		policy2.ID,
		policy2.Name,
		`{"path":"/inside/*","capabilities":["write"]}`,
		policy2.CreatedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see both policies
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM policies`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Get outside transaction should also see both policies
	retrievedPolicy1, err := repo.Get(ctx, policy1.Name)
	require.NoError(t, err)
	assert.Equal(t, policy1.Name, retrievedPolicy1.Name)

	retrievedPolicy2, err := repo.Get(ctx, policy2.Name)
	require.NoError(t, err)
	assert.Equal(t, policy2.Name, retrievedPolicy2.Name)
}
