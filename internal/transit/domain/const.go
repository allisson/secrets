// Package domain defines core transit encryption domain models.
package domain

const (
	// MaxTransitKeyNameLength is the maximum allowed length for transit key names.
	// This limit aligns with database schema constraints (VARCHAR(255)) and prevents
	// excessively long identifiers that could impact performance or cause display issues.
	MaxTransitKeyNameLength = 255

	// AEADNonceSize is the nonce length (in bytes) used in the transit wire format.
	// AES-256-GCM and ChaCha20-Poly1305 both use 12-byte nonces; a future algorithm
	// with a different nonce size would require a transit wire-format version bump.
	AEADNonceSize = 12
)
