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

func TestTokenService_HashToken(t *testing.T) {
	service := NewTokenService()

	t.Run("Success_HashToken", func(t *testing.T) {
		plainToken := "test-token-abc123"

		tokenHash := service.HashToken(plainToken)

		// Assert hash is not empty
		assert.NotEmpty(t, tokenHash)

		// Assert hash is valid SHA-256 hex string (64 characters)
		assert.Len(t, tokenHash, 64, "SHA-256 hash should be 64 hex characters")

		// Assert hash matches expected SHA-256 hash
		expectedHash := sha256.Sum256([]byte(plainToken))
		expectedHashHex := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expectedHashHex, tokenHash)
	})

	t.Run("Success_ConsistentHashing", func(t *testing.T) {
		plainToken := "consistent-token-xyz789"

		hash1 := service.HashToken(plainToken)
		hash2 := service.HashToken(plainToken)

		// Assert same input produces same hash
		assert.Equal(t, hash1, hash2, "hashing should be deterministic")
	})

	t.Run("Success_DifferentTokensProduceDifferentHashes", func(t *testing.T) {
		token1 := "token-one"
		token2 := "token-two"

		hash1 := service.HashToken(token1)
		hash2 := service.HashToken(token2)

		// Assert different inputs produce different hashes
		assert.NotEqual(t, hash1, hash2, "different tokens should have different hashes")
	})

	t.Run("Success_EmptyStringProducesValidHash", func(t *testing.T) {
		plainToken := ""

		tokenHash := service.HashToken(plainToken)

		// Assert hash is generated even for empty string
		assert.NotEmpty(t, tokenHash)
		assert.Len(t, tokenHash, 64)

		// Verify it matches expected SHA-256 of empty string
		expectedHash := sha256.Sum256([]byte(""))
		expectedHashHex := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expectedHashHex, tokenHash)
	})
}

func TestTokenService_GenerateAndVerify(t *testing.T) {
	service := NewTokenService()

	t.Run("Success_GeneratedTokenHashMatchesManualHash", func(t *testing.T) {
		plainToken, generatedHash, err := service.GenerateToken()
		require.NoError(t, err)

		// Manually hash the plain token
		manualHash := service.HashToken(plainToken)

		// Assert generated hash matches manual hash
		assert.Equal(t, generatedHash, manualHash, "generated hash should match manual hash of plain token")
	})
}
