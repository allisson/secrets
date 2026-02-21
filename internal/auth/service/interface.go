// Package service provides technical services for authentication operations.
//
// This package implements reusable services for client secret generation, hashing,
// and validation using industry-standard cryptographic practices, as well as
// cryptographic signing for audit log integrity.
package service

import (
	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// SecretService defines operations for client secret generation and validation.
// Implementations must use cryptographically secure random generation and
// industry-standard hashing algorithms (e.g., bcrypt, argon2).
type SecretService interface {
	// GenerateSecret creates a new cryptographically secure random secret.
	// Returns both the plain text secret (to be shared with the client) and
	// the hashed version (to be stored in the database).
	//
	// The plain secret should be treated as sensitive data and only displayed
	// once to the client during creation.
	GenerateSecret() (plainSecret string, hashedSecret string, error error)

	// HashSecret hashes a plain text secret using a secure hashing algorithm.
	// Used when clients need to regenerate or update their secrets.
	HashSecret(plainSecret string) (hashedSecret string, error error)

	// CompareSecret compares a plain text secret against a hashed secret.
	// Returns true if the plain secret matches the hash, false otherwise.
	// This is constant-time to prevent timing attacks.
	CompareSecret(plainSecret string, hashedSecret string) bool
}

// TokenService defines operations for authentication token generation and hashing.
// Implementations must use cryptographically secure random generation and
// fast hashing algorithms suitable for short-lived tokens (e.g., SHA-256).
type TokenService interface {
	// GenerateToken creates a new cryptographically secure random token.
	// Returns both the plain text token (to be shared with the client) and
	// the hashed version (to be stored in the database).
	//
	// The plain token should be treated as sensitive data and only displayed
	// once to the client during token issuance.
	GenerateToken() (plainToken string, tokenHash string, error error)

	// HashToken hashes a plain text token using SHA-256.
	// Used for token validation by comparing hashes.
	HashToken(plainToken string) string
}

// AuditSigner provides cryptographic signing and verification for audit logs.
// Uses HMAC-SHA256 with KEK-derived signing keys (via HKDF) to detect tampering
// and ensure compliance with PCI DSS Requirement 10.2.2.
type AuditSigner interface {
	// Sign generates HMAC-SHA256 signature for audit log using KEK-derived key.
	// The signing key is derived from the KEK using HKDF-SHA256 with info parameter
	// "audit-log-signing-v1" to separate encryption and signing key usage.
	//
	// Returns 32-byte HMAC signature or error if signing fails. The kekKey parameter
	// must be a valid 32-byte KEK plaintext key.
	Sign(kekKey []byte, log *authDomain.AuditLog) ([]byte, error)

	// Verify checks HMAC-SHA256 signature for audit log using KEK-derived key.
	// Recomputes the expected signature and compares it with the log's signature
	// using constant-time comparison to prevent timing attacks.
	//
	// Returns nil if signature is valid, ErrSignatureInvalid if tampered or invalid.
	// The kekKey must be the same KEK used during signing (referenced by log.KekID).
	Verify(kekKey []byte, log *authDomain.AuditLog) error
}
