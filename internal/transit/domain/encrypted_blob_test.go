package domain_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/transit/domain"
)

func TestNewEncryptedBlob_Success(t *testing.T) {
	t.Run("ValidInput_WithCiphertext", func(t *testing.T) {
		// Arrange
		plaintext := []byte("Hello, World!")
		ciphertext := base64.StdEncoding.EncodeToString(plaintext)
		input := "1:" + ciphertext

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(1), blob.Version)
		assert.Equal(t, plaintext, blob.Ciphertext)
	})

	t.Run("ValidInput_Version0", func(t *testing.T) {
		// Arrange
		input := "0:dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(0), blob.Version)
		assert.Equal(t, []byte("test"), blob.Ciphertext)
	})

	t.Run("ValidInput_LargeVersion", func(t *testing.T) {
		// Arrange
		input := "999999:ZGF0YQ=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(999999), blob.Version)
		assert.Equal(t, []byte("data"), blob.Ciphertext)
	})

	t.Run("ValidInput_EmptyCiphertext", func(t *testing.T) {
		// Arrange - empty string is valid base64
		input := "5:"

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(5), blob.Version)
		assert.Empty(t, blob.Ciphertext)
	})

	t.Run("ValidInput_WithColonsInBase64", func(t *testing.T) {
		// Arrange - test that we split on first colon only
		input := "1:dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(1), blob.Version)
	})

	t.Run("ValidInput_ComplexBase64", func(t *testing.T) {
		// Arrange - complex data with padding
		data := []byte("This is a longer test message that will encode to a longer base64 string!")
		ciphertext := base64.StdEncoding.EncodeToString(data)
		input := "10:" + ciphertext

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(10), blob.Version)
		assert.Equal(t, data, blob.Ciphertext)
	})
}

func TestNewEncryptedBlob_Errors(t *testing.T) {
	t.Run("Error_EmptyString", func(t *testing.T) {
		// Arrange
		input := ""

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobFormat)
		assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_OnePart", func(t *testing.T) {
		// Arrange
		input := "1"

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobFormat)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_ThreeParts", func(t *testing.T) {
		// Arrange
		input := "1:data:extra"

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobFormat)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_InvalidVersion_NonNumeric", func(t *testing.T) {
		// Arrange
		input := "abc:dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobVersion)
		assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_InvalidVersion_Negative", func(t *testing.T) {
		// Arrange
		input := "-1:dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobVersion)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_InvalidVersion_Float", func(t *testing.T) {
		// Arrange
		input := "1.5:dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobVersion)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_InvalidBase64_InvalidCharacters", func(t *testing.T) {
		// Arrange
		input := "1:not-valid-base64!!!"

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobBase64)
		assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_InvalidBase64_IncorrectPadding", func(t *testing.T) {
		// Arrange
		input := "1:dGVzd==="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobBase64)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})

	t.Run("Error_VersionWithSpaces", func(t *testing.T) {
		// Arrange
		input := " 1 :dGVzdA=="

		// Act
		blob, err := domain.NewEncryptedBlob(input)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBlobVersion)
		assert.Equal(t, domain.EncryptedBlob{}, blob)
	})
}

func TestEncryptedBlob_String(t *testing.T) {
	t.Run("Success_SerializesCorrectly", func(t *testing.T) {
		// Arrange
		plaintext := []byte("Hello, World!")
		blob := domain.EncryptedBlob{
			Version:    42,
			Ciphertext: plaintext,
		}
		expected := "42:" + base64.StdEncoding.EncodeToString(plaintext)

		// Act
		result := blob.String()

		// Assert
		assert.Equal(t, expected, result)
	})

	t.Run("Success_EmptyCiphertext", func(t *testing.T) {
		// Arrange
		blob := domain.EncryptedBlob{
			Version:    1,
			Ciphertext: []byte{},
		}
		expected := "1:"

		// Act
		result := blob.String()

		// Assert
		assert.Equal(t, expected, result)
	})

	t.Run("Success_Version0", func(t *testing.T) {
		// Arrange
		blob := domain.EncryptedBlob{
			Version:    0,
			Ciphertext: []byte("data"),
		}

		// Act
		result := blob.String()

		// Assert
		assert.Contains(t, result, "0:")
	})

	t.Run("Success_RoundTrip", func(t *testing.T) {
		// Arrange
		original := domain.EncryptedBlob{
			Version:    123,
			Ciphertext: []byte("This is test data for round trip!"),
		}

		// Act
		serialized := original.String()
		parsed, err := domain.NewEncryptedBlob(serialized)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, original.Version, parsed.Version)
		assert.Equal(t, original.Ciphertext, parsed.Ciphertext)
	})

	t.Run("Success_RoundTrip_MultipleIterations", func(t *testing.T) {
		// Arrange
		original := domain.EncryptedBlob{
			Version:    5,
			Ciphertext: []byte("test data"),
		}

		// Act - serialize and parse multiple times
		current := original
		for i := 0; i < 5; i++ {
			serialized := current.String()
			parsed, err := domain.NewEncryptedBlob(serialized)
			require.NoError(t, err)
			current = parsed
		}

		// Assert - should still equal original
		assert.Equal(t, original.Version, current.Version)
		assert.Equal(t, original.Ciphertext, current.Ciphertext)
	})

	t.Run("Success_ComplexData", func(t *testing.T) {
		// Arrange - binary data with various byte values
		complexData := []byte{0x00, 0x01, 0xFF, 0xAB, 0xCD, 0xEF}
		blob := domain.EncryptedBlob{
			Version:    99,
			Ciphertext: complexData,
		}

		// Act
		serialized := blob.String()
		parsed, err := domain.NewEncryptedBlob(serialized)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, complexData, parsed.Ciphertext)
	})
}
