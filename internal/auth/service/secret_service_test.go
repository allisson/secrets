package service

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecretService(t *testing.T) {
	service := NewSecretService()
	assert.NotNil(t, service)
	assert.IsType(t, &secretService{}, service)
}

func TestSecretService_GenerateSecret(t *testing.T) {
	service := NewSecretService()

	t.Run("Success_GeneratesValidSecret", func(t *testing.T) {
		plainSecret, hashedSecret, err := service.GenerateSecret()
		require.NoError(t, err)

		// Verify plain secret is not empty
		assert.NotEmpty(t, plainSecret)

		// Verify plain secret is valid base64
		decoded, err := base64.URLEncoding.DecodeString(plainSecret)
		require.NoError(t, err)
		assert.Len(t, decoded, 32) // 32 bytes

		// Verify hashed secret is not empty
		assert.NotEmpty(t, hashedSecret)

		// Verify hashed secret is different from plain secret
		assert.NotEqual(t, plainSecret, hashedSecret)

		// Verify hashed secret starts with $argon2id$ (PHC format)
		assert.Contains(t, hashedSecret, "$argon2id$")
	})

	t.Run("Success_GeneratesUniqueSecrets", func(t *testing.T) {
		plainSecret1, hashedSecret1, err := service.GenerateSecret()
		require.NoError(t, err)

		plainSecret2, hashedSecret2, err := service.GenerateSecret()
		require.NoError(t, err)

		// Verify each call generates different secrets
		assert.NotEqual(t, plainSecret1, plainSecret2)
		assert.NotEqual(t, hashedSecret1, hashedSecret2)
	})

	t.Run("Success_GeneratedSecretCanBeVerified", func(t *testing.T) {
		plainSecret, hashedSecret, err := service.GenerateSecret()
		require.NoError(t, err)

		// Verify the generated secret can be compared successfully
		matches := service.CompareSecret(plainSecret, hashedSecret)
		assert.True(t, matches)
	})
}

func TestSecretService_HashSecret(t *testing.T) {
	service := NewSecretService()

	t.Run("Success_HashesSecretCorrectly", func(t *testing.T) {
		plainSecret := "test-secret-123"
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		// Verify hash is not empty
		assert.NotEmpty(t, hashedSecret)

		// Verify hash is different from plain secret
		assert.NotEqual(t, plainSecret, hashedSecret)

		// Verify hash uses Argon2id
		assert.Contains(t, hashedSecret, "$argon2id$")
	})

	t.Run("Success_SameSecretProducesDifferentHashes", func(t *testing.T) {
		plainSecret := "test-secret-123"

		hashedSecret1, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		hashedSecret2, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		// Verify different hashes due to different salts
		assert.NotEqual(t, hashedSecret1, hashedSecret2)

		// But both should verify against the same plain secret
		assert.True(t, service.CompareSecret(plainSecret, hashedSecret1))
		assert.True(t, service.CompareSecret(plainSecret, hashedSecret2))
	})

	t.Run("Success_EmptySecretCanBeHashed", func(t *testing.T) {
		plainSecret := ""
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		assert.NotEmpty(t, hashedSecret)
		assert.True(t, service.CompareSecret(plainSecret, hashedSecret))
	})
}

func TestSecretService_CompareSecret(t *testing.T) {
	service := NewSecretService()

	t.Run("Success_CorrectSecretMatches", func(t *testing.T) {
		plainSecret := "correct-secret"
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		matches := service.CompareSecret(plainSecret, hashedSecret)
		assert.True(t, matches)
	})

	t.Run("Failure_IncorrectSecretDoesNotMatch", func(t *testing.T) {
		plainSecret := "correct-secret"
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		matches := service.CompareSecret("wrong-secret", hashedSecret)
		assert.False(t, matches)
	})

	t.Run("Failure_EmptySecretDoesNotMatch", func(t *testing.T) {
		plainSecret := "correct-secret"
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		matches := service.CompareSecret("", hashedSecret)
		assert.False(t, matches)
	})

	t.Run("Failure_InvalidHashFormat", func(t *testing.T) {
		plainSecret := "correct-secret"

		matches := service.CompareSecret(plainSecret, "invalid-hash-format")
		assert.False(t, matches)
	})

	t.Run("Failure_EmptyHashString", func(t *testing.T) {
		plainSecret := "correct-secret"

		matches := service.CompareSecret(plainSecret, "")
		assert.False(t, matches)
	})

	t.Run("Success_CaseSensitiveComparison", func(t *testing.T) {
		plainSecret := "CaseSensitive"
		hashedSecret, err := service.HashSecret(plainSecret)
		require.NoError(t, err)

		// Correct case matches
		assert.True(t, service.CompareSecret(plainSecret, hashedSecret))

		// Different case does not match
		assert.False(t, service.CompareSecret("casesensitive", hashedSecret))
		assert.False(t, service.CompareSecret("CASESENSITIVE", hashedSecret))
	})
}

func TestSecretService_Integration(t *testing.T) {
	service := NewSecretService()

	t.Run("Success_EndToEndWorkflow", func(t *testing.T) {
		// Generate a new secret
		plainSecret, hashedSecret, err := service.GenerateSecret()
		require.NoError(t, err)
		require.NotEmpty(t, plainSecret)
		require.NotEmpty(t, hashedSecret)

		// Verify the plain secret matches the hash
		assert.True(t, service.CompareSecret(plainSecret, hashedSecret))

		// Verify a different secret does not match
		wrongPlain := "definitely-not-the-right-secret"
		assert.False(t, service.CompareSecret(wrongPlain, hashedSecret))

		// Hash a custom secret
		customSecret := "my-custom-api-key" //nolint:gosec // test fixture, not a real credential
		customHash, err := service.HashSecret(customSecret)
		require.NoError(t, err)

		// Verify custom secret
		assert.True(t, service.CompareSecret(customSecret, customHash))
		assert.False(t, service.CompareSecret("wrong-key", customHash))
	})
}
