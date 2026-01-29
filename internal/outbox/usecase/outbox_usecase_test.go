package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/allisson/go-project-template/internal/outbox/domain"
)

// MockTxManager is a mock implementation of database.TxManager
type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	// Execute the function to test the logic inside
	return fn(ctx)
}

// MockOutboxEventRepository is a mock implementation of OutboxEventRepository
type MockOutboxEventRepository struct {
	mock.Mock
}

func (m *MockOutboxEventRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOutboxEventRepository) GetPendingEvents(
	ctx context.Context,
	limit int,
) ([]*domain.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.OutboxEvent), args.Error(1)
}

func (m *MockOutboxEventRepository) Update(ctx context.Context, event *domain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockEventProcessor is a mock implementation of EventProcessor
type MockEventProcessor struct {
	mock.Mock
}

func (m *MockEventProcessor) Process(ctx context.Context, event *domain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestNewOutboxUseCase(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	assert.NotNil(t, uc)
	assert.Equal(t, config.Interval, uc.config.Interval)
	assert.Equal(t, config.BatchSize, uc.config.BatchSize)
	assert.Equal(t, config.MaxRetries, uc.config.MaxRetries)
}

func TestOutboxUseCase_Start_ContextCancellation(t *testing.T) {
	config := Config{
		Interval:      100 * time.Millisecond,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := uc.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestOutboxUseCase_ProcessEvents_Success(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	uuid2 := uuid.Must(uuid.NewV7())
	events := []*domain.OutboxEvent{
		{
			ID:        uuid1,
			EventType: "user.created",
			Payload:   `{"user_id": 1, "name": "John Doe", "email": "john@example.com"}`,
			Status:    domain.OutboxEventStatusPending,
			Retries:   0,
		},
		{
			ID:        uuid2,
			EventType: "user.created",
			Payload:   `{"user_id": 2, "name": "Jane Doe", "email": "jane@example.com"}`,
			Status:    domain.OutboxEventStatusPending,
			Retries:   0,
		},
	}

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(events, nil)
	eventProcessor.On("Process", ctx, events[0]).Return(nil)
	eventProcessor.On("Process", ctx, events[1]).Return(nil)
	outboxRepo.On("Update", ctx, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		return e.Status == domain.OutboxEventStatusProcessed && e.ProcessedAt != nil
	})).Return(nil).Times(2)

	err := uc.ProcessEvents(ctx)

	assert.NoError(t, err)
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
	eventProcessor.AssertExpectations(t)
}

func TestOutboxUseCase_ProcessEvents_NoEvents(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	emptyEvents := []*domain.OutboxEvent{}

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(emptyEvents, nil)

	err := uc.ProcessEvents(ctx)

	assert.NoError(t, err)
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestOutboxUseCase_ProcessEvents_GetPendingError(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	getError := errors.New("database error")

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(nil, getError)

	err := uc.ProcessEvents(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestOutboxUseCase_ProcessEvents_ProcessorError(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	events := []*domain.OutboxEvent{
		{
			ID:        uuid1,
			EventType: "user.created",
			Payload:   `{"user_id": 1}`,
			Status:    domain.OutboxEventStatusPending,
			Retries:   0,
		},
	}

	processingError := errors.New("processing failed")

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(events, nil)
	eventProcessor.On("Process", ctx, events[0]).Return(processingError)
	outboxRepo.On("Update", ctx, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		return e.ID == uuid1 && e.Retries == 1 && e.LastError != nil
	})).Return(nil)

	err := uc.ProcessEvents(ctx)

	assert.NoError(t, err) // ProcessEvents should not return error, just log and update event
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
	eventProcessor.AssertExpectations(t)
}

func TestOutboxUseCase_ProcessEvents_MaxRetriesReached(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	events := []*domain.OutboxEvent{
		{
			ID:        uuid1,
			EventType: "user.created",
			Payload:   `{"user_id": 1}`,
			Status:    domain.OutboxEventStatusPending,
			Retries:   2, // Will become 3 after this attempt
		},
	}

	processingError := errors.New("processing failed")

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(events, nil)
	eventProcessor.On("Process", ctx, events[0]).Return(processingError)
	outboxRepo.On("Update", ctx, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		return e.ID == uuid1 &&
			e.Retries == 3 &&
			e.Status == domain.OutboxEventStatusFailed &&
			e.LastError != nil
	})).Return(nil)

	err := uc.ProcessEvents(ctx)

	assert.NoError(t, err)
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
	eventProcessor.AssertExpectations(t)
}

func TestOutboxUseCase_ProcessEvents_UpdateError(t *testing.T) {
	config := Config{
		Interval:      5 * time.Second,
		BatchSize:     10,
		MaxRetries:    3,
		RetryInterval: 1 * time.Minute,
	}
	txManager := &MockTxManager{}
	outboxRepo := &MockOutboxEventRepository{}
	eventProcessor := &MockEventProcessor{}

	uc := NewOutboxUseCase(config, txManager, outboxRepo, eventProcessor, nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	events := []*domain.OutboxEvent{
		{
			ID:        uuid1,
			EventType: "user.created",
			Payload:   `{"user_id": 1}`,
			Status:    domain.OutboxEventStatusPending,
			Retries:   0,
		},
	}

	updateError := errors.New("update failed")

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	outboxRepo.On("GetPendingEvents", ctx, config.BatchSize).Return(events, nil)
	eventProcessor.On("Process", ctx, events[0]).Return(nil)
	outboxRepo.On("Update", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(updateError)

	err := uc.ProcessEvents(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
	txManager.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
	eventProcessor.AssertExpectations(t)
}

func TestDefaultEventProcessor_Process_Success(t *testing.T) {
	processor := NewDefaultEventProcessor(nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `{"user_id": 1, "name": "John Doe", "email": "john@example.com"}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	err := processor.Process(ctx, event)

	assert.NoError(t, err)
}

func TestDefaultEventProcessor_Process_UnknownEventType(t *testing.T) {
	processor := NewDefaultEventProcessor(nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "unknown.event",
		Payload:   `{"data": "test"}`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	err := processor.Process(ctx, event)

	assert.NoError(t, err) // Unknown events are just logged as warning
}

func TestDefaultEventProcessor_Process_InvalidJSON(t *testing.T) {
	processor := NewDefaultEventProcessor(nil)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	event := &domain.OutboxEvent{
		ID:        uuid1,
		EventType: "user.created",
		Payload:   `invalid json`,
		Status:    domain.OutboxEventStatusPending,
		Retries:   0,
	}

	err := processor.Process(ctx, event)

	assert.Error(t, err)
}
