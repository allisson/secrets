package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apperrors "github.com/allisson/secrets/internal/errors"
)

func TestErrors_Wrapping(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrTokenizationKeyNotFound",
			err:         ErrTokenizationKeyNotFound,
			expectedMsg: "tokenization key not found",
		},
		{
			name:        "ErrTokenizationKeyAlreadyExists",
			err:         ErrTokenizationKeyAlreadyExists,
			expectedMsg: "tokenization key already exists",
		},
		{
			name:        "ErrTokenNotFound",
			err:         ErrTokenNotFound,
			expectedMsg: "token not found",
		},
		{
			name:        "ErrTokenExpired",
			err:         ErrTokenExpired,
			expectedMsg: "token has expired",
		},
		{
			name:        "ErrTokenRevoked",
			err:         ErrTokenRevoked,
			expectedMsg: "token has been revoked",
		},
		{
			name:        "ErrInvalidFormatType",
			err:         ErrInvalidFormatType,
			expectedMsg: "invalid format type",
		},
		{
			name:        "ErrInvalidTokenLength",
			err:         ErrInvalidTokenLength,
			expectedMsg: "invalid token length for format",
		},
		{
			name:        "ErrValueTooLong",
			err:         ErrValueTooLong,
			expectedMsg: "value exceeds maximum length",
		},
		{
			name:        "ErrPlaintextTooLarge",
			err:         ErrPlaintextTooLarge,
			expectedMsg: "plaintext exceeds maximum size of 64KB",
		},
		{
			name:        "ErrPlaintextEmpty",
			err:         ErrPlaintextEmpty,
			expectedMsg: "plaintext cannot be empty",
		},
		{
			name:        "ErrTokenLengthInvalid",
			err:         ErrTokenLengthInvalid,
			expectedMsg: "token length invalid for format type",
		},
		{
			name:        "ErrTokenizationKeyNameEmpty",
			err:         ErrTokenizationKeyNameEmpty,
			expectedMsg: "tokenization key name cannot be empty",
		},
		{
			name:        "ErrTokenizationKeyVersionInvalid",
			err:         ErrTokenizationKeyVersionInvalid,
			expectedMsg: "tokenization key version must be greater than 0",
		},
		{
			name:        "ErrTokenizationKeyDekIDInvalid",
			err:         ErrTokenizationKeyDekIDInvalid,
			expectedMsg: "tokenization key DEK ID cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.expectedMsg)
		})
	}
}

func TestErrors_Types(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType error
	}{
		{
			name:         "ErrTokenizationKeyNotFound_IsNotFound",
			err:          ErrTokenizationKeyNotFound,
			expectedType: apperrors.ErrNotFound,
		},
		{
			name:         "ErrTokenizationKeyAlreadyExists_IsConflict",
			err:          ErrTokenizationKeyAlreadyExists,
			expectedType: apperrors.ErrConflict,
		},
		{
			name:         "ErrTokenNotFound_IsNotFound",
			err:          ErrTokenNotFound,
			expectedType: apperrors.ErrNotFound,
		},
		{
			name:         "ErrTokenExpired_IsInvalidInput",
			err:          ErrTokenExpired,
			expectedType: apperrors.ErrInvalidInput,
		},
		{
			name:         "ErrTokenRevoked_IsInvalidInput",
			err:          ErrTokenRevoked,
			expectedType: apperrors.ErrInvalidInput,
		},
		{
			name:         "ErrInvalidFormatType_IsInvalidInput",
			err:          ErrInvalidFormatType,
			expectedType: apperrors.ErrInvalidInput,
		},
		{
			name:         "ErrPlaintextTooLarge_IsInvalidInput",
			err:          ErrPlaintextTooLarge,
			expectedType: apperrors.ErrInvalidInput,
		},
		{
			name:         "ErrPlaintextEmpty_IsInvalidInput",
			err:          ErrPlaintextEmpty,
			expectedType: apperrors.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, apperrors.Is(tt.err, tt.expectedType),
				"expected %v to be of type %v", tt.err, tt.expectedType)
		})
	}
}

func TestErrors_Distinct(t *testing.T) {
	// Verify that all errors are distinct
	errors := []error{
		ErrTokenizationKeyNotFound,
		ErrTokenizationKeyAlreadyExists,
		ErrTokenNotFound,
		ErrTokenExpired,
		ErrTokenRevoked,
		ErrInvalidFormatType,
		ErrInvalidTokenLength,
		ErrValueTooLong,
		ErrPlaintextTooLarge,
		ErrPlaintextEmpty,
		ErrTokenLengthInvalid,
		ErrTokenizationKeyNameEmpty,
		ErrTokenizationKeyVersionInvalid,
		ErrTokenizationKeyDekIDInvalid,
	}

	// Check each error against all others
	for i := 0; i < len(errors); i++ {
		for j := i + 1; j < len(errors); j++ {
			assert.NotEqual(t, errors[i].Error(), errors[j].Error(),
				"errors at index %d and %d have the same message", i, j)
		}
	}
}
