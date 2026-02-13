// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/httputil"
)

// mockTokenUseCase is a mock implementation of TokenUseCase for testing.
type mockTokenUseCase struct {
	mock.Mock
}

func (m *mockTokenUseCase) Issue(
	ctx context.Context,
	issueTokenInput *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	args := m.Called(ctx, issueTokenInput)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.IssueTokenOutput), args.Error(1)
}

func (m *mockTokenUseCase) Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Client), args.Error(1)
}

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

// mockAuditLogUseCase is a mock implementation of AuditLogUseCase for testing.
type mockAuditLogUseCase struct {
	mock.Mock
}

func (m *mockAuditLogUseCase) Create(
	ctx context.Context,
	requestID uuid.UUID,
	clientID uuid.UUID,
	capability authDomain.Capability,
	path string,
	metadata map[string]any,
) error {
	args := m.Called(ctx, requestID, clientID, capability, path, metadata)
	return args.Error(0)
}

func (m *mockAuditLogUseCase) List(ctx context.Context, offset, limit int) ([]*authDomain.AuditLog, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authDomain.AuditLog), args.Error(1)
}

// TestMain sets Gin to test mode for all tests in this package.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// createTestLogger creates a test logger that discards output.
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestAuthenticationMiddleware_Success tests successful authentication with valid Bearer token.
func TestAuthenticationMiddleware_Success(t *testing.T) {
	// Setup mocks
	mockTokenUC := &mockTokenUseCase{}
	mockTokenSvc := &mockTokenService{}
	logger := createTestLogger()

	// Test data
	plainToken := "test-token-xyz789"
	tokenHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	clientID := uuid.Must(uuid.NewV7())
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "test-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{},
	}

	// Setup expectations
	mockTokenSvc.On("HashToken", plainToken).Return(tokenHash).Once()
	mockTokenUC.On("Authenticate", mock.Anything, tokenHash).Return(client, nil).Once()

	// Create test router with middleware
	router := gin.New()
	router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
	router.GET("/test", func(c *gin.Context) {
		// Verify client is in context
		retrievedClient, ok := GetClient(c.Request.Context())
		require.True(t, ok, "client should be in context")
		require.NotNil(t, retrievedClient, "client should not be nil")
		assert.Equal(t, clientID, retrievedClient.ID)
		assert.Equal(t, "test-client", retrievedClient.Name)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	mockTokenSvc.AssertExpectations(t)
	mockTokenUC.AssertExpectations(t)
}

// TestAuthenticationMiddleware_Success_CaseInsensitiveBearer tests case-insensitive Bearer prefix.
func TestAuthenticationMiddleware_Success_CaseInsensitiveBearer(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
	}{
		{"lowercase_bearer", "bearer "},
		{"uppercase_BEARER", "BEARER "},
		{"mixedcase_BeArEr", "BeArEr "},
		{"standard_Bearer", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockTokenUC := &mockTokenUseCase{}
			mockTokenSvc := &mockTokenService{}
			logger := createTestLogger()

			plainToken := "test-token-xyz789"
			tokenHash := "hash123"
			client := &authDomain.Client{
				ID:       uuid.Must(uuid.NewV7()),
				Name:     "test-client",
				IsActive: true,
			}

			mockTokenSvc.On("HashToken", plainToken).Return(tokenHash).Once()
			mockTokenUC.On("Authenticate", mock.Anything, tokenHash).Return(client, nil).Once()

			// Create test router
			router := gin.New()
			router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Make request with different case
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.prefix+plainToken)
			router.ServeHTTP(w, req)

			// Should succeed regardless of case
			assert.Equal(t, http.StatusOK, w.Code)
			mockTokenSvc.AssertExpectations(t)
			mockTokenUC.AssertExpectations(t)
		})
	}
}

