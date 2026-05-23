package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenService(t *testing.T) {
	service := NewTokenService()
	assert.NotNil(t, service)
	assert.IsType(t, &tokenService{}, service)
}

func TestTokenService_GenerateToken(t *testing.T) {
	service := NewTokenService()

	t.Run("Success_GenerateToken", func(t *testing.T) {
		plainToken, tokenHash, err := service.GenerateToken()

		// Assert no error
		require.NoError(t, err)

		// Assert plain token is not empty
		assert.NotEmpty(t, plainToken)

		// Assert token hash is not empty
		assert.NotEmpty(t, tokenHash)

		// Assert plain token is base64 URL-encoded
		decodedBytes, err := base64.URLEncoding.DecodeString(plainToken)
		require.NoError(t, err)
		assert.Len(t, decodedBytes, 32, "decoded token should be 32 bytes")

		// Assert token hash is valid SHA-256 hex string (64 characters)
		assert.Len(t, tokenHash, 64, "SHA-256 hash should be 64 hex characters")

		// Assert hash matches manually hashed plain token
		expectedHash := sha256.Sum256([]byte(plainToken))
		expectedHashHex := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expectedHashHex, tokenHash)
	})

	t.Run("Success_GenerateUniqueTokens", func(t *testing.T) {
		plainToken1, tokenHash1, err1 := service.GenerateToken()
		require.NoError(t, err1)

		plainToken2, tokenHash2, err2 := service.GenerateToken()
		require.NoError(t, err2)

		// Assert tokens are different
		assert.NotEqual(t, plainToken1, plainToken2, "generated tokens should be unique")
		assert.NotEqual(t, tokenHash1, tokenHash2, "generated hashes should be unique")
	})
}
