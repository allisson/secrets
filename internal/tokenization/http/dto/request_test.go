package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

func TestCreateTokenizationKeyRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest_UUID", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_Numeric", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "numeric-key",
			FormatType:      "numeric",
			IsDeterministic: true,
			Algorithm:       "chacha20-poly1305",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_LuhnPreserving", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "luhn-key",
			FormatType:      "luhn-preserving",
			IsDeterministic: true,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_Alphanumeric", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "alpha-key",
			FormatType:      "alphanumeric",
			IsDeterministic: false,
			Algorithm:       "chacha20-poly1305",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankName", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "   ",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingFormatType", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "invalid-format",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingAlgorithm", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		req := CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "invalid-algorithm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestRotateTokenizationKeyRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest_UUID", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_Numeric", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "numeric",
			IsDeterministic: true,
			Algorithm:       "chacha20-poly1305",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingFormatType", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "invalid-format",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingAlgorithm", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		req := RotateTokenizationKeyRequest{
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "invalid-algorithm",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestTokenizeRequest_Validate(t *testing.T) {
	ttl := 3600

	t.Run("Success_ValidRequest_WithTTL", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			Metadata:  map[string]any{"key": "value"},
			TTL:       &ttl,
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_WithoutTTL", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "SGVsbG8gV29ybGQ=",
			Metadata:  map[string]any{"key": "value"},
			TTL:       nil,
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_ValidRequest_WithoutMetadata", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "SGVsbG8gV29ybGQ=",
			Metadata:  nil,
			TTL:       &ttl,
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingPlaintext", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "",
			Metadata:  map[string]any{"key": "value"},
			TTL:       &ttl,
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankPlaintext", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "   ",
			Metadata:  map[string]any{"key": "value"},
			TTL:       &ttl,
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		req := TokenizeRequest{
			Plaintext: "not-valid-base64!!!",
			Metadata:  map[string]any{"key": "value"},
			TTL:       &ttl,
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_NegativeTTL", func(t *testing.T) {
		negativeTTL := -1
		req := TokenizeRequest{
			Plaintext: "SGVsbG8gV29ybGQ=",
			Metadata:  map[string]any{"key": "value"},
			TTL:       &negativeTTL,
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestDetokenizeRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := DetokenizeRequest{
			Token: "tok_1234567890",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		req := DetokenizeRequest{
			Token: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankToken", func(t *testing.T) {
		req := DetokenizeRequest{
			Token: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestValidateTokenRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := ValidateTokenRequest{
			Token: "tok_1234567890",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		req := ValidateTokenRequest{
			Token: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankToken", func(t *testing.T) {
		req := ValidateTokenRequest{
			Token: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestRevokeTokenRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := RevokeTokenRequest{
			Token: "tok_1234567890",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingToken", func(t *testing.T) {
		req := RevokeTokenRequest{
			Token: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankToken", func(t *testing.T) {
		req := RevokeTokenRequest{
			Token: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestParseFormatType(t *testing.T) {
	t.Run("Success_UUID", func(t *testing.T) {
		formatType, err := ParseFormatType("uuid")
		assert.NoError(t, err)
		assert.Equal(t, tokenizationDomain.FormatUUID, formatType)
	})

	t.Run("Success_Numeric", func(t *testing.T) {
		formatType, err := ParseFormatType("numeric")
		assert.NoError(t, err)
		assert.Equal(t, tokenizationDomain.FormatNumeric, formatType)
	})

	t.Run("Success_LuhnPreserving", func(t *testing.T) {
		formatType, err := ParseFormatType("luhn-preserving")
		assert.NoError(t, err)
		assert.Equal(t, tokenizationDomain.FormatLuhnPreserving, formatType)
	})

	t.Run("Success_Alphanumeric", func(t *testing.T) {
		formatType, err := ParseFormatType("alphanumeric")
		assert.NoError(t, err)
		assert.Equal(t, tokenizationDomain.FormatAlphanumeric, formatType)
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		_, err := ParseFormatType("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format type")
	})

	t.Run("Error_EmptyFormatType", func(t *testing.T) {
		_, err := ParseFormatType("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format type")
	})

	t.Run("Error_CaseSensitive", func(t *testing.T) {
		_, err := ParseFormatType("UUID")
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
		_, err := ParseAlgorithm("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid algorithm")
	})

	t.Run("Error_EmptyAlgorithm", func(t *testing.T) {
		_, err := ParseAlgorithm("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid algorithm")
	})

	t.Run("Error_CaseSensitive", func(t *testing.T) {
		_, err := ParseAlgorithm("AES-GCM")
		assert.Error(t, err)
	})
}

func TestValidateFormatType(t *testing.T) {
	t.Run("Success_ValidString", func(t *testing.T) {
		err := validateFormatType("uuid")
		assert.NoError(t, err)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		err := validateFormatType(123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a string")
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		err := validateFormatType("invalid")
		assert.Error(t, err)
	})
}

func TestValidateAlgorithm(t *testing.T) {
	t.Run("Success_ValidString", func(t *testing.T) {
		err := validateAlgorithm("aes-gcm")
		assert.NoError(t, err)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		err := validateAlgorithm(123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a string")
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		err := validateAlgorithm("invalid")
		assert.Error(t, err)
	})
}
