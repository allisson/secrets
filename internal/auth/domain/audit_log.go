package domain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditLog records authorization decisions for compliance and security monitoring.
// Captures client identity, requested resource path, required capability, and metadata.
// Used to track access patterns and investigate security incidents.
//
// Cryptographic Integrity: All audit logs are signed with HMAC-SHA256 using KEK-derived
// signing keys to detect tampering. The Signature field
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

// Canonical returns the deterministic byte representation of this audit log used
// for HMAC signing. Format: RequestID || ClientID || capability (length-prefixed) ||
// path (length-prefixed) || metadata JSON (length-prefixed) ||
// created_at as Unix nanoseconds (8 bytes, big-endian).
func (a *AuditLog) Canonical() ([]byte, error) {
	buf := make([]byte, 0, 1024)
	buf = append(buf, a.RequestID[:]...)
	buf = append(buf, a.ClientID[:]...)
	buf = appendLengthPrefixed(buf, []byte(string(a.Capability)))
	buf = appendLengthPrefixed(buf, []byte(a.Path))
	if a.Metadata != nil {
		metadataBytes, err := json.Marshal(a.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		buf = appendLengthPrefixed(buf, metadataBytes)
	} else {
		buf = appendLengthPrefixed(buf, nil)
	}
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(a.CreatedAt.UnixNano()))
	buf = append(buf, timeBytes...)
	return buf, nil
}

// appendLengthPrefixed adds a 4-byte big-endian length prefix followed by data.
func appendLengthPrefixed(buf, data []byte) []byte {
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
