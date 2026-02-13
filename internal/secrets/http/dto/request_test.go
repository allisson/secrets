package dto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdateSecretRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString([]byte("my-secret-value")),
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_LargeValue", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString(make([]byte, 10000)), // 10KB value
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_BinaryData", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		req := CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString(binaryData),
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyValue", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: "",
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value")
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: "not-valid-base64!@#$%",
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base64")
	})

	t.Run("Error_EmptyString", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{}

		err := req.Validate()
		assert.Error(t, err)
	})
}
