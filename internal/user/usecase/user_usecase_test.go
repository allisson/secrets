package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	outboxDomain "github.com/allisson/go-project-template/internal/outbox/domain"
	"github.com/allisson/go-project-template/internal/user/domain"
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

// MockUserRepository is a mock implementation of repository.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	if args.Get(0) != nil {
		// Set the ID to simulate database behavior
		user.ID = uuid.Must(uuid.NewV7())
	}
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockOutboxEventRepository is a mock implementation of repository.OutboxEventRepository
type MockOutboxEventRepository struct {
	mock.Mock
}

func (m *MockOutboxEventRepository) Create(ctx context.Context, event *outboxDomain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOutboxEventRepository) GetPendingEvents(
	ctx context.Context,
	limit int,
) ([]*outboxDomain.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outboxDomain.OutboxEvent), args.Error(1)
}

func (m *MockOutboxEventRepository) Update(ctx context.Context, event *outboxDomain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestNewUserUseCase(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)
	assert.NotNil(t, useCase)
}

func TestUserUseCase_RegisterUser_Success(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	input := RegisterUserInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	outboxRepo.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil)

	user, err := useCase.RegisterUser(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, input.Name, user.Name)
	assert.Equal(t, input.Email, user.Email)
	assert.NotEmpty(t, user.Password)
	assert.NotEqual(t, input.Password, user.Password) // Password should be hashed

	txManager.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestUserUseCase_RegisterUser_CreateUserError(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	input := RegisterUserInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	createError := errors.New("database error")

	// Setup expectations - WithTx will call the function, which should fail on Create
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(createError)

	user, err := useCase.RegisterUser(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, user)
	// The error should be the database error returned by the repository
	assert.Equal(t, createError, err)

	txManager.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestUserUseCase_RegisterUser_CreateOutboxEventError(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	input := RegisterUserInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	outboxError := errors.New("outbox error")

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	outboxRepo.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(outboxError)

	user, err := useCase.RegisterUser(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to create outbox event")

	txManager.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestUserUseCase_RegisterUser_VerifyOutboxPayload(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	input := RegisterUserInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	// Setup expectations
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Capture the outbox event to verify its payload
	var capturedEvent *outboxDomain.OutboxEvent
	outboxRepo.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).
		Run(func(args mock.Arguments) {
			capturedEvent = args.Get(1).(*outboxDomain.OutboxEvent)
		}).
		Return(nil)

	user, err := useCase.RegisterUser(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, capturedEvent)
	assert.Equal(t, "user.created", capturedEvent.EventType)
	assert.Equal(t, outboxDomain.OutboxEventStatusPending, capturedEvent.Status)
	assert.Equal(t, 0, capturedEvent.Retries)

	// Verify payload structure
	var payload map[string]interface{}
	err = json.Unmarshal([]byte(capturedEvent.Payload), &payload)
	assert.NoError(t, err)
	assert.Equal(t, input.Name, payload["name"])
	assert.Equal(t, input.Email, payload["email"])
	assert.NotNil(t, payload["user_id"])

	txManager.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByEmail_Success(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	uuid1 := uuid.New()
	expectedUser := &domain.User{
		ID:    uuid1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	userRepo.On("GetByEmail", ctx, "john@example.com").Return(expectedUser, nil)

	user, err := useCase.GetUserByEmail(ctx, "john@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)

	userRepo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByEmail_NotFound(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	notFoundError := errors.New("user not found")

	userRepo.On("GetByEmail", ctx, "notfound@example.com").Return(nil, notFoundError)

	user, err := useCase.GetUserByEmail(ctx, "notfound@example.com")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, notFoundError, err)

	userRepo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByID_Success(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	uuid1 := uuid.Must(uuid.NewV7())
	expectedUser := &domain.User{
		ID:    uuid1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	userRepo.On("GetByID", ctx, uuid1).Return(expectedUser, nil)

	user, err := useCase.GetUserByID(ctx, uuid1)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)

	userRepo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByID_NotFound(t *testing.T) {
	txManager := &MockTxManager{}
	userRepo := &MockUserRepository{}
	outboxRepo := &MockOutboxEventRepository{}

	useCase, err := NewUserUseCase(txManager, userRepo, outboxRepo)
	require.NoError(t, err)

	ctx := context.Background()
	notFoundError := errors.New("user not found")
	notFoundUUID := uuid.Must(uuid.NewV7())

	userRepo.On("GetByID", ctx, notFoundUUID).Return(nil, notFoundError)

	user, err := useCase.GetUserByID(ctx, notFoundUUID)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, notFoundError, err)

	userRepo.AssertExpectations(t)
}
