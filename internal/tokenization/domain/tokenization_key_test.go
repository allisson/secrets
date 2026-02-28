package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTokenizationKey_Validate(t *testing.T) {
	validID := uuid.Must(uuid.NewV7())
	validDekID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()

	tests := []struct {
		name        string
		key         *TokenizationKey
		expectError bool
		expectedErr error
	}{
		{
			name: "Success_ValidKey",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         1,
				FormatType:      FormatUUID,
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       now,
				DeletedAt:       nil,
			},
			expectError: false,
		},
		{
			name: "Success_ValidKeyWithHighVersion",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         100,
				FormatType:      FormatNumeric,
				IsDeterministic: true,
				DekID:           validDekID,
				CreatedAt:       now,
				DeletedAt:       nil,
			},
			expectError: false,
		},
		{
			name: "Success_ValidKeyAllFormats",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         1,
				FormatType:      FormatLuhnPreserving,
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       now,
				DeletedAt:       nil,
			},
			expectError: false,
		},
		{
			name: "Error_EmptyName",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "",
				Version:         1,
				FormatType:      FormatUUID,
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       now,
			},
			expectError: true,
			expectedErr: ErrTokenizationKeyNameEmpty,
		},
		{
			name: "Error_ZeroVersion",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         0,
				FormatType:      FormatUUID,
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       now,
			},
			expectError: true,
			expectedErr: ErrTokenizationKeyVersionInvalid,
		},
		{
			name: "Error_InvalidFormatType",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         1,
				FormatType:      FormatType("invalid"),
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       now,
			},
			expectError: true,
			expectedErr: ErrInvalidFormatType,
		},
		{
			name: "Error_NilDekID",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         1,
				FormatType:      FormatUUID,
				IsDeterministic: false,
				DekID:           uuid.Nil,
				CreatedAt:       now,
			},
			expectError: true,
			expectedErr: ErrTokenizationKeyDekIDInvalid,
		},
		{
			name: "Error_ZeroCreatedAt",
			key: &TokenizationKey{
				ID:              validID,
				Name:            "test-key",
				Version:         1,
				FormatType:      FormatUUID,
				IsDeterministic: false,
				DekID:           validDekID,
				CreatedAt:       time.Time{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.key.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenizationKey_Fields(t *testing.T) {
	t.Run("Success_AllFieldsSet", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()
		deletedAt := now.Add(24 * time.Hour)

		key := &TokenizationKey{
			ID:              id,
			Name:            "payment-cards",
			Version:         5,
			FormatType:      FormatLuhnPreserving,
			IsDeterministic: true,
			DekID:           dekID,
			CreatedAt:       now,
			DeletedAt:       &deletedAt,
		}

		assert.Equal(t, id, key.ID)
		assert.Equal(t, "payment-cards", key.Name)
		assert.Equal(t, uint(5), key.Version)
		assert.Equal(t, FormatLuhnPreserving, key.FormatType)
		assert.True(t, key.IsDeterministic)
		assert.Equal(t, dekID, key.DekID)
		assert.Equal(t, now, key.CreatedAt)
		assert.NotNil(t, key.DeletedAt)
		assert.Equal(t, deletedAt, *key.DeletedAt)
	})

	t.Run("Success_DeletedAtNil", func(t *testing.T) {
		key := &TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            "test-key",
			Version:         1,
			FormatType:      FormatUUID,
			IsDeterministic: false,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       time.Now().UTC(),
			DeletedAt:       nil,
		}

		assert.Nil(t, key.DeletedAt)
	})
}
