package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	domainMocks "github.com/allisson/secrets/internal/auth/domain/mocks"
	"github.com/allisson/secrets/internal/auth/usecase"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
)

func TestClientUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateNewClient", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)

		var capturedPlain string
		hashSecret := func(plain string) (string, error) {
			capturedPlain = plain
			return "hashed:" + plain, nil
		}

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

		mockClientRepo.EXPECT().Create(ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.Secret == "hashed:"+capturedPlain &&
				client.Name == createInput.Name &&
				client.IsActive == createInput.IsActive
		})).Return(nil).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			hashSecret,
		)
		output, err := uc.Create(ctx, createInput)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotEmpty(t, output.PlainSecret)
		assert.Equal(t, capturedPlain, output.PlainSecret)
	})
}

func TestClientUseCase_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_UpdateClient", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }

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
			hashSecret)
		err := uc.Update(ctx, clientID, updateInput)

		assert.NoError(t, err)
	})
}

func TestClientUseCase_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_GetClient", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }

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
			hashSecret)
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
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }
		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			hashSecret)

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
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }
		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			hashSecret)

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
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }

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
			hashSecret)
		err := uc.Delete(ctx, clientID)

		assert.NoError(t, err)
	})
}

func TestClientUseCase_Unlock(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }

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
			hashSecret)
		err := uc.Unlock(ctx, clientID)

		assert.NoError(t, err)
	})
}

func TestClientUseCase_RotateSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RotateClientSecret", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)

		var capturedPlain string
		hashSecret := func(plain string) (string, error) {
			capturedPlain = plain
			return "hashed:" + plain, nil
		}

		clientID := uuid.Must(uuid.NewV7())
		existingClient := &authDomain.Client{
			ID:       clientID,
			Secret:   "old-hash",
			IsActive: true,
		}

		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Once()

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(existingClient, nil).Once()
		mockClientRepo.EXPECT().Update(ctx, mock.MatchedBy(func(client *authDomain.Client) bool {
			return client.ID == clientID && client.Secret == "hashed:"+capturedPlain
		})).Return(nil).Once()
		mockTokenRepo.EXPECT().RevokeByClientID(ctx, clientID).Return(nil).Once()
		mockAuditLogUseCase.EXPECT().
			Create(ctx, mock.Anything, clientID, authDomain.RotateCapability, "/v1/clients/"+clientID.String()+"/rotate-secret", mock.Anything).
			Return(nil).
			Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			hashSecret,
		)
		output, err := uc.RotateSecret(ctx, clientID)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, clientID, output.ID)
		assert.NotEmpty(t, output.PlainSecret)
		assert.Equal(t, capturedPlain, output.PlainSecret)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockClientRepo := domainMocks.NewMockClientRepository(t)
		mockTokenRepo := domainMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		hashSecret := func(plain string) (string, error) { return "hashed:" + plain, nil }

		clientID := uuid.Must(uuid.NewV7())

		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Once()

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, authDomain.ErrClientNotFound).Once()

		uc := usecase.NewClientUseCase(
			mockTxManager,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			hashSecret)
		output, err := uc.RotateSecret(ctx, clientID)

		assert.ErrorIs(t, err, authDomain.ErrClientNotFound)
		assert.Nil(t, output)
	})
}
