package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenizationKey represents a versioned tokenization key configuration.
// Each key defines the token format and deterministic behavior for tokenization operations.
type TokenizationKey struct {
	// ID is the unique identifier for this specific key version.
	ID uuid.UUID

	// Name is the logical name for this tokenization key (e.g., "payment-cards", "ssn").
	// Multiple versions can share the same name.
	Name string

	// Version is the key version number, starting at 1 and incremented on rotation.
	// Higher versions are preferred for tokenization; all versions support detokenization.
	Version uint

	// FormatType defines the token format (UUID, Numeric, Luhn-Preserving, Alphanumeric).
	FormatType FormatType

	// IsDeterministic indicates whether the same plaintext always produces the same token.
	// When true, enables efficient duplicate detection; when false, provides better privacy.
	IsDeterministic bool

	// DekID is the reference to the Data Encryption Key used to encrypt values for this version.
	DekID uuid.UUID

	// CreatedAt is the timestamp when this key version was created (UTC).
	CreatedAt time.Time

	// DeletedAt is the timestamp when this key was soft-deleted (nil if active).
	DeletedAt *time.Time
}

// Validate checks if the TokenizationKey has valid field values.
// Returns an error if any field constraint is violated.
func (tk *TokenizationKey) Validate() error {
	if tk.Name == "" {
		return ErrTokenizationKeyNameEmpty
	}
	if tk.Version == 0 {
		return ErrTokenizationKeyVersionInvalid
	}
	if err := tk.FormatType.Validate(); err != nil {
		return ErrInvalidFormatType
	}
	if tk.DekID == uuid.Nil {
		return ErrTokenizationKeyDekIDInvalid
	}
	if tk.CreatedAt.IsZero() {
		return ErrInvalidFormatType // Using existing error for now
	}
	return nil
}
