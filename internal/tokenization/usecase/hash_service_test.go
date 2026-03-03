package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewSHA256HashService tests the constructor.
func TestNewSHA256HashService(t *testing.T) {
	hashService := NewSHA256HashService()
	assert.NotNil(t, hashService)
	assert.IsType(t, &sha256HashService{}, hashService)
}

// TestSHA256HashService_Hash tests the Hash method.
func TestSHA256HashService_Hash(t *testing.T) {
	hashService := NewSHA256HashService()

	t.Run("Success_HashEmptyInputNoSalt", func(t *testing.T) {
		// Empty input should produce the SHA-256 hash of empty string when salt is empty
		input := []byte{}
		result := hashService.Hash(input, nil)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result)) // SHA-256 produces 32 bytes = 64 hex chars

		// Verify it matches expected SHA-256 hash of empty string
		expectedHash := sha256.Sum256([]byte{})
		expected := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expected, result)
	})

	t.Run("Success_HashWithSalt", func(t *testing.T) {
		input := []byte("hello")
		salt := []byte("secret-salt")
		result := hashService.Hash(input, salt)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected HMAC-SHA256 hash
		h := hmac.New(sha256.New, salt)
		h.Write(input)
		expected := hex.EncodeToString(h.Sum(nil))
		assert.Equal(t, expected, result)

		// Verify it is different from unsalted hash
		unsaltedHash := sha256.Sum256(input)
		unsalted := hex.EncodeToString(unsaltedHash[:])
		assert.NotEqual(t, unsalted, result)
	})

	t.Run("Success_DifferentSaltsProduceDifferentHashes", func(t *testing.T) {
		input := []byte("same-plaintext")
		salt1 := []byte("salt-1")
		salt2 := []byte("salt-2")

		result1 := hashService.Hash(input, salt1)
		result2 := hashService.Hash(input, salt2)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("Success_HashLargeInputWithSalt", func(t *testing.T) {
		// Create a large input (10KB)
		input := []byte(strings.Repeat("A", 10240))
		salt := []byte("large-input-salt")
		result := hashService.Hash(input, salt)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected HMAC-SHA256 hash
		h := hmac.New(sha256.New, salt)
		h.Write(input)
		expected := hex.EncodeToString(h.Sum(nil))
		assert.Equal(t, expected, result)
	})

	t.Run("Success_HashBinaryDataWithSalt", func(t *testing.T) {
		// Test with binary data (not just ASCII)
		input := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		salt := []byte{0xAA, 0xBB, 0xCC}
		result := hashService.Hash(input, salt)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected HMAC-SHA256 hash
		h := hmac.New(sha256.New, salt)
		h.Write(input)
		expected := hex.EncodeToString(h.Sum(nil))
		assert.Equal(t, expected, result)
	})

	t.Run("Success_ConsistencyCheck", func(t *testing.T) {
		// Same input and salt should always produce the same hash
		input := []byte("test-consistency")
		salt := []byte("test-salt")
		result1 := hashService.Hash(input, salt)
		result2 := hashService.Hash(input, salt)
		result3 := hashService.Hash(input, salt)

		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
	})

	t.Run("Success_KnownTestVectorHMAC", func(t *testing.T) {
		// Test with a known HMAC-SHA256 test vector
		// HMAC_SHA256(key="key", data="The quick brown fox jumps over the lazy dog")
		// = f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8
		input := []byte("The quick brown fox jumps over the lazy dog")
		salt := []byte("key")
		result := hashService.Hash(input, salt)

		expected := "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8"
		assert.Equal(t, expected, result)
	})

	t.Run("Success_UnicodeInputWithSalt", func(t *testing.T) {
		// Test with Unicode characters
		input := []byte("Hello 世界 🌍")
		salt := []byte("unicode-salt-🚀")
		result := hashService.Hash(input, salt)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify consistency
		result2 := hashService.Hash(input, salt)
		assert.Equal(t, result, result2)
	})
}
