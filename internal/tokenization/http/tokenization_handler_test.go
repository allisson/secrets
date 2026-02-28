package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	"github.com/allisson/secrets/internal/tokenization/http/dto"
	"github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// setupTestTokenizationHandler creates a test handler with mocked dependencies.
func setupTestTokenizationHandler(t *testing.T) (*TokenizationHandler, *mocks.MockTokenizationUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockTokenizationUseCase := mocks.NewMockTokenizationUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewTokenizationHandler(mockTokenizationUseCase, logger)

	return handler, mockTokenizationUseCase
}

func TestTokenizationHandler_TokenizeHandler(t *testing.T) {
	t.Run("Success_TokenizeValue", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		plaintext := []byte("test-value")
		plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)
		metadata := map[string]any{"last4": "alue"}
		ttl := 3600

		request := dto.TokenizeRequest{
			Plaintext: plaintextB64,
			Metadata:  metadata,
			TTL:       &ttl,
		}

		expectedToken := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_123456",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			Metadata:          metadata,
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         func() *time.Time { t := time.Now().UTC().Add(1 * time.Hour); return &t }(),
		}

		mockUseCase.EXPECT().
			Tokenize(
				mock.Anything,
				"test-key",
				plaintext,
				metadata,
				mock.MatchedBy(func(expiresAt *time.Time) bool {
					return expiresAt != nil
				}),
			).
			Return(expectedToken, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TokenizeResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "tok_123456", response.Token)
		assert.Equal(t, metadata, response.Metadata)
		assert.NotNil(t, response.ExpiresAt)
	})

	t.Run("Success_TokenizeWithoutTTL", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		plaintext := []byte("test-value")
		plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

		request := dto.TokenizeRequest{
			Plaintext: plaintextB64,
			Metadata:  nil,
			TTL:       nil,
		}

		expectedToken := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_123456",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
		}

		mockUseCase.EXPECT().
			Tokenize(
				mock.Anything,
				"test-key",
				plaintext,
				mock.Anything,
				mock.Anything,
			).
			Return(expectedToken, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TokenizeResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "tok_123456", response.Token)
		assert.Nil(t, response.ExpiresAt)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/tokenize", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_MissingPlaintext", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		request := dto.TokenizeRequest{
			Plaintext: "",
			Metadata:  nil,
			TTL:       nil,
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		request := dto.TokenizeRequest{
			Plaintext: "not-valid-base64!!!",
			Metadata:  nil,
			TTL:       nil,
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_MissingKeyName", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		plaintext := []byte("test-value")
		plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

		request := dto.TokenizeRequest{
			Plaintext: plaintextB64,
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys//tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: ""}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_KeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		plaintext := []byte("test-value")
		plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

		request := dto.TokenizeRequest{
			Plaintext: plaintextB64,
		}

		mockUseCase.EXPECT().
			Tokenize(mock.Anything, "nonexistent-key", plaintext, mock.Anything, mock.Anything).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/nonexistent-key/tokenize", request)
		c.Params = gin.Params{{Key: "name", Value: "nonexistent-key"}}

		handler.TokenizeHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTokenizationHandler_DetokenizeHandler(t *testing.T) {
	t.Run("Success_Detokenize", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		plaintext := []byte("original-value")
		plaintextCopy := make([]byte, len(plaintext))
		copy(plaintextCopy, plaintext)
		metadata := map[string]any{"last4": "alue"}

		request := dto.DetokenizeRequest{
			Token: "tok_123456",
		}

		mockUseCase.EXPECT().
			Detokenize(mock.Anything, "tok_123456").
			Return(plaintext, metadata, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", request)

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.DetokenizeResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		decodedPlaintext, err := base64.StdEncoding.DecodeString(response.Plaintext)
		assert.NoError(t, err)
		assert.Equal(t, plaintextCopy, decodedPlaintext)
		assert.Equal(t, metadata, response.Metadata)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		request := dto.DetokenizeRequest{
			Token: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", request)

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.DetokenizeRequest{
			Token: "tok_nonexistent",
		}

		mockUseCase.EXPECT().
			Detokenize(mock.Anything, "tok_nonexistent").
			Return(nil, nil, tokenizationDomain.ErrTokenNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", request)

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error_TokenExpired", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.DetokenizeRequest{
			Token: "tok_expired",
		}

		mockUseCase.EXPECT().
			Detokenize(mock.Anything, "tok_expired").
			Return(nil, nil, tokenizationDomain.ErrTokenExpired).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", request)

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_TokenRevoked", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.DetokenizeRequest{
			Token: "tok_revoked",
		}

		mockUseCase.EXPECT().
			Detokenize(mock.Anything, "tok_revoked").
			Return(nil, nil, tokenizationDomain.ErrTokenRevoked).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/detokenize", request)

		handler.DetokenizeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestTokenizationHandler_ValidateHandler(t *testing.T) {
	t.Run("Success_ValidToken", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.ValidateTokenRequest{
			Token: "tok_valid",
		}

		mockUseCase.EXPECT().
			Validate(mock.Anything, "tok_valid").
			Return(true, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/validate", request)

		handler.ValidateHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ValidateTokenResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Valid)
	})

	t.Run("Success_InvalidToken", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.ValidateTokenRequest{
			Token: "tok_invalid",
		}

		mockUseCase.EXPECT().
			Validate(mock.Anything, "tok_invalid").
			Return(false, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/validate", request)

		handler.ValidateHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ValidateTokenResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Valid)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/validate", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.ValidateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		request := dto.ValidateTokenRequest{
			Token: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/validate", request)

		handler.ValidateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.ValidateTokenRequest{
			Token: "tok_test",
		}

		dbError := errors.New("database error")

		mockUseCase.EXPECT().
			Validate(mock.Anything, "tok_test").
			Return(false, dbError).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/validate", request)

		handler.ValidateHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestTokenizationHandler_RevokeHandler(t *testing.T) {
	t.Run("Success_RevokeToken", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.RevokeTokenRequest{
			Token: "tok_revoke",
		}

		mockUseCase.EXPECT().
			Revoke(mock.Anything, "tok_revoke").
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/revoke", request)

		handler.RevokeHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/revoke", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.RevokeHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		handler, _ := setupTestTokenizationHandler(t)

		request := dto.RevokeTokenRequest{
			Token: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/revoke", request)

		handler.RevokeHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.RevokeTokenRequest{
			Token: "tok_nonexistent",
		}

		mockUseCase.EXPECT().
			Revoke(mock.Anything, "tok_nonexistent").
			Return(tokenizationDomain.ErrTokenNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/revoke", request)

		handler.RevokeHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestTokenizationHandler(t)

		request := dto.RevokeTokenRequest{
			Token: "tok_test",
		}

		dbError := errors.New("database error")

		mockUseCase.EXPECT().
			Revoke(mock.Anything, "tok_test").
			Return(dbError).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/revoke", request)

		handler.RevokeHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