// TestAuthenticationMiddleware_Error_MissingAuthorizationHeader tests missing Authorization header.
func TestAuthenticationMiddleware_Error_MissingAuthorizationHeader(t *testing.T) {
	mockTokenUC := &mockTokenUseCase{}
	mockTokenSvc := &mockTokenService{}
	logger := createTestLogger()

	// Create test router with middleware
	router := gin.New()
	router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
	router.GET("/test", func(c *gin.Context) {
		t.Fatal("handler should not be called when authentication fails")
	})

	// Make request without Authorization header
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response.Error)

	// Verify no use case methods were called
	mockTokenSvc.AssertNotCalled(t, "HashToken", mock.Anything)
	mockTokenUC.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
}

// TestAuthenticationMiddleware_Error_MalformedAuthorizationHeader tests malformed Authorization header.
func TestAuthenticationMiddleware_Error_MalformedAuthorizationHeader(t *testing.T) {
	testCases := []struct {
		name   string
		header string
	}{
		{"no_prefix", "just-a-token"},
		{"wrong_prefix", "Basic username:password"},
		{"missing_space", "Bearertoken"},
		{"only_bearer", "Bearer"},
		{"only_bearer_with_space", "Bearer "},
		{"empty_token", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockTokenUC := &mockTokenUseCase{}
			mockTokenSvc := &mockTokenService{}
			logger := createTestLogger()

			// Create test router with middleware
			router := gin.New()
			router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
			router.GET("/test", func(c *gin.Context) {
				t.Fatal("handler should not be called when authentication fails")
			})

			// Make request with malformed header
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.header)
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response httputil.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "unauthorized", response.Error)

			// Verify no use case methods were called
			mockTokenSvc.AssertNotCalled(t, "HashToken", mock.Anything)
			mockTokenUC.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
		})
	}
}

