package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/http/dto"
	serviceMocks "github.com/allisson/secrets/internal/auth/service/mocks"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func setupTokenTestHandler(
	t *testing.T,
) (*TokenHandler, *usecaseMocks.MockTokenUseCase, *serviceMocks.MockTokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mockTokenUseCase := usecaseMocks.NewMockTokenUseCase(t)
	mockTokenService := serviceMocks.NewMockTokenService(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTokenHandler(mockTokenUseCase, mockTokenService, logger)
	return handler, mockTokenUseCase, mockTokenService
}

func TestTokenHandler_IssueTokenHandler(t *testing.T) {
	t.Run("Success_ValidCredentials", func(t *testing.T) {
		handler, mockUseCase, _ := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		plainToken := "tok_12345"
		expiresAt := time.Now().UTC().Add(1 * time.Hour)

		request := dto.IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret",
		}

		mockUseCase.EXPECT().Issue(mock.Anything, mock.MatchedBy(func(in *authDomain.IssueTokenInput) bool {
			return in.ClientID == clientID && in.ClientSecret == "test_secret"
		})).Return(&authDomain.IssueTokenOutput{
			PlainToken: plainToken,
			ExpiresAt:  expiresAt,
		}, nil).Once()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body, _ := json.Marshal(request)
		c.Request, _ = http.NewRequest(http.MethodPost, "/v1/token", bytes.NewBuffer(body))

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response dto.IssueTokenResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, plainToken, response.Token)

	})
}

func TestTokenHandler_RevokeTokenHandler(t *testing.T) {
	t.Run("Success_RevokeCurrentToken", func(t *testing.T) {
		handler, mockUseCase, mockTokenService := setupTokenTestHandler(t)

		token := "valid-token"
		tokenHash := "hashed-token"

		mockTokenService.EXPECT().HashToken(token).Return(tokenHash).Once()
		mockUseCase.EXPECT().Revoke(mock.Anything, tokenHash).Return(nil).Once()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodDelete, "/v1/token", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		handler.RevokeTokenHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		handler, _, _ := setupTokenTestHandler(t)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodDelete, "/v1/token", nil)

		handler.RevokeTokenHandler(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
