package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog records authorization decisions for compliance and security monitoring.
// Captures client identity, requested resource path, required capability, and metadata.
// Used to track access patterns and investigate security incidents.
//
// Cryptographic Integrity: All audit logs are signed with HMAC-SHA256 using KEK-derived
// signing keys to detect tampering (PCI DSS Requirement 10.2.2). The Signature field
// contains the 32-byte HMAC, KekID references the KEK used for signing, and IsSigned
// distinguishes signed logs from legacy unsigned logs created before the feature.
type AuditLog struct {
	ID         uuid.UUID
	RequestID  uuid.UUID
	ClientID   uuid.UUID
	Capability Capability
	Path       string
	Metadata   map[string]any
	Signature  []byte     // HMAC-SHA256 signature (32 bytes) for tamper detection
	KekID      *uuid.UUID // KEK used for signing (NULL for legacy unsigned logs)
	IsSigned   bool       // True if signed, false for legacy logs
	CreatedAt  time.Time
}

// HasValidSignature checks if the audit log has complete signature data.
// Returns true only if the log is marked as signed, has a KEK ID, and contains
// a 32-byte HMAC signature.
func (a *AuditLog) HasValidSignature() bool {
	return a.IsSigned && a.KekID != nil && len(a.Signature) == 32
}

// IsLegacy returns true if this is an unsigned legacy audit log created before
// cryptographic integrity was implemented. Legacy logs have no signature, no KEK ID,
// and are marked as unsigned.
func (a *AuditLog) IsLegacy() bool {
	return !a.IsSigned && a.KekID == nil && len(a.Signature) == 0
}
