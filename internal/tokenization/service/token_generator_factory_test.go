package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

func TestNewTokenGenerator(t *testing.T) {
	tests := []struct {
		name         string
		formatType   tokenizationDomain.FormatType
		expectError  bool
		expectedType string
	}{
		{
			name:         "Success_UUID",
			formatType:   tokenizationDomain.FormatUUID,
			expectError:  false,
			expectedType: "*service.uuidGenerator",
		},
		{
			name:         "Success_Numeric",
			formatType:   tokenizationDomain.FormatNumeric,
			expectError:  false,
			expectedType: "*service.numericGenerator",
		},
		{
			name:         "Success_LuhnPreserving",
			formatType:   tokenizationDomain.FormatLuhnPreserving,
			expectError:  false,
			expectedType: "*service.luhnGenerator",
		},
		{
			name:         "Success_Alphanumeric",
			formatType:   tokenizationDomain.FormatAlphanumeric,
			expectError:  false,
			expectedType: "*service.alphanumericGenerator",
		},
		{
			name:        "Error_InvalidFormatType",
			formatType:  tokenizationDomain.FormatType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewTokenGenerator(tt.formatType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, gen)
				assert.ErrorIs(t, err, tokenizationDomain.ErrInvalidFormatType)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gen)
			}
		})
	}
}

func TestNewTokenGenerator_FunctionalTest(t *testing.T) {
	// Test that each generator can actually generate tokens
	formatTypes := []tokenizationDomain.FormatType{
		tokenizationDomain.FormatUUID,
		tokenizationDomain.FormatNumeric,
		tokenizationDomain.FormatLuhnPreserving,
		tokenizationDomain.FormatAlphanumeric,
	}

	for _, formatType := range formatTypes {
		t.Run("Generate_"+formatType.String(), func(t *testing.T) {
			gen, err := NewTokenGenerator(formatType)
			assert.NoError(t, err)
			assert.NotNil(t, gen)

			// Generate a token
			length := 16
			if formatType == tokenizationDomain.FormatUUID {
				length = 0 // UUID ignores length
			}

			token, err := gen.Generate(length)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate the generated token
			err = gen.Validate(token)
			assert.NoError(t, err)
		})
	}
}
