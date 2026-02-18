package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/config"
)

// mockTokenService is a mock implementation of TokenService for testing.
type mockTokenService struct {
	mock.Mock
}

func (m *mockTokenService) GenerateToken() (plainToken string, tokenHash string, error error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockTokenService) HashToken(plainToken string) string {
	args := m.Called(plainToken)
	return args.String(0)
}

// mockTokenRepository is a mock implementation of TokenRepository for testing.
type mockTokenRepository struct {
	mock.Mock
}

func (m *mockTokenRepository) Create(ctx context.Context, token *authDomain.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockTokenRepository) Update(ctx context.Context, token *authDomain.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*authDomain.Token, error) {
	args := m.Called(ctx, tokenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Token), args.Error(1)
}

func (m *mockTokenRepository) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*authDomain.Token, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Token), args.Error(1)
}

func TestTokenUseCase_Issue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_IssueTokenWithValidCredentials", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		// Test data
		clientID := uuid.Must(uuid.NewV7())
		clientSecret := "test-client-secret-abc123"                //nolint:gosec // test fixture, not a real credential
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential
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
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		mockSecretService.On("CompareSecret", clientSecret, hashedSecret).
			Return(true).
			Once()

		mockTokenService.On("GenerateToken").
			Return(plainToken, tokenHash, nil).
			Once()

		mockTokenRepo.On("Create", ctx, mock.MatchedBy(func(token *authDomain.Token) bool {
			return token.TokenHash == tokenHash &&
				token.ClientID == clientID &&
				token.RevokedAt == nil &&
				!token.ExpiresAt.IsZero() &&
				!token.CreatedAt.IsZero()
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, plainToken, output.PlainToken)
		mockClientRepo.AssertExpectations(t)
		mockSecretService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "some-secret",
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert - should return generic error to prevent enumeration
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed-secret",
			Name:     "inactive-client",
			IsActive: false, // Client is inactive
			Policies: []authDomain.PolicyDocument{},
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "client-secret",
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrClientInactive, err)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_InvalidClientSecret", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		wrongSecret := "wrong-secret"
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential

		client := &authDomain.Client{
			ID:       clientID,
			Secret:   hashedSecret,
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: wrongSecret,
		}

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		mockSecretService.On("CompareSecret", wrongSecret, hashedSecret).
			Return(false). // Secret doesn't match
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockClientRepo.AssertExpectations(t)
		mockSecretService.AssertExpectations(t)
	})

	t.Run("Error_TokenGenerationFails", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		clientSecret := "test-client-secret"
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential

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

		expectedErr := errors.New("failed to generate random token")

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		mockSecretService.On("CompareSecret", clientSecret, hashedSecret).
			Return(true).
			Once()

		mockTokenService.On("GenerateToken").
			Return("", "", expectedErr).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
		mockClientRepo.AssertExpectations(t)
		mockSecretService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("Error_RepositoryCreateFails", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		clientSecret := "test-client-secret"
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential
		plainToken := "test-token"
		tokenHash := "token-hash"

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

		expectedErr := errors.New("database error")

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		mockSecretService.On("CompareSecret", clientSecret, hashedSecret).
			Return(true).
			Once()

		mockTokenService.On("GenerateToken").
			Return(plainToken, tokenHash, nil).
			Once()

		mockTokenRepo.On("Create", ctx, mock.AnythingOfType("*domain.Token")).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
		mockClientRepo.AssertExpectations(t)
		mockSecretService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Success_TokenExpirationSetFromConfig", func(t *testing.T) {
		// Setup mocks with specific expiration duration
		customExpiration := 48 * time.Hour
		mockConfig := &config.Config{
			AuthTokenExpiration: customExpiration,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		clientSecret := "test-client-secret"
		hashedSecret := "$argon2id$v=19$m=65536,t=3,p=4$test-hash" //nolint:gosec // test fixture, not a real credential
		plainToken := "test-token"
		tokenHash := "token-hash"

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

		// Capture the created token to verify expiration
		var createdToken *authDomain.Token
		now := time.Now().UTC()

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		mockSecretService.On("CompareSecret", clientSecret, hashedSecret).
			Return(true).
			Once()

		mockTokenService.On("GenerateToken").
			Return(plainToken, tokenHash, nil).
			Once()

		mockTokenRepo.On("Create", ctx, mock.MatchedBy(func(token *authDomain.Token) bool {
			createdToken = token
			return true
		})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotNil(t, createdToken)

		// Verify expiration is set correctly (within 1 second tolerance)
		expectedExpiration := now.Add(customExpiration)
		assert.WithinDuration(t, expectedExpiration, createdToken.ExpiresAt, time.Second)

		mockClientRepo.AssertExpectations(t)
		mockSecretService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryGetReturnsUnexpectedError", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		issueInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "some-secret",
		}

		expectedErr := errors.New("unexpected database error")

		// Setup expectations
		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		output, err := uc.Issue(ctx, issueInput)

		// Assert - should return the original error, not ErrInvalidCredentials
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
		mockClientRepo.AssertExpectations(t)
	})
}

