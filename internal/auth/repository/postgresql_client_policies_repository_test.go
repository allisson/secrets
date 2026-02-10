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

func TestNewPostgreSQLClientPoliciesRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLClientPoliciesRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLClientPoliciesRepository{}, repo)
}

func TestPostgreSQLClientPoliciesRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err = policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	// Create the client-policy relationship
	clientPolicies := &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy.ID,
	}

	err = repo.Create(ctx, clientPolicies)
	require.NoError(t, err)

	// Verify the relationship was created by querying the database
	var count int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM client_policies WHERE client_id = $1 AND policy_id = $2`,
		client.ID,
		policy.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostgreSQLClientPoliciesRepository_Create_Duplicate(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err = policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	// Create the client-policy relationship
	clientPolicies := &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy.ID,
	}

	err = repo.Create(ctx, clientPolicies)
	require.NoError(t, err)

	// Try to create the same relationship again - should fail
	err = repo.Create(ctx, clientPolicies)
	assert.Error(t, err)
}

func TestPostgreSQLClientPoliciesRepository_Create_InvalidClientID(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err := policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Try to create a relationship with a non-existent client ID
	clientPolicies := &authDomain.ClientPolicies{
		ClientID: uuid.Must(uuid.NewV7()),
		PolicyID: policy.ID,
	}

	err = repo.Create(ctx, clientPolicies)
	assert.Error(t, err)
}

func TestPostgreSQLClientPoliciesRepository_Create_InvalidPolicyID(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Try to create a relationship with a non-existent policy ID
	clientPolicies := &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: uuid.Must(uuid.NewV7()),
	}

	err = repo.Create(ctx, clientPolicies)
	assert.Error(t, err)
}

func TestPostgreSQLClientPoliciesRepository_Create_MultiplePoliciesToOneClient(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

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
	err = policyRepo.Create(ctx, policy1)
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
	err = policyRepo.Create(ctx, policy2)
	require.NoError(t, err)

	// Attach both policies to the same client
	err = repo.Create(ctx, &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy1.ID,
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy2.ID,
	})
	require.NoError(t, err)

	// Verify both relationships exist
	var count int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM client_policies WHERE client_id = $1`,
		client.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPostgreSQLClientPoliciesRepository_Delete_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err = policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	// Create the client-policy relationship
	clientPolicies := &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy.ID,
	}

	err = repo.Create(ctx, clientPolicies)
	require.NoError(t, err)

	// Delete the relationship
	err = repo.Delete(ctx, client.ID, policy.ID)
	require.NoError(t, err)

	// Verify the relationship was deleted
	var count int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM client_policies WHERE client_id = $1 AND policy_id = $2`,
		client.ID,
		policy.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPostgreSQLClientPoliciesRepository_Delete_NonExistent(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Try to delete a non-existent relationship
	clientID := uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())

	err := repo.Delete(ctx, clientID, policyID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, authDomain.ErrClientPoliciesNotFound)
}

func TestPostgreSQLClientPoliciesRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err = policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create relationship within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO client_policies (client_id, policy_id) VALUES ($1, $2)`,
		client.ID,
		policy.ID,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the relationship was not created (rollback worked)
	var count int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM client_policies WHERE client_id = $1 AND policy_id = $2`,
		client.ID,
		policy.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPostgreSQLClientPoliciesRepository_Delete_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	clientRepo := NewPostgreSQLClientRepository(db)
	policyRepo := NewPostgreSQLPolicyRepository(db)
	repo := NewPostgreSQLClientPoliciesRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create a policy
	policy := &authDomain.Policy{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "test-policy",
		Document: authDomain.PolicyDocument{
			Path:         "/secrets/*",
			Capabilities: []string{"read"},
		},
		CreatedAt: time.Now().UTC(),
	}
	err = policyRepo.Create(ctx, policy)
	require.NoError(t, err)

	// Create the client-policy relationship
	err = repo.Create(ctx, &authDomain.ClientPolicies{
		ClientID: client.ID,
		PolicyID: policy.ID,
	})
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Delete within transaction
	_, err = tx.ExecContext(
		ctx,
		`DELETE FROM client_policies WHERE client_id = $1 AND policy_id = $2`,
		client.ID,
		policy.ID,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the relationship still exists (rollback worked)
	var count int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM client_policies WHERE client_id = $1 AND policy_id = $2`,
		client.ID,
		policy.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
