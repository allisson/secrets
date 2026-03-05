package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	serviceMocks "github.com/allisson/secrets/internal/auth/service/mocks"
	"github.com/allisson/secrets/internal/auth/usecase"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
)

func TestClientUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateNewClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)

		// Test data
		plainSecret := "test-plain-secret-abc123"                  //nolint:gosec
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec
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
		mockSecretService.EXPECT().GenerateSecret().Return(plainSecret, hashedSecret, nil).Once()
		mockClientRepo.EXPECT().Create(ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.Secret == hashedSecret &&
				client.Name == createInput.Name &&
				client.IsActive == createInput.IsActive
		})).Return(nil).Once()

		// Execute
		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)
		output, err := uc.Create(ctx, createInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, plainSecret, output.PlainSecret)
	})
}

func TestClientUseCase_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_UpdateClient", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Name:     "old-name",
			IsActive: true,
		}

		updateInput := &authDomain.UpdateClientInput{
			Name:     "new-name",
			IsActive: false,
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(existingClient, nil).Once()
		mockClientRepo.EXPECT().Update(ctx, mock.AnythingOfType("*domain.Client")).Return(nil).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)
		err := uc.Update(ctx, clientID, updateInput)

		assert.NoError(t, err)
	})
}

func TestClientUseCase_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_GetClient", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)

		clientID := uuid.Must(uuid.NewV7())
		expectedClient := &authDomain.Client{
			ID:   clientID,
			Name: "test-client",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(expectedClient, nil).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)
		client, err := uc.Get(ctx, clientID)

		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, clientID, client.ID)
	})
}

func TestClientUseCase_RevokeTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RevokeTokens", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)

		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{ID: clientID}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockTokenRepo.EXPECT().RevokeByClientID(ctx, clientID).Return(nil).Once()
		mockAuditLogUseCase.EXPECT().
			Create(ctx, mock.Anything, clientID, authDomain.DeleteCapability, "/v1/clients/"+clientID.String()+"/tokens", mock.Anything).
			Return(nil).
			Once()

		err := uc.RevokeTokens(ctx, clientID)
		assert.NoError(t, err)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)

		clientID := uuid.Must(uuid.NewV7())

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, authDomain.ErrClientNotFound).Once()

		err := uc.RevokeTokens(ctx, clientID)
		assert.ErrorIs(t, err, authDomain.ErrClientNotFound)
	})
}

func TestClientUseCase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_SoftDeleteClient", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			IsActive: true,
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(existingClient, nil).Once()
		mockClientRepo.EXPECT().Update(ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.ID == clientID && client.IsActive == false
		})).Return(nil).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)
		err := uc.Delete(ctx, clientID)

		assert.NoError(t, err)
	})
}

func TestClientUseCase_Unlock(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			IsActive: true,
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(existingClient, nil).Once()
		mockClientRepo.EXPECT().UpdateLockState(ctx, clientID, 0, (*time.Time)(nil)).Return(nil).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
		)
		err := uc.Unlock(ctx, clientID)

		assert.NoError(t, err)
	})
}
