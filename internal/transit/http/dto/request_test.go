package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

func TestCreateTransitKeyRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest_AESGCM", func(t *testing.T) {
		req := CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "aes-gcm",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_ChaCha20", func(t *testing.T) {
		req := CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "chacha20-poly1305",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		req := CreateTransitKeyRequest{
			Name:      "",
			Algorithm: "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingAlgorithm", func(t *testing.T) {
		req := CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		req := CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "invalid-algorithm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_NameTooLong", func(t *testing.T) {
		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'a'
		}

		req := CreateTransitKeyRequest{
			Name:      string(longName),
			Algorithm: "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestRotateTransitKeyRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest_AESGCM", func(t *testing.T) {
		req := RotateTransitKeyRequest{
			Algorithm: "aes-gcm",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_ChaCha20", func(t *testing.T) {
		req := RotateTransitKeyRequest{
			Algorithm: "chacha20-poly1305",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingAlgorithm", func(t *testing.T) {
		req := RotateTransitKeyRequest{
			Algorithm: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		req := RotateTransitKeyRequest{
			Algorithm: "invalid-algorithm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestEncryptRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := EncryptRequest{
			Plaintext: []byte("my secret data"),
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyPlaintext", func(t *testing.T) {
		req := EncryptRequest{
			Plaintext: []byte{},
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_NilPlaintext", func(t *testing.T) {
		req := EncryptRequest{
			Plaintext: nil,
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestDecryptRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := DecryptRequest{
			Ciphertext: "1:ZW5jcnlwdGVkLWRhdGE=",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyCiphertext", func(t *testing.T) {
		req := DecryptRequest{
			Ciphertext: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankCiphertext", func(t *testing.T) {
		req := DecryptRequest{
			Ciphertext: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestParseAlgorithm(t *testing.T) {
	t.Run("Success_AESGCM", func(t *testing.T) {
		alg, err := ParseAlgorithm("aes-gcm")
		assert.NoError(t, err)
		assert.Equal(t, cryptoDomain.AESGCM, alg)
	})

	t.Run("Success_ChaCha20", func(t *testing.T) {
		alg, err := ParseAlgorithm("chacha20-poly1305")
		assert.NoError(t, err)
		assert.Equal(t, cryptoDomain.ChaCha20, alg)
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		alg, err := ParseAlgorithm("invalid")
		assert.Error(t, err)
		assert.Empty(t, alg)
		assert.Contains(t, err.Error(), "invalid algorithm")
	})

	t.Run("Error_EmptyString", func(t *testing.T) {
		alg, err := ParseAlgorithm("")
		assert.Error(t, err)
		assert.Empty(t, alg)
	})
}
