package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	serviceMocks "github.com/allisson/secrets/internal/auth/service/mocks"
	"github.com/allisson/secrets/internal/auth/usecase"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
	"github.com/allisson/secrets/internal/config"
)

func TestTokenUseCase_Issue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_IssueTokenWithValidCredentials", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  10,
			LockoutDuration:     30 * time.Minute,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		// Test data
		clientID := uuid.Must(uuid.NewV7())
		clientSecret := "test-client-secret-abc123"                //nolint:gosec
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec
		plainToken := "test-token-xyz789"
		tokenHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

		client := &authDomain.Client{
			ID:       clientID,
			Secret:   hashedSecret,
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}

		// Setup expectations
		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret(clientSecret, hashedSecret).Return(true).Once()
		mockTokenService.EXPECT().GenerateToken().Return(plainToken, tokenHash, nil).Once()
		mockTokenRepo.EXPECT().Create(ctx, mock.AnythingOfType("*domain.Token")).Return(nil).Once()

		// Execute
		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, plainToken, output.PlainToken)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "some-secret",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, authDomain.ErrClientNotFound).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "inactive-client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{},
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "client-secret",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrClientInactive, err)
	})

	t.Run("Error_InvalidClientSecret", func(t *testing.T) {
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  10,
			LockoutDuration:     30 * time.Minute,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		wrongSecret := "wrong-secret"
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec

		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			Name:           "test-client",
			IsActive:       true,
			Policies:       []authDomain.PolicyDocument{},
			FailedAttempts: 0,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: wrongSecret,
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret(wrongSecret, hashedSecret).Return(false).Once()
		mockClientRepo.EXPECT().UpdateLockState(ctx, clientID, 1, (*time.Time)(nil)).Return(nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
	})

	t.Run("Error_AccountLocked", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		lockedUntil := time.Now().UTC().Add(30 * time.Minute)
		client := &authDomain.Client{
			ID:          clientID,
			IsActive:    true,
			LockedUntil: &lockedUntil,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "any-secret",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.ErrorIs(t, err, authDomain.ErrClientLocked)
		assert.Nil(t, output)
	})

	t.Run("Error_FailedAttemptsReachesLockThreshold", func(t *testing.T) {
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  3,
			LockoutDuration:     30 * time.Minute,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			IsActive:       true,
			FailedAttempts: 2,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "wrong",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("wrong", hashedSecret).Return(false).Once()
		mockClientRepo.EXPECT().
			UpdateLockState(ctx, clientID, 3, mock.AnythingOfType("*time.Time")).
			Return(nil).
			Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, output)
	})

	t.Run("Success_ResetsCounterAfterSuccessfulAuth", func(t *testing.T) {
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  10,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			IsActive:       true,
			FailedAttempts: 5,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "correct",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("correct", hashedSecret).Return(true).Once()
		mockClientRepo.EXPECT().UpdateLockState(ctx, clientID, 0, (*time.Time)(nil)).Return(nil).Once()
		mockTokenService.EXPECT().GenerateToken().Return("plain", "hash", nil).Once()
		mockTokenRepo.EXPECT().Create(ctx, mock.Anything).Return(nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.NoError(t, err)
		assert.NotNil(t, output)
	})

	t.Run("Success_LockExpiredAllowsAuth", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		lockedUntil := time.Now().UTC().Add(-1 * time.Minute)
		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			IsActive:       true,
			LockedUntil:    &lockedUntil,
			FailedAttempts: 3,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "correct",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("correct", hashedSecret).Return(true).Once()
		mockClientRepo.EXPECT().UpdateLockState(ctx, clientID, 0, (*time.Time)(nil)).Return(nil).Once()
		mockTokenService.EXPECT().GenerateToken().Return("plain", "hash", nil).Once()
		mockTokenRepo.EXPECT().Create(ctx, mock.Anything).Return(nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.NoError(t, err)
		assert.NotNil(t, output)
	})

	t.Run("Error_TokenGenerationFails", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   hashedSecret,
			IsActive: true,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "correct",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("correct", hashedSecret).Return(true).Once()
		mockTokenService.EXPECT().GenerateToken().Return("", "", errors.New("gen error")).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
	})

	t.Run("Error_RepositoryCreateFails", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   hashedSecret,
			IsActive: true,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "correct",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("correct", hashedSecret).Return(true).Once()
		mockTokenService.EXPECT().GenerateToken().Return("plain", "hash", nil).Once()
		mockTokenRepo.EXPECT().Create(ctx, mock.Anything).Return(errors.New("db error")).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
	})

	t.Run("Success_TokenExpirationSetFromConfig", func(t *testing.T) {
		expiration := 12 * time.Hour
		mockConfig := &config.Config{AuthTokenExpiration: expiration}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   hashedSecret,
			IsActive: true,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "correct",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("correct", hashedSecret).Return(true).Once()
		mockTokenService.EXPECT().GenerateToken().Return("plain", "hash", nil).Once()

		var capturedToken *authDomain.Token
		mockTokenRepo.EXPECT().
			Create(ctx, mock.Anything).
			Run(func(ctx context.Context, token *authDomain.Token) {
				capturedToken = token
			}).
			Return(nil).
			Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.WithinDuration(t, time.Now().UTC().Add(expiration), capturedToken.ExpiresAt, time.Second)
	})

	t.Run("Success_UpdateLockStateFails", func(t *testing.T) {
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  10,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			IsActive:       true,
			FailedAttempts: 5,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "wrong",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("wrong", hashedSecret).Return(false).Once()
		mockClientRepo.EXPECT().
			UpdateLockState(ctx, clientID, 6, (*time.Time)(nil)).
			Return(errors.New("db error")).
			Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		// Should still return ErrInvalidCredentials and not the db error
		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, output)
	})

	t.Run("Success_FirstFailureTriggersLock", func(t *testing.T) {
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
			LockoutMaxAttempts:  1,
			LockoutDuration:     30 * time.Minute,
		}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		hashedSecret := "hashed"
		client := &authDomain.Client{
			ID:             clientID,
			Secret:         hashedSecret,
			IsActive:       true,
			FailedAttempts: 0,
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "wrong",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()
		mockSecretService.EXPECT().CompareSecret("wrong", hashedSecret).Return(false).Once()
		mockClientRepo.EXPECT().
			UpdateLockState(ctx, clientID, 1, mock.AnythingOfType("*time.Time")).
			Return(nil).
			Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, output)
	})

	t.Run("Error_RepositoryGetReturnsUnexpectedError", func(t *testing.T) {
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "some-secret",
		}

		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, errors.New("unexpected error")).Once()

		uc := usecase.NewTokenUseCase(
			nil,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		output, err := uc.Issue(ctx, issueInput)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, "unexpected error", err.Error())
	})
}