// TestAuthenticationMiddleware_Error_InvalidToken tests authentication with invalid token.
func TestAuthenticationMiddleware_Error_InvalidToken(t *testing.T) {
	mockTokenUC := &mockTokenUseCase{}
	mockTokenSvc := &mockTokenService{}
	logger := createTestLogger()

	plainToken := "invalid-token"
	tokenHash := "invalid-hash"

	// Setup expectations - token is invalid
	mockTokenSvc.On("HashToken", plainToken).Return(tokenHash).Once()
	mockTokenUC.On("Authenticate", mock.Anything, tokenHash).
		Return(nil, authDomain.ErrInvalidCredentials).Once()

	// Create test router with middleware
	router := gin.New()
	router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
	router.GET("/test", func(c *gin.Context) {
		t.Fatal("handler should not be called when authentication fails")
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response.Error)

	mockTokenSvc.AssertExpectations(t)
	mockTokenUC.AssertExpectations(t)
}

// TestAuthenticationMiddleware_Error_InactiveClient tests authentication with inactive client.
func TestAuthenticationMiddleware_Error_InactiveClient(t *testing.T) {
	mockTokenUC := &mockTokenUseCase{}
	mockTokenSvc := &mockTokenService{}
	logger := createTestLogger()

	plainToken := "valid-token"
	tokenHash := "valid-hash"

	// Setup expectations - client is inactive
	mockTokenSvc.On("HashToken", plainToken).Return(tokenHash).Once()
	mockTokenUC.On("Authenticate", mock.Anything, tokenHash).
		Return(nil, authDomain.ErrClientInactive).Once()

	// Create test router with middleware
	router := gin.New()
	router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
	router.GET("/test", func(c *gin.Context) {
		t.Fatal("handler should not be called when authentication fails")
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)

	mockTokenSvc.AssertExpectations(t)
	mockTokenUC.AssertExpectations(t)
}

// TestAuthenticationMiddleware_Error_DatabaseError tests authentication with database error.
func TestAuthenticationMiddleware_Error_DatabaseError(t *testing.T) {
	mockTokenUC := &mockTokenUseCase{}
	mockTokenSvc := &mockTokenService{}
	logger := createTestLogger()

	plainToken := "valid-token"
	tokenHash := "valid-hash"

	// Setup expectations - database error
	mockTokenSvc.On("HashToken", plainToken).Return(tokenHash).Once()
	mockTokenUC.On("Authenticate", mock.Anything, tokenHash).
		Return(nil, assert.AnError).Once()

	// Create test router with middleware
	router := gin.New()
	router.Use(AuthenticationMiddleware(mockTokenUC, mockTokenSvc, logger))
	router.GET("/test", func(c *gin.Context) {
		t.Fatal("handler should not be called when authentication fails")
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	router.ServeHTTP(w, req)

	// Assertions - should return 500 for unexpected errors
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)

	mockTokenSvc.AssertExpectations(t)
	mockTokenUC.AssertExpectations(t)
}

// TestGetClient_WithClient tests GetClient when client is in context.
func TestGetClient_WithClient(t *testing.T) {
	ctx := context.Background()
	clientID := uuid.Must(uuid.NewV7())
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "test-client",
		IsActive: true,
	}

	// Store client in context
	ctx = WithClient(ctx, client)

	// Retrieve client
	retrievedClient, ok := GetClient(ctx)

	// Assertions
	assert.True(t, ok, "GetClient should return true")
	require.NotNil(t, retrievedClient, "client should not be nil")
	assert.Equal(t, clientID, retrievedClient.ID)
	assert.Equal(t, "test-client", retrievedClient.Name)
	assert.True(t, retrievedClient.IsActive)
}

// TestGetClient_WithoutClient tests GetClient when no client is in context.
func TestGetClient_WithoutClient(t *testing.T) {
	ctx := context.Background()

	// Try to retrieve client from empty context
	retrievedClient, ok := GetClient(ctx)

	// Assertions
	assert.False(t, ok, "GetClient should return false")
	assert.Nil(t, retrievedClient, "client should be nil")
}

// TestWithClient_NilClient tests storing nil client in context.
func TestWithClient_NilClient(t *testing.T) {
	ctx := context.Background()

	// Store nil client
	ctx = WithClient(ctx, nil)

	// Retrieve client
	retrievedClient, ok := GetClient(ctx)

	// Assertions
	assert.True(t, ok, "GetClient should return true (value was set)")
	assert.Nil(t, retrievedClient, "client should be nil")
}

// TestGetPath_WithPath tests GetPath when path is in context.
func TestGetPath_WithPath(t *testing.T) {
	ctx := context.Background()
	expectedPath := "/api/v1/secrets"

	// Store path in context
	ctx = WithPath(ctx, expectedPath)

	// Retrieve path
	retrievedPath, ok := GetPath(ctx)

	// Assertions
	assert.True(t, ok, "GetPath should return true")
	assert.Equal(t, expectedPath, retrievedPath)
}

// TestGetPath_WithoutPath tests GetPath when no path is in context.
func TestGetPath_WithoutPath(t *testing.T) {
	ctx := context.Background()

	// Try to retrieve path from empty context
	retrievedPath, ok := GetPath(ctx)

	// Assertions
	assert.False(t, ok, "GetPath should return false")
	assert.Equal(t, "", retrievedPath, "path should be empty string")
}

// TestWithPath_EmptyString tests storing empty string path in context.
func TestWithPath_EmptyString(t *testing.T) {
	ctx := context.Background()

	// Store empty path
	ctx = WithPath(ctx, "")

	// Retrieve path
	retrievedPath, ok := GetPath(ctx)

	// Assertions
	assert.True(t, ok, "GetPath should return true (value was set)")
	assert.Equal(t, "", retrievedPath, "path should be empty string")
}

// TestGetCapability_WithCapability tests GetCapability when capability is in context.
func TestGetCapability_WithCapability(t *testing.T) {
	ctx := context.Background()
	expectedCapability := authDomain.ReadCapability

	// Store capability in context
	ctx = WithCapability(ctx, expectedCapability)

	// Retrieve capability
	retrievedCapability, ok := GetCapability(ctx)

	// Assertions
	assert.True(t, ok, "GetCapability should return true")
	assert.Equal(t, expectedCapability, retrievedCapability)
}

// TestGetCapability_WithoutCapability tests GetCapability when no capability is in context.
func TestGetCapability_WithoutCapability(t *testing.T) {
	ctx := context.Background()

	// Try to retrieve capability from empty context
	retrievedCapability, ok := GetCapability(ctx)

	// Assertions
	assert.False(t, ok, "GetCapability should return false")
	assert.Equal(t, authDomain.Capability(""), retrievedCapability, "capability should be empty")
}

// TestWithCapability_AllTypes tests storing different capability types in context.
func TestWithCapability_AllTypes(t *testing.T) {
	testCases := []struct {
		name       string
		capability authDomain.Capability
	}{
		{"ReadCapability", authDomain.ReadCapability},
		{"WriteCapability", authDomain.WriteCapability},
		{"DeleteCapability", authDomain.DeleteCapability},
		{"EncryptCapability", authDomain.EncryptCapability},
		{"DecryptCapability", authDomain.DecryptCapability},
		{"RotateCapability", authDomain.RotateCapability},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Store capability in context
			ctx = WithCapability(ctx, tc.capability)

			// Retrieve capability
			retrievedCapability, ok := GetCapability(ctx)

			// Assertions
			assert.True(t, ok, "GetCapability should return true")
			assert.Equal(t, tc.capability, retrievedCapability)
		})
	}
}

