package usecase

import (
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

	t.Run("Success_HashEmptyInput", func(t *testing.T) {
		// Empty input should produce the SHA-256 hash of empty string
		input := []byte{}
		result := hashService.Hash(input)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result)) // SHA-256 produces 32 bytes = 64 hex chars

		// Verify it matches expected SHA-256 hash of empty string
		expectedHash := sha256.Sum256([]byte{})
		expected := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expected, result)
	})

	t.Run("Success_HashSmallInput", func(t *testing.T) {
		input := []byte("hello")
		result := hashService.Hash(input)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected SHA-256 hash
		expectedHash := sha256.Sum256(input)
		expected := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expected, result)
	})

	t.Run("Success_HashLargeInput", func(t *testing.T) {
		// Create a large input (10KB)
		input := []byte(strings.Repeat("A", 10240))
		result := hashService.Hash(input)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected SHA-256 hash
		expectedHash := sha256.Sum256(input)
		expected := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expected, result)
	})

	t.Run("Success_HashBinaryData", func(t *testing.T) {
		// Test with binary data (not just ASCII)
		input := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		result := hashService.Hash(input)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify it matches expected SHA-256 hash
		expectedHash := sha256.Sum256(input)
		expected := hex.EncodeToString(expectedHash[:])
		assert.Equal(t, expected, result)
	})

	t.Run("Success_ConsistencyCheck", func(t *testing.T) {
		// Same input should always produce the same hash
		input := []byte("test-consistency")
		result1 := hashService.Hash(input)
		result2 := hashService.Hash(input)
		result3 := hashService.Hash(input)

		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
	})

	t.Run("Success_DifferentInputsProduceDifferentHashes", func(t *testing.T) {
		// Different inputs should produce different hashes
		input1 := []byte("plaintext1")
		input2 := []byte("plaintext2")

		result1 := hashService.Hash(input1)
		result2 := hashService.Hash(input2)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("Success_SensitivityToSmallChanges", func(t *testing.T) {
		// Even a single bit change should produce a completely different hash
		input1 := []byte("plaintext")
		input2 := []byte("Plaintext") // Only first letter capitalized

		result1 := hashService.Hash(input1)
		result2 := hashService.Hash(input2)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("Success_ResultIsValidHexString", func(t *testing.T) {
		input := []byte("test")
		result := hashService.Hash(input)

		// Verify result contains only valid hex characters
		for _, char := range result {
			assert.True(t,
				(char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
				"Result should only contain hex characters (0-9, a-f)")
		}
	})

	t.Run("Success_KnownTestVector", func(t *testing.T) {
		// Test with a known SHA-256 test vector
		// SHA-256("abc") = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
		input := []byte("abc")
		result := hashService.Hash(input)

		expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		assert.Equal(t, expected, result)
	})

	t.Run("Success_MultipleInstancesProduceSameHash", func(t *testing.T) {
		// Multiple instances of the hash service should produce the same result
		hashService1 := NewSHA256HashService()
		hashService2 := NewSHA256HashService()

		input := []byte("test-multiple-instances")
		result1 := hashService1.Hash(input)
		result2 := hashService2.Hash(input)

		assert.Equal(t, result1, result2)
	})

	t.Run("Success_UnicodeInput", func(t *testing.T) {
		// Test with Unicode characters
		input := []byte("Hello 世界 🌍")
		result := hashService.Hash(input)

		// Verify result is non-empty and valid hex
		assert.NotEmpty(t, result)
		assert.Equal(t, 64, len(result))

		// Verify consistency
		result2 := hashService.Hash(input)
		assert.Equal(t, result, result2)
	})
}
