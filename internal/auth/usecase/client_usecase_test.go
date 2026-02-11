package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
)

// mockSecretService is a mock implementation of SecretService for testing.
type mockSecretService struct {
	mock.Mock
}

func (m *mockSecretService) GenerateSecret() (plainSecret string, hashedSecret string, error error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockSecretService) HashSecret(plainSecret string) (hashedSecret string, error error) {
	args := m.Called(plainSecret)
	return args.String(0), args.Error(1)
}

func (m *mockSecretService) CompareSecret(plainSecret string, hashedSecret string) bool {
	args := m.Called(plainSecret, hashedSecret)
	return args.Bool(0)
}

// mockClientRepository is a mock implementation of ClientRepository for testing.
type mockClientRepository struct {
	mock.Mock
}

func (m *mockClientRepository) Create(ctx context.Context, client *authDomain.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *mockClientRepository) Update(ctx context.Context, client *authDomain.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *mockClientRepository) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Client), args.Error(1)
}

func TestClientUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateNewClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		// Test data
		plainSecret := "test-plain-secret-abc123"                  //nolint:gosec // test fixture, not a real credential
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential
		createInput := &authDomain.CreateClientInput{
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		// Setup expectations
		mockSecretService.On("GenerateSecret").
			Return(plainSecret, hashedSecret, nil).
			Once()

		mockClientRepo.On("Create", ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.Secret == hashedSecret &&
				client.Name == createInput.Name &&
				client.IsActive == createInput.IsActive &&
				len(client.Policies) == len(createInput.Policies)
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		output, err := uc.Create(ctx, createInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotEqual(t, uuid.Nil, output.ID)
		assert.Equal(t, plainSecret, output.PlainSecret)
		mockSecretService.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_SecretGenerationFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		createInput := &authDomain.CreateClientInput{
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		expectedErr := errors.New("failed to generate random secret")

		// Setup expectations
		mockSecretService.On("GenerateSecret").
			Return("", "", expectedErr).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		output, err := uc.Create(ctx, createInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
		mockSecretService.AssertExpectations(t)
	})

	t.Run("Error_RepositoryCreateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		plainSecret := "test-plain-secret-abc123"                  //nolint:gosec // test fixture, not a real credential
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential
		createInput := &authDomain.CreateClientInput{
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		expectedErr := errors.New("database error")

		// Setup expectations
		mockSecretService.On("GenerateSecret").
			Return(plainSecret, hashedSecret, nil).
			Once()

		mockClientRepo.On("Create", ctx, mock.AnythingOfType("*domain.Client")).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		output, err := uc.Create(ctx, createInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
		mockSecretService.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Success_CreateInactiveClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		plainSecret := "test-plain-secret-xyz789"                    //nolint:gosec // test fixture, not a real credential
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash-2" //nolint:gosec // test fixture, not a real credential
		createInput := &authDomain.CreateClientInput{
			Name:     "inactive-client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockSecretService.On("GenerateSecret").
			Return(plainSecret, hashedSecret, nil).
			Once()

		mockClientRepo.On("Create", ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.IsActive == false
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		output, err := uc.Create(ctx, createInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		mockSecretService.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})
}

func TestClientUseCase_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_UpdateClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "existing-hash",
			Name:     "old-name",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		updateInput := &authDomain.UpdateClientInput{
			Name:     "new-name",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "config/*",
					Capabilities: []authDomain.Capability{authDomain.WriteCapability},
				},
			},
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(existingClient, nil).
			Once()

		mockClientRepo.On("Update", ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.ID == clientID &&
				client.Name == updateInput.Name &&
				client.IsActive == updateInput.IsActive &&
				len(client.Policies) == len(updateInput.Policies) &&
				client.Secret == existingClient.Secret // Secret should remain unchanged
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Update(ctx, clientID, updateInput)

		// Assert
		assert.NoError(t, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		updateInput := &authDomain.UpdateClientInput{
			Name:     "new-name",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Update(ctx, clientID, updateInput)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, authDomain.ErrClientNotFound, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryUpdateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "existing-hash",
			Name:     "old-name",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		updateInput := &authDomain.UpdateClientInput{
			Name:     "new-name",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		expectedErr := errors.New("database update error")

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(existingClient, nil).
			Once()

		mockClientRepo.On("Update", ctx, mock.AnythingOfType("*domain.Client")).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Update(ctx, clientID, updateInput)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClientRepo.AssertExpectations(t)
	})
}

func TestClientUseCase_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_GetClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		expectedClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(expectedClient, nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		client, err := uc.Get(ctx, clientID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, expectedClient.ID, client.ID)
		assert.Equal(t, expectedClient.Name, client.Name)
		assert.Equal(t, expectedClient.IsActive, client.IsActive)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		client, err := uc.Get(ctx, clientID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Equal(t, authDomain.ErrClientNotFound, err)
		mockClientRepo.AssertExpectations(t)
	})
}

func TestClientUseCase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_SoftDeleteClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(existingClient, nil).
			Once()

		mockClientRepo.On("Update", ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.ID == clientID && client.IsActive == false
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Delete(ctx, clientID)

		// Assert
		assert.NoError(t, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Delete(ctx, clientID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, authDomain.ErrClientNotFound, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryUpdateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		expectedErr := errors.New("database update error")

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(existingClient, nil).
			Once()

		mockClientRepo.On("Update", ctx, mock.AnythingOfType("*domain.Client")).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Delete(ctx, clientID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Success_DeleteAlreadyInactiveClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := &mockClientRepository{}
		mockSecretService := &mockSecretService{}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "test-client",
			IsActive: false, // Already inactive
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(existingClient, nil).
			Once()

		mockClientRepo.On("Update", ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.ID == clientID && client.IsActive == false
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewClientUseCase(mockTxManager, mockClientRepo, mockSecretService)
		err := uc.Delete(ctx, clientID)

		// Assert
		assert.NoError(t, err)
		mockClientRepo.AssertExpectations(t)
	})
}
