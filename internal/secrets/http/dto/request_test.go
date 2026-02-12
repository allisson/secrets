package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdateSecretRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: []byte("my-secret-value"),
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Success_LargeValue", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: make([]byte, 10000), // 10KB value
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyValue", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: []byte{},
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value")
	})

	t.Run("Error_NilValue", func(t *testing.T) {
		req := CreateOrUpdateSecretRequest{
			Value: nil,
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}
