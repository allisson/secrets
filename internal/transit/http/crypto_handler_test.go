package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	"github.com/allisson/secrets/internal/transit/http/dto"
	"github.com/allisson/secrets/internal/transit/usecase/mocks"
)

// setupTestCryptoHandler creates a test crypto handler with mocked dependencies.
func setupTestCryptoHandler(t *testing.T) (*CryptoHandler, *mocks.MockTransitKeyUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockTransitKeyUseCase := mocks.NewMockTransitKeyUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewCryptoHandler(mockTransitKeyUseCase, nil, logger)

	return handler, mockTransitKeyUseCase
}

func TestCryptoHandler_EncryptHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		plaintext := []byte("my secret data")

		request := dto.EncryptRequest{
			Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		}

		encryptedBlob := &transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: []byte("encrypted-data"),
		}

		mockUseCase.EXPECT().
			Encrypt(mock.Anything, "test-key", plaintext).
			Return(encryptedBlob, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/encrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.EncryptResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, encryptedBlob.String(), response.Ciphertext)
		assert.Equal(t, uint(1), response.Version)
	})

	t.Run("Error_EmptyName", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		request := dto.EncryptRequest{
			Plaintext: base64.StdEncoding.EncodeToString([]byte("my secret data")),
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys//encrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: ""}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/encrypt", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_ValidationFailed_EmptyPlaintext", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		request := dto.EncryptRequest{
			Plaintext: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/encrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ValidationFailed_InvalidBase64", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		request := dto.EncryptRequest{
			Plaintext: "not-valid-base64!@#$",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/encrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		plaintext := []byte("my secret data")

		request := dto.EncryptRequest{
			Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		}

		mockUseCase.EXPECT().
			Encrypt(mock.Anything, "nonexistent-key", plaintext).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/nonexistent-key/encrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "nonexistent-key"}}

		handler.EncryptHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCryptoHandler_DecryptHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		plaintext := []byte("my secret data")
		expectedPlaintext := make([]byte, len(plaintext))
		copy(expectedPlaintext, plaintext) // Save expected value before handler zeros it
		ciphertext := []byte("encrypted-data")
		ciphertextString := "1:ZW5jcnlwdGVkLWRhdGE=" // version:base64(ciphertext)

		request := dto.DecryptRequest{
			Ciphertext: ciphertextString,
		}

		decryptedBlob := &transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: ciphertext,
			Plaintext:  plaintext,
		}

		mockUseCase.EXPECT().
			Decrypt(mock.Anything, "test-key", ciphertextString).
			Return(decryptedBlob, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.DecryptResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, base64.StdEncoding.EncodeToString(expectedPlaintext), response.Plaintext)
		assert.Equal(t, uint(1), response.Version)
	})

	t.Run("Error_EmptyName", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		request := dto.DecryptRequest{
			Ciphertext: "1:ZW5jcnlwdGVkLWRhdGE=",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys//decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: ""}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_ValidationFailed_EmptyCiphertext", func(t *testing.T) {
		handler, _ := setupTestCryptoHandler(t)

		request := dto.DecryptRequest{
			Ciphertext: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidBlobFormat", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		request := dto.DecryptRequest{
			Ciphertext: "invalid-format",
		}

		mockUseCase.EXPECT().
			Decrypt(mock.Anything, "test-key", "invalid-format").
			Return(nil, transitDomain.ErrInvalidBlobFormat).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		request := dto.DecryptRequest{
			Ciphertext: "1:invalid-base64!!!",
		}

		mockUseCase.EXPECT().
			Decrypt(mock.Anything, "test-key", "1:invalid-base64!!!").
			Return(nil, transitDomain.ErrInvalidBlobBase64).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		ciphertextString := "1:ZW5jcnlwdGVkLWRhdGE="

		request := dto.DecryptRequest{
			Ciphertext: ciphertextString,
		}

		mockUseCase.EXPECT().
			Decrypt(mock.Anything, "nonexistent-key", ciphertextString).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/nonexistent-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "nonexistent-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestCryptoHandler(t)

		ciphertextString := "1:ZW5jcnlwdGVkLWRhdGE="

		request := dto.DecryptRequest{
			Ciphertext: ciphertextString,
		}

		mockUseCase.EXPECT().
			Decrypt(mock.Anything, "test-key", ciphertextString).
			Return(nil, apperrors.ErrInvalidInput).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/decrypt", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.DecryptHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}
