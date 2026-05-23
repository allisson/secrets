package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

type auditSigner struct{}

// NewAuditSigner creates a new HMAC-based audit log signer using HKDF-SHA256
// for key derivation and HMAC-SHA256 for signature generation.
func NewAuditSigner() AuditSigner {
	return &auditSigner{}
}

// deriveSigningKey uses HKDF-SHA256 to derive a 32-byte signing key from KEK.
// Separates encryption key usage from signing key usage (cryptographic best practice).
// Info parameter: "audit-log-signing-v1" (versioned for future algorithm changes).
func (a *auditSigner) deriveSigningKey(kekKey []byte) ([]byte, error) {
	info := []byte("audit-log-signing-v1")
	hash := sha256.New
	hkdf := hkdf.New(hash, kekKey, nil, info)

	signingKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdf, signingKey); err != nil {
		return nil, err
	}

	return signingKey, nil
}

// Sign generates HMAC-SHA256 signature for the audit log.
// Returns 32-byte signature or error if signing fails.
func (a *auditSigner) Sign(kekKey []byte, log *authDomain.AuditLog) ([]byte, error) {
	signingKey, err := a.deriveSigningKey(kekKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive signing key: %w", err)
	}
	defer zero(signingKey) // Clear derived key from memory

	canonical, err := log.Canonical()
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize log: %w", err)
	}

	mac := hmac.New(sha256.New, signingKey)
	mac.Write(canonical)
	signature := mac.Sum(nil)

	return signature, nil
}

// Verify checks if the audit log signature is valid.
// Returns nil if valid, ErrSignatureInvalid if tampered or invalid.
func (a *auditSigner) Verify(kekKey []byte, log *authDomain.AuditLog) error {
	expectedSig, err := a.Sign(kekKey, log)
	if err != nil {
		return fmt.Errorf("failed to compute expected signature: %w", err)
	}

	if !hmac.Equal(log.Signature, expectedSig) {
		return authDomain.ErrSignatureInvalid
	}

	return nil
}

// zero overwrites sensitive data in memory with zeros.
// Prevents key material from lingering in memory after use.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