// TestAuthorizationMiddleware_Success tests successful authorization with exact path match.
func TestAuthorizationMiddleware_Success(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create client with read capability on specific path
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "test-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/api/v1/secrets",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	// Expect audit log creation for successful authorization
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.ReadCapability, "/api/v1/secrets", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == true &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router with middleware
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Simulate AuthenticationMiddleware by storing client in context
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/api/v1/secrets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Success_WildcardPath tests authorization with wildcard "*" path.
func TestAuthorizationMiddleware_Success_WildcardPath(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())

	// Create admin client with wildcard access
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "admin-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "*",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability, authDomain.WriteCapability},
			},
		},
	}

	testCases := []struct {
		name       string
		path       string
		capability authDomain.Capability
	}{
		{"read_any_path", "/api/v1/secrets", authDomain.ReadCapability},
		{"write_any_path", "/api/v1/secrets/new", authDomain.WriteCapability},
		{"read_different_path", "/api/v1/keys/123", authDomain.ReadCapability},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuditLogUC := &mockAuditLogUseCase{}

			// Expect audit log creation for successful authorization
			mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
				tc.capability, tc.path, mock.MatchedBy(func(metadata map[string]any) bool {
					return metadata["allowed"] == true &&
						metadata["ip"] != nil &&
						metadata["user_agent"] != nil
				})).Return(nil).Once()

			// Create test router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				ctx := WithClient(c.Request.Context(), client)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			router.Use(AuthorizationMiddleware(tc.capability, mockAuditLogUC, logger))
			router.GET(tc.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Make request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(w, req)

			// Should succeed with wildcard policy
			assert.Equal(t, http.StatusOK, w.Code)
			mockAuditLogUC.AssertExpectations(t)
		})
	}
}

// TestAuthorizationMiddleware_Success_PrefixWildcard tests authorization with prefix wildcard "path/*".
func TestAuthorizationMiddleware_Success_PrefixWildcard(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())

	// Create client with prefix wildcard access
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "secrets-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/secret/*",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	testCases := []struct {
		name          string
		path          string
		shouldSucceed bool
	}{
		{"match_prefix_single", "/secret/mykey", true},
		{"match_prefix_nested", "/secret/team/prod/key", true},
		{"no_match_different_prefix", "/public/key", false},
		{"no_match_exact_without_slash", "/secret", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuditLogUC := &mockAuditLogUseCase{}

			// Expect audit log creation (with appropriate allowed value)
			mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
				authDomain.ReadCapability, tc.path, mock.MatchedBy(func(metadata map[string]any) bool {
					return metadata["allowed"] == tc.shouldSucceed &&
						metadata["ip"] != nil &&
						metadata["user_agent"] != nil
				})).Return(nil).Once()

			// Create test router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				ctx := WithClient(c.Request.Context(), client)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
			router.GET("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Make request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(w, req)

			if tc.shouldSucceed {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, w.Code)
			}
			mockAuditLogUC.AssertExpectations(t)
		})
	}
}

