package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransitKey represents a versioned encryption key for transit encryption operations.
// Supports key rotation by maintaining multiple versions with the same name. The active
// version (highest number) is used for encryption while older versions remain available
// for decryption. Soft deletion via DeletedAt field preserves keys for historical decryption.
type TransitKey struct {
	ID        uuid.UUID  // Unique identifier for this specific transit key version
	Name      string     // Human-readable name (shared across all versions of this key)
	Version   uint       // Key version number (increments with rotation, starts at 1)
	DekID     uuid.UUID  // Reference to the Data Encryption Key used to encrypt this transit key
	CreatedAt time.Time  // Timestamp when this key version was created (UTC)
	DeletedAt *time.Time // Soft deletion timestamp (nil if active, set when deleted)
}

// Validate checks if the transit key contains valid data.
// Returns an error if any field violates domain constraints.
func (tk *TransitKey) Validate() error {
	if tk.Name == "" {
		return errors.New("transit key name cannot be empty")
	}

	if len(tk.Name) > MaxTransitKeyNameLength {
		return fmt.Errorf("transit key name exceeds maximum length of %d characters", MaxTransitKeyNameLength)
	}

	if tk.Version == 0 {
		return errors.New("transit key version must be greater than 0")
	}

	if tk.DekID == uuid.Nil {
		return errors.New("transit key must have a valid DEK ID")
	}

	if tk.CreatedAt.IsZero() {
		return errors.New("transit key must have a valid created_at timestamp")
	}

	return nil
}
