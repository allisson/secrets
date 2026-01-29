package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/go-project-template/internal/outbox/domain"
	"github.com/allisson/go-project-template/internal/testutil"
)

func TestNewPostgreSQLOutboxEventRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestPostgreSQLOutboxEventRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `{"id": 1}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	err := repo.Create(ctx, event)
	assert.NoError(t, err)

	// Verify the event was created
	events, err := repo.GetPendingEvents(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, event.ID, events[0].ID)
	assert.Equal(t, event.EventType, events[0].EventType)
}

func TestPostgreSQLOutboxEventRepository_GetPendingEvents(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	uuid2 := uuid.Must(uuid.NewV7())

	event1 := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `{"id": 1}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}
	event2 := &domain.OutboxEvent{
		ID:        uuid2,
		EventType: "user.created",
		Payload:   `{"id": 2}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	// Create events
	err := repo.Create(ctx, event1)
	require.NoError(t, err)
	err = repo.Create(ctx, event2)
	require.NoError(t, err)

	// Get pending events
	events, err := repo.GetPendingEvents(ctx, 10)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 2)
	assert.Equal(t, uuid1, events[0].ID)
	assert.Equal(t, uuid2, events[1].ID)
}

func TestPostgreSQLOutboxEventRepository_GetPendingEvents_Empty(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	ctx := context.Background()

	events, err := repo.GetPendingEvents(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestPostgreSQLOutboxEventRepository_Update(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `{"id": 1}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	// Create event
	err := repo.Create(ctx, event)
	require.NoError(t, err)

	// Update event
	now := time.Now()
	event.Status = domain.OutboxEventStatusProcessed
	event.ProcessedAt = &now

	err = repo.Update(ctx, event)
	assert.NoError(t, err)

	// Verify no pending events
	events, err := repo.GetPendingEvents(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestPostgreSQLOutboxEventRepository_Update_Error(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLOutboxEventRepository(db)
	ctx := context.Background()

	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `{"id": 1}`,
		Status:    domain.OutboxEventStatusProcessed,
	}

	// Update non-existent event - should not return error but also shouldn't affect any rows
	err := repo.Update(ctx, event)
	assert.NoError(t, err)
}
