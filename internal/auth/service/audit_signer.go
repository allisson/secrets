package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
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

// canonicalizeLog converts audit log to canonical byte representation for signing.
// Format: request_id || client_id || capability || path || metadata || created_at
// Uses length-prefixed encoding for variable-length fields to prevent ambiguity.
func (a *auditSigner) canonicalizeLog(log *authDomain.AuditLog) ([]byte, error) {
	// Estimate capacity to reduce allocations (typical log ~1KB)
	buf := make([]byte, 0, 1024)

	// Append UUIDs (16 bytes each)
	buf = append(buf, log.RequestID[:]...)
	buf = append(buf, log.ClientID[:]...)

	// Append capability string (length-prefixed for safety)
	buf = appendLengthPrefixed(buf, []byte(string(log.Capability)))

	// Append path string (length-prefixed)
	buf = appendLengthPrefixed(buf, []byte(log.Path))

	// Append metadata JSON (length-prefixed, deterministic serialization)
	if log.Metadata != nil {
		// Serialize metadata to JSON for deterministic representation
		metadataBytes, err := json.Marshal(log.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		buf = appendLengthPrefixed(buf, metadataBytes)
	} else {
		// Empty metadata = 0 length prefix
		buf = appendLengthPrefixed(buf, nil)
	}

	// Append timestamp (Unix nano for precision)
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(log.CreatedAt.UnixNano()))
	buf = append(buf, timeBytes...)

	return buf, nil
}

// appendLengthPrefixed adds a 4-byte big-endian length prefix followed by data.
// Format: [length (4 bytes)] + [data (length bytes)]
// Panics if data length exceeds uint32 max (4GB) to prevent integer overflow.
func appendLengthPrefixed(buf []byte, data []byte) []byte {
	dataLen := len(data)
	if dataLen > 0xFFFFFFFF {
		panic("data length exceeds uint32 max (4GB)")
	}
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(dataLen))
	buf = append(buf, length...)
	buf = append(buf, data...)
	return buf
}

// Sign generates HMAC-SHA256 signature for the audit log.
// Returns 32-byte signature or error if signing fails.
func (a *auditSigner) Sign(kekKey []byte, log *authDomain.AuditLog) ([]byte, error) {
	signingKey, err := a.deriveSigningKey(kekKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive signing key: %w", err)
	}
	defer zero(signingKey) // Clear derived key from memory

	canonical, err := a.canonicalizeLog(log)
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