func TestTokenUseCase_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_AuthenticateWithValidToken", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		tokenHash := "abcdef1234567890"

		token := &authDomain.Token{
			ID:        uuid.Must(uuid.NewV7()),
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			RevokedAt: nil,
		}

		client := &authDomain.Client{
			ID:       clientID,
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()
		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		result, err := uc.Authenticate(ctx, tokenHash)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, client.ID, result.ID)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, nil, nil, nil)

		tokenHash := "not-found"
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(nil, authDomain.ErrTokenNotFound).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, result)
	})

	t.Run("Error_TokenExpired", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, nil, nil, nil)

		tokenHash := "expired"
		token := &authDomain.Token{
			ExpiresAt: time.Now().UTC().Add(-1 * time.Minute),
		}
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, result)
	})

	t.Run("Error_TokenRevoked", func(t *testing.T) {
		mockConfig := &config.Config{AuthTokenExpiration: 24 * time.Hour}
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		mockSecretService := serviceMocks.NewMockSecretService(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		tokenHash := "revoked-token-hash"
		revokedAt := time.Now().UTC().Add(-1 * time.Hour)

		token := &authDomain.Token{
			TokenHash: tokenHash,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			RevokedAt: &revokedAt,
		}

		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()

		uc := usecase.NewTokenUseCase(
			mockConfig,
			mockClientRepo,
			mockTokenRepo,
			mockAuditLogUseCase,
			mockSecretService,
			mockTokenService,
		)
		result, err := uc.Authenticate(ctx, tokenHash)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		uc := usecase.NewTokenUseCase(nil, mockClientRepo, mockTokenRepo, nil, nil, nil)

		tokenHash := "hash"
		clientID := uuid.Must(uuid.NewV7())
		token := &authDomain.Token{
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()
		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, authDomain.ErrClientNotFound).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.ErrorIs(t, err, authDomain.ErrInvalidCredentials)
		assert.Nil(t, result)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		uc := usecase.NewTokenUseCase(nil, mockClientRepo, mockTokenRepo, nil, nil, nil)

		tokenHash := "hash"
		clientID := uuid.Must(uuid.NewV7())
		token := &authDomain.Token{
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		client := &authDomain.Client{
			ID:       clientID,
			IsActive: false,
		}
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()
		mockClientRepo.EXPECT().Get(ctx, clientID).Return(client, nil).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.ErrorIs(t, err, authDomain.ErrClientInactive)
		assert.Nil(t, result)
	})

	t.Run("Error_RepositoryGetTokenFails", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, nil, nil, nil)

		tokenHash := "hash"
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(nil, errors.New("db error")).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("Error_RepositoryGetClientFails", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockClientRepo := usecaseMocks.NewMockClientRepository(t)
		uc := usecase.NewTokenUseCase(nil, mockClientRepo, mockTokenRepo, nil, nil, nil)

		tokenHash := "hash"
		clientID := uuid.Must(uuid.NewV7())
		token := &authDomain.Token{
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()
		mockClientRepo.EXPECT().Get(ctx, clientID).Return(nil, errors.New("db error")).Once()

		result, err := uc.Authenticate(ctx, tokenHash)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestTokenUseCase_Revoke(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RevokeToken", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, mockAuditLogUseCase, nil, nil)

		tokenHash := "test-token-hash"
		token := &authDomain.Token{ID: uuid.New(), ClientID: uuid.New()}

		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(token, nil).Once()
		mockTokenRepo.EXPECT().RevokeByTokenID(ctx, token.ID).Return(nil).Once()
		mockAuditLogUseCase.EXPECT().
			Create(ctx, mock.Anything, token.ClientID, authDomain.DeleteCapability, "/v1/token", mock.Anything).
			Return(nil).
			Once()

		err := uc.Revoke(ctx, tokenHash)
		assert.NoError(t, err)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		mockAuditLogUseCase := usecaseMocks.NewMockAuditLogUseCase(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, mockAuditLogUseCase, nil, nil)

		tokenHash := "test-token-hash"

		mockTokenRepo.EXPECT().GetByTokenHash(ctx, tokenHash).Return(nil, authDomain.ErrTokenNotFound).Once()

		err := uc.Revoke(ctx, tokenHash)
		assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
	})
}

func TestTokenUseCase_PurgeExpiredAndRevoked(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_PurgeTokens", func(t *testing.T) {
		mockTokenRepo := usecaseMocks.NewMockTokenRepository(t)
		uc := usecase.NewTokenUseCase(nil, nil, mockTokenRepo, nil, nil, nil)

		days := 30
		mockTokenRepo.EXPECT().
			PurgeExpiredAndRevoked(ctx, mock.AnythingOfType("time.Time")).
			Return(int64(5), nil).
			Once()

		count, err := uc.PurgeExpiredAndRevoked(ctx, days)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Error_InvalidDays", func(t *testing.T) {
		uc := usecase.NewTokenUseCase(nil, nil, nil, nil, nil, nil)

		count, err := uc.PurgeExpiredAndRevoked(ctx, -1)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
	})
}