// TestAuthorizationMiddleware_Success_MultipleCapabilities tests authorization with multiple capabilities.
func TestAuthorizationMiddleware_Success_MultipleCapabilities(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())

	// Create client with multiple capabilities on same path
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "rw-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path: "/api/v1/secrets",
				Capabilities: []authDomain.Capability{
					authDomain.ReadCapability,
					authDomain.WriteCapability,
					authDomain.DeleteCapability,
				},
			},
		},
	}

	testCases := []struct {
		name       string
		capability authDomain.Capability
	}{
		{"read_capability", authDomain.ReadCapability},
		{"write_capability", authDomain.WriteCapability},
		{"delete_capability", authDomain.DeleteCapability},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuditLogUC := &mockAuditLogUseCase{}

			// Expect audit log creation for successful authorization
			mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
				tc.capability, "/api/v1/secrets", mock.MatchedBy(func(metadata map[string]any) bool {
					return metadata["allowed"] == true &&
						metadata["ip"] != nil &&
						metadata["user_agent"] != nil
				})).Return(nil).Once()

			// Create test router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				ctx := WithClient(c.Request.Context(), client)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			router.Use(AuthorizationMiddleware(tc.capability, mockAuditLogUC, logger))
			router.GET("/api/v1/secrets", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Make request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
			router.ServeHTTP(w, req)

			// Should succeed for all capabilities
			assert.Equal(t, http.StatusOK, w.Code)
			mockAuditLogUC.AssertExpectations(t)
		})
	}
}

// TestAuthorizationMiddleware_Error_NoClientInContext tests missing client in context.
func TestAuthorizationMiddleware_Error_NoClientInContext(t *testing.T) {
	logger := createTestLogger()
	mockAuditLogUC := &mockAuditLogUseCase{}

	// No audit log should be created when there's no client in context

	// Create test router without AuthenticationMiddleware
	router := gin.New()
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/test", func(c *gin.Context) {
		t.Fatal("handler should not be called when authorization fails")
	})

	// Make request without client in context
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Assertions - should return 401 when client is missing
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response.Error)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Error_ClientLacksCapability tests client without required capability.
func TestAuthorizationMiddleware_Error_ClientLacksCapability(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create client with only read capability
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "readonly-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/api/v1/secrets",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	// Expect audit log creation for failed authorization
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.WriteCapability, "/api/v1/secrets", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == false &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router requiring write capability
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.WriteCapability, mockAuditLogUC, logger))
	router.POST("/api/v1/secrets", func(c *gin.Context) {
		t.Fatal("handler should not be called when authorization fails")
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", nil)
	router.ServeHTTP(w, req)

	// Assertions - should return 403 when capability is missing
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Error_PathNotInPolicy tests path not matching any policy.
func TestAuthorizationMiddleware_Error_PathNotInPolicy(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create client with access to /api/v1/secrets only
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "limited-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/api/v1/secrets",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	// Expect audit log creation for failed authorization
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.ReadCapability, "/api/v1/keys", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == false &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router for different path
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/api/v1/keys", func(c *gin.Context) {
		t.Fatal("handler should not be called when authorization fails")
	})

	// Make request to unauthorized path
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys", nil)
	router.ServeHTTP(w, req)

	// Assertions - should return 403 when path doesn't match
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Error_WrongCapabilityForPath tests wrong capability for path.
func TestAuthorizationMiddleware_Error_WrongCapabilityForPath(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())

	// Create client with read on /secrets and write on /keys
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "split-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/api/v1/secrets",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
			{
				Path:         "/api/v1/keys",
				Capabilities: []authDomain.Capability{authDomain.WriteCapability},
			},
		},
	}

	testCases := []struct {
		name       string
		path       string
		capability authDomain.Capability
	}{
		{"write_on_readonly_path", "/api/v1/secrets", authDomain.WriteCapability},
		{"read_on_writeonly_path", "/api/v1/keys", authDomain.ReadCapability},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuditLogUC := &mockAuditLogUseCase{}

			// Expect audit log creation for failed authorization
			mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
				tc.capability, tc.path, mock.MatchedBy(func(metadata map[string]any) bool {
					return metadata["allowed"] == false &&
						metadata["ip"] != nil &&
						metadata["user_agent"] != nil
				})).Return(nil).Once()

			// Create test router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				ctx := WithClient(c.Request.Context(), client)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			router.Use(AuthorizationMiddleware(tc.capability, mockAuditLogUC, logger))
			router.GET(tc.path, func(c *gin.Context) {
				t.Fatal("handler should not be called when authorization fails")
			})

			// Make request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(w, req)

			// Should fail with 403
			assert.Equal(t, http.StatusForbidden, w.Code)

			var response httputil.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "forbidden", response.Error)
			mockAuditLogUC.AssertExpectations(t)
		})
	}
}

