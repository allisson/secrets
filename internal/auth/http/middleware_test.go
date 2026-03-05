// Package http provides HTTP middleware and utilities for authentication.
package http

import (
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
	serviceMocks "github.com/allisson/secrets/internal/auth/service/mocks"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
	"github.com/allisson/secrets/internal/httputil"
)

func TestAuthenticationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("Success_ValidToken", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{
			ID:       clientID,
			Name:     "test-client",
			IsActive: true,
		}

		token := "test-token"
		tokenHash := "hashed-token"

		mockTokenService.EXPECT().HashToken(token).Return(tokenHash).Once()
		mockTokenUC.EXPECT().Authenticate(mock.Anything, tokenHash).Return(client, nil).Once()

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			val, _ := GetClient(c.Request.Context())
			assert.Equal(t, client, val)
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error_MissingAuthorizationHeader", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", resp.Error)
		assert.Equal(t, "Authentication is required", resp.Message)
	})

	t.Run("Error_InvalidAuthorizationFormat", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat token")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", resp.Error)
		assert.Equal(t, "Authentication is required", resp.Message)
	})

	t.Run("Error_InvalidToken", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		token := "invalid-token"
		tokenHash := "hashed-invalid-token" //nolint:gosec

		mockTokenService.EXPECT().HashToken(token).Return(tokenHash).Once()

		mockTokenUC.EXPECT().
			Authenticate(mock.Anything, tokenHash).
			Return(nil, authDomain.ErrInvalidCredentials).
			Once()

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", resp.Error)
		assert.Equal(t, "Authentication is required", resp.Message)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		token := "token"
		tokenHash := "hashed-token"

		mockTokenService.EXPECT().HashToken(token).Return(tokenHash).Once()
		mockTokenUC.EXPECT().
			Authenticate(mock.Anything, tokenHash).
			Return(nil, authDomain.ErrClientInactive).
			Once()

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "forbidden", resp.Error)
		assert.Equal(t, "You don't have permission to access this resource", resp.Message)
	})

	t.Run("Error_UnexpectedUseCaseError", func(t *testing.T) {
		mockTokenUC := usecaseMocks.NewMockTokenUseCase(t)
		mockTokenService := serviceMocks.NewMockTokenService(t)

		token := "token"
		tokenHash := "hashed-token"

		mockTokenService.EXPECT().HashToken(token).Return(tokenHash).Once()
		mockTokenUC.EXPECT().Authenticate(mock.Anything, tokenHash).Return(nil, os.ErrNotExist).Once()

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthenticationMiddleware(mockTokenUC, mockTokenService, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "internal_error", resp.Error)
	})
}

func TestAuthorizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("Success_Allowed", func(t *testing.T) {
		mockAuditUC := usecaseMocks.NewMockAuditLogUseCase(t)
		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{
			ID: clientID,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		mockAuditUC.EXPECT().
			Create(mock.Anything, mock.Anything, clientID, authDomain.ReadCapability, "/test", mock.Anything).
			Return(nil).
			Once()

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(WithClient(c.Request.Context(), client))
			c.Next()
		})
		r.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditUC, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error_Forbidden", func(t *testing.T) {
		mockAuditUC := usecaseMocks.NewMockAuditLogUseCase(t)
		clientID := uuid.Must(uuid.NewV7())
		client := &authDomain.Client{
			ID: clientID,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/other/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		mockAuditUC.EXPECT().
			Create(mock.Anything, mock.Anything, clientID, authDomain.ReadCapability, "/test", mock.Anything).
			Return(nil).
			Once()

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(WithClient(c.Request.Context(), client))
			c.Next()
		})
		r.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditUC, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp httputil.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "forbidden", resp.Error)
		assert.Equal(t, "You don't have permission to access this resource", resp.Message)
	})

	t.Run("Error_MissingClientContext", func(t *testing.T) {
		mockAuditUC := usecaseMocks.NewMockAuditLogUseCase(t)
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthorizationMiddleware(authDomain.ReadCapability, mockAuditUC, logger))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
