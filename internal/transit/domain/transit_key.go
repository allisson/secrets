package domain

import (
	"time"

	"github.com/google/uuid"
)

// TransitKey represents a versioned encryption key for transit encryption operations.
// Supports key rotation by maintaining multiple versions with the same name. The active
// version (highest number) is used for encryption while older versions remain available
// for decryption. Soft deletion via DeletedAt field preserves keys for historical decryption.
type TransitKey struct {
	ID        uuid.UUID
	Name      string
	Version   uint
	DekID     uuid.UUID
	CreatedAt time.Time
	DeletedAt *time.Time
}