func TestTokenUseCase_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_AuthenticateWithValidToken", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		// Test data
		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // Valid token, not expired
			RevokedAt: nil,                                  // Not revoked
			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		}

		//nolint:gosec // test fixture data
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "$argon2id$v=19$m=65536,t=3,p=4$test-hash", //nolint:gosec // test fixture
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, client.ID, result.ID)
		assert.Equal(t, client.Name, result.Name)
		assert.Equal(t, client.IsActive, result.IsActive)
		mockTokenRepo.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		tokenHash := "nonexistent-token-hash"

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(nil, authDomain.ErrTokenNotFound).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert - should return generic error to prevent enumeration
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_TokenExpired", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "expired-token-hash"

		// Token that expired 1 hour ago
		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(-1 * time.Hour), // Expired
			RevokedAt: nil,
			CreatedAt: time.Now().UTC().Add(-25 * time.Hour),
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_TokenRevoked", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "revoked-token-hash"
		revokedAt := time.Now().UTC().Add(-1 * time.Hour)

		// Token that was revoked
		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // Not expired
			RevokedAt: &revokedAt,                           // But revoked
			CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "orphaned-token-hash"

		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			RevokedAt: nil,
			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert - should return generic error to prevent enumeration
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrInvalidCredentials, err)
		mockTokenRepo.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "inactive-client-token-hash"

		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			RevokedAt: nil,
			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		}

		//nolint:gosec // test fixture data
		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "$argon2id$v=19$m=65536,t=3,p=4$test-hash", //nolint:gosec // test fixture
			Name:     "inactive-client",
			IsActive: false, // Client is inactive
			Policies: []authDomain.PolicyDocument{},
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		mockClientRepo.On("Get", ctx, clientID).
			Return(client, nil).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, authDomain.ErrClientInactive, err)
		mockTokenRepo.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryGetTokenFails", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		tokenHash := "database-error-token-hash"
		expectedErr := errors.New("database connection error")

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert - should propagate the original error
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryGetClientFails", func(t *testing.T) {
		// Setup mocks
		mockConfig := &config.Config{
			AuthTokenExpiration: 24 * time.Hour,
		}
		mockClientRepo := &mockClientRepository{}
		mockTokenRepo := &mockTokenRepository{}
		mockSecretService := &mockSecretService{}
		mockTokenService := &mockTokenService{}

		clientID := uuid.Must(uuid.NewV7())
		tokenID := uuid.Must(uuid.NewV7())
		tokenHash := "client-db-error-token-hash" //nolint:gosec // test fixture, not a real credential
		expectedErr := errors.New("database connection error")

		token := &authDomain.Token{
			ID:        tokenID,
			TokenHash: tokenHash,
			ClientID:  clientID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			RevokedAt: nil,
			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		}

		// Setup expectations
		mockTokenRepo.On("GetByTokenHash", ctx, tokenHash).
			Return(token, nil).
			Once()

		mockClientRepo.On("Get", ctx, clientID).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewTokenUseCase(mockConfig, mockClientRepo, mockTokenRepo, mockSecretService, mockTokenService)
		result, err := uc.Authenticate(ctx, tokenHash)

		// Assert - should propagate the original error
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockTokenRepo.AssertExpectations(t)
		mockClientRepo.AssertExpectations(t)
	})
}