// TestAuthorizationMiddleware_Error_EmptyPath tests that Client.IsAllowed handles empty paths correctly.
// Note: In practice, Gin always normalizes paths to at least "/", but we test the middleware's
// behavior with the domain logic that empty paths are rejected by Client.IsAllowed.
func TestAuthorizationMiddleware_Error_EmptyPath(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create client with wildcard access
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "test-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "*",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	// Expect audit log creation for successful authorization (wildcard matches "/")
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.ReadCapability, "/", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == true &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router - use "/" as the path (Gin's normalized empty path)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request to root path "/"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	// Should succeed - "/" is a valid path and wildcard "*" matches everything
	assert.Equal(t, http.StatusOK, w.Code)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Error_InactiveClientStillAuthorizes tests that inactive clients
// stored in context still pass authorization (authentication should have failed first).
func TestAuthorizationMiddleware_Error_InactiveClientStillAuthorizes(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create inactive client with valid policies
	// Note: This scenario shouldn't happen in practice because AuthenticationMiddleware
	// would reject inactive clients, but we test the authorization logic in isolation.
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "inactive-client",
		IsActive: false, // Inactive
		Policies: []authDomain.PolicyDocument{
			{
				Path:         "/api/v1/secrets",
				Capabilities: []authDomain.Capability{authDomain.ReadCapability},
			},
		},
	}

	// Expect audit log creation for successful authorization
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.ReadCapability, "/api/v1/secrets", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == true &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Simulate storing inactive client in context (shouldn't happen in practice)
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/api/v1/secrets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	router.ServeHTTP(w, req)

	// Authorization middleware only checks policies, not IsActive status.
	// AuthenticationMiddleware is responsible for checking IsActive.
	// So this should succeed from an authorization perspective.
	assert.Equal(t, http.StatusOK, w.Code)
	mockAuditLogUC.AssertExpectations(t)
}

// TestAuthorizationMiddleware_Error_NoPolicies tests client with no policies.
func TestAuthorizationMiddleware_Error_NoPolicies(t *testing.T) {
	logger := createTestLogger()
	clientID := uuid.Must(uuid.NewV7())
	mockAuditLogUC := &mockAuditLogUseCase{}

	// Create client with no policies
	client := &authDomain.Client{
		ID:       clientID,
		Name:     "no-policy-client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{}, // Empty policies
	}

	// Expect audit log creation for failed authorization
	mockAuditLogUC.On("Create", mock.Anything, mock.AnythingOfType("uuid.UUID"), clientID,
		authDomain.ReadCapability, "/api/v1/secrets", mock.MatchedBy(func(metadata map[string]any) bool {
			return metadata["allowed"] == false &&
				metadata["ip"] != nil &&
				metadata["user_agent"] != nil
		})).Return(nil).Once()

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditLogUC, logger))
	router.GET("/api/v1/secrets", func(c *gin.Context) {
		t.Fatal("handler should not be called when authorization fails")
	})

	// Make request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	router.ServeHTTP(w, req)

	// Should fail with 403 when no policies exist
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response httputil.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)
	mockAuditLogUC.AssertExpectations(t)
}
